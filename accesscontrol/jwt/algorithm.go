package jwt

type (
	Algorithm int
)

const (
	AlgorithmUnknown Algorithm = iota - 1
	_
	AlgorithmRSA256
	AlgorithmRSA384
	AlgorithmRSA512
	AlgorithmHMAC256
	AlgorithmHMAC384
	AlgorithmHMAC512
	AlgorithmECDSA256
	AlgorithmECDSA384
	AlgorithmECDSA512
)

var RSAAlgorithms = []Algorithm{AlgorithmRSA256, AlgorithmRSA384, AlgorithmRSA512}
var ECDSAlgorithms = []Algorithm{AlgorithmECDSA256, AlgorithmECDSA384, AlgorithmECDSA512}

func NewAlgorithm(a string) Algorithm {
	switch a {
	case "RS256":
		return AlgorithmRSA256
	case "RS384":
		return AlgorithmRSA384
	case "RS512":
		return AlgorithmRSA512
	case "HS256":
		return AlgorithmHMAC256
	case "HS384":
		return AlgorithmHMAC384
	case "HS512":
		return AlgorithmHMAC512
	case "ES256":
		return AlgorithmECDSA256
	case "ES384":
		return AlgorithmECDSA384
	case "ES512":
		return AlgorithmECDSA512
	default:
		return AlgorithmUnknown
	}
}

func (a Algorithm) IsHMAC() bool {
	switch a {
	case AlgorithmHMAC256, AlgorithmHMAC384, AlgorithmHMAC512:
		return true
	default:
		return false
	}
}

func (a Algorithm) String() string {
	switch a {
	case AlgorithmRSA256:
		return "RS256"
	case AlgorithmRSA384:
		return "RS384"
	case AlgorithmRSA512:
		return "RS512"
	case AlgorithmHMAC256:
		return "HS256"
	case AlgorithmHMAC384:
		return "HS384"
	case AlgorithmHMAC512:
		return "HS512"
	case AlgorithmECDSA256:
		return "ES256"
	case AlgorithmECDSA384:
		return "ES384"
	case AlgorithmECDSA512:
		return "ES512"
	default:
		return "Unknown"
	}
}
