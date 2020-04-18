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
	"testing"
)

func TestIoBufferPoolWithCount(t *testing.T) {
	buf := GetIoBuffer(0)
	bytes := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	buf.Write(bytes)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer len not match write bytes' size")
	}
	// Add a count, need put twice to free buffer
	buf.Count(1)
	PutIoBuffer(buf)
	if buf.Len() != len(bytes) {
		t.Error("iobuffer expected put ignore")
	}
	PutIoBuffer(buf)
	if buf.Len() != 0 {
		t.Error("iobuffer expected put success")
	}
}

func TestIoBufferPooPutduplicate(t *testing.T) {
	buf := GetIoBuffer(0)
	err := PutIoBuffer(buf)
	if err != nil {
		t.Errorf("iobuffer put error:%v", err)
	}
	err = PutIoBuffer(buf)
	if err == nil {
		t.Errorf("iobuffer should be error: Put IoBuffer duplicate")
	}
}

func Test_IoBufferPool_Slice_Increase(t *testing.T) {
	str := "IoBufferPool Test"
	// []byte slice increase
	buffer := GetIoBuffer(1)
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Slice Increase Failed")
	}
	t.Log("IoBufferPool Test Slice Increase Sucess")
}
func Test_IoBufferPool_Alloc_Free(t *testing.T) {
	str := "IoBufferPool Test"
	buffer := GetIoBuffer(100)
	buffer.Free()
	buffer.Alloc(1)
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Alloc Free Failed")
	}
	t.Log("IoBufferPool Test Alloc Free Sucess")
}

func Test_IoBufferPool(t *testing.T) {
	str := "IoBufferPool Test"
	buffer := GetIoBuffer(len(str))
	buffer.Write([]byte(str))

	b := make([]byte, 32)
	_, err := buffer.Read(b)

	if err != nil {
		t.Fatal(err)
	}

	PutIoBuffer(buffer)

	if string(b[:len(str)]) != str {
		t.Fatal("IoBufferPool Test Failed")
	}
	t.Log("IoBufferPool Test Sucess")
}

// Test IoBufferPool
const Size = 2048

var Buffer [Size]byte

func testiobufferpool() IoBuffer {
	b := GetIoBuffer(Size)
	b.Write(Buffer[:])
	return b
}

func testiobuffer() IoBuffer {
	b := newIoBuffer(Size)
	b.Write(Buffer[:])
	return b
}

func BenchmarkIoBufferPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testiobufferpool()
		PutIoBuffer(buf)
	}
}

func BenchmarkIoBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testiobuffer()
	}
}

func BenchmarkNewIoBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := testiobuffer()
		PutIoBuffer(buf)
	}
}

