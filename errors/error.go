package errors

import (
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	// client: synopsis
	// log: synopsis label message inner(Error())
	httpStatus int
	inner      error    // wrapped error
	kinds      []string // error_handler "event" names and relation
	label      string   // mostly the user configured label for e.g. access_control or backend
	message    string   // additional custom message
	synopsis   string   // seen by client
	Contexts   []string // context block types (e.g. api or endpoint)
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

// Equals returns true if the base type and kinds are equal.
func Equals(a, b error) bool {
	aerr, oka := a.(*Error)
	berr, okb := b.(*Error)
	if !oka || !okb {
		return a == b
	}
	return aerr.synopsis == berr.synopsis &&
		strings.Join(aerr.kinds, "") == strings.Join(berr.kinds, "")
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

	if eer, ok := e.inner.(*Error); ok {
		k := eer.Kinds()
		if len(k) > 0 {
			reversed = append(reversed, k[0])
		}
	}

	return reversed
}

// Context appends the given context block type to the existing ones.
func (e *Error) Context(name string) *Error {
	err := e.clone()
	err.Contexts = append(err.Contexts, name)
	return err
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
	if inner == nil {
		return e
	}
	err := e.clone()
	err.inner = inner
	return err
}

func (e *Error) clone() *Error {
	err := *e
	err.kinds = e.kinds[:]
	err.Contexts = e.Contexts[:]
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
	msg := AppendMsg(e.synopsis, e.label, e.message)

	if e.inner != nil {
		if innr, ok := e.inner.(*Error); ok {
			if Equals(e, innr) {
				innr.synopsis = "" // at least for one level, prevent duplicated synopsis
			}
			return AppendMsg(msg, innr.LogError())
		}
		msg = AppendMsg(msg, e.inner.Error())
	}

	return msg
}

// HTTPStatus returns the configured http status code this error should be served with.
func (e *Error) HTTPStatus() int {
	if e.httpStatus == 0 {
		return http.StatusInternalServerError
	}
	return e.httpStatus
}

// AppendMsg chains the given strings with ": " as separator.
func AppendMsg(target string, messages ...string) string {
	result := target
	for _, m := range messages {
		if result != "" && m != "" {
			result += ": "
		}
		result += m
	}
	return result
}
