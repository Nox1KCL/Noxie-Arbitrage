// Package syncutils provides synchronization utilities.
package syncutils

import "sync"

type MyWaitGroup struct {
	sync.WaitGroup
}

func (wg *MyWaitGroup) Go(fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}
