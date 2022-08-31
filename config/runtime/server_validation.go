package runtime

import (
	"fmt"
	"regexp"
)

// reValidFormat validates the format only, validating for a valid host or port is out of scope.
var reValidFormat = regexp.MustCompile(`^([a-z0-9.-]+|\*)(:\*|:\d{1,5})?$`)

func validateHosts(serverName string, hosts []string, isHostsMandatory bool) error {
	if isHostsMandatory && len(hosts) == 0 {
		return fmt.Errorf("the hosts attribute is mandatory for multiple servers: %q", serverName)
	}

	for _, host := range hosts {
		if !reValidFormat.MatchString(host) {
			return fmt.Errorf("the host format is invalid: %q", host)
		}
	}

	return nil
}
