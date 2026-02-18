package store

import (
	"maps"
	"slices"
	"sync"
)

type ram[K comparable, V any] struct {
	lock sync.RWMutex
	data map[K]V
}

func NewRam[K comparable, V any]() Store[K, V] {
	return &ram[K, V]{data: make(map[K]V)}
}

func (c *ram[K, V]) Store(key K, value V) {
	c.lock.Lock()
	c.data[key] = value
	c.lock.Unlock()
}

func (c *ram[K, V]) Delete(key K) {
	c.lock.Lock()
	delete(c.data, key)
	c.lock.Unlock()
}

func (c *ram[K, V]) Range(f func(key K, value V) bool) {
	c.lock.RLock()
	for k, v := range c.data {
		f(k, v)
	}
	c.lock.RUnlock()
}

func (c *ram[K, V]) Load(key K) (value V, ok bool) {
	c.lock.RLock()
	value, ok = c.data[key]
	c.lock.RUnlock()
	return
}

func (c *ram[K, V]) GetKeys() []K {
	c.lock.RLock()
	var keys = slices.Collect(maps.Keys(c.data))
	c.lock.RUnlock()
	return keys
}
