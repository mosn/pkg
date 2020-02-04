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

package buffer

import (
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func intRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func intN(n int) int {
	return rand.Intn(n) + 1
}

func TestByteBufferPoolSmallBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := intN(1 << minShift)
		bp := pool.take(size)

		if cap(*bp) != 1<<minShift {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolMediumBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := minShift; i < maxShift; i++ {
		size := intRange((1<<uint(i))+1, 1<<uint(i+1))
		bp := pool.take(size)

		if cap(*bp) != 1<<uint(i+1) {
			t.Errorf("Expect get the slab size (%d) from pool, but got %d", 1<<uint(i+1), cap(*bp))
		}

		//Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesBufferPoolLargeBytes(t *testing.T) {
	pool := newByteBufferPool()

	for i := 0; i < 1024; i++ {
		size := 1<<maxShift + intN(i+1)
		bp := pool.take(size)

		if cap(*bp) != size {
			t.Errorf("Expect get the %d bytes from pool, but got %d", size, cap(*bp))
		}

		// Puts the bytes to pool
		pool.give(bp)
	}
}

func TestBytesSlot(t *testing.T) {
	pool := newByteBufferPool()

	if pool.slot(pool.minSize-1) != 0 {
		t.Errorf("Expect get the 0 slot")
	}

	if pool.slot(pool.minSize) != 0 {
		t.Errorf("Expect get the 0 slot")
	}

	if pool.slot(pool.minSize+1) != 1 {
		t.Errorf("Expect get the 1 slot")
	}

	if pool.slot(pool.maxSize-1) != maxShift-minShift {
		t.Errorf("Expect get the %d slot", maxShift-minShift)
	}

	if pool.slot(pool.maxSize) != maxShift-minShift {
		t.Errorf("Expect get the %d slot", maxShift-minShift)
	}

	if pool.slot(pool.maxSize+1) != errSlot {
		t.Errorf("Expect get errSlot")
	}
}

func Test_ByteBufferPool(t *testing.T) {
	str := "ByteBufferPool Test"
	b := GetBytes(len(str))
	buf := *b
	copy(buf, str)

	if string(buf) != str {
		t.Fatal("ByteBufferPool Test Failed")
	}
	PutBytes(b)

	b = GetBytes(len(str))
	buf = *b
	copy(buf, str)

	if string(buf) != str {
		t.Fatal("ByteBufferPool Test Failed")
	}
	PutBytes(b)
	t.Log("ByteBufferPool Test Sucess")
}

// Test byteBufferPool
func testbytepool() *[]byte {
	b := GetBytes(Size)
	buf := *b
	for i := 0; i < Size; i++ {
		buf[i] = 1
	}
	return b
}

func testbyte() []byte {
	buf := make([]byte, Size)
	for i := 0; i < Size; i++ {
		buf[i] = 1
	}
	return buf
}

func BenchmarkBytePool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testbytepool()
		PutBytes(buf)
	}
}

func BenchmarkByteMake(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testbyte()
	}
}
