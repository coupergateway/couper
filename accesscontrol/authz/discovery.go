package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/coupergateway/couper/resource"
)

const defaultConfigurationTTL = time.Hour

// authzenConfiguration is the metadata document of a policy decision point,
// served at /.well-known/authzen-configuration.
type authzenConfiguration struct {
	AccessEvaluationEndpoint  string `json:"access_evaluation_endpoint"`
	AccessEvaluationsEndpoint string `json:"access_evaluations_endpoint"`
	PolicyDecisionPoint       string `json:"policy_decision_point"`
}

// discovery resolves the callout endpoint from the metadata of a decision point.
type discovery struct {
	batched            bool
	configurationURL   string
	expectedIdentifier string
	syncedResource     *resource.SyncedResource
}

func newDiscovery(ctx context.Context, name, configurationURL string, batched bool,
	ttl, maxStale time.Duration, transport http.RoundTripper, log *logrus.Entry) (*discovery, error) {

	d := &discovery{
		batched:          batched,
		configurationURL: configurationURL,
	}

	// The identifier the metadata must claim is the URL without the well-known suffix. A
	// decision point that claims a different one is not the one Couper asked, which is how
	// the specification prevents a mix-up between decision points.
	identifier, err := policyDecisionPointIdentifier(configurationURL)
	if err != nil {
		return nil, err
	}
	d.expectedIdentifier = identifier

	d.syncedResource, err = resource.NewSyncedResource(ctx, "", "",
		configurationURL, transport, roundTripName, ttl, maxStale, d, log)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Unmarshal implements the resource.ResourceUnmarshaller interface.
func (d *discovery) Unmarshal(raw []byte) (interface{}, error) {
	configuration := &authzenConfiguration{}
	if err := json.Unmarshal(raw, configuration); err != nil {
		return nil, err
	}

	if configuration.PolicyDecisionPoint != d.expectedIdentifier {
		return nil, fmt.Errorf("policy_decision_point %q does not match %q",
			configuration.PolicyDecisionPoint, d.expectedIdentifier)
	}

	if configuration.AccessEvaluationEndpoint == "" {
		return nil, fmt.Errorf("missing access_evaluation_endpoint in authzen configuration")
	}

	if d.batched && configuration.AccessEvaluationsEndpoint == "" {
		return nil, fmt.Errorf("missing access_evaluations_endpoint in authzen configuration")
	}

	return configuration, nil
}

// endpoint returns the callout URL of the decision point. The endpoint must stay on the
// origin Couper already talks to: a backend rewrites scheme and host of every request, so a
// foreign origin would silently send the credentials of the client to the configured host.
func (d *discovery) endpoint() (string, error) {
	data, err := d.syncedResource.Data()
	if err != nil {
		return "", err
	}

	configuration, ok := data.(*authzenConfiguration)
	if !ok {
		return "", fmt.Errorf("data not authzen configuration data: %#v", data)
	}

	endpoint := configuration.AccessEvaluationEndpoint
	if d.batched {
		endpoint = configuration.AccessEvaluationsEndpoint
	}

	if err = sameOrigin(d.configurationURL, endpoint); err != nil {
		return "", err
	}

	return endpoint, nil
}

func policyDecisionPointIdentifier(configurationURL string) (string, error) {
	parsed, err := url.Parse(configurationURL)
	if err != nil {
		return "", err
	}

	const wellKnown = "/.well-known/authzen-configuration"
	if !strings.Contains(parsed.Path, wellKnown) {
		return "", fmt.Errorf("configuration_url must contain %q", wellKnown)
	}

	// A multi-tenant decision point appends the tenant to the well-known path, and the
	// identifier keeps that suffix as its path.
	base, tenant, _ := strings.Cut(parsed.Path, wellKnown)
	parsed.Path = base + tenant
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimSuffix(parsed.String(), "/"), nil
}

func sameOrigin(configurationURL, endpoint string) error {
	base, err := url.Parse(configurationURL)
	if err != nil {
		return err
	}

	target, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	if !target.IsAbs() {
		return nil
	}

	if target.Scheme != base.Scheme || target.Host != base.Host {
		return fmt.Errorf("endpoint %q is not on the origin of the authzen configuration", endpoint)
	}

	return nil
}
