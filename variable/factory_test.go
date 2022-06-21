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

package variable

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testVarConnectionID = NewVariable("connection_id", nil, nil, DefaultSetter, 0)
	testVarStreamID     = NewVariable("stream_id", nil, nil, DefaultSetter, 0)
	testVarGetter       = NewStringVariable("test_getter", nil, func(ctx context.Context, _ *IndexedValue, _ interface{}) (string, error) {
		v := ctx.Value("test")
		return v.(string), nil
	}, nil, 0)
)

func init() {
	Register(testVarConnectionID)
	Register(testVarStreamID)
	Register(testVarGetter)
}

func _getIntVariable(ctx context.Context, v Variable) (int, bool) {
	i, err := GetVariable(ctx, v)
	if err != nil {
		return 0, false
	}
	value, ok := i.(int)
	return value, ok
}

// In MOSN case, some variables is based on connection
// some variables is based on stream.
// stream context should inherit variabels from connection,
// and should not take effect on connection. (with context Clone)
func TestNewVariableContext(t *testing.T) {

	connCtx := NewVariableContext(context.Background())
	SetVariable(connCtx, testVarConnectionID, 1)
	// stream context is inherit from connection context
	streamCtx := NewVariableContext(connCtx)
	SetVariable(streamCtx, testVarStreamID, 1)
	// verify

	// connection context should not get stream id
	cid, ok := _getIntVariable(connCtx, testVarConnectionID)
	require.True(t, ok)
	require.Equal(t, 1, cid)
	_, ok = _getIntVariable(connCtx, testVarStreamID)
	require.False(t, ok)

	// stream context should get connection id
	cid, ok = _getIntVariable(streamCtx, testVarConnectionID)
	require.True(t, ok)
	require.Equal(t, 1, cid)
	sid, ok := _getIntVariable(streamCtx, testVarStreamID)
	require.True(t, ok)
	require.Equal(t, 1, sid)

	// if stream context modify var in connection
	// should not take effect on connection context
	SetVariable(streamCtx, testVarConnectionID, 2)
	cid, ok = _getIntVariable(streamCtx, testVarConnectionID)
	require.True(t, ok)
	require.Equal(t, 2, cid)

	cid, ok = _getIntVariable(connCtx, testVarConnectionID)
	require.True(t, ok)
	require.Equal(t, 1, cid)

	// new stream context from connection context should be independent
	streamCtx2 := NewVariableContext(connCtx)
	cid, ok = _getIntVariable(streamCtx2, testVarConnectionID)
	require.True(t, ok)
	require.Equal(t, 1, sid)
	_, ok = _getIntVariable(streamCtx2, testVarStreamID)
	require.False(t, ok)
}

// if a variable is a getter variable, the context can not be variable context
// but the inherit is same as setter variable
func TestGetterVariable(t *testing.T) {
	ctx := context.WithValue(context.Background(), "test", "value")

	vi, err := GetVariable(ctx, testVarGetter)
	require.Nil(t, err)
	require.Equal(t, "value", vi.(string))

	newCtx := NewVariableContext(ctx)

	vi, err = GetVariable(newCtx, testVarGetter)
	require.Nil(t, err)
	require.Equal(t, "value", vi.(string))

	newCtx = context.WithValue(newCtx, "test", "new value")

	vi, err = GetVariable(newCtx, testVarGetter)
	require.Nil(t, err)
	require.Equal(t, "new value", vi.(string))

	vi, err = GetVariable(ctx, testVarGetter)
	require.Nil(t, err)
	require.Equal(t, "value", vi.(string))

}
