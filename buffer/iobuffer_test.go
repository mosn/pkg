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
	"bytes"
	"encoding/binary"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randN(n int) int {
	return rand.Intn(n) + 1
}

func TestNewIoBufferString(t *testing.T) {
	for i := 0; i < 1024; i++ {
		s := randString(i)
		b := NewIoBufferString(s)
		if b.String() != s {
			t.Errorf("Expect %s but got %s", s, b.String())
		}
	}
}

func TestNewIoBufferBytes(t *testing.T) {
	for i := 0; i < 1024; i++ {
		s := randString(i)
		b := NewIoBufferBytes([]byte(s))
		if !bytes.Equal(b.Bytes(), []byte(s)) {
			t.Errorf("Expect %s but got %s", s, b.String())
		}
	}
}

func TestIoBufferCopy(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	n := randN(1024) + 1
	b.copy(n)
	if cap(b.buf) < 2*1+n {
		t.Errorf("b.copy(%d) should expand to at least %d, but got %d", n, 2*1+n, cap(b.buf))
	}
}

func TestIoBufferGrowCopy(t *testing.T) {
	bi := newIoBuffer(MaxThreshold)
	b := bi.(*ioBuffer)
	n := randN(1024) + 1
	b.copy(n)
	l := cap(*GetBytes(MaxThreshold+MaxThreshold/4+n))
	if cap(b.buf) != l {
		t.Errorf("b.copy(%d) should expand to %d, but got %d", n, l, cap(b.buf))
	}
}

func TestIoBufferWrite(t *testing.T) {
	b := newIoBuffer(1)
	n := randN(64)

	for i := 0; i < n; i++ {
		s := randString(i + 16)
		n, err := b.Write([]byte(s))
		if err != nil {
			t.Fatal(err)
		}

		if n != len(s) {
			t.Errorf("Expect write %d bytes, but got %d", len(s), n)
		}

		if !bytes.Equal(b.Peek(len(s)), []byte(s)) {
			t.Errorf("Expect peek %s but got %s", s, string(b.Peek(len(s))))
		}

		b.Drain(len(s))
	}

	input := make([]byte, 0, 1024)
	writer := bytes.NewBuffer(nil)

	for i := 0; i < n; i++ {
		s := randString(i + 16)
		n, err := b.Write([]byte(s))
		if err != nil {
			t.Fatal(err)
		}

		if n != len(s) {
			t.Errorf("Expect write %d bytes, but got %d", len(s), n)
		}

		input = append(input, []byte(s)...)
	}

	_, err := b.WriteTo(writer)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(writer.Bytes(), input) {
		t.Errorf("Expect %s but got %s", input, string(writer.Bytes()))
	}
}

func TestIoBufferAppend(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	n := randN(64)
	for i := 0; i < n; i++ {
		s := randString(i + 16)
		err := b.Append([]byte(s))
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(b.Peek(len(s)), []byte(s)) {
			t.Errorf("Expect peek %s but got %s", s, string(b.Peek(len(s))))
		}

		b.Drain(len(s))
	}
}

func TestIoBufferAppendByte(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	input := make([]byte, 0, 1024)
	n := randN(1024)

	for i := 0; i < n; i++ {
		err := b.AppendByte(byte(i))
		if err != nil {
			t.Fatal(err)
		}
		input = append(input, byte(i))
	}

	if b.Len() != n {
		t.Errorf("Expect %d bytes, but got %d", n, b.Len())
	}

	if !bytes.Equal(b.Peek(n), input) {
		t.Errorf("Expect %x, but got %x", input, b.Peek(n))
	}
}

func TestIoBufferWriteByte(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	input := make([]byte, 0, 1024)
	n := randN(1024)

	for i := 0; i < n; i++ {
		err := b.WriteByte(byte(i))
		if err != nil {
			t.Fatal(err)
		}
		input = append(input, byte(i))
	}

	if b.Len() != n {
		t.Errorf("Expect %d bytes, but got %d", n, b.Len())
	}

	if !bytes.Equal(b.Peek(n), input) {
		t.Errorf("Expect %x, but got %x", input, b.Peek(n))
	}
}

func TestIoBufferWriteUin16(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	input := make([]byte, 0, 1024)
	n := randN(512)
	var temp [2]byte

	for i := 0; i < n; i++ {
		err := b.WriteUint16(uint16(i))
		if err != nil {
			t.Fatal(err)
		}
		binary.BigEndian.PutUint16(temp[0:], uint16(i))
		input = append(input, temp[0])
		input = append(input, temp[1])
	}

	if b.Len() != n*2 {
		t.Errorf("Expect %d bytes, but got %d", n, b.Len())
	}

	if !bytes.Equal(b.Peek(n*2), input) {
		t.Errorf("Expect %x, but got %x", input, b.Peek(n))
	}
}

func TestIoBufferWriteUint32(t *testing.T) {
	bi := newIoBuffer(1)
	b := bi.(*ioBuffer)
	input := make([]byte, 0, 1024)
	n := randN(256)
	var temp [4]byte

	for i := 0; i < n; i++ {
		err := b.WriteUint32(uint32(i))
		if err != nil {
			t.Fatal(err)
		}
		binary.BigEndian.PutUint32(temp[0:], uint32(i))
		input = append(input, temp[0])
		input = append(input, temp[1])
		input = append(input, temp[2])
		input = append(input, temp[3])
	}

	if b.Len() != n*4 {
		t.Errorf("Expect %d bytes, but got %d", n, b.Len())
	}

	if !bytes.Equal(b.Peek(n*4), input) {
		t.Errorf("Expect %x, but got %x", input, b.Peek(n))
	}
}

func TestIoBufferRead(t *testing.T) {
	b := newIoBuffer(0)
	data := make([]byte, 1)

	n, err := b.Read(data)
	if err != io.EOF {
		t.Errorf("Expect io.EOF but got %s", err)
	}

	if n != 0 {
		t.Errorf("Expect 0 bytes but got %d", n)
	}

	n, err = b.Read(nil)
	if n != 0 || err != nil {
		t.Errorf("Expect (0, nil) but got (%d, %s)", n, err)
	}

	b = newIoBuffer(1)
	s := randString(1024)
	reader := bytes.NewReader([]byte(s))

	nr, err := b.ReadFrom(reader)
	if err != nil {
		t.Errorf("Expect nil but got %s", err)
	}

	if nr != int64(len(s)) {
		t.Errorf("Expect %d bytes but got %d", len(s), nr)
	}

	if !bytes.Equal(b.Peek(len(s)), []byte(s)) {
		t.Errorf("Expect peek %s but got %s", s, string(b.Peek(len(s))))
	}
}

func TestIoBufferReadOnce(t *testing.T) {
	// test small data
	b := newIoBuffer(1)
	s := randString(25)
	reader := bytes.NewReader([]byte(s))

	n, err := b.ReadOnce(reader)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != len(s) {
		t.Errorf("Expect %d bytes but got %d", len(s), n)
	}
	if b.Cap() != 1<<minShift {
		t.Errorf("Expect %d bytes but got %d", 1<<minShift, b.Cap())
	}

	if !bytes.Equal(b.Bytes(), []byte(s)) {
		t.Errorf("Expect got %s but got %s", s, b.Bytes())
	}

	// test big data
	b = newIoBuffer(1)
	bsize := 1025
	s = randString(bsize)
	reader = bytes.NewReader([]byte(s))

	// first read  1<<minShift
	n, err = b.ReadOnce(reader)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != 1<<minShift {
		t.Errorf("Expect %d bytes but got %d", 1<<minShift, n)
	}
	// expand buf
	if b.Cap() != MinRead<<1 {
		t.Errorf("Expect %d bytes but got %d", MinRead<<1, b.Cap())
	}

	//second read  MinRead<<1 - 1<<minShift
	n, err = b.ReadOnce(reader)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != MinRead<<1-1<<minShift {
		t.Errorf("Expect %d bytes but got %d", MinRead<<1-1<<minShift, n)
	}
	// expand buf
	if b.Cap() != MinRead<<2 {
		t.Errorf("Expect %d bytes but got %d", MinRead<<2, b.Cap())
	}

	// third read  bsize - MinRead<<1
	n, err = b.ReadOnce(reader)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != bsize-MinRead<<1 {
		t.Errorf("Expect %d bytes but got %d", bsize-MinRead<<1-1<<minShift, n)
	}
	// not expand buf
	if b.Cap() != MinRead<<2 {
		t.Errorf("Expect %d bytes but got %d", MinRead<<2, b.Cap())
	}
	if !bytes.Equal(b.Bytes(), []byte(s)) {
		t.Errorf("Expect got %s but got %s", s, b.Bytes())
	}

}

func TestIoBufferClone(t *testing.T) {
	for i := 16; i < 1024+16; i++ {
		s := randString(i)
		buffer := NewIoBufferString(s)
		nb := buffer.Clone()
		if nb.String() != s {
			t.Errorf("Clone() expect %s but got %s", s, nb.String())
		}
	}
}

func TestIoBufferCut(t *testing.T) {
	for i := 16; i < 1024+16; i++ {
		s := randString(i)
		bi := NewIoBufferString(s)
		b := bi.(*ioBuffer)
		offset := randN(i) - 1
		nb := b.Cut(offset)
		if nb.String() != s[:offset] {
			t.Errorf("Cut(%d) expect %s but got %s", offset, s[:offset], nb.String())
		}
	}
}

func TestIoBufferAllocAndFree(t *testing.T) {
	b := newIoBuffer(0)
	for i := 0; i < 1024; i++ {
		b.Alloc(i)
		if b.Cap() < i {
			t.Errorf("Expect alloc at least %d bytes but allocated %d", i, b.Cap())
		}
	}

	b.Reset()

	for i := 0; i < 1024; i++ {
		b.Alloc(i)
		if b.Cap() < i {
			t.Errorf("Expect alloc at least %d bytes but allocated %d", i, b.Cap())
		}
		b.Free()
		if b.Cap() != 0 {
			t.Errorf("Expect free to 0 bytes but got %d", b.Cap())
		}
	}
}

func TestIoBufferZero(t *testing.T) {
	writer := bytes.NewBuffer(nil)
	b := newIoBuffer(0)
	_, err := b.WriteTo(writer)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.Bytes()) != 0 {
		t.Errorf("Expect 0, but got %s", string(writer.Bytes()))
	}

	b = NewIoBufferBytes(nil)
	_, err = b.WriteTo(writer)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.Bytes()) != 0 {
		t.Errorf("Expect 0, but got %s", string(writer.Bytes()))
	}

	b = NewIoBufferString("")
	_, err = b.WriteTo(writer)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.Bytes()) != 0 {
		t.Errorf("Expect 0, but got %s", string(writer.Bytes()))
	}

	b = NewIoBufferEOF()
	_, err = b.WriteTo(writer)
	if err != nil {
		t.Fatal(err)
	}

	if len(writer.Bytes()) != 0 {
		t.Errorf("Expect 0, but got %s", string(writer.Bytes()))
	}

	b = newIoBuffer(0)

	if b.String() != "" {
		t.Errorf("Expect \"\", but got %s", string(b.String()))
	}

	if len(b.Bytes()) != 0 {
		t.Errorf("Expect 0, but got %d", len(b.Bytes()))
	}

	if len(b.Peek(0)) != 0 {
		t.Errorf("Expect 0, but got %d", len(b.Bytes()))
	}
}

func TestIoBufferMaxBufferReadOnce(t *testing.T) {
	b := newIoBuffer(1)
	s := randString(MaxBufferLength + 1)
	input := make([]byte, 0, 1024)
	reader := bytes.NewReader([]byte(s))
	countbytes := 0
	for {
		n, err := b.ReadOnce(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		countbytes = countbytes + int(n)
		if countbytes >= MaxBufferLength {
			input = append(input, b.Peek(int(countbytes))...)
			b.Drain(int(countbytes))
			countbytes = 0
		}
	}
	if countbytes > 0 {
		input = append(input, b.Peek(int(countbytes))...)
		b.Drain(int(countbytes))
	}
	if !bytes.Equal(input, []byte(s)) {
		t.Errorf("Expect got %s but got %s", s, string(input))
	}
	if b.Cap() > MaxBufferLength {
		t.Errorf("Expect got length %d", b.Cap())
	}
}

func TestPipe_CloseWithError(t *testing.T) {
	pipe := NewPipeBuffer(0)
	var w sync.WaitGroup
	w.Add(1)
	go func() {
		defer w.Done()
		bs := make([]byte, 100)
		_, err := pipe.Read(bs)
		if err != io.EOF {
			t.Fatal(err)
		}

	}()
	time.Sleep(1000)
	pipe.CloseWithError(io.EOF)
	w.Wait()
}

func TestPipe_ReadAndWrite(t *testing.T) {
	pipe := NewPipeBuffer(0)
	var w sync.WaitGroup
	bbs := []byte("aaabbbbvvvbbbbvvv")
	w.Add(1)
	go func() {
		defer w.Done()
		bs := make([]byte, len(bbs))
		_, _ = pipe.Read(bs)
		if !bytes.Equal(bs, bbs) {
			t.Fatalf("test failed")
		}
	}()
	_, _ = pipe.Write(bbs)
	w.Wait()
}
