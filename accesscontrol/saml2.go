package accesscontrol

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"

	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
)

type Saml2 struct {
	arrayAttributes []string
	name            string
	sp              *saml2.SAMLServiceProvider
}

func NewSAML2ACS(metadataFile string, name string, acsUrl string, spEntityId string, arrayAttributes []string) (*Saml2, error) {
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
		return err
	}

	encodedResponse := req.FormValue("SAMLResponse")
	req.ContentLength = 0

	assertionInfo, err := s.sp.RetrieveAssertionInfo(encodedResponse)
	if err != nil {
		return err
	}

	err = s.ValidateAssertionInfo(assertionInfo)
	if err != nil {
		return err
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
		return errors.Types["saml2"].Message("wrong audience")
	}

	return nil
}

func (s *Saml2) GetAssertionData(assertionInfo *saml2.AssertionInfo) map[string]interface{} {
	attributes := make(map[string]interface{})
	for _, attribute := range assertionInfo.Values {
		if !contains(s.arrayAttributes, attribute.Name) {
			for _, attributeValue := range attribute.Values {
				attributes[attribute.Name] = attributeValue.Value
			}
		} else {
			var attributeValues []string
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
