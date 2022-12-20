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

	"mosn.io/api"
	mosnctx "mosn.io/pkg/internal/context"
)

// GetString return the value of string-typed variable
func GetString(ctx context.Context, v interface{}) (string, error) {
	v, err := Get(ctx, v)
	if err != nil {
		return "", err
	}

	if s, ok := v.(string); ok {
		return s, nil
	}

	return "", errors.New(errVariableNotString)
}

// SetString set the value of string-typed variable
func SetString(ctx context.Context, v interface{}, value string) error {
	if ctx == nil {
		return errors.New(errInvalidContext)
	}

	return Set(ctx, v, value)
}

// Get the value of variable.
func Get(ctx context.Context, i interface{}) (interface{}, error) {
	switch v := i.(type) {
	case string:
		return getByName(ctx, v)
	case Variable:
		return getByVariable(ctx, v)
	default:
		return nil, invalidVariableIndex
	}
}

// getByVariable returns the value of variable in the context
func getByVariable(ctx context.Context, variable Variable) (interface{}, error) {
	// 1.1 check indexed value
	if indexer, ok := variable.(Indexer); ok {
		return getFlushedValue(ctx, indexer.GetIndex())
	}
	// 1.2 use variable.Getter() to get value
	getter := variable.Getter()
	if getter == nil {
		return "", errors.New(errValueNotFound + variable.Name())
	}
	return getter.Get(ctx, nil, variable.Data())
}

// get the value of variable by name
func getByName(ctx context.Context, name string) (interface{}, error) {
	// 1. find built-in variables
	if variable, ok := variables[name]; ok {
		return getByVariable(ctx, variable)
	}

	// 2. find prefix variables
	for prefix, variable := range prefixVariables {
		if strings.HasPrefix(name, prefix) {
			getter := variable.Getter()
			if getter == nil {
				return "", errors.New(errValueNotFound + name)
			}
			return getter.Get(ctx, nil, name)
		}
	}

	// 3. find protocol resource variables
	if v, e := GetProtocolResource(ctx, api.ProtocolResourceName(name)); e == nil {
		return v, nil
	}

	return "", errors.New(errUndefinedVariable + name)
}

// Set the value of variable.
func Set(ctx context.Context, i interface{}, value interface{}) error {
	switch v := i.(type) {
	case string:
		return setByName(ctx, v, value)
	case Variable:
		return setByVariable(ctx, v, value)
	default:
		return invalidVariableIndex
	}
}

// SetVariable sets value into variable in context
func setByVariable(ctx context.Context, variable Variable, value interface{}) error {
	// 1.1 check indexed value
	if indexer, ok := variable.(Indexer); ok {
		return setFlushedValue(ctx, indexer.GetIndex(), value)
	}
	return errors.New(errSupportIndexedOnly + ": set variable value")
}

// Set the value of variable by name
func setByName(ctx context.Context, name string, value interface{}) error {
	if ctx == nil {
		return errors.New(errInvalidContext)
	}

	// find built-in & indexed variables, prefix and non-indexed are not supported
	if variable, ok := variables[name]; ok {
		return setByVariable(ctx, variable, value)
	}

	return errors.New(errSupportIndexedOnly + ": set variable value")
}

// TODO: provide direct access to this function, so the cost of variable name finding could be optimized
func getFlushedValue(ctx context.Context, index uint32) (interface{}, error) {
	if variables := ctx.Value(mosnctx.KeyVariables); variables != nil {
		if values, ok := variables.([]IndexedValue); ok {
			value := &values[index]
			if value.Valid {
				return value.data, nil

			}

			return getIndexedValue(ctx, value, index)
		}
	}

	return "", errors.New(errNoVariablesInContext)
}

func getIndexedValue(ctx context.Context, value *IndexedValue, index uint32) (interface{}, error) {
	variable := indexedVariables[index]

	getter := variable.Getter()
	if getter == nil {
		return "", errors.New(errValueNotFound + variable.Name())
	}
	vdata, err := getter.Get(ctx, value, variable.Data())
	if err != nil {
		value.Valid = false
		return vdata, err
	}

	value.data = vdata
	value.Valid = true
	return value.data, nil
}

func setFlushedValue(ctx context.Context, index uint32, value interface{}) error {
	if variables := ctx.Value(mosnctx.KeyVariables); variables != nil {
		if values, ok := variables.([]IndexedValue); ok {
			variable := indexedVariables[index]
			variableValue := &values[index]

			setter := variable.Setter()
			if setter == nil {
				return errors.New(errSetterNotFound + variable.Name())
			}

			// should invalidate the cached value before setting it to a new one
			variableValue.Valid = false
			return setter.Set(ctx, variableValue, value)
		}
	}

	return errors.New(errNoVariablesInContext)
}
