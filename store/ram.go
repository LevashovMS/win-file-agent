package store

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"mediamagi.ru/win-file-agent/log"
)

type ram[K comparable, V any] struct {
	lock   sync.RWMutex
	data   map[K]V
	timers map[K]*time.Timer
}

func NewRam[K comparable, V any](ctx context.Context) Store[K, V] {
	var r = &ram[K, V]{
		data:   make(map[K]V),
		timers: make(map[K]*time.Timer),
	}
	go r.timersStop(ctx)

	return r
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

func (c *ram[K, V]) SetTimeout(key K, t time.Time) {
	var dur = time.Until(t)
	c.lock.Lock()
	var _, ok = c.data[key]
	if ok {
		c.timers[key] = time.AfterFunc(dur, func() {
			c.handler(key)
		})
	}
	c.lock.Unlock()
}

func (c *ram[K, V]) timersStop(ctx context.Context) {
	<-ctx.Done()
	c.lock.Lock()
	for key, v := range c.timers {
		log.Debug("Timer stop: %v\n", key)
		v.Stop()
	}
	c.data = make(map[K]V)
	c.timers = make(map[K]*time.Timer)
	c.lock.Unlock()

}

func (c *ram[K, V]) handler(key K) {
	log.Debug("Ожидание завершено по ключу: %v\n", key)
	c.lock.Lock()
	var _, ok = c.data[key]
	if ok {
		delete(c.data, key)
		delete(c.timers, key)
	}
	c.lock.Unlock()
}
