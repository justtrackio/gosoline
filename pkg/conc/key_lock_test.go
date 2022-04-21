package conc_test

import (
	"os"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/conc"
)

func Test_keyLock_Lock(t *testing.T) {
	var (
		unlockFuncA conc.UnlockFunc
		unlockFuncB conc.UnlockFunc
	)
	c := make(chan bool, 2)
	l := conc.NewKeyLock()

	time.AfterFunc(time.Second*2, func() {
		t.Errorf("waiting to read from a channel for too long")
		os.Exit(0)
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
}
