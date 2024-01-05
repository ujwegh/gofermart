package errors

type ResponseCodeError struct {
	err  error
	msg  string
	code int
}

func New(err error, msg string) error {
	return ResponseCodeError{err: err, msg: msg, code: 500}
}
func NewWithCode(err error, msg string, code int) error {
	return ResponseCodeError{err: err, msg: msg, code: code}
}
func (rce ResponseCodeError) Error() string {
	return rce.err.Error()
}
func (rce ResponseCodeError) Msg() string {
	return rce.msg
}
func (rce ResponseCodeError) Code() int {
	return rce.code
}
func (rce ResponseCodeError) Unwrap() error {
	return rce.err
}
