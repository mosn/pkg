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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

type errorLogCase struct {
	level Level
	f     func(format string, args ...interface{})
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

var preMapping = map[Level]string{
	FATAL: FatalPre,
	ERROR: ErrorPre,
	WARN:  WarnPre,
	INFO:  InfoPre,
	DEBUG: DebugPre,
	TRACE: TracePre,
}

func TestErrorLog(t *testing.T) {
	logName := "/tmp/mosn/error_log_print.log"
	os.Remove(logName)
	rlg, err := GetOrCreateLogger(logName, nil)
	if err != nil {
		t.Fatal("create logger failed")
	}
	lg := &SimpleErrorLog{
		Level:  RAW,
		Logger: rlg,
	}
	cases := []errorLogCase{
		{
			level: ERROR,
			f:     lg.Errorf,
		},
		{
			level: WARN,
			f:     lg.Warnf,
		},
		{
			level: INFO,
			f:     lg.Infof,
		},
		{
			level: DEBUG,
			f:     lg.Debugf,
		},
		{
			level: TRACE,
			f:     lg.Tracef,
		},
	}
	for _, c := range cases {
		lg.SetLogLevel(c.level)
		c.f("testdata")
	}
	lg.Toggle(true) // disable
	for _, c := range cases {
		lg.SetLogLevel(c.level)
		c.f("testdata") // write nothing
	}
	time.Sleep(time.Second) // wait buffer flush
	// read lines
	lines, err := readLines(logName)
	if err != nil {
		t.Fatal(err)
	}
	// verify count
	if len(lines) != len(cases) {
		t.Fatalf("logger write lines not expected, writes: %d, expected: %d", len(lines), len(cases))
	}
	// verify log in order if channel buffer is not full
	for i, l := range lines {
		// l format
		// 2006/01/02 15:04:05 [Level] {Count}
		qs := strings.Split(l, " ")
		c := cases[i]
		if !(len(qs) >= 4 && qs[2] == preMapping[c.level]) {
			t.Errorf("level: %v write format is not expected", c)
		}
	}
}

func testLogFatal(logName string) {
	os.Remove(logName)
	rlg, err := GetOrCreateLogger(logName, nil)
	if err != nil {
		fmt.Println("create logger failed: ", err)
		return
	}
	lg := &SimpleErrorLog{
		Level:  ERROR,
		Logger: rlg,
	}
	lg.Fatalf("test_fatal")
}

func TestLogFatal(t *testing.T) {
	logName := "/tmp/mosn/fatal_log.log"
	if os.Getenv("FATAL_TEST") == "true" {
		testLogFatal(logName)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestLogFatal")
	cmd.Env = append(os.Environ(), "FATAL_TEST=true")
	o, err := cmd.Output()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		lines, err := readLines(logName)
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 {
			t.Fatalf("logger write lines not expected, writes: %d", len(lines))
		}
		qs := strings.SplitN(lines[0], " ", 4)
		if !(len(qs) == 4 &&
			qs[2] == "[FATAL]" &&
			qs[3] == "test_fatal") {
			t.Fatalf("output data is unexpected: %s", lines[0])
		}
	} else {
		t.Fatalf("want a fatal exit, output: %s", string(o))
	}
}

func BenchmarkLog(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rlg, err := GetOrCreateLogger("/tmp/mosn_bench/benchmark.log", nil)
	if err != nil {
		b.Fatal("create logger failed")
	}
	l := &SimpleErrorLog{
		Level:  DEBUG,
		Logger: rlg,
	}
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		l.Debugf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
	}
}

func BenchmarkLogParallel(b *testing.B) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rlg, err := GetOrCreateLogger("/tmp/mosn_bench/benchmark.log", nil)
	if err != nil {
		b.Fatal("create logger failed")
	}
	l := &SimpleErrorLog{
		Level:  DEBUG,
		Logger: rlg,
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Debugf("BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog BenchmarkLog %v", l)
		}
	})
}
