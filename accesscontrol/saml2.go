package accesscontrol

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/lib"
)

type Saml2 struct {
	arrayAttributes []string
	name            string
	sp              *saml2.SAMLServiceProvider
}

func NewSAML2ACS(metadata []byte, name string, acsURL string, spEntityID string, arrayAttributes []string) (*Saml2, error) {
	metadataEntity := &types.EntityDescriptor{}
	if err := xml.Unmarshal(metadata, metadataEntity); err != nil {
		return nil, err
	}

	certStore := dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}

	for _, kd := range metadataEntity.IDPSSODescriptor.KeyDescriptors {
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
		AssertionConsumerServiceURL: acsURL,
		AudienceURI:                 spEntityID,
		IDPCertificateStore:         &certStore,
		IdentityProviderIssuer:      metadataEntity.EntityID,
	}
	if arrayAttributes != nil {
		sort.Strings(arrayAttributes)
	}
	samlObj := &Saml2{
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

func (s *Saml2) Validate(req *http.Request) error {
	err := req.ParseForm()
	if err != nil {
		return errors.Saml.With(err)
	}

	origin := eval.NewRawOrigin(req.URL)
	absAcsURL, err := lib.AbsoluteURL(s.sp.AssertionConsumerServiceURL, origin)
	if err != nil {
		return errors.Saml.With(err)
	}
	s.sp.AssertionConsumerServiceURL = absAcsURL

	encodedResponse := req.FormValue("SAMLResponse")
	req.ContentLength = 0

	assertionInfo, err := s.sp.RetrieveAssertionInfo(encodedResponse)
	if err != nil {
		return errors.Saml.With(err)
	}

	err = s.ValidateAssertionInfo(assertionInfo)
	if err != nil {
		return errors.Saml.With(err)
	}

	ass := s.GetAssertionData(assertionInfo)

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

func (s *Saml2) ValidateAssertionInfo(assertionInfo *saml2.AssertionInfo) error {
	if assertionInfo.WarningInfo.NotInAudience {
		return fmt.Errorf("wrong audience")
	}

	return nil
}

func (s *Saml2) GetAssertionData(assertionInfo *saml2.AssertionInfo) map[string]interface{} {
	attributes := make(map[string]interface{})
	// default empty slice for all arrayAttributes
	for _, arrayAttrName := range s.arrayAttributes {
		attributes[arrayAttrName] = []string{}
	}
	for _, attribute := range assertionInfo.Values {
		if !contains(s.arrayAttributes, attribute.Name) {
			for _, attributeValue := range attribute.Values {
				attributes[attribute.Name] = attributeValue.Value
			}
		} else {
			// default empty slice for this arrayAttribute (instead of nil slice)
			attributeValues := []string{}
			for _, attributeValue := range attribute.Values {
				attributeValues = append(attributeValues, attributeValue.Value)
			}
			attributes[attribute.Name] = attributeValues
		}
	}

	ass := make(map[string]interface{})
	ass["attributes"] = attributes
	ass["sub"] = assertionInfo.NameID
	if assertionInfo.SessionNotOnOrAfter != nil {
		ass["exp"] = assertionInfo.SessionNotOnOrAfter.Unix()
	}

	return ass
}
