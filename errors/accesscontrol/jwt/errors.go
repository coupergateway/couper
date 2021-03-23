//go:generate stringer -type=Error -output=./errors_string.go

package jwt

var _ error = Error(0)

type Error uint8

const (
	NotConfigured Error = iota
	AlgorithmNotSupported
	BearerRequired
	ClaimRequired
	ClaimValueInvalid
	ClaimValueInvalidType
	KeyRequired
	SignatureInvalid
	SourceInvalid
	TokenExpired
	TokenRequired
	Unauthorized
)

func (i Error) Error() string {
	return i.String()
}
