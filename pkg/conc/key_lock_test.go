package conc_test

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
)

func Test_keyLock_Lock(t *testing.T) {
	var (
		unlockFuncA conc.UnlockFunc
		unlockFuncB conc.UnlockFunc
		done        int32
	)
	c := make(chan bool, 2)
	l := conc.NewKeyLock()

	time.AfterFunc(time.Second*2, func() {
		if atomic.LoadInt32(&done) != 1 {
			t.Errorf("waiting to read from a channel for too long")
			os.Exit(1)
		}
	})

	go func(chan bool) {
		unlockFuncA = l.Lock("a")
		c <- true
	}(c)

	go func(chan bool) {
		unlockFuncB = l.Lock("b")
		c <- true
	}(c)

	go func(chan bool) {
		unlockFuncA = l.Lock("a")
		c <- true
	}(c)

	<-c
	<-c
	unlockFuncA()
	unlockFuncB()
	<-c
	unlockFuncA()

	atomic.StoreInt32(&done, 1)
}

func TestKeyLockHighTraffic(t *testing.T) {
	cfn := coffin.New()
	count := 0

	cfn.Go(func() error {
		l := conc.NewKeyLock()

		var inCCS int32

		for i := 0; i < 100; i++ {
			cfn.Go(func() error {
				for j := 0; j < 1000; j++ {
					unlock := l.Lock("a")

					// in critical section!
					success := atomic.CompareAndSwapInt32(&inCCS, 0, 1)
					assert.True(t, success, "we should be the only one in the critical section!")

					count++

					success = atomic.CompareAndSwapInt32(&inCCS, 1, 0)
					assert.True(t, success, "we should be the only one in the critical section!")

					unlock()
				}

				return nil
			})
		}

		return nil
	})

	err := cfn.Wait()
	assert.NoError(t, err)

	assert.Equal(t, 100*1000, count)
}
