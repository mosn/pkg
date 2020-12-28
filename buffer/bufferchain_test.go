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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//Test write
func TestBufferWrite(t *testing.T) {
	chain := NewIoBufferChain(10)
	write := func(i *int32) error {
		bytes := make([]byte, 1)
		_, err := chain.Write(bytes)
		if err == nil {
			atomic.AddInt32(i, 1)
		}
		return err
	}
	var i int32
	go func() {
		var err error
		for i <= 20 {
			err = write(&i)
			if err != nil {
				break
			}
		}
		assert.Equal(t, i, int32(10))
		err = write(&i)
		assert.Error(t, err)
	}()
	time.Sleep(2 * time.Second)
	chain.CloseWithError(nil)
	assert.Equal(t, i, int32(10))
}

func TestBufferReade(t *testing.T) {
	chain := NewIoBufferChain(10)
	chain.Write(make([]byte, 1))
	chain.Write(make([]byte, 1))
	var i int32
	reader := func(i *int32) {
		_ = chain.Bytes()
		atomic.AddInt32(i, 1)
	}
	go func() {
		for {
			reader(&i)
		}
	}()
	time.Sleep(1 * time.Second)
	chain.CloseWithError(nil)
	assert.Equal(t, i, int32(2))
}
