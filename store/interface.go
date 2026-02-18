package store

import "time"

type Store[K any, V any] interface {
	Store(key K, value V)
	Range(f func(key K, value V) bool)
	Delete(key K)
	Load(key K) (value V, ok bool)
	GetKeys() []K
	SetTimeout(key K, t time.Time)
}
