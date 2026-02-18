package store

import "sync"

type Store[K, V any] interface {
	Store(key K, value V)
	Range(f func(key K, value V) bool)
	Delete(key K)
	Load(key K) (value V, ok bool)
	GetKeys() []V
}

type ram struct {
	childProcs sync.Map // map[taskID]*exec.Cmd
}

func NewRam() Store[any, any] {
	return &ram{}
}

func (c *ram) Store(key any, value any) {
	c.childProcs.Store(key, value)
}

func (c *ram) Delete(key any) {
	c.childProcs.Delete(key)
}

func (c *ram) Range(f func(key, value any) bool) {
	c.childProcs.Range(f)
}

func (c *ram) Load(key any) (value any, ok bool) {
	return c.childProcs.Load(key)
}

func (c *ram) GetKeys() []any {
	var keys = make([]any, 0)
	c.childProcs.Range(func(key, value any) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}
