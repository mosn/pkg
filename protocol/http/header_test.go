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

package http

import (
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestRequestHeader_Add(t *testing.T) {
	header := RequestHeader{&fasthttp.RequestHeader{}}

	// Add headers using Mosn APIs
	header.Add("custom-empty", "") // Add first to ensure no readback bug.
	header.Add("custom", "1")
	header.Add("x-forwarded-for", "client, proxy1")
	header.Add("x-forwarded-for", "proxy2")
	header.Add("content-type", "text/plain")

	// Now, Verify headers using fasthttp APIs

	// Lookup by lowercase key
	if have := header.Peek("custom-empty"); string(have) != "" {
		t.Error("expected to set an empty header")
	}
	if want, have := "1", header.Peek("custom"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	// We expect the first value to be retained, in order.
	if want, have := "client, proxy1", header.Peek("x-forwarded-for"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	if want, have := "text/plain", header.Peek("content-type"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, want := range []string{
		"Custom-Empty: \r\n",
		"Custom: 1\r\n",
		"X-Forwarded-For: client, proxy1\r\n",
		"X-Forwarded-For: proxy2\r\n", // We expect the second value!
		"Content-Type: text/plain\r\n",
	} {
		if !strings.Contains(have, want) {
			t.Errorf("unexpected representation, want: %s, have: %s", want, have)
		}
	}
}

func TestResponseHeader_Add(t *testing.T) {
	header := ResponseHeader{&fasthttp.ResponseHeader{}}

	// Add headers using Mosn APIs
	header.Add("custom-empty", "") // Add first to ensure no readback bug.
	header.Add("custom", "1")
	header.Add("set-cookie", "a=b")
	header.Add("set-cookie", "c=d")
	header.Add("content-type", "text/plain")

	// Now, Verify headers using fasthttp APIs

	// Lookup by lowercase key
	if have := header.Peek("custom-empty"); string(have) != "" {
		t.Error("expected to set an empty header")
	}
	if want, have := "1", header.Peek("custom"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	// We expect fasthttp to join the set-cookie header in the standard way.
	if want, have := "a=b; c=d", header.Peek("set-cookie"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	if want, have := "text/plain", header.Peek("content-type"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, want := range []string{
		"Custom-Empty: \r\n",
		"Custom: 1\r\n",
		"Set-Cookie: a=b\r\n",
		"Set-Cookie: c=d\r\n", // We expect the literal representation still two values!
		"Content-Type: text/plain\r\n",
	} {
		if !strings.Contains(have, want) {
			t.Errorf("unexpected representation, want: %s, have: %s", want, have)
		}
	}
}

func TestRequestHeader_Set(t *testing.T) {
	header := RequestHeader{&fasthttp.RequestHeader{}}

	// Set headers using Mosn APIs
	header.Set("custom", "")
	header.Set("custom", "1") // we expect this to overwrite
	header.Set("x-forwarded-for", "client, proxy1")
	header.Set("x-forwarded-for", "proxy2") // we expect this to overwrite
	header.Set("content-type", "text/plain")

	// Now, Verify headers using fasthttp APIs

	// Lookup by lowercase key
	if want, have := "1", header.Peek("custom"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	// We expect the first value to be overwritten.
	if want, have := "proxy2", header.Peek("x-forwarded-for"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	if want, have := "text/plain", header.Peek("content-type"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, want := range []string{
		"Custom: 1\r\n",
		"X-Forwarded-For: proxy2\r\n", // We expect the second value!
		"Content-Type: text/plain\r\n",
	} {
		if !strings.Contains(have, want) {
			t.Errorf("unexpected representation, want: %s, have: %s", want, have)
		}
	}
}

func TestResponseHeader_Set(t *testing.T) {
	header := ResponseHeader{&fasthttp.ResponseHeader{}}

	// Set headers using Mosn APIs
	header.Set("custom", "")
	header.Set("custom", "1") // we expect this to overwrite
	header.Set("set-cookie", "a=b")
	header.Set("set-cookie", "c=d") // we expect this to overwrite
	header.Set("content-type", "text/plain")

	// Now, Verify headers using fasthttp APIs

	// Lookup by lowercase key
	if want, have := "1", header.Peek("custom"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	// We expect the first value to be overwritten.
	if want, have := "c=d", header.Peek("set-cookie"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}
	if want, have := "text/plain", header.Peek("content-type"); want != string(have) {
		t.Errorf("unexpected value, want: %s, have: %s", want, have)
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, want := range []string{
		"Custom: 1\r\n",
		"Set-Cookie: c=d\r\n", // We expect the second value!
		"Content-Type: text/plain\r\n",
	} {
		if !strings.Contains(have, want) {
			t.Errorf("unexpected representation, want: %s, have: %s", want, have)
		}
	}
}

// Note: These tests exclude content-type as you can't delete it in
// fasthttp, only zero it out.
func TestRequestHeader_Del(t *testing.T) {
	header := RequestHeader{&fasthttp.RequestHeader{}}

	// Add headers using Mosn APIs
	header.Add("custom-empty", "")
	header.Add("custom", "1")
	header.Add("x-forwarded-for", "client, proxy1")
	header.Add("x-forwarded-for", "proxy2")

	for _, name := range []string{
		"custom-empty",
		"custom",
		"x-forwarded-for",
	} {
		// Now, delete them using Mosn APIs
		header.Del(name)

		// Ensure it doesn't exist in fasthttp
		if have := header.Peek(name); have != nil {
			t.Errorf("expected to delete: %s, have: %s", name, have)
		}
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, dontWant := range []string{
		"Custom-Empty:",
		"Custom:",
		"X-Forwarded-For:",
		"Content-Type:",
	} {
		if strings.Contains(have, dontWant) {
			t.Errorf("unexpected representation, expected to delete: %s, have: %s", dontWant, have)
		}
	}
}

// Note: These tests exclude content-type as you can't delete it in
// fasthttp, only zero it out.
func TestResponseHeader_Del(t *testing.T) {
	header := ResponseHeader{&fasthttp.ResponseHeader{}}

	// Add headers using Mosn APIs.
	header.Add("custom-empty", "") // Del first to ensure no readback bug.
	header.Add("custom", "1")
	header.Add("set-cookie", "a=b")
	header.Add("set-cookie", "c=d")

	for _, name := range []string{
		"custom-empty",
		"custom",
		"set-cookie",
	} {
		// Now, delete them using Mosn APIs
		header.Del(name)

		// Ensure it doesn't exist in fasthttp
		if have := header.Peek(name); have != nil {
			t.Errorf("expected to delete: %s, have: %s", name, have)
		}
	}

	// Verify the literal request is in canonical case format.
	have := header.String()
	for _, dontWant := range []string{
		"Custom-Empty:",
		"Custom:",
		"Set-Cookie:",
		"Content-Type:",
	} {
		if strings.Contains(have, dontWant) {
			t.Errorf("unexpected representation, expected to delete: %s, have: %s", dontWant, have)
		}
	}
}
