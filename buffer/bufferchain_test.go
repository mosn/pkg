package buffer

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//Test write
func TestBufferWrite(t *testing.T) {
	chain := NewIoBufferChain(10)
	write := func(i *int32) error {
		bytes := make([]byte, 1)
		_, err := chain.Write(bytes)
		if err != nil {
			t.Logf("errs: %v", err)
			return err
		}
		atomic.AddInt32(i, 1)
		return nil
	}
	var i int32
	go func() {
		for i < 20 {
			err := write(&i)
			if err != nil {
				break
			}
		}
	}()
	time.Sleep(1 * time.Second / 2)
	chain.CloseWithError(nil)
	assert.Equal(t, i, int32(10))
}

func TestBufferReade(t *testing.T) {
	chain := NewIoBufferChain(10)
	chain.Write(make([]byte, 1))
	chain.Write(make([]byte, 1))
	var i int32
	reader := func(i *int32) {
		_ = chain.Bytes()
		atomic.AddInt32(i, 1)
	}
	go func() {
		for {
			reader(&i)
		}
	}()
	time.Sleep(1 * time.Second)
	chain.CloseWithError(nil)
	assert.Equal(t, i, int32(2))
}
