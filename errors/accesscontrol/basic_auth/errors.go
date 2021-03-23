//go:generate stringer -type=Error -output=./errors_string.go

package jwt

var _ error = Error(0)

type Error uint8

const (
	NotConfigured Error = iota
	MissingCredentials
	Unauthorized
	ParseErrorLineInvalid
	ParseErrorLineLengthExceeded
	ParseErrorMalformedPassword
	ParseErrorMultipleUser
	ParseErrorAlgorithmNotSupported
)

func (i Error) Error() string {
	return i.String()
}
