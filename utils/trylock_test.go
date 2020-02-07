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

func TestTrylock(t *testing.T) {
	m := NewMutex()
	ok := m.TryLock(time.Second)
	if !ok {
		t.Error("it should be lock suc but failed!")
	}

	ok = m.TryLock(time.Second * 3)
	if ok {
		t.Error("it should be lock failed but suc")
	}

	m.Unlock()

	ok = m.TryLock(time.Second)
	if !ok {
		t.Error("it should be lock suc but failed!")
	}

}

func BenchmarkTrylock(b *testing.B) {
	var number int
	lock := NewMutex()
	for i := 0; i < b.N; i++ {
		go func() {
			defer lock.Unlock()
			lock.TryLock(time.Second * 3)
			number++
		}()
	}
}
