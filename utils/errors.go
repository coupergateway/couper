package utils

const unknownError = "Unknown error"

var errorsMap = map[int]string{
	1001: "Route not found",
	2001: "SPA route not found",
	3001: "Files route not found",
	4001: "API route not found",
}

// GetErrorMessage returns an error message associated with an error code
func GetErrorMessage(code int) string {
	if m, ok := errorsMap[code]; ok {
		return m
	}

	return unknownError
}
