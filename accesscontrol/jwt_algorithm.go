package accesscontrol

const (
	AlgorithmUnknown Algorithm = iota - 1
	_
	AlgorithmRSA256
	AlgorithmRSA384
	AlgorithmRSA512
	AlgorithmHMAC256
	AlgorithmHMAC384
	AlgorithmHMAC512
)

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
	default:
		return AlgorithmUnknown
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
	default:
		return "Unknown"
	}
}
