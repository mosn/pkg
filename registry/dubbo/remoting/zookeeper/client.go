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

package zookeeper

import (
	"path"
	"strings"
	"sync"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"

	"mosn.io/pkg/registry/dubbo/common/constant"
	"mosn.io/pkg/registry/dubbo/common/logger"
	perrors "github.com/pkg/errors"
)

const (
	// ConnDelay connection delay interval
	ConnDelay = 3
	// MaxFailTimes max fail times
	MaxFailTimes = 15
)

var (
	errNilZkClientConn = perrors.New("zookeeper client{conn} is nil")
	errNilChildren     = perrors.Errorf("has none children")
	errNilNode         = perrors.Errorf("node does not exist")
)

// ZookeeperClient represents zookeeper client Configuration
type ZookeeperClient struct {
	name         string
	ZkAddrs      []string
	sync.RWMutex // for conn
	Conn         *zk.Conn
	Timeout      time.Duration
	exit         chan struct{}
	Wait         sync.WaitGroup

	eventRegistry     map[string][]*chan struct{}
	eventRegistryLock sync.RWMutex
}

// nolint
func StateToString(state zk.State) string {
	switch state {
	case zk.StateDisconnected:
		return "zookeeper disconnected"
	case zk.StateConnecting:
		return "zookeeper connecting"
	case zk.StateAuthFailed:
		return "zookeeper auth failed"
	case zk.StateConnectedReadOnly:
		return "zookeeper connect readonly"
	case zk.StateSaslAuthenticated:
		return "zookeeper sasl authenticated"
	case zk.StateExpired:
		return "zookeeper connection expired"
	case zk.StateConnected:
		return "zookeeper connected"
	case zk.StateHasSession:
		return "zookeeper has session"
	case zk.StateUnknown:
		return "zookeeper unknown state"
	case zk.State(zk.EventNodeDeleted):
		return "zookeeper node deleted"
	case zk.State(zk.EventNodeDataChanged):
		return "zookeeper node data changed"
	default:
		return state.String()
	}
}

// nolint
type Options struct {
	zkName string
	client *ZookeeperClient

	ts *zk.TestCluster
}

// Option will define a function of handling Options
type Option func(*Options)

// WithZkName sets zk client name
func WithZkName(name string) Option {
	return func(opt *Options) {
		opt.zkName = name
	}
}

// ValidateZookeeperClient validates client and sets options
func ValidateZookeeperClient(container ZkClientFacade, opts ...Option) error {
	var (
		err error
	)
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}
	connected := false

	lock := container.ZkClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	if container.ZkClient() == nil {
		// in dubbo, every registry only connect one node, so this is []string{r.Address}
		timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.Location)
		}
		zkAddresses := strings.Split(url.Location, ",")
		newClient, err := NewZookeeperClient(options.zkName, zkAddresses, timeout)
		if err != nil {
			logger.Warnf("newZookeeperClient(name{%s}, zk address{%v}, timeout{%d}) = error{%v}",
				options.zkName, url.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.Location)
		}
		container.SetZkClient(newClient)
		connected = true
	}

	if container.ZkClient().Conn == nil {
		var event <-chan zk.Event
		container.ZkClient().Conn, event, err = zk.Connect(container.ZkClient().ZkAddrs, container.ZkClient().Timeout)
		if err == nil {
			container.ZkClient().Wait.Add(1)
			connected = true
			go container.ZkClient().HandleZkEvent(event)
		}
	}

	if connected {
		logger.Infof("Connect to zookeeper successfully, name{%s}, zk address{%v}", options.zkName, url.Location)
		container.WaitGroup().Add(1) // zk client start successful, then registry wg +1
	}

	return perrors.WithMessagef(err, "newZookeeperClient(address:%+v)", url.PrimitiveURL)
}

// nolint
func NewZookeeperClient(name string, zkAddrs []string, timeout time.Duration) (*ZookeeperClient, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *ZookeeperClient
	)

	z = &ZookeeperClient{
		name:          name,
		ZkAddrs:       zkAddrs,
		Timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}
	// connect to zookeeper
	z.Conn, event, err = zk.Connect(zkAddrs, timeout)
	if err != nil {
		return nil, perrors.WithMessagef(err, "zk.Connect(zkAddrs:%+v)", zkAddrs)
	}

	z.Wait.Add(1)
	go z.HandleZkEvent(event)

	return z, nil
}

// WithTestCluster sets test cluster for zk client
func WithTestCluster(ts *zk.TestCluster) Option {
	return func(opt *Options) {
		opt.ts = ts
	}
}

// NewMockZookeeperClient returns a mock client instance
func NewMockZookeeperClient(name string, timeout time.Duration, opts ...Option) (*zk.TestCluster, *ZookeeperClient, <-chan zk.Event, error) {
	var (
		err   error
		event <-chan zk.Event
		z     *ZookeeperClient
		ts    *zk.TestCluster
	)

	z = &ZookeeperClient{
		name:          name,
		ZkAddrs:       []string{},
		Timeout:       timeout,
		exit:          make(chan struct{}),
		eventRegistry: make(map[string][]*chan struct{}),
	}

	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// connect to zookeeper
	if options.ts != nil {
		ts = options.ts
	} else {
		ts, err = zk.StartTestCluster(1, nil, nil)
		if err != nil {
			return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
		}
	}

	z.Conn, event, err = ts.ConnectWithOptions(timeout)
	if err != nil {
		return nil, nil, nil, perrors.WithMessagef(err, "zk.Connect")
	}

	return ts, z, event, nil
}

// HandleZkEvent handles zookeeper events
func (z *ZookeeperClient) HandleZkEvent(session <-chan zk.Event) {
	var (
		state int
		event zk.Event
	)

	defer func() {
		z.Wait.Done()
		logger.Infof("zk{path:%v, name:%s} connection goroutine game over.", z.ZkAddrs, z.name)
	}()

	for {
		select {
		case <-z.exit:
			return
		case event = <-session:
			logger.Infof("client{%s} get a zookeeper event{type:%s, server:%s, path:%s, state:%d-%s, err:%v}",
				z.name, event.Type, event.Server, event.Path, event.State, StateToString(event.State), event.Err)
			switch (int)(event.State) {
			case (int)(zk.StateDisconnected):
				logger.Warnf("zk{addr:%s} state is StateDisconnected, so close the zk client{name:%s}.", z.ZkAddrs, z.name)
				z.stop()
				z.Lock()
				conn := z.Conn
				z.Conn = nil
				z.Unlock()
				if conn != nil {
					conn.Close()
				}
				return
			case (int)(zk.EventNodeDataChanged), (int)(zk.EventNodeChildrenChanged):
				logger.Infof("zkClient{%s} get zk node changed event{path:%s}", z.name, event.Path)
				z.eventRegistryLock.RLock()
				for p, a := range z.eventRegistry {
					if strings.HasPrefix(p, event.Path) {
						logger.Infof("send event{state:zk.EventNodeDataChange, Path:%s} notify event to path{%s} related listener",
							event.Path, p)
						for _, e := range a {
							*e <- struct{}{}
						}
					}
				}
				z.eventRegistryLock.RUnlock()
			case (int)(zk.StateConnecting), (int)(zk.StateConnected), (int)(zk.StateHasSession):
				if state == (int)(zk.StateHasSession) {
					continue
				}
				z.eventRegistryLock.RLock()
				if a, ok := z.eventRegistry[event.Path]; ok && 0 < len(a) {
					for _, e := range a {
						*e <- struct{}{}
					}
				}
				z.eventRegistryLock.RUnlock()
			}
			state = (int)(event.State)
		}
	}
}

// RegisterEvent registers zookeeper events
func (z *ZookeeperClient) RegisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" || event == nil {
		return
	}

	z.eventRegistryLock.Lock()
	defer z.eventRegistryLock.Unlock()
	a := z.eventRegistry[zkPath]
	a = append(a, event)
	z.eventRegistry[zkPath] = a
	logger.Debugf("zkClient{%s} register event{path:%s, ptr:%p}", z.name, zkPath, event)
}

// UnregisterEvent unregisters zookeeper events
func (z *ZookeeperClient) UnregisterEvent(zkPath string, event *chan struct{}) {
	if zkPath == "" {
		return
	}

	z.eventRegistryLock.Lock()
	defer z.eventRegistryLock.Unlock()
	infoList, ok := z.eventRegistry[zkPath]
	if !ok {
		return
	}
	for i, e := range infoList {
		if e == event {
			infoList = append(infoList[:i], infoList[i+1:]...)
			logger.Infof("zkClient{%s} unregister event{path:%s, event:%p}", z.name, zkPath, event)
		}
	}
	logger.Debugf("after zkClient{%s} unregister event{path:%s, event:%p}, array length %d",
		z.name, zkPath, event, len(infoList))
	if len(infoList) == 0 {
		delete(z.eventRegistry, zkPath)
	} else {
		z.eventRegistry[zkPath] = infoList
	}
}

// nolint
func (z *ZookeeperClient) Done() <-chan struct{} {
	return z.exit
}

func (z *ZookeeperClient) stop() bool {
	select {
	case <-z.exit:
		return true
	default:
		close(z.exit)
	}

	return false
}

// ZkConnValid validates zookeeper connection
func (z *ZookeeperClient) ZkConnValid() bool {
	select {
	case <-z.exit:
		return false
	default:
	}

	z.RLock()
	defer z.RUnlock()
	return z.Conn != nil
}

// nolint
func (z *ZookeeperClient) Close() {
	if z == nil {
		return
	}

	z.stop()
	z.Wait.Wait()
	z.Lock()
	conn := z.Conn
	z.Conn = nil
	z.Unlock()
	if conn != nil {
		logger.Infof("zkClient Conn{name:%s, zk addr:%d} exit now.", z.name, conn.SessionID())
		conn.Close()
	}

	logger.Infof("zkClient{name:%s, zk addr:%s} exit now.", z.name, z.ZkAddrs)
}

// Create will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
// And the value for the basePath is ""
func (z *ZookeeperClient) Create(basePath string) error {
	return z.CreateWithValue(basePath, []byte(""))
}

// CreateWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first.
func (z *ZookeeperClient) CreateWithValue(basePath string, value []byte) error {
	var (
		err     error
		tmpPath string
	)

	logger.Debugf("zookeeperClient.Create(basePath{%s})", basePath)
	conn := z.getConn()
	err = errNilZkClientConn
	if conn == nil {
		return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
	}

	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		_, err = conn.Create(tmpPath, value, 0, zk.WorldACL(zk.PermAll))

		if err != nil {
			if err == zk.ErrNodeExists {
				logger.Debugf("zk.create(\"%s\") exists", tmpPath)
			} else {
				logger.Errorf("zk.create(\"%s\") error(%v)", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

// CreateTempWithValue will create the node recursively, which means that if the parent node is absent,
// it will create parent node first，and set value in last child path
// If the path exist, it will update data
func (z *ZookeeperClient) CreateTempWithValue(basePath string, value []byte) error {
	var (
		err     error
		tmpPath string
	)

	logger.Debugf("zookeeperClient.Create(basePath{%s})", basePath)
	conn := z.getConn()
	err = errNilZkClientConn
	if conn == nil {
		return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
	}

	pathSlice := strings.Split(basePath, "/")[1:]
	length := len(pathSlice)
	for i, str := range pathSlice {
		tmpPath = path.Join(tmpPath, "/", str)
		// last child need be ephemeral
		if i == length-1 {
			_, err = conn.Create(tmpPath, value, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
			if err == zk.ErrNodeExists {
				return err
			}
		} else {
			_, err = conn.Create(tmpPath, []byte{}, 0, zk.WorldACL(zk.PermAll))
		}
		if err != nil {
			if err == zk.ErrNodeExists {
				logger.Debugf("zk.create(\"%s\") exists", tmpPath)
			} else {
				logger.Errorf("zk.create(\"%s\") error(%v)", tmpPath, perrors.WithStack(err))
				return perrors.WithMessagef(err, "zk.Create(path:%s)", basePath)
			}
		}
	}

	return nil
}

// nolint
func (z *ZookeeperClient) Delete(basePath string) error {
	err := errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		err = conn.Delete(basePath, -1)
	}

	return perrors.WithMessagef(err, "Delete(basePath:%s)", basePath)
}

// RegisterTemp registers temporary node by @basePath and @node
func (z *ZookeeperClient) RegisterTemp(basePath string, node string) (string, error) {
	var (
		err     error
		zkPath  string
		tmpPath string
	)

	err = errNilZkClientConn
	zkPath = path.Join(basePath) + "/" + node
	conn := z.getConn()
	if conn != nil {
		tmpPath, err = conn.Create(zkPath, []byte(""), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}

	if err != nil {
		logger.Warnf("conn.Create(\"%s\", zk.FlagEphemeral) = error(%v)", zkPath, perrors.WithStack(err))
		return zkPath, perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s", z.name, tmpPath)

	return tmpPath, nil
}

// RegisterTempSeq register temporary sequence node by @basePath and @data
func (z *ZookeeperClient) RegisterTempSeq(basePath string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		tmpPath, err = conn.Create(
			path.Join(basePath)+"/",
			data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll),
		)
	}

	logger.Debugf("zookeeperClient.RegisterTempSeq(basePath{%s}) = tempPath{%s}", basePath, tmpPath)
	if err != nil && err != zk.ErrNodeExists {
		logger.Errorf("zkClient{%s} conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)",
			z.name, basePath, string(data), err)
		return "", perrors.WithStack(err)
	}
	logger.Debugf("zkClient{%s} create a temp zookeeper node:%s", z.name, tmpPath)

	return tmpPath, nil
}

// GetChildrenW gets children watch by @path
func (z *ZookeeperClient) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		watcher  *zk.Watcher
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		children, stat, watcher, err = conn.ChildrenW(path)
	}

	if err != nil {
		if err == zk.ErrNoChildrenForEphemerals {
			return nil, nil, errNilChildren
		}
		if err == zk.ErrNoNode {
			return nil, nil, errNilNode
		}
		logger.Errorf("zk.ChildrenW(path{%s}) = error(%v)", path, err)
		return nil, nil, perrors.WithMessagef(err, "zk.ChildrenW(path:%s)", path)
	}
	if stat == nil {
		return nil, nil, perrors.Errorf("path{%s} get stat is nil", path)
	}
	if len(children) == 0 {
		return nil, nil, errNilChildren
	}

	return children, watcher.EvtCh, nil
}

// GetChildren gets children by @path
func (z *ZookeeperClient) GetChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		children, stat, err = conn.Children(path)
	}

	if err != nil {
		if err == zk.ErrNoNode {
			return nil, perrors.Errorf("path{%s} has none children", path)
		}
		logger.Errorf("zk.Children(path{%s}) = error(%v)", path, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.Children(path:%s)", path)
	}
	if stat == nil {
		return nil, perrors.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, errNilChildren
	}

	return children, nil
}

// ExistW to judge watch whether it exists or not by @zkPath
func (z *ZookeeperClient) ExistW(zkPath string) (<-chan zk.Event, error) {
	var (
		exist   bool
		err     error
		watcher *zk.Watcher
	)

	err = errNilZkClientConn
	conn := z.getConn()
	if conn != nil {
		exist, _, watcher, err = conn.ExistsW(zkPath)
	}

	if err != nil {
		logger.Warnf("zkClient{%s}.ExistsW(path{%s}) = error{%v}.", z.name, zkPath, perrors.WithStack(err))
		return nil, perrors.WithMessagef(err, "zk.ExistsW(path:%s)", zkPath)
	}
	if !exist {
		logger.Warnf("zkClient{%s}'s App zk path{%s} does not exist.", z.name, zkPath)
		return nil, perrors.Errorf("zkClient{%s} App zk path{%s} does not exist.", z.name, zkPath)
	}

	return watcher.EvtCh, nil
}

// GetContent gets content by @zkPath
func (z *ZookeeperClient) GetContent(zkPath string) ([]byte, *zk.Stat, error) {
	return z.Conn.Get(zkPath)
}

// nolint
func (z *ZookeeperClient) SetContent(zkPath string, content []byte, version int32) (*zk.Stat, error) {
	return z.Conn.Set(zkPath, content, version)
}

// getConn gets zookeeper connection safely
func (z *ZookeeperClient) getConn() *zk.Conn {
	z.RLock()
	defer z.RUnlock()
	return z.Conn
}
