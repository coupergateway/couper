package cache

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Storage interface {
	Del(string)                      // TODO: return bool or int?
	Get(string) (interface{}, error) // TODO: return (string, ok=bool, error)? value=string?
	Set(string, interface{}, int64)  // TODO: return bool? value=string?

	/*
		// ZooKeeper soll performance-mäßig grauenhaft sein. In diversen Foren wird
		// davon dringend abgeraten, dieses Tool als KV-Store zu nutzen.

		--------------------------------------------------------------------------------

		// Consule ist nur bedingt sinnvoll, da es hier gewisse einschränkungen
		// zu TTL gibt. Hier die Doku zu TTL:

		// TTL (string: "") - Specifies the duration of a session (between 10s and 86400s).
		// If provided, the session is invalidated if it is not renewed before the TTL expires.
		// The lowest practical TTL should be used to keep the number of managed sessions low.
		// When locks are forcibly expired, such as when following the leader election pattern
		// in an application, sessions may not be reaped for up to double this TTL, so long TTL
		// values (> 1 hour) should be avoided. Valid time units include "s", "m" and "h".

		--------------------------------------------------------------------------------

		// Lock uses SetNX for Redis (and Memory - Dummy), or "go.etcd.io/etcd/clientv3/concurrency"
		// Session/Lock for ETCD.
		Lock(string) bool

		// Unlock uses Del() of this interface for Redis (and Memory - Dummy), or
		// "go.etcd.io/etcd/clientv3/concurrency" Session/Unlock for ETCD.
		Unlock(string) bool

		--------------------------------------------------------------------------------

		// Zu klären: Was passiert, wenn zwischen Lock und Unlock ein Netzwerk-Problem
		// entsteht und der Unlock nicht mehr ausgeführt werden kann?

		--------------------------------------------------------------------------------


	*/
}

// New creates a <Storage> instance depending on the given <configURL>.
func New(ctx context.Context, configURL string, log *logrus.Entry, quitCh <-chan struct{}) (Storage, error) {
	if configURL == "" {
		return NewMemory(log, quitCh), nil
	}

	return NewRedis(ctx, configURL)
}
