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

//nolint
package variable

import (
	"context"
	"errors"
	"fmt"

	"mosn.io/api"
)

var (
	errUnregisterProtocolResource = "unregister Protocol resource, Protocol: "
	protocolVar                   map[string]string
)

func init() {
	protocolVar = make(map[string]string)
}

// RegisterProtocolResource registers the resource as ProtocolResourceName
// forexample protocolVar[Http1+api.URI] = http_request_uri var
func RegisterProtocolResource(protocol api.ProtocolName, resource api.ProtocolResourceName, varname string) error {
	pr := convert(protocol, resource)
	if _, ok := protocolVar[pr]; ok {
		return errors.New("protocol resource already exists, name: " + pr)
	}

	protocolVar[pr] = fmt.Sprintf("%s_%s", protocol, varname)

	return nil
}

func convert(p api.ProtocolName, name api.ProtocolResourceName) string {
	return string(p) + string(name)
}

// GetProtocol returns the protocol name in the context.
// This allows user defines the way to get protocol.
// If the GetProtocol is undefined, the GetProtocolResource always returns an error.
var GetProtocol func(ctx context.Context) (api.ProtocolName, error)

// GetProtocolResource get URI,PATH,ARG var depends on ProtocolResourceName
func GetProtocolResource(ctx context.Context, name api.ProtocolResourceName, data ...interface{}) (string, error) {
	if GetProtocol == nil {
		return "", errNoGetProtocol
	}
	p, err := GetProtocol(ctx)
	if err != nil {
		return "", err
	}
	if v, ok := protocolVar[convert(p, name)]; ok {
		// apend data behind if data exists
		if len(data) == 1 {
			v = fmt.Sprintf("%s%s", v, data[0])
		}

		return GetString(ctx, v)
	}
	return "", errors.New(errUnregisterProtocolResource + string(p))
}
