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

	"github.com/stretchr/testify/assert"
)

func TestGetVariableValue_normal(t *testing.T) {
	name := "ApiGet"
	value := "Getter Result"

	// register test variable
	Register(NewStringVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		return value, nil
	}, nil, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	vv, err := GetString(ctx, name)
	if err != nil {
		t.Error(err)
	}

	if vv != value {
		t.Errorf("get value not equal, expected: %s, acutal: %s", value, vv)
	}

	// test Check
	if _, err := Check(name); err != nil {
		t.Errorf("Check existed variable failed：%v", err)
	}

	// test prefix variable
	name = "prefix_var_"
	value = "prefix value"
	RegisterPrefix(name, NewStringVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		return value, nil
	}, nil, 0))

	vv, err = GetString(ctx, name)
	if err != nil {
		t.Error(err)
	}

	if vv != value {
		t.Errorf("get prefix variable value not equal, expected: %s, acutal: %s", value, vv)
	}

	// test Check
	if _, err := Check(name); err != nil {
		t.Errorf("Check existed variable failed：%v", err)
	}

	name = "unknown"
	if _, err := Check(name); err == nil {
		t.Error("Check unknown variable failed")
	}

}

func TestSetShouldInvalidateCachedValue(t *testing.T) {
	name := "SetShouldInvalidateCachedValue"
	value := "1"

	getter := func(ctx context.Context, v *IndexedValue, data interface{}) (string, error) {
		return value, nil
	}

	setter := func(ctx context.Context, variableValue *IndexedValue, v string) error {
		value = v
		return nil
	}

	// register test variable
	Register(NewStringVariable(name, nil, getter, setter, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	s, _ := GetString(ctx, name) // make sure the value is cached
	assert.Equal(t, s, "1")

	_ = SetString(ctx, name, "2") // set to a new value, the cached value should be invalidated
	assert.Equal(t, value, "2")

	s, _ = GetString(ctx, name)
	assert.Equal(t, s, "2")
}

func TestSetVariableValue_normal(t *testing.T) {
	name := "ApiSet"
	value := "Setter Value"

	// register test variable
	Register(NewStringVariable(name, nil, nil, DefaultStringSetter, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	err := SetString(ctx, name, value)
	if err != nil {
		t.Error(err)
	}

	vv, err := GetString(ctx, name)
	if err != nil {
		t.Error(err)
	}

	if vv != value {
		t.Errorf("get/set value not equal, expected: %s, acutal: %s", value, vv)
	}

	ii, err := Get(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, ii.(string), value)
}

func TestInterfaceVariableGetter(t *testing.T) {
	name := "testInterfaceGetter"
	value := struct{}{}

	getter := func(ctx context.Context, v *IndexedValue, data interface{}) (interface{}, error) {
		return value, nil
	}
	Register(NewVariable(name, nil, getter, nil, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	vv, err := Get(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, vv, value)
}

func TestInterfaceVariableSetter(t *testing.T) {
	name := "testInterfaceSetter"
	value := struct{}{}

	getter := func(ctx context.Context, v *IndexedValue, data interface{}) (interface{}, error) {
		return value, nil
	}
	Register(NewVariable(name, nil, getter, DefaultSetter, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	vv, err := Get(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, vv, value)

	// set int
	err = Set(ctx, name, int(1))
	assert.Nil(t, err)

	i, err := Get(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, i.(int), 1)

	// set string
	err = Set(ctx, name, "someString")
	assert.Nil(t, err)

	s, err := Get(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, s.(string), "someString")
}

func TestVarNotGetterHint(t *testing.T) {
	name := "testVarNotGetterHint"
	Register(NewVariable(name, nil, nil, DefaultSetter, 0))

	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	_, err := Get(ctx, name)
	assert.Equal(t, err.Error(), errValueNotFound+name)

	_, err2 := Get(ctx, name)
	assert.Equal(t, err2.Error(), errValueNotFound+name)
}

func TestVariableGetSetCached(t *testing.T) {
	name := "cache getter"
	cacheValue := "cached"
	getterCall := 0
	Register(NewStringVariable(name, nil, func(ctx context.Context, variableValue *IndexedValue, data interface{}) (s string, err error) {
		getterCall++
		return cacheValue, nil
	}, DefaultStringSetter, 0))
	ctx := NewVariableContext(context.Background())
	value, err := GetString(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, cacheValue, value)
	assert.Equal(t, 1, getterCall)
	// get from cache
	value, err = GetString(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, cacheValue, value)
	assert.Equal(t, 1, getterCall)
	// set will overwrite cache
	SetString(ctx, name, "overwrite")
	value, err = GetString(ctx, name)
	assert.Nil(t, err)
	assert.Equal(t, "overwrite", value)
	assert.Equal(t, 1, getterCall)

}

func BenchmarkGetVariableValue2(b *testing.B) {
	name := "benchmarkGet"
	value := "someValue"

	_ = Register(NewStringVariable(name, nil, nil, DefaultStringSetter, 0))
	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	_ = SetString(ctx, name, value)

	for i := 0; i < b.N; i++ {
		_, _ = GetString(ctx, name)
	}
}

func BenchmarkSetVariableValue2(b *testing.B) {
	name := "benchmarkSet"
	value := "someValue"

	_ = Register(NewStringVariable(name, nil, nil, DefaultStringSetter, 0))
	ctx := context.Background()
	ctx = NewVariableContext(ctx)

	_ = SetString(ctx, name, value)

	for i := 0; i < b.N; i++ {
		_ = SetString(ctx, name, value)
	}
}
