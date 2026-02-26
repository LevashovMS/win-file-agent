package server

import (
	"fmt"
	"reflect"
	"runtime"
)

type ArgsHandler func(*server)

func (c *server) verification() error {
	if c.port < 80 {
		c.port = 8099
	}

	return nil
}

func Port(port int) ArgsHandler {
	return func(o *server) {
		o.port = port
	}
}

func Handler[T any](method, path string, h routerAction[T]) ArgsHandler {
	return func(o *server) {
		var pc = reflect.ValueOf(h).Pointer()
		var name = runtime.FuncForPC(pc).Name()
		var handler = (&router[T]{h: h, name: name}).generalHandler

		o.mux.HandleFunc(fmt.Sprintf("%s %s", method, path), handler)
	}
}
