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

import "context"

// some built-in variable names for common case
const (
	downStreamProtocol    = "bultin_variable_downstream_protocol"
	upstreamProtocol      = "bultin_variable_upstream_protocol"
	downStreamReqHeaders  = "bultin_variable_downstream_req_headers"
	downStreamRespHeaders = "bultin_variable_downstream_resp_headers"
	traceSpan             = "bultin_variable_trace_span"
)

// some built-in variables for common case
var (
	VarDownStreamProtocol    = NewVariable(downStreamProtocol, nil, nil, DefaultSetter, 0)
	VarUpstreamProtocol      = NewVariable(upstreamProtocol, nil, nil, DefaultSetter, 0)
	VarDownStreamReqHeaders  = NewVariable(downStreamReqHeaders, nil, nil, DefaultSetter, 0)
	VarDownStreamRespHeaders = NewVariable(downStreamRespHeaders, nil, nil, DefaultSetter, 0)
	VarTraceSpan             = NewVariable(traceSpan, nil, nil, DefaultSetter, 0)
)

func init() {
	builtinVariables := []Variable{
		VarDownStreamProtocol,
		VarUpstreamProtocol,
		VarDownStreamReqHeaders,
		VarDownStreamRespHeaders,
		VarTraceSpan,
	}
	for _, v := range builtinVariables {
		Register(v)
	}
}

// ContextSet is a function wrapper likes context usage
func ContextSet(ctx context.Context, v Variable, value interface{}) context.Context {
	_ = SetVariable(ctx, v, value)
	return ctx
}

// ContextGet is a functuon wrapper likes context usage
func ContextGet(ctx context.Context, v Variable) interface{} {
	value, err := GetVariable(ctx, v)
	if err != nil {
		return nil
	}
	return value

}
