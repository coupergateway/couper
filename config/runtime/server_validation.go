package runtime

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/avenga/couper/internal/seetie"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/hashicorp/hcl/v2"
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
func validatePortHosts(conf *config.CouperFile, configuredPort int) (ports, hosts, error) {
	portMap := make(ports)
	hostMap := make(hosts)
	isHostsMandatory := len(conf.Server) > 1

	for _, srv := range conf.Server {
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

func validateOrigin(ctx *hcl.EvalContext, inline config.Inline) error {
	content, _, diags := inline.Body().PartialContent(inline.Schema(true))
	if seetie.SetSeverityLevel(diags).HasErrors() {
		return diags
	}

	ctxRange := inline.Body().MissingItemRange()

	originAttr, ok := content.Attributes["origin"]
	if !ok {
		return hcl.Diagnostics{&hcl.Diagnostic{
			Subject: &ctxRange,
			Summary: "missing backend.origin attribute",
		}}
	}
	return nil // TODO: read byte range

	originValue, diags := originAttr.Expr.Value(ctx)
	if seetie.SetSeverityLevel(diags).HasErrors() {
		return diags
	}

	origin := seetie.ValueToString(originValue)

	diagErr := &hcl.Diagnostic{
		Subject: &ctxRange,
		Summary: "invalid backend.origin value",
	}

	if origin == "" {
		diagErr.Detail = "origin attribute is required"
		return hcl.Diagnostics{diagErr}
	}

	// if origin contains fallback content with variables
	origin = strings.ReplaceAll(strings.ReplaceAll(origin, "}", ""), "${", "")

	u, err := url.Parse(origin)
	if err != nil {
		diagErr.Detail = fmt.Sprintf("url parse error: %v", err)
		return hcl.Diagnostics{diagErr}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		diagErr.Detail = fmt.Sprintf("valid http scheme required for origin: %q", origin)
		return hcl.Diagnostics{diagErr}
	}
	return nil
}

func validateACName(accessControls ac.Map, name, acType string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return name, fmt.Errorf("Missing a non-empty label for %q", acType)
	}

	if _, ok := accessControls[name]; ok {
		return name, fmt.Errorf("Label %q already exists in the ACL", name)
	}

	return name, nil
}

func isUnique(endpoints map[string]bool, pattern string) (bool, string) {
	pattern = reCleanPattern.ReplaceAllString(pattern, "{}")

	return !endpoints[pattern], pattern
}
