// +build !windows

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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestSetHijackStdPipeline(t *testing.T) {
	// init
	stderrFile := "/tmp/test_stderr"
	os.Remove(stderrFile)
	// call, test std error only
	SetHijackStdPipeline(stderrFile, false, true)
	time.Sleep(time.Second) // wait goroutine run
	fmt.Fprintf(os.Stderr, "test stderr")
	// verify
	if !verifyFile(stderrFile, "test stderr") {
		t.Error("stderr hijack failed")
	}
	ResetHjiackStdPipeline()
	fmt.Fprintf(os.Stderr, "repaired\n")
}

func verifyFile(p string, data string) bool {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return false
	}
	return string(b) == data
}

func TestNextDay(t *testing.T) {
	t.Run("test dst in America/Los_Angeles", func(t *testing.T) {
		loc, _ := time.LoadLocation("America/Los_Angeles")
		firstTimeStr := "2020-11-01 00:00:00 -0700 PDT"
		ft, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", firstTimeStr, loc)
		d := nextDayDuration(ft, loc)
		if d != 25*time.Hour { // rollback an hour, so next day is 25 hour
			t.Fatalf("next day duration is %s", d)
		}

		secondTimeStr := "2021-03-14 00:00:00 -0800 PST"
		st, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", secondTimeStr, loc)
		d2 := nextDayDuration(st, loc)
		if d2 != 23*time.Hour { // dst, next day is 23 hour
			t.Fatalf("next day duration is %s", d)
		}
	})
	t.Run("test utc time zone", func(t *testing.T) {
		loc, _ := time.LoadLocation("UTC")
		firstTimeStr := "2020-11-01 00:00:00 +0000 UTC"
		ft, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", firstTimeStr, loc)
		d := nextDayDuration(ft, loc)
		if d != 24*time.Hour {
			t.Fatalf("next day duration is %s", d)
		}

		secondTimeStr := "2021-03-14 00:00:00 +0000 UTC"
		st, _ := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", secondTimeStr, loc)
		d2 := nextDayDuration(st, loc)
		if d2 != 24*time.Hour {
			t.Fatalf("next day duration is %s", d)
		}

	})
	t.Run("test time.loal", func(t *testing.T) {
		firstTimeStr := "2020-11-01 00:00:00"
		ft, _ := time.ParseInLocation("2006-01-02 15:04:05", firstTimeStr, time.Local)
		d := nextDayDuration(ft, time.Local)
		if d != 24*time.Hour {
			t.Fatalf("next day duration is %s", d)
		}

		secondTimeStr := "2021-03-14 00:00:00"
		st, _ := time.ParseInLocation("2006-01-02 15:04:05", secondTimeStr, time.Local)
		d2 := nextDayDuration(st, time.Local)
		if d2 != 24*time.Hour {
			t.Fatalf("next day duration is %s", d)
		}
	})
}
