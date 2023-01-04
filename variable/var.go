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
)

// NewVariable creates a variable with name
func NewVariable(name string, data interface{}, getter GetterFunc, setter SetterFunc, flags uint32) Variable {
	basic := BasicVariable{
		getter: &getterImpl{name: name, getter: getter},
		setter: &setterImpl{name: name, setter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	if setter != nil {
		return &IndexedVariable{BasicVariable: basic}
	}

	return &basic
}

// DefaultSetter used for interface-typed variable value setting
func DefaultSetter(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	variableValue.data = value
	variableValue.Valid = true
	return nil
}

// NewStringVariable is a wrapper for NewVariable
// nolint:lll
func NewStringVariable(name string, data interface{}, getter StringGetterFunc, setter StringSetterFunc, flags uint32) Variable {
	basic := BasicVariable{
		getter: &getterImpl{name: name, strGetter: getter},
		setter: &setterImpl{name: name, strSetter: setter},
		name:   name,
		data:   data,
		flags:  flags,
	}

	if setter != nil {
		return &IndexedVariable{BasicVariable: basic}
	}

	return &basic
}

// DefaultStringSetter used for string-typed variable value setting only, and would not affect any real data structure, like headers.
func DefaultStringSetter(ctx context.Context, variableValue *IndexedValue, value string) error {
	return DefaultSetter(ctx, variableValue, value)
}

// variable.Variable
type BasicVariable struct {
	getter *getterImpl
	setter *setterImpl

	name  string
	data  interface{}
	flags uint32
}

// Name returns variable's name
func (bv *BasicVariable) Name() string {
	return bv.name
}

// Data returns variable's value
func (bv *BasicVariable) Data() interface{} {
	return bv.data
}

// Getter is the variable's value get function if the variable contains it
func (bv *BasicVariable) Getter() Getter {
	return bv.getter
}

// Setter is the variable's value set function if the variable contains it
func (bv *BasicVariable) Setter() Setter {
	return bv.setter
}

// IndexedVariable contains index for set search
type IndexedVariable struct {
	BasicVariable

	index uint32
}

// SetIndex sets the variable's index for search
func (iv *IndexedVariable) SetIndex(index uint32) {
	iv.index = index
}

// GetIndex returns the variable's index for search
func (iv *IndexedVariable) GetIndex() uint32 {
	return iv.index
}

type setterImpl struct {
	name      string
	strSetter StringSetterFunc
	setter    SetterFunc
}

func (s *setterImpl) Set(ctx context.Context, variableValue *IndexedValue, value interface{}) error {
	if s.strSetter != nil {
		if v, ok := value.(string); ok {
			return s.strSetter(ctx, variableValue, v)
		}
		return errors.New(errValueNotString)
	}

	if s.setter != nil {
		return s.setter(ctx, variableValue, value)
	}

	return errors.New(errSetterNotFound + s.name)
}

type getterImpl struct {
	name      string
	strGetter StringGetterFunc
	getter    GetterFunc
}

func (g *getterImpl) Get(ctx context.Context, value *IndexedValue, data interface{}) (interface{}, error) {
	if g.strGetter != nil {
		return g.strGetter(ctx, value, data)
	}

	if g.getter != nil {
		return g.getter(ctx, value, data)
	}

	return nil, errors.New(errValueNotFound + g.name)
}
