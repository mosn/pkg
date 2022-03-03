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
	"sync"
	"testing"
)

func TestSyncListPushBack(t *testing.T) {
	l := NewSyncList()

	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			l.PushBack(1)
		}()
	}
	wg.Wait()
	len := l.Len()
	if len != 100 {
		t.Errorf("sync list length expect 100 while got: %v", len)
	}
}

func BenchmarkSyncListPushBack(b *testing.B) {
	l := NewSyncList()
	for i := 0; i < b.N; i++ {
		l.PushBack(1)
	}
}

func TestSyncListRemove(t *testing.T) {
	l := NewSyncList()
	wg := sync.WaitGroup{}
	wg.Add(200)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			l.PushBack(1)
		}()
		go func() {
			defer wg.Done()
			e := l.PushBack(2)
			l.Remove(e)
		}()
	}
	wg.Wait()
	len := l.Len()
	if len != 100 {
		t.Errorf("sync list length expect 100 while got: %v", len)
	}
}

func BenchmarkSyncListRemove(b *testing.B) {
	l := NewSyncList()
	for i := 0; i < b.N; i++ {
		e := l.PushBack(1)
		l.Remove(e)
	}
}

func TestSyncListVisitSafe(t *testing.T) {
	l := NewSyncList()
	for i := 0; i < 100; i++ {
		l.PushBack(i)
	}
	count := 0
	l.VisitSafe(func(_ interface{}) {
		count++
	})
	if count != 100 {
		t.Errorf("count expected 100 while got: %v", count)
	}
	len := l.Len()
	if len != 100 {
		t.Errorf("sync list length expect 100 while got: %v", len)
	}
}

func TestSyncListVisitSafeWithRemove(t *testing.T) {
	l := NewSyncList()
	for i := 0; i < 100; i++ {
		l.PushBack(i)
	}
	count := 0
	l.VisitSafe(func(v interface{}) {
		n := v.(int)
		first := l.list.Front().Value.(int)

		if n != first {
			t.Errorf("visit value not match the first value, %v vs %v", n, first)
		}
		l.Remove(l.list.Front().Next())
		l.Remove(l.list.Front())

		count++
	})
	if count != 50 {
		t.Errorf("count expected 50 while got: %v", count)
	}
	len := l.Len()
	if len != 0 {
		t.Errorf("sync list length expect 0 while got: %v", len)
	}
}

func BenchmarkSyncListVisitSafe(b *testing.B) {
	l := NewSyncList()
	for i := 0; i < 100; i++ {
		l.PushBack(i)
	}
	for i := 0; i < b.N; i++ {
		l.VisitSafe(func(v interface{}) {
			n := v.(int)
			n++
		})
	}
}

func TestSyncListVisitSafeWithPushBack(t *testing.T) {
	l := NewSyncList()
	for i := 0; i < 100; i++ {
		l.PushBack(i)
	}
	count := 0
	l.VisitSafe(func(v interface{}) {
		n := v.(int)
		if n%2 == 0 && n > 10 {
			l.PushBack(n / 2)
		}

		count++
	})
	if count != 171 {
		t.Errorf("count expected 50 while got: %v", count)
	}
	len := l.Len()
	if len != count {
		t.Errorf("sync list length expect %v while got: %v", count, len)
	}
}
