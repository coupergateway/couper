package errors

import "fmt"

type Error struct {
	// client: synopsis
	// log: synopsis label message inner(Error())
	httpStatus int
	inner      error    // wrapped error
	kinds      []string // error_handler "event" names and relation
	label      string   // mostly the user configured label for e.g. access_control or backend
	message    string   // additional custom message
	synopsis   string   // seen by client
}

type GoError interface {
	error
	HTTPStatus() int
	LogError() string
}

var _ GoError = &Error{}

func New() *Error {
	return Server
}

// Status configures the http status-code which will be
// written along with this error.
func (e *Error) Status(s int) *Error {
	err := e.clone()
	err.httpStatus = s
	return err
}

// Kind appends the given kind name to the existing ones.
// Latest added should be the more specific ones.
func (e *Error) Kind(name string) *Error {
	err := e.clone()
	err.kinds = append(err.kinds, name)
	return err
}

// Kinds returns all configured kinds in reversed order so the
// most specific one gets evaluated first.
func (e *Error) Kinds() []string {
	var reversed []string
	for i := len(e.kinds); i > 0; i-- {
		reversed = append(reversed, e.kinds[i-1])
	}
	return reversed
}

func (e *Error) Label(name string) *Error {
	err := e.clone()
	err.label = name
	return err
}

func (e *Error) Message(msg string) *Error {
	err := e.clone()
	err.message = msg
	return err
}

func (e *Error) Messagef(msg string, args ...interface{}) *Error {
	return e.Message(fmt.Sprintf(msg, args...))
}

func (e *Error) With(inner error) *Error {
	err := e.clone()
	err.inner = inner
	return err
}

func (e *Error) clone() *Error {
	err := *e
	err.kinds = e.kinds[:]
	return &err
}

func (e *Error) Error() string {
	return e.synopsis
}

func (e *Error) Unwrap() error {
	return e.inner
}

// LogError contains additional context which should be used for logging purposes only.
func (e *Error) LogError() string {
	msg := appendMsg(e.synopsis, e.label, e.message)

	if e.inner != nil {
		msg = appendMsg(msg, e.inner.Error())
	}
	return msg
}

// HTTPStatus returns the configured http status code this error should be served with.
func (e *Error) HTTPStatus() int {
	return e.httpStatus
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
