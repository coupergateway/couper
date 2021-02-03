package cache_test

import (
	"testing"
	"time"

	"github.com/avenga/couper/cache"
)

func TestCache_All(t *testing.T) {
	ms := cache.New()

	if v := ms.Get("key"); v != "" {
		t.Errorf("Empty string expected, given %q", v)
	}

	ms.Set("key", "val", 2)
	ms.Set("del", "del", 30)

	time.Sleep(300 * time.Millisecond)

	if v := ms.Get("key"); v != "val" {
		t.Errorf("Expected 'val', given %q", v)
	}
	if v := ms.Get("del"); v != "del" {
		t.Errorf("Expected 'del', given %q", v)
	}

	time.Sleep(1700 * time.Millisecond)

	if v := ms.Get("key"); v != "" {
		t.Errorf("Empty string expected, given %q", v)
	}
	if v := ms.Get("del"); v != "del" {
		t.Errorf("Expected 'del', given %q", v)
	}
}
