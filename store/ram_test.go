package store

import (
	"sync"
	"testing"
)

func TestRam(t *testing.T) {
	var r = NewRam[int, int]()

	var wg sync.WaitGroup
	wg.Add(7)

	//write
	go func() {
		defer wg.Done()
		for idx := range 1000000 {
			r.Store(idx, idx)
		}
	}()

	//write2
	go func() {
		defer wg.Done()
		for idx := range 100000 {
			r.Store(idx, idx)
		}
	}()

	//write3
	go func() {
		defer wg.Done()
		for idx := range 10000 {
			r.Store(idx, idx)
		}
	}()

	//read
	go func() {
		defer wg.Done()
		for idx := range 100000 {
			r.Load(idx)
		}
	}()

	//range
	go func() {
		defer wg.Done()
		for range 10000 {
			r.Range(func(key, value int) bool {
				return true
			})
		}
	}()

	//delete
	go func() {
		defer wg.Done()
		for idx := range 2000 {
			r.Delete(idx)
		}
	}()

	//keys
	go func() {
		defer wg.Done()
		for range 10000 {
			r.GetKeys()
		}
	}()

	wg.Wait()
}
