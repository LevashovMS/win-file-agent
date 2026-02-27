package server

import (
	"encoding/json"
	"net/http"

	"mediamagi.ru/win-file-agent/errors"
	"mediamagi.ru/win-file-agent/log"
)

type routerAction[T any] func(*http.Request) (T, error)

type router[T any] struct {
	name string
	h    routerAction[*T]
}

func (c *router[T]) generalHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("Method: %s, Path: %s -> %s\n", req.Method, req.URL.Path, c.name)

	var data, err = c.h(req)
	if err != nil {
		switch t := err.(type) {
		case *StCode:
			if len(t.externalMsg) > 0 {
				http.Error(w, t.externalMsg, t.statusCode)
			} else {
				w.WriteHeader(t.statusCode)
			}
			if t.innerErr != nil {
				log.Error("%+v", t.innerErr)
			}
		default:
			log.Error("%+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if data == nil {
		return
	}

	buffer, err := json.Marshal(data)
	if err != nil {
		log.Error("%+v", errors.WithStack(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(buffer)
	if err != nil {
		log.Error("%+v\n", errors.WithStack(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
