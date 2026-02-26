package server

var _ error = (*StCode)(nil)

// StCode sets the http response status code
type StCode struct {
	statusCode  int
	innerErr    error
	externalMsg string
}

func (c *StCode) Error() string { return c.externalMsg }

func StatusCode(val int) *StCode {
	return &StCode{statusCode: val}
}

func StatusErr(val int, err error) *StCode {
	return &StCode{statusCode: val, innerErr: err}
}

func StatusMsgErr(val int, msg string, err error) *StCode {
	return &StCode{statusCode: val, innerErr: err, externalMsg: msg}
}
