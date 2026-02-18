package server

import "strconv"

// StatusCode sets the http response status code
type StatusCode int

func (s StatusCode) Error() string {
	return strconv.Itoa(int(s))
}
