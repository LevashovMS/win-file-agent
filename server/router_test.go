package server

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"mediamagi.ru/win-file-agent/errors"
	"mediamagi.ru/win-file-agent/worker"
)

type testGeneralHandler func(http.ResponseWriter, *http.Request)

func TestHandler(t *testing.T) {
	var arr []testGeneralHandler
	arr = append(arr, (&router[worker.Task]{
		name: "test",
		h: func(r *http.Request) (*worker.Task, error) {
			return nil, nil
		},
	}).generalHandler)

	arr = append(arr, (&router[string]{
		name: "test2",
		h: func(r *http.Request) (*string, error) {
			var str = "data"
			return &str, nil
		},
	}).generalHandler)

	arr = append(arr, (&router[worker.Task]{
		name: "test3",
		h: func(r *http.Request) (*worker.Task, error) {
			return nil, StatusMsgErr(500, "Внешняя ошибка", errors.New("TEST3"))
		},
	}).generalHandler)

	arr = append(arr, (&router[worker.Task]{
		name: "test4",
		h: func(r *http.Request) (*worker.Task, error) {
			return &worker.Task{}, StatusCode(201)
		},
	}).generalHandler)

	arr = append(arr, (&router[worker.Task]{
		name: "test5",
		h: func(r *http.Request) (*worker.Task, error) {
			return nil, errors.New("TEST5")
		},
	}).generalHandler)

	arr = append(arr, (&router[worker.Task]{
		name: "test6",
		h: func(r *http.Request) (*worker.Task, error) {
			return nil, StatusErr(409, errors.New("TEST5"))
		},
	}).generalHandler)

	for _, it := range arr {
		it(&testResponseWriter{header: make(map[string][]string)},
			&http.Request{
				URL: &url.URL{},
			})
	}
}

var _ http.ResponseWriter = (*testResponseWriter)(nil)

type testResponseWriter struct {
	header http.Header
}

func (c *testResponseWriter) Header() http.Header { return c.header }
func (c *testResponseWriter) Write(buffer []byte) (int, error) {
	println(string(buffer))
	return 0, nil
}
func (c *testResponseWriter) WriteHeader(statusCode int) {
	println(strconv.Itoa(statusCode))
}
