package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"runtime"
)

type routerAction func(*http.Request) (any, error)

type router struct {
	mux *http.ServeMux
}

func newRouter() *router {
	return &router{
		mux: http.NewServeMux(),
	}
}

func (c *router) regHandler(method, path string, h routerAction) {
	c.mux.HandleFunc(fmt.Sprintf("%s %s", method, path), (&routerExt{h: h}).generalHandler)
}

type routerExt struct {
	h routerAction
}

func (c *routerExt) generalHandler(res http.ResponseWriter, req *http.Request) {
	var pc = reflect.ValueOf(c.h).Pointer()
	var name = runtime.FuncForPC(pc).Name()
	log.Printf("Method: %s, Path: %s -> %s\n", req.Method, req.URL.Path, name)

	var data, err = c.h(req)
	if err != nil {
		switch t := err.(type) {
		case StatusCode:
			res.WriteHeader(int(t))
			if data == nil {
				return
			}
		default:
			log.Printf("%+v\n", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if data == nil {
		res.WriteHeader(http.StatusOK)
		return
	}

	buffer, err := json.Marshal(data)
	if err != nil {
		log.Printf("%+v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	_, err = res.Write(buffer)
	if err != nil {
		log.Printf("%+v\n", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}
