package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/coupergateway/couper/cache"
	"github.com/coupergateway/couper/internal/test"
)

func TestCache_All(t *testing.T) {
	log, _ := test.NewLogger()
	logger := log.WithContext(context.Background())

	quitCh := make(chan struct{})
	defer close(quitCh)
	ms := cache.New(logger, quitCh)

	if v := ms.Get("key"); v != nil {
		t.Errorf("Nil expected, given %q", v)
	}

	go ms.Set("key", "val", 2)
	go ms.Set("del", "del", 30)
	go func() {
		ms.Get("key")
	}()

	time.Sleep(300 * time.Millisecond)

	if v := ms.Get("key"); v != "val" {
		t.Errorf("Expected 'val', given %q", v)
	}
	if v := ms.Get("del"); v != "del" {
		t.Errorf("Expected 'del', given %q", v)
	}

	time.Sleep(1700 * time.Millisecond)

	if v := ms.Get("key"); v != nil {
		t.Errorf("Nil expected, given %q", v)
	}
	if v := ms.Get("del"); v != "del" {
		t.Errorf("Expected 'del', given %q", v)
	}
}
