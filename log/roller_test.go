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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRoller(t *testing.T) {
	defer func() {
		// reset
		defaultRoller = Roller{MaxTime: defaultRotateTime, Handler: rollerHandler}
	}()
	errorPraseArgs := "size=100 age=10 keep=10 compress=1"
	roller, err := ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	errorPraseArgs = "size = 100 age = 10 keep = 10"
	roller, err = ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	errorPraseArgs = "size=100, age=10, keep=10, compress=off"
	roller, err = ParseRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	praseArgs := "size=100 age=10 keep=10 compress=on"
	roller, err = ParseRoller(praseArgs)
	if roller == nil {
		t.Errorf("ParseRoller failed")
	}
	if roller.MaxSize != 100 || roller.MaxAge != 10 || roller.MaxBackups != 10 || roller.Compress != true {
		t.Errorf("ParseRoller failed")
	}

	praseArgs = "size=100"
	roller, err = ParseRoller(praseArgs)
	if roller == nil {
		t.Errorf("ParseRoller failed")
	}
	if roller.MaxSize != 100 || roller.Compress != false {
		t.Errorf("ParseRoller failed")
	}

	errorPraseArgs = "A=3"
	err = InitGlobalRoller(errorPraseArgs)
	if err == nil {
		t.Errorf("ParseRoller should be failed")
	}

	praseArgs = "size=100"
	err = InitGlobalRoller(praseArgs)
	if err != nil {
		t.Errorf("ParseRoller failed")
	}
	if defaultRoller.MaxSize != 100 || defaultRoller.Compress != false {
		t.Errorf("ParseRoller failed")
	}

	errorPraseArgs = "time=12"
	err = InitGlobalRoller(errorPraseArgs)
	if err != nil {
		t.Errorf("ParseRoller failed")
	}

	if defaultRoller.MaxTime != 12*60*60 {
		t.Errorf("ParseRoller failed")
	}

	errorPraseArgs = "time=36" // out of max
	err = InitGlobalRoller(errorPraseArgs)
	if err != nil {
		t.Errorf("ParseRoller failed")
	}
	if defaultRoller.MaxTime != 24*60*60 {
		t.Errorf("ParseRoller out of max time")
	}
}

func TestInitDefaultRoller(t *testing.T) {

	lg, err := GetOrCreateLogger("/tmp/test_roller_init.log", nil)
	if err != nil {
		t.Fatal(lg)
	}
	if lg.roller.MaxTime != defaultRotateTime {
		t.Errorf("unexpected default roller, got %d", lg.roller.MaxTime)
	}
	InitGlobalRoller("time=1")
	defer func() {
		// reset
		defaultRoller = Roller{MaxTime: defaultRotateTime, Handler: rollerHandler}
		os.RemoveAll("/tmp/test_roller_init.log")
	}()
	if lg.roller.MaxTime != 60*60 {
		t.Errorf("expected roller reset, but not, got: %d", lg.roller.MaxTime)
	}

}

func TestRollerHandler(t *testing.T) {
	p := "/tmp/rollertest/"
	name := path.Join(p, "roller.log")
	os.RemoveAll(p)
	os.MkdirAll(p, 0755)
	mockWriteLog := func(data []byte) error {
		return ioutil.WriteFile(name, data, 0644)
	}
	linfo := &LoggerInfo{
		LogRoller: &Roller{
			MaxTime:    defaultRotateTime,
			MaxBackups: 10,
		},
		FileName:   name,
		CreateTime: time.Now(),
	}
	for i := 0; i < 3; i++ {
		str := fmt.Sprintf("%s.%d\n", "test", i)
		if err := mockWriteLog([]byte(str)); err != nil {
			t.Fatalf("mock write log error: %v", err)
		}
		rollerHandler(linfo)
	}
	// verify
	files, err := ioutil.ReadDir(p)
	if err != nil {
		t.Fatalf("read dir %s error: %v", p, err)
	}
	prefix := "roller.log." + linfo.CreateTime.Format("2006-01-02")
	for _, finfo := range files {
		last := strings.TrimPrefix(finfo.Name(), prefix)
		if last == "" {
			last = ".0"
		}
		expected := fmt.Sprintf("%s%s\n", "test", last)
		fname := path.Join(p, finfo.Name())
		b, _ := ioutil.ReadFile(fname)
		if !strings.EqualFold(string(b), expected) {
			t.Fatalf("file %s read data %s, expected %s", fname, string(b), expected)
		}
	}
}

func TestRollerGetLogWriter(t *testing.T) {
	roller := defaultRoller
	roller.Filename = "test"
	io1 := roller.GetLogWriter()
	io2 := roller.GetLogWriter()
	assert.Equal(t, io1, io2)
}
