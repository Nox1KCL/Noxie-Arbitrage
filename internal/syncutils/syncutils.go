// Package syncutils provides synchronization utilities.
package syncutils

import (
	"sync"

	"log/slog"

	"github.com/joho/godotenv"
)

var sulog = slog.With("service", "syncutils")

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

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		sulog.Warn("env file was not found", "error", err)
		return
	}
}
