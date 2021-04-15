package errors

type Error struct {
	httpStatus int
	inner      error // details
	label      string
	message    string // log message
	synopsis   string // client message
}

type GoError interface {
	error
	GoStatus() int
	GoError() string
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

func (e *Error) Label(lbl string) *Error {
	err := *e
	err.label = lbl
	return &err
}

func (e *Error) Message(msg string) *Error {
	err := *e
	err.message = msg
	return &err
}

func (e *Error) With(inner error) *Error {
	err := *e
	err.inner = inner
	return &err
}

func (e *Error) Error() string {
	var msg string
	if e.label != "" {
		msg += e.label + ": "
	}
	if e.synopsis != "" {
		msg += e.synopsis
	}
	return msg
}

func (e *Error) GoError() string {
	var msg string
	if e.label != "" {
		msg += e.label + ": "
	}
	if e.message != "" {
		msg += e.message
	}
	if e.inner != nil {
		msg += ": " + e.inner.Error()
	}
	return msg
}

func (e *Error) GoStatus() int {
	return e.httpStatus
}
