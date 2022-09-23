package cache

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const maxExpiresIn = 86400

type entry struct {
	value interface{}
	expAt int64
}

// MemoryStore represents the <MemoryStore> object.
type MemoryStore struct {
	db     map[string]*entry
	mu     sync.RWMutex
	log    *logrus.Entry
	quitCh <-chan struct{}
}

// NewMemory creates a new <MemoryStore> object.
func NewMemory(log *logrus.Entry, quitCh <-chan struct{}) Storage {
	store := &MemoryStore{
		db:     make(map[string]*entry),
		log:    log,
		quitCh: quitCh,
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
func (ms *MemoryStore) Get(k string) (interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if v, ok := ms.db[k]; ok {
		if time.Now().Unix() >= v.expAt {
			return nil, nil
		}
		return v.value, nil
	}

	return nil, nil
}

// Set stores a key/value pair for <ttl> second(s) into the <MemoryStore>.
func (ms *MemoryStore) Set(k string, v interface{}, ttl int64) {
	if ttl < 0 {
		ttl = 0
	} else if ttl > maxExpiresIn {
		ttl = maxExpiresIn
	}

	ms.mu.Lock()
	ms.db[k] = &entry{
		value: v,
		expAt: time.Now().Unix() + ttl,
	}
	ms.mu.Unlock()
}

func (ms *MemoryStore) gc() {
	ticker := time.NewTicker(time.Second)

	defer func() {
		if rc := recover(); rc != nil {
			ms.log.WithField("panic", string(debug.Stack())).Panic(rc)
		}
		ticker.Stop()
	}()

	for {
		select {
		case <-ms.quitCh:
			return
		case now := <-ticker.C:
			ms.mu.Lock()

			for k, v := range ms.db {
				if now.Unix() >= v.expAt {
					delete(ms.db, k)
				}
			}

			ms.mu.Unlock()
		}
	}
}
