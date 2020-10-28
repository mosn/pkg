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
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	ch := make(chan int, 1)
	_ = NewTimer(time.Second, func() {
		ch <- 100
	})
	select {
	case i := <-ch:
		if i != 100 {
			t.Fatalf("unexpected channel %d", i)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout")
	}
}

func TestTimerStop(t *testing.T) {
	ch := make(chan int, 1)
	timer := NewTimer(time.Second, func() {
		ch <- 100
	})
	timer.Stop()
	// func never be called
	select {
	case <-ch:
		t.Fatalf("receive channel from func")
	case <-time.After(2 * time.Second):
		// expected, pass
	}
}

func TestTimerReset(t *testing.T) {
	ch := make(chan int, 1)
	timer := NewTimer(100*time.Second, func() {
		ch <- 100
	})
	timer.Reset(0) // timeout right now
	select {
	case i := <-ch:
		if i != 100 {
			t.Fatalf("unexpected channel %d", i)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("unexpected timeout")
	}
}
