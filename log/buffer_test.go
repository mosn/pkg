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
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/api"
)

func TestLogBufferPool(t *testing.T) {
	buf := GetLogBuffer(0)
	bufaddr := fmt.Sprintf("%p", buf.buffer())
	PutLogBuffer(buf)
	buf2 := GetLogBuffer(0)
	buf2addr := fmt.Sprintf("%p", buf2.buffer())
	require.Equal(t, bufaddr, buf2addr)
}

// use an interface to benchmark
// use struct directly is better than use an interface or a pointer
type iLogBuf interface {
	buffer() api.IoBuffer
}

func getILogBuf() iLogBuf {
	return GetLogBuffer(0)
}

func getILogBufByPointer() iLogBuf {
	return getLogBufferPointer()
}

func putILogBuf(buf iLogBuf) {
	logPool.PutIoBuffer(buf.buffer())
}

func getLogBufferPointer() *LogBuffer {
	return &LogBuffer{
		logbuffer: logPool.GetIoBuffer(0),
	}
}

func putLogBufferPointer(bp *LogBuffer) {
	logPool.PutIoBuffer(bp.buffer())
}

func BenchmarkLogBuffer(b *testing.B) {
	// use channel write & read to mock log print
	b.Run("raw iobuffer", func(b *testing.B) {
		ch := make(chan api.IoBuffer, 1)
		for i := 0; i < b.N; i++ {
			func(bf api.IoBuffer) {
				ch <- bf
				buf := <-ch
				logPool.PutIoBuffer(buf)
			}(logPool.GetIoBuffer(0))
		}
	})

	b.Run("struct", func(b *testing.B) {
		ch := make(chan LogBuffer, 1)
		for i := 0; i < b.N; i++ {
			func(bf LogBuffer) {
				ch <- bf
				buf := <-ch
				PutLogBuffer(buf)
			}(GetLogBuffer(0))
		}
	})

	b.Run("pointer", func(b *testing.B) {
		ch := make(chan *LogBuffer, 1)
		for i := 0; i < b.N; i++ {
			func(bf *LogBuffer) {
				ch <- bf
				bp := <-ch
				putLogBufferPointer(bp)
			}(getLogBufferPointer())
		}
	})

	b.Run("struct interface", func(b *testing.B) {
		ch := make(chan iLogBuf, 1)
		for i := 0; i < b.N; i++ {
			func(bf iLogBuf) {
				ch <- bf
				buf := <-ch
				putILogBuf(buf)
			}(getILogBuf())
		}
	})

	b.Run("pointer interface", func(b *testing.B) {
		ch := make(chan iLogBuf, 1)
		for i := 0; i < b.N; i++ {
			func(bf iLogBuf) {
				ch <- bf
				buf := <-ch
				putILogBuf(buf)
			}(getILogBufByPointer())
		}
	})
}
