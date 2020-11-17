package buffer

import (
	"errors"
	"io"
	"sync"
	"time"
)

var ErrWriteCovered = errors.New("chain write covered")

type ioBufferchain struct {
	IoBuffer
	bufferchain chan []byte
	errChan     chan error
	mutex       sync.Mutex
}

func NewIoBufferChain(capacity int) *ioBufferchain {
	if capacity == 0 {
		capacity = 1 << 9
	}
	return &ioBufferchain{
		bufferchain: make(chan []byte, capacity),
		errChan:     make(chan error),
	}
}

func (bc *ioBufferchain) Bytes() (p []byte) {
	p, b := <-bc.bufferchain
	if !b {
		return nil
	}
	return p
}

func (bc *ioBufferchain) Write(p []byte) (n int, err error) {
	bytes := *GetBytes(len(p))
	copy(bytes, p)
	select {
	case <-bc.errChan:
		return 0, io.EOF
	default:
		bc.mutex.Lock()
		defer bc.mutex.Unlock()
		select {
		case bc.bufferchain <- bytes:
			return len(bytes), nil
		default:
			// chain is full conn goutine wait 1s to consumer
			ticker := time.NewTicker(1 * time.Second)
			select {
			case <-bc.errChan:
				return 0, io.EOF
			case bc.bufferchain <- bytes:
				ticker.Stop()
				return len(bytes), nil
			case <-ticker.C:
				return 0, ErrWriteCovered
			}
		}
	}
}

func (bc *ioBufferchain) CloseWithError(err error) {
	select {
	case <-bc.errChan:
		return
	default:
		close(bc.errChan)
		bc.mutex.Lock()
		defer bc.mutex.Unlock()
		close(bc.bufferchain)
	}
}

func (bc *ioBufferchain) Count(int32) int32 {
	return 1
}

func (bc *ioBufferchain) Len() int {
	return 0
}
