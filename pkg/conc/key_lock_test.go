package conc_test

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/conc"
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
