package base64url

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func Decode(data string) ([]byte, error) {
	data = strings.Replace(data, "-", "+", -1)
	data = strings.Replace(data, "_", "/", -1)

	switch len(data) % 4 {
	case 0:
		// No pad chars in this case
	case 2:
		// Two pad chars
		data += "=="
		// One pad char
	case 3:
		data += "="
	default:
		return nil, fmt.Errorf("invalid base64url string")
	}

	return base64.StdEncoding.DecodeString(data)
}

func Encode(data []byte) string {
	result := base64.StdEncoding.EncodeToString(data)
	result = strings.Replace(result, "+", "-", -1)
	result = strings.Replace(result, "/", "_", -1)
	result = strings.TrimRight(result, "=")

	return result
}
