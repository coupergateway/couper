package accesscontrol

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/avenga/couper/config/request"
)

type SAML2ACS struct {
	arrayAttributes []string
	name            string
	sp              *saml2.SAMLServiceProvider
}

func NewSAML2ACS(metadataFile string, name string, acsUrl string, spEntityId string, arrayAttributes []string) (*SAML2ACS, error) {
	p, err := filepath.Abs(metadataFile)
	if err != nil {
		return nil, err
	}

	rawMetadata, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	metadata := &types.EntityDescriptor{}
	err = xml.Unmarshal(rawMetadata, metadata)
	if err != nil {
		return nil, err
	}

	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	for _, kd := range metadata.IDPSSODescriptor.KeyDescriptors {
		for idx, xcert := range kd.KeyInfo.X509Data.X509Certificates {
			if xcert.Data == "" {
				return nil, fmt.Errorf("metadata certificate(%d) must not be empty", idx)
			}
			certData, err := base64.StdEncoding.DecodeString(xcert.Data)
			if err != nil {
				return nil, err
			}

			idpCert, err := x509.ParseCertificate(certData)
			if err != nil {
				return nil, err
			}

			certStore.Roots = append(certStore.Roots, idpCert)
		}
	}

	sp := &saml2.SAMLServiceProvider{
		AssertionConsumerServiceURL: acsUrl,
		AudienceURI:                 spEntityId,
		IDPCertificateStore:         &certStore,
		IdentityProviderIssuer:      metadata.EntityID,
		IdentityProviderSSOURL:      metadata.IDPSSODescriptor.SingleSignOnServices[0].Location,
		SignAuthnRequests:           false,
	}
	if arrayAttributes != nil {
		sort.Strings(arrayAttributes)
	}
	samlObj := &SAML2ACS{
		arrayAttributes: arrayAttributes,
		name:            name,
		sp:              sp,
	}
	return samlObj, nil
}

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	return i < len(s) && s[i] == searchterm
}

func (s *SAML2ACS) Validate(req *http.Request) error {
	err := req.ParseForm()
	if err != nil {
		return err
	}

	encodedResponse := req.FormValue("SAMLResponse")
	req.ContentLength = 0 // TODO remove when used with response {}

	assertionInfo, err := s.sp.RetrieveAssertionInfo(encodedResponse)
	if err != nil {
		return err
	}

	if assertionInfo.WarningInfo.NotInAudience {
		// TODO do we want this?
		return errors.New("Audience mismatch")
	}

	assertion := assertionInfo.Assertions[0]
	conditions := assertion.Conditions
	notBefore, err := time.Parse(time.RFC3339, conditions.NotBefore)
	if err != nil {
		return err
	}
	notOnOrAfter, err := time.Parse(time.RFC3339, conditions.NotOnOrAfter)
	if err != nil {
		return err
	}

	attributes := make(map[string]interface{})
	for _, attribute := range assertionInfo.Values {
		if !contains(s.arrayAttributes, attribute.Name) {
			for _, attributeValue := range attribute.Values {
				attributes[attribute.Name] = attributeValue.Value
			}
		} else {
			attributeValues := []string{}
			for _, attributeValue := range attribute.Values {
				attributeValues = append(attributeValues, attributeValue.Value)
			}
			attributes[attribute.Name] = attributeValues
		}
	}

	audiences := []string{}
	for _, audienceRestriction := range conditions.AudienceRestrictions {
		for _, audience := range audienceRestriction.Audiences {
			audiences = append(audiences, audience.Value)
		}
	}

	ass := make(map[string]interface{})
	ass["attributes"] = attributes
	ass["aud"] = audiences
	ass["sub"] = assertionInfo.NameID
	ass["iss"] = assertion.Issuer.Value
	ass["exp"] = notOnOrAfter.Unix()
	ass["iat"] = assertion.IssueInstant.Unix()
	ass["nbf"] = notBefore.Unix()

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[s.name] = ass
	ctx = context.WithValue(ctx, request.AccessControls, acMap)
	*req = *req.WithContext(ctx)

	return nil
}
