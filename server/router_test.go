package server

import (
	"net/http"
	"testing"
)

func TestHandler(t *testing.T) {
	var r = newRouter()
	r.regHandler(http.MethodGet, "/v1/test/:qwerty", func(r *http.Request) (any, error) { return nil, nil })
}
