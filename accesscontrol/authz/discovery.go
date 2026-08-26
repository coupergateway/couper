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
	tenant             string
}

func newDiscovery(ctx context.Context, configurationURL string, batched bool,
	ttl, maxStale time.Duration, transport http.RoundTripper, log *logrus.Entry) (*discovery, error) {

	d := &discovery{
		batched:          batched,
		configurationURL: configurationURL,
	}

	// The identifier the metadata must claim is the URL without the well-known suffix. A
	// decision point that claims a different one is not the one Couper asked, which is how
	// the specification prevents a mix-up between decision points.
	identifier, tenant, err := policyDecisionPointIdentifier(configurationURL)
	if err != nil {
		return nil, err
	}
	d.expectedIdentifier = identifier
	d.tenant = tenant

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

	if err := d.verifyIdentifier(configuration.PolicyDecisionPoint); err != nil {
		return nil, err
	}

	if configuration.AccessEvaluationEndpoint == "" {
		return nil, fmt.Errorf("missing access_evaluation_endpoint in authzen configuration")
	}

	if d.batched && configuration.AccessEvaluationsEndpoint == "" {
		return nil, fmt.Errorf("missing access_evaluations_endpoint in authzen configuration")
	}

	// The endpoints must stay on the origin Couper already talks to: a backend rewrites
	// scheme and host of every request, so a foreign origin would silently send the
	// credentials of the client to the configured host. Checked here so an invalid document
	// never enters the cache.
	for _, endpoint := range []string{
		configuration.AccessEvaluationEndpoint, configuration.AccessEvaluationsEndpoint,
	} {
		if endpoint == "" {
			continue
		}
		if err := sameOrigin(d.configurationURL, endpoint); err != nil {
			return nil, err
		}
	}

	return configuration, nil
}

// endpoint returns the callout URL of the decision point.
func (d *discovery) endpoint() (string, error) {
	data, err := d.syncedResource.Data()
	// A refresh error keeps the stale document usable until max_stale invalidates it.
	configuration, ok := data.(*authzenConfiguration)
	if !ok {
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("data not authzen configuration data: %#v", data)
	}

	if d.batched {
		return configuration.AccessEvaluationsEndpoint, nil
	}

	return configuration.AccessEvaluationEndpoint, nil
}

// verifyIdentifier accepts the exact identifier of the configuration URL, or a same-origin
// one whose path ends with the tenant: a multi-tenant decision point may root its
// identifiers under a path of its own (OpenFGA claims <origin>/stores/<store_id>), and the
// pinned origin plus the pinned tenant still rule out a mix-up between decision points.
func (d *discovery) verifyIdentifier(published string) error {
	if published == d.expectedIdentifier {
		return nil
	}

	mismatchErr := fmt.Errorf("policy_decision_point %q does not match %q",
		published, d.expectedIdentifier)

	if d.tenant == "" {
		return mismatchErr
	}

	base, err := url.Parse(d.configurationURL)
	if err != nil {
		return mismatchErr
	}
	parsed, err := url.Parse(published)
	if err != nil || parsed.Scheme != base.Scheme || parsed.Host != base.Host ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return mismatchErr
	}

	// The tenant keeps its leading slash, so the suffix match ends on a segment boundary.
	if !strings.HasSuffix(strings.TrimSuffix(parsed.Path, "/"), d.tenant) {
		return mismatchErr
	}

	return nil
}

func policyDecisionPointIdentifier(configurationURL string) (identifier, tenant string, err error) {
	parsed, err := url.Parse(configurationURL)
	if err != nil {
		return "", "", err
	}
	// A relative reference has no origin to check the endpoints against.
	if !parsed.IsAbs() || parsed.Host == "" {
		return "", "", fmt.Errorf("configuration_url must be an absolute URL")
	}

	const wellKnown = "/.well-known/authzen-configuration"
	base, tenant, found := strings.Cut(parsed.Path, wellKnown)
	if !found || (tenant != "" && !strings.HasPrefix(tenant, "/")) {
		return "", "", fmt.Errorf("configuration_url must contain %q", wellKnown)
	}

	// A multi-tenant decision point appends the tenant to the well-known path, and the
	// identifier keeps that suffix as its path.
	parsed.Path = base + tenant
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimSuffix(parsed.String(), "/"), strings.TrimSuffix(tenant, "/"), nil
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
		// A protocol-relative reference names a foreign host without a scheme.
		if target.Host != "" {
			return fmt.Errorf("endpoint %q is not on the origin of the authzen configuration", endpoint)
		}
		return nil
	}

	if target.Scheme != base.Scheme || target.Host != base.Host {
		return fmt.Errorf("endpoint %q is not on the origin of the authzen configuration", endpoint)
	}

	return nil
}
