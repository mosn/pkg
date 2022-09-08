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
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"mosn.io/api"
)

const (
	HTTP1        api.ProtocolName = "Http1"
	Dubbo        api.ProtocolName = "Dubbo"
	ProtocolName                  = "protocol"
)

func TestMain(m *testing.M) {
	GetProtocol = func(ctx context.Context) (api.ProtocolName, error) {
		v := ctx.Value(ProtocolName)
		if proto, ok := v.(api.ProtocolName); ok {
			return proto, nil
		}
		return api.ProtocolName(""), errors.New("no protocol found")
	}
	os.Exit(m.Run())
}

func newVariableContextWithProtocol(proto api.ProtocolName) context.Context {
	ctx := context.WithValue(context.Background(), ProtocolName, proto)
	return NewVariableContext(ctx)
}

func TestGetProtocolResource(t *testing.T) {
	request_path := "request_path"

	httpKey := string(HTTP1) + "_" + request_path
	dubboKey := string(Dubbo) + "_" + request_path
	m := map[string]string{
		httpKey:  "/http",
		dubboKey: "/dubbo",
	}

	for k, _ := range m {
		val := m[k]
		// register test variable
		Register(NewStringVariable(k, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
			return val, nil
		}, nil, 0))
	}

	// register HTTP protocol resource var
	RegisterProtocolResource(HTTP1, api.PATH, request_path)

	ctx := newVariableContextWithProtocol(HTTP1)
	vv, err := GetProtocolResource(ctx, api.PATH)
	require.Nil(t, err)
	require.Equal(t, m[httpKey], vv)

	ctx = newVariableContextWithProtocol(Dubbo)
	vv, err = GetProtocolResource(ctx, api.PATH)
	require.EqualError(t, err, errUnregisterProtocolResource+string(Dubbo))
}

func BenchmarkGetProtocolResource(b *testing.B) {
	ctx := prepareProtocolResource()
	for i := 0; i < b.N; i++ {
		_, err := GetProtocolResource(ctx, api.PATH)
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func BenchmarkGetValue(b *testing.B) {

	ctx := prepareProtocolResource()
	for i := 0; i < b.N; i++ {
		_, err := GetString(ctx, string(api.PATH))
		if err != nil {
			b.Error("get variable failed:", err)
		}
	}
}

func prepareProtocolResource() context.Context {
	name := "http_request_path"
	value := "/path"

	// register test variable
	Register(NewStringVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		return value, nil
	}, nil, 0))

	// register HTTP protocol resource var
	RegisterProtocolResource(HTTP1, api.PATH, name)

	ctx := newVariableContextWithProtocol(HTTP1)
	return ctx
}
