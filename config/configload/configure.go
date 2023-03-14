package configload

import (
	"fmt"

	"github.com/avenga/couper/config"
	hclbody "github.com/avenga/couper/config/body"
	"github.com/avenga/couper/config/configload/collect"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func configureBlocks(helper *helper) error {
	var err error

	for _, outerBlock := range helper.content.Blocks {
		switch outerBlock.Type {
		case definitions:
			backendContent, leftOver, diags := outerBlock.Body.PartialContent(backendBlockSchema)
			if diags.HasErrors() {
				return diags
			}

			// backends first
			if backendContent != nil {
				for _, be := range backendContent.Blocks {
					helper.addBackend(be)
				}

				if err = helper.configureDefinedBackends(); err != nil {
					return err
				}
			}

			// decode all other blocks into definition struct
			if diags = gohcl.DecodeBody(leftOver, helper.context, helper.config.Definitions); diags.HasErrors() {
				return diags
			}

			if err = helper.configureACBackends(); err != nil {
				return err
			}

			acErrorHandler := collect.ErrorHandlerSetters(helper.config.Definitions)
			if err = configureErrorHandler(acErrorHandler, helper); err != nil {
				return err
			}

		case settings:
			if diags := gohcl.DecodeBody(outerBlock.Body, helper.context, helper.config.Settings); diags.HasErrors() {
				return diags
			}
		}
	}

	return nil
}

func configureJWTSigningProfile(helper *helper) *errors.Error {
	for _, profile := range helper.config.Definitions.JWTSigningProfile {
		if profile.Headers != nil {
			expression, _ := profile.Headers.Value(nil)
			headers := seetie.ValueToMap(expression)

			if _, exists := headers["alg"]; exists {
				return errors.Configuration.Label(profile.Name).With(fmt.Errorf(`"alg" cannot be set via "headers"`))
			}
		}
	}

	return nil
}

func configureSAML(helper *helper) *errors.Error {
	for _, saml := range helper.config.Definitions.SAML {
		metadata, err := reader.ReadFromFile("saml2 idp_metadata_file", saml.IdpMetadataFile)
		if err != nil {
			return errors.Configuration.Label(saml.Name).With(err)
		}

		saml.MetadataBytes = metadata
	}

	return nil
}

func configureJWTSigningConfig(helper *helper) (map[string]*lib.JWTSigningConfig, *errors.Error) {
	jwtSigningConfigs := make(map[string]*lib.JWTSigningConfig)

	for _, profile := range helper.config.Definitions.JWTSigningProfile {
		signConf, err := lib.NewJWTSigningConfigFromJWTSigningProfile(profile, nil)
		if err != nil {
			return nil, errors.Configuration.Label(profile.Name).With(err)
		}

		jwtSigningConfigs[profile.Name] = signConf
	}

	for _, jwt := range helper.config.Definitions.JWT {
		signConf, err := lib.NewJWTSigningConfigFromJWT(jwt)
		if err != nil {
			return nil, errors.Configuration.Label(jwt.Name).With(err)
		}

		if signConf != nil {
			jwtSigningConfigs[jwt.Name] = signConf
		}
	}

	return jwtSigningConfigs, nil
}

// Reads per server block and merge backend settings which results in a final server configuration.
func configureServers(helper *helper, definedACs map[string]struct{}, body *hclsyntax.Body) error {
	var err error

	for _, serverBlock := range hclbody.BlocksOfType(body, server) {
		serverConfig := &config.Server{}
		if diags := gohcl.DecodeBody(serverBlock.Body, helper.context, serverConfig); diags.HasErrors() {
			return diags
		}

		// Set the server name since gohcl.DecodeBody decoded the body and not the block.
		if len(serverBlock.Labels) > 0 {
			serverConfig.Name = serverBlock.Labels[0]
		}

		if err = checkReferencedAccessControls(serverBlock.Body, serverConfig.AccessControl, serverConfig.DisableAccessControl, definedACs); err != nil {
			return err
		}

		for _, fileConfig := range serverConfig.Files {
			if err := checkReferencedAccessControls(fileConfig.HCLBody(), fileConfig.AccessControl, fileConfig.DisableAccessControl, definedACs); err != nil {
				return err
			}
		}

		for _, spaConfig := range serverConfig.SPAs {
			if err := checkReferencedAccessControls(spaConfig.HCLBody(), spaConfig.AccessControl, spaConfig.DisableAccessControl, definedACs); err != nil {
				return err
			}
		}

		err = configureAPIs(helper, serverConfig.APIs, definedACs)
		if err != nil {
			return err
		}

		// Standalone endpoints
		err = refineEndpoints(helper, serverConfig.Endpoints, true, definedACs)
		if err != nil {
			return err
		}

		helper.config.Servers = append(helper.config.Servers, serverConfig)
	}

	return nil
}

// Reads api blocks and merge backends with server and definitions backends.
func configureAPIs(helper *helper, apis config.APIs, definedACs map[string]struct{}) error {
	var err error

	for _, apiConfig := range apis {
		apiBody := apiConfig.HCLBody()

		if apiConfig.AllowedMethods != nil && len(apiConfig.AllowedMethods) > 0 {
			if err = validMethods(apiConfig.AllowedMethods, apiBody.Attributes["allowed_methods"]); err != nil {
				return err
			}
		}

		if err := checkReferencedAccessControls(apiBody, apiConfig.AccessControl, apiConfig.DisableAccessControl, definedACs); err != nil {
			return err
		}

		rp := apiBody.Attributes["required_permission"]
		if rp != nil {
			apiConfig.RequiredPermission = rp.Expr
		}

		err = refineEndpoints(helper, apiConfig.Endpoints, true, definedACs)
		if err != nil {
			return err
		}

		err = checkPermissionMixedConfig(apiConfig)
		if err != nil {
			return err
		}

		apiConfig.CatchAllEndpoint = newCatchAllEndpoint()

		apiErrorHandler := collect.ErrorHandlerSetters(apiConfig)
		if err = configureErrorHandler(apiErrorHandler, helper); err != nil {
			return err
		}
	}

	return nil
}

func configureJobs(helper *helper) error {
	var err error

	for _, job := range helper.config.Definitions.Job {
		attrs := job.Remain.(*hclsyntax.Body).Attributes
		r := attrs["interval"].Expr.Range()

		job.IntervalDuration, err = config.ParseDuration("interval", job.Interval, -1)
		if err != nil {
			return newDiagErr(&r, err.Error())
		} else if job.IntervalDuration == -1 {
			return newDiagErr(&r, "invalid duration")
		}

		endpointConf := &config.Endpoint{
			Pattern:  job.Name, // for error messages
			Remain:   job.Remain,
			Requests: job.Requests,
		}

		err = refineEndpoints(helper, config.Endpoints{endpointConf}, false, nil)
		if err != nil {
			return err
		}

		job.Endpoint = endpointConf
	}

	return nil
}
