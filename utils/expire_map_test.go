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

func TestExpiredMap(t *testing.T) {
	handler := func(key interface{}) (interface{}, bool) {
		return "val2", true
	}

	// Test sync update mod
	expireMap := NewExpiredMap(handler, true)

	val, ok := expireMap.Get("nokey")
	if ok {
		t.Error("want get nil, but got ok")
	}

	expireMap.Set("key1", "val1", time.Duration(2)*time.Millisecond)
	val, ok = expireMap.Get("key1")
	if val != "val1" || !ok {
		t.Errorf("want get val %s, but got val %s", "val1", val)
	}

	// If it expires, it should be automatically updated
	time.Sleep(time.Millisecond * 3)
	val, ok = expireMap.Get("key1")
	if val != "val2" || !ok {
		t.Errorf("want get val %s, but got val %s", "val2", val)
	}

	// Check sync mode and update the slow scene
	handlerSlow := func(key interface{}) (interface{}, bool) {
		time.Sleep(time.Millisecond * 3)
		return "val3", true
	}

	expireMap.UpdateHandler = handlerSlow

	time.Sleep(time.Millisecond * 3)
	val, ok = expireMap.Get("key1")
	if val != "val3" || !ok {
		t.Errorf("want get val %s, but got val %s", "val3", val)
	}

	// Test async update mod
	asyncHandler := func(key interface{}) (interface{}, bool) {
		time.Sleep(time.Millisecond * 1)
		return "val5", true
	}

	expireMap = NewExpiredMap(asyncHandler, false)

	expireMap.Set("key1", "val4", time.Duration(2)*time.Millisecond)
	val, ok = expireMap.Get("key1")
	if val != "val4" || !ok {
		t.Errorf("want get val %s, but got val %s", "val4", val)
	}

	time.Sleep(time.Millisecond * 3)
	val, ok = expireMap.Get("key1")
	if val != "val4" || ok {
		t.Errorf("async mod want get val %s, but got val %s", "val4", val)
	}

	time.Sleep(time.Millisecond * 2)
	val, ok = expireMap.Get("key1")
	if val != "val5" || !ok {
		t.Errorf("async mod want get val %s, but got val %s", "val5", val)
	}

	// Test NeverExpire

	expireMap = NewExpiredMap(nil, false)
	expireMap.Set("key1", "val6", NeverExpire)
	time.Sleep(time.Second * 1)
	val, ok = expireMap.Get("key1")
	if !ok {
		t.Error("want never expire, but got expired")
	}

}

func BenchmarkExpiredMapSync(b *testing.B) {
	handler := func(key interface{}) (interface{}, bool) {
		return "val2", true
	}

	// Test sync update mod
	expireMap := NewExpiredMap(handler, true)

	for i := 0; i < b.N; i++ {
		expireMap.Set("key", i, time.Duration(10))
		expireMap.Get("key")
	}
}

func BenchmarkExpiredMapAsync(b *testing.B) {
	handler := func(key interface{}) (interface{}, bool) {
		return "val2", true
	}

	// Test async update mod
	expireMap := NewExpiredMap(handler, false)

	for i := 0; i < b.N; i++ {
		expireMap.Set("key", i, time.Duration(10))
		expireMap.Get("key")
	}
}

func BenchmarkExpiredMapAsyncParallel(b *testing.B) {
	handler := func(key interface{}) (interface{}, bool) {
		return "val2", true
	}

	// Test async update mod
	expireMap := NewExpiredMap(handler, false)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			expireMap.Set("key", 1, time.Duration(10))
			expireMap.Get("key")
		}
	})

}

func BenchmarkExpiredMapSyncParallel(b *testing.B) {
	handler := func(key interface{}) (interface{}, bool) {
		return "val2", true
	}

	// Test sync update mod
	expireMap := NewExpiredMap(handler, true)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			expireMap.Set("key", 1, time.Duration(10))
			expireMap.Get("key")
		}
	})

}
