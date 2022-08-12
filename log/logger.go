/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gsyslog "github.com/hashicorp/go-syslog"
	"mosn.io/pkg/utils"
)

var (
	// error
	ErrReopenUnsupported = errors.New("reopen unsupported")

	remoteSyslogPrefixes = map[string]string{
		"syslog+tcp://": "tcp",
		"syslog+udp://": "udp",
		"syslog://":     "udp",
	}
)

// Logger is a basic sync logger implement, contains unexported fields
// The Logger Function contains:
// Print(buffer LogBuffer, discard bool) error
// Printf(format string, args ...interface{})
// Println(args ...interface{})
// Fatalf(format string, args ...interface{})
// Fatal(args ...interface{})
// Fatalln(args ...interface{})
// Close() error
// Reopen() error
// Toggle(disable bool)
type Logger struct {
	// output is the log's output path
	// if output is empty(""), it is equals to stderr
	output string
	// writer writes the log, created by output
	writer io.Writer
	// roller rotates the log, if the output is a file path
	roller *Roller
	// disable presents the logger state. if disable is true, the logger will write nothing
	// the default value is false
	disable bool
	// implementation elements
	create          time.Time
	once            sync.Once
	rollerUpdate    chan bool
	stopRotate      chan struct{}
	reopenChan      chan struct{}
	closeChan       chan struct{}
	writeBufferChan chan LogBuffer
}

type LoggerInfo struct {
	LogRoller  *Roller
	FileName   string
	CreateTime time.Time
}

// loggers keeps all Logger we created
// key is output, same output reference the same Logger
var loggers sync.Map // map[string]*Logger

func ToggleLogger(p string, disable bool) bool {
	// find Logger
	if lg, ok := loggers.Load(p); ok {
		lg.(*Logger).Toggle(disable)
		return true
	}
	return false
}

// Reopen all logger
func Reopen() (err error) {
	loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		err = logger.Reopen()
		if err != nil {
			return false
		}
		return true
	})
	return
}

// CloseAll logger
func CloseAll() (err error) {
	loggers.Range(func(key, value interface{}) bool {
		logger := value.(*Logger)
		err = logger.Close()
		if err != nil {
			return false
		}
		return true
	})
	return
}

// ClearAll created logger, just for test
func ClearAll() {
	loggers = sync.Map{}
}

// defaultBufferSize indicates the amount that can be cached in a logger
const defaultBufferSize = 500

func GetOrCreateLogger(output string, roller *Roller) (*Logger, error) {
	if lg, ok := loggers.Load(output); ok {
		return lg.(*Logger), nil
	}

	notify := make(chan bool, 1)
	if roller == nil {
		roller = &defaultRoller
		// use defaultRoller, add a notify
		registeNofify(notify)
	}

	if roller.Handler == nil {
		roller.Handler = rollerHandler
	}

	lg := &Logger{
		output:          output,
		roller:          roller,
		writeBufferChan: make(chan LogBuffer, defaultBufferSize),
		reopenChan:      make(chan struct{}),
		closeChan:       make(chan struct{}),
		stopRotate:      make(chan struct{}),
		rollerUpdate:    notify,
		// writer and create will be setted in start()
	}
	err := lg.start()
	if err == nil { // only keeps start success logger
		loggers.Store(output, lg)
	}
	return lg, err
}

func (l *Logger) start() error {
	switch l.output {
	case "", "stderr", "/dev/stderr":
		l.writer = os.Stderr
	case "stdout", "/dev/stdout":
		l.writer = os.Stdout
	case "syslog":
		writer, err := gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "mosn")
		if err != nil {
			return err
		}
		l.writer = writer
	default:
		if address := parseSyslogAddress(l.output); address != nil {
			writer, err := gsyslog.DialLogger(address.network, address.address, gsyslog.LOG_ERR, "LOCAL0", "mosn")
			if err != nil {
				return err
			}
			l.writer = writer
		} else { // write to file
			if err := os.MkdirAll(filepath.Dir(l.output), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(l.output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				return err
			}
			if l.roller.MaxTime == 0 {
				file.Close()
				l.roller.Filename = l.output
				l.writer = l.roller.GetLogWriter()
			} else {
				// time.Now() faster than reported timestamps from filesystem (https://github.com/golang/go/issues/33510)
				// init logger
				if l.create.IsZero() {
					stat, err := file.Stat()
					if err != nil {
						return err
					}
					l.create = stat.ModTime()
				} else {
					l.create = time.Now()
				}
				l.writer = file
				l.mill()
				l.once.Do(l.startRotate) // start rotate, only once
			}
		}
	}
	// TODO: recover?
	go l.handler()
	return nil
}

func (l *Logger) handler() {
	defer func() {
		if p := recover(); p != nil {
			debug.PrintStack()
			// TODO: recover?
			go l.handler()
		}
	}()
	for {
		select {
		case <-l.reopenChan:
			// reopen is used for roller
			err := l.reopen()
			if err == nil {
				return
			}
			fmt.Fprintf(os.Stderr, "%s reopen failed : %v\n", l.output, err)
		case <-l.closeChan:
			// flush all buffers before close
			// make sure all logs are outputed
			// a closed logger can not write anymore
			for {
				select {
				case buf := <-l.writeBufferChan:
					l.Write(buf.Bytes())
					PutLogBuffer(buf)
				default:
					l.stop()
					close(l.stopRotate)
					return
				}
			}
		case buf := <-l.writeBufferChan:

			l.Write(buf.Bytes())
			PutLogBuffer(buf)
		}
	}
}

func (l *Logger) stop() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return nil
	}

	if closer, ok := l.writer.(io.WriteCloser); ok {
		err := closer.Close()
		return err
	}

	return nil
}

func (l *Logger) reopen() error {
	if l.writer == os.Stdout || l.writer == os.Stderr {
		return ErrReopenUnsupported
	}
	if closer, ok := l.writer.(io.WriteCloser); ok {
		// ignore the close error, always try to start a new file
		// record the error info
		if err := closer.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "logger %s close error when restart, error: %v", l.output, err)
		}
		return l.start()
	}
	return ErrReopenUnsupported
}

var ErrChanFull = errors.New("channel is full")

// Print writes the final buffere to the buffer chan
// if discard is true and the buffer is full, returns an error
// If a LogBuffer needs to call Print N(N>1) times, the LogBuffer.Count(N-1) should be called
// or call LogBuffer.Count(1) N-1 times.
// If the N is 1, LogBuffer.Count should not be called.
func (l *Logger) Print(buf LogBuffer, discard bool) error {
	if l.disable {
		// free the buf
		PutLogBuffer(buf)
		return nil
	}
	select {
	case l.writeBufferChan <- buf:
	default:
		// todo: configurable
		if discard {
			return ErrChanFull
		} else {
			l.writeBufferChan <- buf
		}
	}
	return nil
}

func (l *Logger) Println(args ...interface{}) {
	if l.disable {
		return
	}
	s := fmt.Sprintln(args...)
	buf := GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	if l.disable {
		return
	}
	s := fmt.Sprintf(format, args...)
	buf := GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	l.Print(buf, true)
}

// Fatal cannot be disabled
func (l *Logger) Fatalf(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	buf := GetLogBuffer(len(s))
	buf.WriteString(s)
	buf.WriteString("\n")
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) Fatal(args ...interface{}) {
	s := fmt.Sprint(args...)
	buf := GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) Fatalln(args ...interface{}) {
	s := fmt.Sprintln(args...)
	buf := GetLogBuffer(len(s))
	buf.WriteString(s)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf.WriteString("\n")
	}
	buf.WriteTo(l.writer)
	os.Exit(1)
}

func (l *Logger) calculateInterval(now time.Time) time.Duration {
	// caculate the next time need to rotate
	_, localOffset := now.Zone()
	return time.Duration(l.roller.MaxTime-(now.Unix()+int64(localOffset))%l.roller.MaxTime) * time.Second
}

func (l *Logger) startRotate() {
	utils.GoWithRecover(func() {
		// roller not by time
		if l.create.IsZero() {
			return
		}
		var interval time.Duration
		// check need to rotate right now
		now := time.Now()
		if now.Sub(l.create) > time.Duration(l.roller.MaxTime)*time.Second {
			interval = 0
		} else {
			interval = l.calculateInterval(now)
		}
		doRotate(l, interval)
	}, func(r interface{}) {
		l.startRotate()
	})
}

var doRotate func(l *Logger, interval time.Duration) = doRotateFunc

func doRotateFunc(l *Logger, interval time.Duration) {
	timer := time.NewTimer(interval)
	for {
		select {
		case <-l.stopRotate:
			return
		case <-l.rollerUpdate:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			now := time.Now()
			interval = l.calculateInterval(now)
		case <-timer.C:
			now := time.Now()
			info := LoggerInfo{FileName: l.output, CreateTime: l.create}
			info.LogRoller = l.roller
			l.roller.Handler(&info)
			l.create = now
			go l.Reopen()

			if interval == 0 { // recalculate interval
				interval = l.calculateInterval(now)
			} else {
				interval = time.Duration(l.roller.MaxTime) * time.Second
			}
		}
		timer.Reset(interval)
	}
}

func (l *Logger) Write(p []byte) (n int, err error) {
	return l.writer.Write(p)
}

func (l *Logger) Close() error {
	l.closeChan <- struct{}{}
	return nil
}

func (l *Logger) Reopen() error {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()
	l.reopenChan <- struct{}{}
	return nil
}

func (l *Logger) Toggle(disable bool) {
	l.disable = disable
}

func (l *Logger) Disable() bool {
	return l.disable
}

// syslogAddress
type syslogAddress struct {
	network string
	address string
}

func parseSyslogAddress(location string) *syslogAddress {
	for prefix, network := range remoteSyslogPrefixes {
		if strings.HasPrefix(location, prefix) {
			return &syslogAddress{
				network: network,
				address: strings.TrimPrefix(location, prefix),
			}
		}
	}

	return nil
}

const (
	compressSuffix = ".gz"
)

// millRunOnce performs compression and removal of stale log files.
// Log files are compressed if enabled via configuration and old log
// files are removed, keeping at most l.MaxBackups files, as long as
// none of them are older than MaxAge.
func (l *Logger) millRunOnce() error {
	files, err := l.oldLogFiles()
	if err != nil {
		return err
	}

	compress, remove := l.screeningCompressFile(files)

	for _, f := range remove {
		_ = os.Remove(filepath.Join(l.dir(), f.FileName))
	}
	var wg sync.WaitGroup
	for _, f := range compress {
		var fnCompress, fileName string
		fileName = f.FileName
		wg.Add(1)
		fnCompress, err = l.findCompressFile(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger %s findCompressFile , error: %v", l.output, err)
			return err
		}
		go func(fnCompress, fileName string, wg *sync.WaitGroup) {
			wg.Done()
			err = l.compressLogFile(fileName, fnCompress)
			if err != nil {
				fmt.Fprintf(os.Stderr, "logger %s compressLogFile , error: %v", l.output, err)
			}
		}(fnCompress, fileName, &wg)
	}
	wg.Wait()
	return err
}

func (l *Logger) screeningCompressFile(files []LoggerInfo) (compress, remove []LoggerInfo) {
	resFiles, removeByMaxAge := l.screeningCompressFileByMaxAge(files)
	resFiles, remove = l.screeningCompressFileByMaxBackups(resFiles, removeByMaxAge)

	if l.roller.Compress {
		for i := range resFiles {
			if !strings.HasSuffix(resFiles[i].FileName, compressSuffix) {
				compress = append(compress, resFiles[i])
			}
		}
	}
	return
}

func (l *Logger) screeningCompressFileByMaxAge(files []LoggerInfo) (resFiles, remove []LoggerInfo) {
	if l.roller.MaxAge > 0 {
		diff := time.Duration(int64(maxRotateHour*time.Hour) * int64(l.roller.MaxAge))
		cutoff := time.Now().Add(-1 * diff)

		for i := range files {
			if files[i].CreateTime.Before(cutoff) {
				remove = append(remove, files[i])
			} else {
				resFiles = append(resFiles, files[i])
			}
		}
	} else {
		resFiles = files
	}
	return
}

func (l *Logger) screeningCompressFileByMaxBackups(files, remove []LoggerInfo) (resFiles, resRemove []LoggerInfo) {
	if l.roller.MaxBackups > 0 && l.roller.MaxBackups < len(files) {
		preserved := make(map[string]bool)

		for i := range files {
			// Only count the uncompressed log file or the
			// compressed log file, not both.
			fn := files[i].FileName

			preserved[strings.TrimSuffix(fn, compressSuffix)] = true

			if len(preserved) > l.roller.MaxBackups {
				remove = append(remove, files[i])
			} else {
				resFiles = append(resFiles, files[i])
			}
		}
	} else {
		resFiles = files
	}
	resRemove = remove
	return
}

//findCompressFile Find the compressed file name based on the file name ，compressed file is not exist。
func (l *Logger) findCompressFile(fileName string) (string, error) {
	var (
		num      = 1
		statName = fileName
		err      error
	)

	for i := 0; i <= l.roller.MaxBackups; i++ {
		if _, err = os.Stat(l.dir() + statName + compressSuffix); os.IsNotExist(err) {
			return statName + compressSuffix, nil
		}
		statName = fileName + "." + strconv.Itoa(num)
		num++
	}
	return fileName, err
}

func (l *Logger) mill() {
	if l.roller.MaxBackups != defaultRotateKeep || l.roller.MaxAge != defaultRotateAge || l.roller.Compress {
		_ = l.millRunOnce()
	}
}

// oldLogFiles returns the list of backup log files stored in the same
// directory as the current log file, sorted by ModTime
func (l *Logger) oldLogFiles() ([]LoggerInfo, error) {
	files, err := ioutil.ReadDir(l.dir())
	if err != nil {
		return nil, err
	}
	logFiles := []LoggerInfo{}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if !strings.HasPrefix(f.Name(), filepath.Base(l.output)+".") {
			continue
		}
		logFiles = append(logFiles, LoggerInfo{l.roller, f.Name(), f.ModTime()})
	}
	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

// dir returns the directory for the current filename.
func (l *Logger) dir() string {
	return filepath.Dir(l.output)
}

// compressLogFile compresses the given log file, removing the
// uncompressed log file if successful.
func (l *Logger) compressLogFile(srcFile, dstFile string) error {
	f, err := os.Open(filepath.Join(l.dir(), filepath.Clean(srcFile)))
	if err != nil {
		return err
	}

	defer func() {
		_ = f.Close()
	}()

	gzf, err := os.OpenFile(filepath.Join(l.dir(), filepath.Clean(dstFile)), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	defer func() {
		_ = gzf.Close()
		if err != nil {
			_ = os.Remove(filepath.Join(l.dir(), filepath.Clean(dstFile)))
		}
	}()

	gz := gzip.NewWriter(gzf)

	if _, err = io.Copy(gz, f); err != nil {
		return err
	}

	if err = gz.Close(); err != nil {
		return err
	}

	return os.Remove(filepath.Join(l.dir(), filepath.Clean(srcFile)))
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []LoggerInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].CreateTime.After(b[j].CreateTime)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
