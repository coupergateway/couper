package cache

import (
	"sync"
	"time"
)

const maxExpiresIn = 86400

type entry struct {
	value string
	expAt int64
}

// MemoryStore represents the <MemoryStore> object.
type MemoryStore struct {
	db map[string]*entry
	mu sync.Mutex
}

// New creates a new <MemoryStore> object.
func New() *MemoryStore {
	store := &MemoryStore{
		db: make(map[string]*entry),
	}

	go store.gc()

	return store
}

// Del deletes the value by the key from the <MemoryStore>.
func (ms *MemoryStore) Del(k string) {
	ms.mu.Lock()

	delete(ms.db, k)

	ms.mu.Unlock()
}

// Get return the value by the key if the ttl is not expired from the <MemoryStore>.
func (ms *MemoryStore) Get(k string) string {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if v, ok := ms.db[k]; ok {
		if time.Now().Unix() >= v.expAt {
			return ""
		}

		return v.value
	}

	return ""
}

// Set stores a key/value pair for <ttl> second(s) into the <MemoryStore>.
func (ms *MemoryStore) Set(k, v string, ttl int64) {
	go ms.set(k, v, ttl)
}

func (ms *MemoryStore) gc() {
	for now := range time.Tick(time.Second) {
		ms.mu.Lock()

		for k, v := range ms.db {
			if now.Unix() >= v.expAt {
				delete(ms.db, k)
			}
		}

		ms.mu.Unlock()
	}
}

func (ms *MemoryStore) set(k, v string, ttl int64) {
	ms.mu.Lock()

	if ttl < 0 {
		ttl = 0
	} else if ttl > maxExpiresIn {
		ttl = maxExpiresIn
	}

	ms.db[k] = &entry{
		value: v,
		expAt: time.Now().Unix() + ttl,
	}

	ms.mu.Unlock()
}
