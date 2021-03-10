package test

import "context"

const Key = "test_bug2kguvvhfvuij7gcsg"

func NewContext() context.Context {
	return context.WithValue(context.Background(), Key, true)
}
