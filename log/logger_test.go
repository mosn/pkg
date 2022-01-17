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
	"io/ioutil"
	"math"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"mosn.io/pkg/buffer"
)

func TestLogPrintDiscard(t *testing.T) {
	l, err := GetOrCreateLogger("/tmp/mosn_bench/benchmark.log", nil)
	if err != nil {
		t.Fatal(err)
	}
	buf := GetLogBuffer(100)
	buf.WriteString("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog")
	l.Close()
	runtime.Gosched()
	// writeBufferChan is 1000
	// l.Printf is discard, non block
	for i := 0; i < 1001; i++ {
		l.Printf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
	lchan := make(chan struct{})
	go func() {
		// block
		l.Print(buf, false)
		lchan <- struct{}{}
	}()

	select {
	case <-lchan:
		t.Errorf("test Print diacard failed, should be block")
	case <-time.After(time.Second * 3):
	}
}

func TestLogPrintnull(t *testing.T) {
	logName := "/tmp/mosn_bench/printnull.log"
	os.Remove(logName)
	l, err := GetOrCreateLogger(logName, nil)
	if err != nil {
		t.Fatal(err)
	}
	buf := GetLogBuffer(0)
	buf.WriteString("testlog")
	l.Print(buf, false)
	buf = GetLogBuffer(0)
	buf.WriteString("")
	l.Print(buf, false)
	l.Close()
	time.Sleep(time.Second)
	f, _ := os.Open(logName)
	b := make([]byte, 1024)
	n, _ := f.Read(b)
	f.Close()

	if n != len("testlog") {
		t.Errorf("Printnull error")
	}
	if string(b[:n]) != "testlog" {
		t.Errorf("Printnull error")
	}
}

// force set rotate interval for test
// we use this for test file rotate action, not the rotate interval
func testRotate(l *Logger, interval time.Duration) {
	doRotateFunc(l, 10*time.Second)
}

func newLogBufferString(s string) LogBuffer {
	return LogBuffer{
		buffer.NewIoBufferString(s),
	}
}

func TestLogDefaultRollerTime(t *testing.T) {
	logName := "/tmp/mosn_bench/printdefaultroller.log"
	rollerName := logName + "." + time.Now().Format("2006-01-02_15")
	os.Remove(logName)
	os.Remove(rollerName)
	// replace rotate interval for test
	doRotate = testRotate
	defer func() {
		doRotate = doRotateFunc
	}()
	logger, err := GetOrCreateLogger(logName, &Roller{MaxTime: 10, Handler: rollerHandler})
	if err != nil {
		t.Fatal(err)
	}
	// 1111 will be rotated to rollerName
	logger.Print(newLogBufferString("1111111"), false)
	time.Sleep(11 * time.Second)
	// 2222 will be writed in logName
	logger.Print(newLogBufferString("2222222"), false)
	time.Sleep(1 * time.Second)
	logger.Close() // stop the rotate

	lines, err := readLines(logName)
	if err != nil {
		t.Fatalf("read %s error: %v", logName, err)
	}
	if len(lines) != 1 || lines[0] != "2222222" {
		t.Fatalf("read %s data: %v, not expected", logName, lines)
	}

	lines, err = readLines(rollerName)
	if err != nil {
		t.Fatalf("read %s error: %v", rollerName, err)
	}
	if len(lines) != 1 || lines[0] != "1111111" {
		t.Fatalf("read %s data: %v, not expected", rollerName, lines)
	}
}

func TestLogDefaultRollerAfterDelete(t *testing.T) {
	logName := "/tmp/log_roller_delete.log"
	rollerName := logName + "." + time.Now().Format("2006-01-02")
	os.Remove(logName)
	os.Remove(rollerName)
	// replace rotate interval for test
	doRotate = testRotate
	defer func() {
		doRotate = doRotateFunc
	}()

	logger, err := GetOrCreateLogger(logName, &Roller{MaxTime: 10, Handler: rollerHandler})
	if err != nil {
		t.Fatal(err)
	}
	// remove the log file, the log is output to no where
	os.Remove(logName)
	logger.Print(newLogBufferString("nowhere"), false)
	// wait roller
	time.Sleep(11 * time.Second)
	logger.Print(newLogBufferString("data"), false)
	time.Sleep(100 * time.Millisecond) // wait write flush
	logger.Print(newLogBufferString("output"), false)
	logger.Close() // force flush
	b, err := ioutil.ReadFile(logName)
	if err != nil {
		t.Fatalf("read log file failed: %v", err)
	}
	if string(b) != "dataoutput" {
		t.Errorf("read file data: %s", string(b))
	}
	// rollerName should not be exists
	if _, err := os.Stat(rollerName); err == nil {
		t.Errorf("roller file exists, but expected not: %v", err)
	}
}

func TestLogReopen(t *testing.T) {
	l, err := GetOrCreateLogger("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.reopen(); err != ErrReopenUnsupported {
		t.Errorf("test log reopen failed")
	}
	l, err = GetOrCreateLogger("/tmp/mosn_bench/testlogreopen.log", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.reopen(); err != nil {
		t.Errorf("test log reopen failed %v", err)
	}
}

func TestLoglocalOffset(t *testing.T) {
	_, offset := time.Now().Zone()
	var defaultRollerTime int64 = 24 * 60 * 60
	t1 := time.Date(2018, time.December, 25, 23, 59, 59, 0, time.Local)
	t2 := time.Date(2018, time.December, 26, 00, 00, 01, 0, time.Local)
	if (t1.Unix()+int64(offset))/defaultRollerTime+1 != (t2.Unix()+int64(offset))/defaultRollerTime {
		t.Errorf("test localOffset failed")
	}
	t.Logf("t1=%d t2=%d offset=%d rollertime=%d\n", t1.Unix(), t2.Unix(), offset, defaultRollerTime)
	t.Logf("%d %d\n", (t1.Unix())/defaultRollerTime, (t1.Unix() / defaultRollerTime))
	t.Logf("%d %d\n", (t1.Unix()+int64(offset))/defaultRollerTime, (t2.Unix()+int64(offset))/defaultRollerTime)
}

type testRecord struct {
	closed   bool
	interval time.Duration
}

func (r *testRecord) overwriteRotateForTest(l *Logger, interval time.Duration) {
	r.interval = interval
	<-l.stopRotate
	r.closed = true
}

func TestRotateClose(t *testing.T) {
	r := &testRecord{}
	doRotate = r.overwriteRotateForTest
	defer func() {
		doRotate = doRotateFunc // recover
	}()
	logName := "/tmp/test.log"
	logger, err := GetOrCreateLogger(logName, nil) // create default roller
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond) // wait logger start
	logger.Close()
	time.Sleep(10 * time.Millisecond) // wait logger closed
	if !r.closed {
		t.Fatalf("want to close rotate func, but not")
	}
}

// we use these cases for test rotate interval
func TestRotateInteval(t *testing.T) {
	r := &testRecord{}
	doRotate = r.overwriteRotateForTest
	defer func() {
		doRotate = doRotateFunc // recover
	}()
	now := time.Now()
	year, month, day := now.Date()
	// 17:00:00
	t1 := time.Date(year, month, day, 17, 00, 0, 0, time.Local)
	lg := &Logger{
		create: t1,
		roller: &defaultRoller,
	}
	lg.startRotate()
	time.Sleep(10 * time.Millisecond)
	// expected interval is not determined by create file, but current time
	// allow some deviation
	tomorrow := time.Date(year, month, day+1, 00, 00, 0, 0, time.Local)
	expected := tomorrow.Sub(now)
	diff := time.Duration(math.Abs(float64(r.interval - expected)))
	if diff > 3*time.Second { // out of deviation
		t.Fatalf("rotate interval is not expected, interval:%v, expected: %v", r.interval, expected)
	}
	t2 := time.Date(year, month, day, now.Hour(), 05, 8, 0, time.Local)
	lg2 := &Logger{
		create: t2,
		roller: &Roller{MaxTime: 3600, Handler: rollerHandler},
	}
	lg2.startRotate()
	time.Sleep(10 * time.Millisecond)
	// expected interval is not determined by create file, but current time
	// allow some deviation
	nextHour := time.Date(year, month, day, now.Hour()+1, 00, 0, 0, time.Local)
	expectedHour := nextHour.Sub(now)
	diffHour := time.Duration(math.Abs(float64(r.interval - expectedHour)))
	if diffHour > 3*time.Second { // out of deviation
		t.Fatalf("rotate interval is not expected, %v", r.interval)
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func TestRotateRightNow(t *testing.T) {
	defer func() {
		doRotate = doRotateFunc
	}()
	// mock rotate
	doRotate = func(l *Logger, interval time.Duration) {
		<-time.After(interval)
		now := time.Now()
		if l.roller.MaxTime == defaultRotateTime {
			os.Rename(l.output, l.output+"."+l.create.Format("2006-01-02"))
		} else {
			os.Rename(l.output, l.output+"."+l.create.Format("2006-01-02_15"))
		}
		l.create = now
		// mock reopen
		os.OpenFile(l.output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	}
	// init
	logpath := "/tmp/mosn_test"
	os.RemoveAll(logpath)
	os.MkdirAll(logpath, 0755)
	logfile := path.Join(logpath, "test_rotate_rightnow.log")
	os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	// make a logger, hack the create time
	ct := time.Now().Add(-24 * time.Hour) // file created at last day
	lg := &Logger{
		output: logfile,
		create: ct,
		roller: &defaultRoller,
	}
	lg.startRotate()
	time.Sleep(10 * time.Millisecond)
	rotated := path.Join(logpath, "test_rotate_rightnow.log."+ct.Format("2006-01-02"))
	if !(exists(rotated) && exists(logfile)) {
		t.Fatalf("log rotate is not expected")
	}
	// rotate by hour
	ct2 := time.Now().Add(-1 * time.Hour) // file created at last hour
	lg2 := &Logger{
		output: logfile,
		create: ct2,
		roller: &Roller{MaxTime: 3600, Handler: rollerHandler},
	}
	lg2.startRotate()
	time.Sleep(10 * time.Millisecond)
	rotatedHour := path.Join(logpath, "test_rotate_rightnow.log."+ct2.Format("2006-01-02_15"))
	if !(exists(rotatedHour) && exists(logfile)) {
		t.Fatalf("log rotate is not expected")
	}
}

func TestDynamicLocalOffset(t *testing.T) {
	l := &Logger{
		roller: &Roller{
			MaxTime: int64(24 * 60 * 60),
		},
	}
	// simulate a location
	loc, _ := time.LoadLocation("America/Los_Angeles")
	today, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", "2020-11-01 00:00:00 -0700 PDT", loc)
	for i := 0; i < 3; i++ {
		interval := l.calculateInterval(today)
		tomorrow := today.Add(interval)
		// rotate
		today = tomorrow
	}
	ts := today.Format("2006-01-02 15:04:05")
	if ts != "2020-11-03 00:00:00" {
		t.Fatalf("time rotate not expected: %v", today)
	}
}

func TestDoRotateFunc(t *testing.T) {
	notify := make(chan bool, 1)
	registeNofify(notify)
	roller := Roller{
		MaxTime: 1000,                   // init by 1000 seconds
		Handler: func(l *LoggerInfo) {}, // ignore, do nothing
	}
	l := &Logger{
		output:       "test.log",
		roller:       &roller,
		stopRotate:   make(chan struct{}),
		reopenChan:   make(chan struct{}, 100),
		rollerUpdate: notify,
	}
	go doRotateFunc(l, 1000*time.Second)
	select {
	case <-l.reopenChan:
		t.Fatalf("expected no reopen called, but received a reopen")
	case <-time.After(2 * time.Second):
	}
	// mock update roller
	roller = Roller{
		MaxTime: 1,
		Handler: func(l *LoggerInfo) {}, // ignore, do nothing
	}
	sendNotify()
	reopens := 0
WAIT:
	for {
		select {
		case <-l.reopenChan:
			reopens++
			if reopens >= 2 {
				break WAIT
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("atfer 3 seconds, got %d reopens, expected at least 2", reopens)
		}
	}
	t.Logf("received %d reopens", reopens)
	close(l.stopRotate)
}
