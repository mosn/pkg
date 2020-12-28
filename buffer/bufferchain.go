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
	"errors"
	"io"
	"sync"
	"time"
)

// ErrWriteCovered bufferchain queue is full.
var ErrWriteCovered = errors.New("chain write covered")

const defaultCapacity = 1 << 9

/*
 * ioBufferchain
 * For HTTP2 stream, in order not to break the structure-adaptation interface.
 */
type ioBufferchain struct {
	bufferchain chan []byte
	errChan     chan error
	mutex       sync.Mutex
}

// NewIoBufferChain returns bufferChain.
func NewIoBufferChain(capacity int) IoBuffer {
	if capacity == 0 {
		capacity = defaultCapacity
	}

	return &ioBufferchain{
		bufferchain: make(chan []byte, capacity),
		errChan:     make(chan error),
	}
}

func (bc *ioBufferchain) Bytes() (p []byte) {
	p, b := <-bc.bufferchain
	if !b {
		return nil
	}

	return p
}

func (bc *ioBufferchain) Write(p []byte) (n int, err error) {
	bytes := *GetBytes(len(p))
	copy(bytes, p)
	select {
	case <-bc.errChan:
		return 0, io.EOF
	default:
		bc.mutex.Lock()
		defer bc.mutex.Unlock()
		select {
		case bc.bufferchain <- bytes:
			return len(bytes), nil
		default:
			// chain is full conn goutine wait 1s to consumer
			ticker := time.NewTicker(1 * time.Second)
			select {
			case <-bc.errChan:
				return 0, io.EOF
			case bc.bufferchain <- bytes:
				ticker.Stop()

				return len(bytes), nil
			case <-ticker.C:
				return 0, ErrWriteCovered
			}
		}
	}
}

func (bc *ioBufferchain) CloseWithError(_ error) {
	select {
	case <-bc.errChan:
		return
	default:
		close(bc.errChan)
		bc.mutex.Lock()
		defer bc.mutex.Unlock()
		close(bc.bufferchain)
	}
}

func (bc *ioBufferchain) Count(int32) int32 {
	return 1
}

func (bc *ioBufferchain) Len() int {
	return 0
}

func (bc *ioBufferchain) Read(p []byte) (n int, err error) {
	return 0, EOF
}

func (bc *ioBufferchain) ReadOnce(r io.Reader) (n int64, err error) {
	return 0, EOF
}

func (bc *ioBufferchain) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, EOF
}

func (bc *ioBufferchain) Grow(n int) error {
	return EOF
}

func (bc *ioBufferchain) WriteString(s string) (n int, err error) {
	return 0, EOF
}

func (bc *ioBufferchain) WriteByte(p byte) error {
	return EOF
}

func (bc *ioBufferchain) WriteUint16(p uint16) error {
	return EOF
}

func (bc *ioBufferchain) WriteUint32(p uint32) error {
	return EOF
}

func (bc *ioBufferchain) WriteUint64(p uint64) error {
	return EOF
}

func (bc *ioBufferchain) WriteTo(w io.Writer) (n int64, err error) {
	return 0, EOF
}

func (bc *ioBufferchain) Peek(n int) []byte {
	return nil
}

func (bc *ioBufferchain) Drain(offset int) {
}

func (bc *ioBufferchain) Cap() int {
	return 0
}

func (bc *ioBufferchain) Reset() {}

func (bc *ioBufferchain) Clone() IoBuffer {
	return nil
}

func (bc *ioBufferchain) String() string {
	return ""
}

func (bc *ioBufferchain) Alloc(int) {
}

func (bc *ioBufferchain) Free() {
}

func (bc *ioBufferchain) EOF() bool {
	return true
}

func (bc *ioBufferchain) SetEOF(eof bool) {
}

func (bc *ioBufferchain) Append(data []byte) error {
	return EOF
}
