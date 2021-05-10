package runtime

import (
	"fmt"
	"regexp"
)

var (
	// reValidFormat validates the format only, validating for a valid host or port is out of scope.
	reValidFormat  = regexp.MustCompile(`^([a-z0-9.-]+|\*)(:\*|:\d{1,5})?$`)
	reCleanPattern = regexp.MustCompile(`{([^}]+)}`)
)

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

func isUnique(endpoints map[string]bool, pattern string) (bool, string) {
	pattern = reCleanPattern.ReplaceAllString(pattern, "{}")

	return !endpoints[pattern], pattern
}
