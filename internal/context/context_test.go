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

package context

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNodeNum = 10

var randomTable [testNodeNum]Key

func TestMain(m *testing.M) {
	// init random table per-run for all benchmark scenario, so the performance will not be affected by random functions.
	for i := 0; i < testNodeNum; i++ {
		randomTable[i] = Key(rand.Intn(int(KeyEnd)))
	}
	os.Exit(m.Run())
}

func TestContext(t *testing.T) {
	ctx := context.Background()
	// Test Set
	expected := "test value"
	ctx = WithValue(ctx, KeyBufferPoolCtx, expected)
	// Test Get
	vi := ctx.Value(KeyBufferPoolCtx)
	value, ok := vi.(string)
	require.True(t, ok)
	require.Equal(t, expected, value)

	// parent is valueCtx, withValue test
	e2 := []string{"1", "2"}
	ctx2 := WithValue(ctx, KeyVariables, e2)
	vi2 := ctx2.Value(KeyVariables)
	value2, ok := vi2.([]string)
	require.True(t, ok)
	require.Len(t, value2, 2)

	// mosn context is different from the std context
	// if you add a k v in the child context
	// the parent context will also change
	pvi2 := ctx.Value(KeyVariables)
	pvalue2, ok := pvi2.([]string)
	require.True(t, ok)
	require.Len(t, pvalue2, 2)

	// not context type key, should go to the other branch of ctx.Value()
	invalid_value := ctx.Value("test")
	assert.Nil(t, invalid_value)

	// another way to get
	vi3 := Get(ctx, KeyBufferPoolCtx)
	value3, ok := vi3.(string)
	require.True(t, ok)
	require.Equal(t, expected, value3)

	// std context
	stdv := Get(context.TODO(), KeyVariables)
	assert.Nil(t, stdv)

	// Test Clone
	ctxNew := Clone(ctx)

	nvi := ctxNew.Value(KeyBufferPoolCtx)
	require.NotNil(t, nvi)

	// clone std context
	ctxBaseNew := context.TODO()
	ctxNew = Clone(ctxBaseNew)
	assert.Equal(t, ctxNew, ctxBaseNew)
}

func BenchmarkCompatibleGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			ctx.Value(randomTable[i])
		}
	}

}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			Get(ctx, randomTable[i])
		}
	}

}

func BenchmarkSet(b *testing.B) {
	// based on 10 k-v

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		for i := 0; i < testNodeNum; i++ {
			ctx = WithValue(ctx, randomTable[i], struct{}{})
		}
	}
}

func BenchmarkRawGet(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < testNodeNum; i++ {
		ctx = context.WithValue(ctx, randomTable[i], struct{}{})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// get all index
		for i := 0; i < testNodeNum; i++ {
			ctx.Value(randomTable[i])
		}
	}
}

func BenchmarkRawSet(b *testing.B) {
	// based on 10 k-v
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		for i := 0; i < testNodeNum; i++ {
			ctx = context.WithValue(ctx, randomTable[i], struct{}{})
		}

	}
}
