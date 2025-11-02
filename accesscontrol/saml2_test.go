package accesscontrol_test

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	saml2 "github.com/russellhaering/gosaml2"
	"github.com/russellhaering/gosaml2/types"

	ac "github.com/coupergateway/couper/accesscontrol"
	"github.com/coupergateway/couper/config/reader"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/internal/test"
)

func Test_NewSAML2ACS(t *testing.T) {
	helper := test.New(t)

	type testCase struct {
		metadataFile, acsURL, spEntityID string
		arrayAttributes                  []string
		expErrMsg                        string
		shouldFail                       bool
	}

	for _, tc := range []testCase{
		{"testdata/idp-metadata.xml", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{}, "", false},
		{"not-there.xml", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{}, "not-there.xml: no such file or directory", true},
	} {
		metadata, err := reader.ReadFromAttrFile("saml2", "", tc.metadataFile)
		if err != nil {
			readErr := err.(errors.GoError)
			if tc.shouldFail {
				if !strings.HasSuffix(readErr.LogError(), tc.expErrMsg) {
					t.Errorf("Want: %q, got: %q", tc.expErrMsg, readErr.LogError())
				}
				continue
			}
			t.Error(err)
			continue
		}

		_, err = ac.NewSAML2ACS(metadata, "test", tc.acsURL, tc.spEntityID, tc.arrayAttributes)
		helper.Must(err)
	}
}

func Test_SAML2ACS_Validate(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	if err != nil || metadata == nil {
		t.Fatal("Expected a metadata object")
	}
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	type testCase struct {
		name    string
		payload string
		wantErr bool
	}
	for _, tc := range []testCase{
		{
			"invalid body",
			"1qp4ghn1pin",
			true,
		},
		{
			"invalid SAMLResponse",
			"SAMLResponse=1qp4ghn1pin",
			true,
		},
		{
			"invalid url-encoded SAMLResponse",
			"SAMLResponse=" + url.QueryEscape("abcde"),
			true,
		},
		{
			"invalid base64- and url-encoded SAMLResponse",
			"SAMLResponse=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte("abcde"))),
			true,
		},
		// TODO how to make test for valid SAMLResponse?
	} {
		t.Run(tc.name, func(subT *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tc.payload))
			if err := sa.Validate(req); (err != nil) != tc.wantErr {
				subT.Errorf("%s: Validate() error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func Test_SAML2ACS_ValidateAssertionInfo(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	if err != nil {
		t.Fatal(err)
	}
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	type testCase struct {
		name          string
		assertionInfo *saml2.AssertionInfo
		wantErr       bool
	}
	for _, tc := range []testCase{
		{
			"assertion mismatch",
			&saml2.AssertionInfo{
				WarningInfo: &saml2.WarningInfo{
					NotInAudience: true,
				},
			},
			true,
		},
		{
			"assertion match",
			&saml2.AssertionInfo{
				WarningInfo: &saml2.WarningInfo{},
			},
			false,
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			if err = sa.ValidateAssertionInfo(tc.assertionInfo); (err != nil) != tc.wantErr {
				subT.Errorf("%s: ValidateAssertionInfo() error = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func Test_SAML2ACS_GetAssertionData(t *testing.T) {
	metadata, err := reader.ReadFromAttrFile("saml2", "", "testdata/idp-metadata.xml")
	if err != nil || metadata == nil {
		t.Fatal("Expected a metadata object")
	}
	sa, err := ac.NewSAML2ACS(metadata, "test", "http://www.examle.org/saml/acs", "my-sp-entity-id", []string{"memberOf"})
	if err != nil || sa == nil {
		t.Fatal("Expected a saml acs object")
	}

	valuesWith2MemberOf := saml2.Values{
		"displayName": types.Attribute{
			Name: "displayName",
			Values: []types.AttributeValue{
				{
					Value: "John Doe",
				},
				{
					Value: "Jane Doe",
				},
			},
		},
		"memberOf": types.Attribute{
			Name: "memberOf",
			Values: []types.AttributeValue{
				{
					Value: "group1",
				},
				{
					Value: "group2",
				},
			},
		},
	}
	valuesWith1MemberOf := saml2.Values{
		"displayName": types.Attribute{
			Name: "displayName",
			Values: []types.AttributeValue{
				{
					Value: "Jane Doe",
				},
			},
		},
		"memberOf": types.Attribute{
			Name: "memberOf",
			Values: []types.AttributeValue{
				{
					Value: "group1",
				},
			},
		},
	}
	valuesEmptyMemberOf := saml2.Values{
		"displayName": types.Attribute{
			Name: "displayName",
			Values: []types.AttributeValue{
				{
					Value: "Jane Doe",
				},
			},
		},
		"memberOf": types.Attribute{
			Name:   "memberOf",
			Values: []types.AttributeValue{},
		},
	}
	valuesMissingMemberOf := saml2.Values{
		"displayName": types.Attribute{
			Name: "displayName",
			Values: []types.AttributeValue{
				{
					Value: "Jane Doe",
				},
			},
		},
	}
	var authnStatement types.AuthnStatement
	err = xml.Unmarshal([]byte(`<AuthnStatement xmlns="urn:oasis:names:tc:SAML:2.0:assertion" SessionNotOnOrAfter="2020-11-13T17:06:00Z"/>`), &authnStatement)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		name          string
		assertionInfo *saml2.AssertionInfo
		want          map[string]interface{}
	}
	for _, tc := range []testCase{
		{
			"without exp, with 2 memberOf",
			&saml2.AssertionInfo{
				NameID: "abc12345",
				Values: valuesWith2MemberOf,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf": []string{
						"group1",
						"group2",
					},
				},
			},
		},
		{
			"without exp, with 1 memberOf",
			&saml2.AssertionInfo{
				NameID: "abc12345",
				Values: valuesWith1MemberOf,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf": []string{
						"group1",
					},
				},
			},
		},
		{
			"with exp, with memberOf",
			&saml2.AssertionInfo{
				NameID:              "abc12345",
				SessionNotOnOrAfter: authnStatement.SessionNotOnOrAfter,
				Values:              valuesWith2MemberOf,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"exp": int64(1605287160),
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf": []string{
						"group1",
						"group2",
					},
				},
			},
		},
		{
			"without exp, empty memberOf",
			&saml2.AssertionInfo{
				NameID: "abc12345",
				Values: valuesEmptyMemberOf,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf":    []string{},
				},
			},
		},
		{
			"without exp, without memberOf",
			&saml2.AssertionInfo{
				NameID: "abc12345",
				Values: valuesMissingMemberOf,
			},
			map[string]interface{}{
				"sub": "abc12345",
				"attributes": map[string]interface{}{
					"displayName": "Jane Doe",
					"memberOf":    []string{},
				},
			},
		},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			assertionData := sa.GetAssertionData(tc.assertionInfo)
			if !cmp.Equal(tc.want, assertionData) {
				subT.Errorf("%s", cmp.Diff(tc.want, assertionData))
			}
		})
	}
}
