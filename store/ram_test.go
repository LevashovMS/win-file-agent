package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRam(t *testing.T) {
	var r = NewRam[int, int](context.TODO())

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

func TestTimeout(t *testing.T) {
	var ctx, cf = context.WithTimeout(context.TODO(), 2*time.Second)
	defer cf()
	// contructor
	var r = &ram[int, int]{
		data:   make(map[int]int),
		timers: make(map[int]*time.Timer),
	}
	go r.timersStop(ctx)
	//##########################
	var wg sync.WaitGroup
	wg.Add(2)
	var count = 100

	//write
	go func() {
		defer wg.Done()
		for idx := range count {
			r.Store(idx, idx)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	//timeout
	go func() {
		defer wg.Done()
		for idx := range count {
			if idx%2 == 0 {
				fmt.Printf("SetTimeout key: %d\n", idx)
				r.SetTimeout(idx, time.Now().Add(5*time.Millisecond))
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	println("Sleep")
	time.Sleep(500 * time.Millisecond)

	var keys = r.GetKeys()
	fmt.Printf("keys: %+v\n", keys)
	r.Range(func(key, value int) bool {
		fmt.Printf("key: %d, value: %d\n", key, value)
		return true
	})
}
