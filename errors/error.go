package errors

import "fmt"

type Error struct {
	// client: label synopsis
	// log: label message inner(Error())
	httpStatus int
	inner      error  // log details
	kind       string // error_handler "event"
	label      string
	message    string // log
	synopsis   string // client
}

type GoError interface {
	error
	GoStatus() int
	GoError() string
	Type() string
}

var _ GoError = &Error{}

func New() *Error {
	return Server
}

func (e *Error) Status(s int) *Error {
	err := *e
	err.httpStatus = s
	return &err
}

func (e *Error) Kind(name string) *Error {
	err := *e
	err.kind = name
	return &err
}

func (e *Error) PrefixKind(name string) *Error {
	err := *e
	if err.kind != "" {
		err.kind = name + "_" + err.kind
	} else {
		err.kind = name
	}
	return &err
}

func (e *Error) Label(name string) *Error {
	err := *e
	err.label = name
	return &err
}

func (e *Error) Message(msg string) *Error {
	err := *e
	err.message = msg
	return &err
}

func (e *Error) Messagef(msg string, args ...interface{}) *Error {
	return e.Message(fmt.Sprintf(msg, args...))
}

func (e *Error) With(inner error) *Error {
	err := *e
	err.inner = inner
	return &err
}

func (e *Error) Error() string {
	return appendMsg(e.label, e.synopsis)
}

func (e *Error) Unwrap() error {
	return e.inner
}

func (e *Error) GoError() string {
	msg := appendMsg(e.label, e.message)

	if e.inner != nil {
		appendMsg(msg, e.inner.Error())
	}
	return msg
}

func (e *Error) GoStatus() int {
	return e.httpStatus
}

func (e *Error) Type() string {
	return e.kind
}

// appendMsg chains the given strings with ": " as separator.
func appendMsg(target string, messages ...string) string {
	result := target
	for _, m := range messages {
		if result != "" && m != "" {
			result += ": "
		}
		result += m
	}
	return result
}
