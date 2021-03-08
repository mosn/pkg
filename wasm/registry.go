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

package wasm

import (
	"errors"
	"mosn.io/pkg/wasm/api"
)

var (
	ErrInvalidParam = errors.New("invalid param")
)

var vmMap = make(map[string]api.WasmVM)

// RegisterWasmEngine registers a wasm vm(engine).
func RegisterWasmEngine(name string, engine api.WasmVM) error {
	if name == "" || engine == nil {
		return ErrInvalidParam
	}

	vmMap[name] = engine

	return nil
}

// GetWasmEngine returns the wasm vm(engine) by name.
func GetWasmEngine(name string) api.WasmVM {
	if engine, ok := vmMap[name]; ok {
		return engine
	}

	return nil
}
