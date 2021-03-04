package runtime

import (
	"fmt"
	"regexp"
	"strings"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
)

var (
	// reValidFormat validates the format only, validating for a valid host or port is out of scope.
	reValidFormat  = regexp.MustCompile(`^([a-z0-9.-]+|\*)(:\*|:\d{1,5})?$`)
	reCleanPattern = regexp.MustCompile(`{([^}]+)}`)
	rePortCheck    = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})$`)
)

// validatePortHosts ensures expected host:port formats and unique hosts per port.
// Host options:
//	"*:<port>"					listen for all hosts on given port
//	"*:<port(configuredPort)>	given port equals configured default port, listen for all hosts
//	"*"							equals to "*:configuredPort"
//	"host:*"					equals to "host:configuredPort"
//	"host"						listen on configured default port for given host
func validatePortHosts(conf *config.Couper, configuredPort int) (ports, hosts, error) {
	portMap := make(ports)
	hostMap := make(hosts)
	isHostsMandatory := len(conf.Servers) > 1

	for _, srv := range conf.Servers {
		if isHostsMandatory && len(srv.Hosts) == 0 {
			return nil, nil, fmt.Errorf("hosts attribute is mandatory for multiple servers: %q", srv.Name)
		}

		srvPortMap := make(ports)
		for _, host := range srv.Hosts {
			if !reValidFormat.MatchString(host) {
				return nil, nil, fmt.Errorf("host format is invalid: %q", host)
			}

			ho, po, err := splitWildcardHostPort(host, configuredPort)
			if err != nil {
				return nil, nil, err
			}

			if _, ok := srvPortMap[po]; !ok {
				srvPortMap[po] = make(hosts)
			}

			srvPortMap[po][ho] = true

			hostMap[fmt.Sprintf("%s:%d", ho, po)] = true
		}

		// srvPortMap contains all unique host port combinations for
		// the current server and should not exist multiple times.
		for po, ho := range srvPortMap {
			if _, ok := portMap[po]; !ok {
				portMap[po] = make(hosts)
			}

			for h := range ho {
				if _, ok := portMap[po][h]; ok {
					return nil, nil, fmt.Errorf("conflict: host %q already defined for port: %d", h, po)
				}

				portMap[po][h] = true
			}
		}
	}

	return portMap, hostMap, nil
}

func validateACName(accessControls ac.Map, name, acType string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return name, fmt.Errorf("access control: label required: '%s'", acType)
	}

	if _, ok := accessControls[name]; ok {
		return name, fmt.Errorf("access control: '%s' already exists", name)
	}

	return name, nil
}

func isUnique(endpoints map[string]bool, pattern string) (bool, string) {
	pattern = reCleanPattern.ReplaceAllString(pattern, "{}")

	return !endpoints[pattern], pattern
}
