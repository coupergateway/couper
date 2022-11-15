package errors

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

type Error struct {
	// client: synopsis
	// log: synopsis label message inner(Error())
	httpStatus int
	inner      error    // wrapped error
	isParent   bool     // is a parent (no leaf) node in the error type hierarchy
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
	Unwrap() error
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
	e.isParent = true
	err.isParent = false
	err.kinds = append([]string{name}, err.kinds...)
	return err
}

// Kinds returns all configured kinds, the
// most specific one gets evaluated first.
func (e *Error) Kinds() []string {
	var kinds []string

	if eer, ok := e.inner.(*Error); ok {
		kinds = eer.Kinds()[:]
	}

	return append(kinds, e.kinds...)
}

// Context appends the given context block type to the existing ones.
func (e *Error) Context(name string) *Error {
	err := e.clone()
	err.Contexts = append(err.Contexts, name)
	return err
}

func (e *Error) IsParent() bool {
	return e.isParent
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
	err.isParent = e.isParent
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
	if diags := e.getDiags(); diags != nil {
		return diags.Error()
	}

	msg := AppendMsg(e.synopsis, e.label, e.message)

	if e.inner != nil {
		if inner, ok := e.inner.(*Error); ok {
			if Equals(e, inner) {
				inner.synopsis = "" // at least for one level, prevent duplicated synopsis
			}
			return AppendMsg(msg, inner.LogError())
		}
		msg = AppendMsg(msg, e.inner.Error())
	}

	return msg
}

func (e *Error) getDiags() hcl.Diagnostics {
	if e.inner != nil {
		if diags, ok := e.inner.(hcl.Diagnostics); ok {
			return diags
		}

		if inner, ok := e.inner.(*Error); ok {
			return inner.getDiags()
		}
	}

	return nil
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
