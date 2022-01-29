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

package utils

import (
	"container/list"
	"sync"
)

type SyncList struct {
	list     *list.List
	curr     *list.Element
	mux      sync.Mutex
	visitMux sync.Mutex
}

func NewSyncList() *SyncList {
	return &SyncList{
		list: list.New(),
	}
}

func (l *SyncList) PushBack(v interface{}) *list.Element {
	l.mux.Lock()
	defer l.mux.Unlock()
	return l.list.PushBack(v)
}

func (l *SyncList) Remove(e *list.Element) interface{} {
	l.mux.Lock()
	defer l.mux.Unlock()
	if e == l.curr {
		l.curr = l.curr.Prev()
	}

	return l.list.Remove(e)
}

// VisitSafe means the visit function f can visit each element safely even f may block some time.
// Also, it won't block other operations(e.g. Remove) when the visit function f is blocked.
// But, it can not run parallel since there is an instance level curr point.
func (l *SyncList) VisitSafe(f func(v interface{})) {
	l.visitMux.Lock()
	defer l.visitMux.Unlock()

	// just in case there is some dirty data left from the previous call.
	l.mux.Lock()
	l.curr = nil
	l.mux.Unlock()

	for {
		l.mux.Lock()
		if l.curr == nil {
			l.curr = l.list.Front()
		} else {
			l.curr = l.curr.Next()
		}
		curr := l.curr
		l.mux.Unlock()

		if curr == nil {
			break
		}
		f(curr.Value)
	}
}

func (l *SyncList) Len() int {
	l.mux.Lock()
	defer l.mux.Unlock()

	n := 0
	for e := l.list.Front(); e != nil; e = e.Next() {
		n++
	}
	return n
}
