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
	"runtime"
	"strings"
	"sync"

	mosnctx "mosn.io/pkg/internal/context"
	"mosn.io/pkg/log"
)

var (
	// global scope
	mux              sync.RWMutex
	variables        = make(map[string]Variable, 32) // all built-in variable definitions
	prefixVariables  = make(map[string]Variable, 32) // all prefix getter definitions
	indexedVariables = make([]Variable, 0, 32)       // indexed variables

	// error message
	errVariableDuplicated   = "duplicate variable register, name: "
	errPrefixDuplicated     = "duplicate prefix variable register, prefix: "
	errUndefinedVariable    = "undefined variable, name: "
	errInvalidContext       = "invalid context"
	errNoVariablesInContext = "no variables found in context"
	errSupportIndexedOnly   = "this operation only support indexed variable"
	errSetterNotFound       = "setter function undefined, variable name: "
	errValueNotFound        = "variable value not found, variable name: "
	errVariableNotString    = "variable type is not string"
	errValueNotString       = "set string variable with non-string type"
	invalidVariableIndex    = errors.New("get variable support name index or variable directly")
	errNoGetProtocol        = errors.New("no way to get protocol, get protocol resource variable failed")
)

// ResetVariableForTest is a test function for reset the variables.
// DONOT call it in any non-test functions
func ResetVariableForTest() {
	mux.Lock()
	defer mux.Unlock()

	variables = make(map[string]Variable, 32)
	prefixVariables = make(map[string]Variable, 32)
	indexedVariables = make([]Variable, 0, 32)
}

// Check return the variable related to name, return error if not registered
// nolint
func Check(name string) (Variable, error) {
	mux.Lock()
	defer mux.Unlock()

	// find built-in variables
	if variable, ok := variables[name]; ok {
		return variable, nil
	}

	// check prefix variables
	for prefix, variable := range prefixVariables {
		if strings.HasPrefix(name, prefix) {
			return variable, nil

			// todo: index fast-path solution
			//// make it into indexed variables
			//indexed := NewStringVariable(name, name, variable.Getter(), variable.Setter(), variable.Flags())
			//// register indexed one
			//if err := Register(indexed); err != nil {
			//      return nil, err
			//}
			//return indexed, nil
		}
	}

	return nil, errors.New(errUndefinedVariable + name)
}

// Register a new variable
func Register(variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	name := variable.Name()

	// check conflict
	if old, ok := variables[name]; ok {
		oldCaller := ""
		if recorder, ok := old.(CallerRecorder); ok {
			oldCaller = recorder.GetCaller()
		}

		log.DefaultLogger.Warnf("[variable] duplicate register variable: %s, the last one is registered by: %v",
			name, oldCaller)
	}

	if recoder, ok := variable.(CallerRecorder); ok {
		if _, file, _, ok := runtime.Caller(1); ok {
			recoder.SetCaller(file)
		}
	}

	// register
	variables[name] = variable

	// check index
	if indexer, ok := variable.(Indexer); ok {
		index := len(indexedVariables)
		indexer.SetIndex(uint32(index))

		indexedVariables = append(indexedVariables, variable)
	}
	return nil
}

// Register a new variable with prefix
func RegisterPrefix(prefix string, variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	// check conflict
	if old, ok := prefixVariables[prefix]; ok {
		oldCaller := ""
		if recorder, ok := old.(CallerRecorder); ok {
			oldCaller = recorder.GetCaller()
		}

		log.DefaultLogger.Warnf("[variable] duplicate register prefix variable: %s, the last one is registered by: %v",
			prefix, oldCaller)
	}

	if recoder, ok := variable.(CallerRecorder); ok {
		if _, file, _, ok := runtime.Caller(1); ok {
			recoder.SetCaller(file)
		}
	}

	// register
	prefixVariables[prefix] = variable
	return nil
}

//nolint
func NewVariableContext(ctx context.Context) context.Context {
	// TODO: sync.Pool reuse
	values := make([]IndexedValue, len(indexedVariables)) // TODO: pre-alloc buffer for runtime variable

	// Inherit index variables from parent
	v := mosnctx.Get(ctx, mosnctx.KeyVariables)
	if ivalues, ok := v.([]IndexedValue); ok {
		copy(values, ivalues)
	}

	return mosnctx.WithValue(mosnctx.Clone(ctx), mosnctx.KeyVariables, values)
}
