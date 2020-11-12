package test

import "context"

const Key = "test_bug2kguvvhfvuij7gcsg"

func NewContext(background context.Context) context.Context {
	return context.WithValue(background, Key, true)
}
