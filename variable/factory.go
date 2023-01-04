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
	errVariableNotRegister  = "override unregistered variable, name: "
	errPrefixDuplicated     = "duplicate prefix variable register, prefix: "
	errPrefixNotRegister    = "override unregistered prefix variable, prefix: "
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
	if _, ok := variables[name]; ok {
		log.DefaultLogger.Errorf("[variable] duplicate register variable: %s", name)
		return errors.New(errVariableDuplicated + name)
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

// Override a variable, return error if the variable haven't been registered
func Override(variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	name := variable.Name()

	// ensure already registered
	oldVar, ok := variables[name]
	if !ok {
		log.DefaultLogger.Errorf("[variable] override unregistered variable: %s", name)
		return errors.New(errVariableNotRegister + name)
	}

	// override
	variables[name] = variable

	// check index
	if newIndexer, ok := variable.(Indexer); ok {
		if oldIndexer, ok := oldVar.(Indexer); ok {  // reuse old index
			index := oldIndexer.GetIndex()
			newIndexer.SetIndex(index)

			indexedVariables[index] = variable
		} else {
			index := len(indexedVariables) // assign a new index
			newIndexer.SetIndex(uint32(index))

			indexedVariables = append(indexedVariables, variable)
		}
	}
	return nil
}


// Register a new variable with prefix
func RegisterPrefix(prefix string, variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	// check conflict
	if _, ok := prefixVariables[prefix]; ok {
		return errors.New(errPrefixDuplicated + prefix)
	}

	// register
	prefixVariables[prefix] = variable
	return nil
}

// Override a variable with prefix, return error if the variable haven't been registered
func OverridePrefix(prefix string, variable Variable) error {
	mux.Lock()
	defer mux.Unlock()

	// ensure already registered
	if _, ok := prefixVariables[prefix]; !ok {
		return errors.New(errPrefixNotRegister + prefix)
	}

	// override
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
