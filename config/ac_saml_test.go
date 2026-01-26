package config

import (
	"testing"
)

func TestSAML_check(t *testing.T) {
	tests := []struct {
		name           string
		saml           *SAML
		expectedErrMsg string
	}{
		{
			name: "valid file-based config",
			saml: &SAML{
				IdpMetadataFile: "/path/to/metadata.xml",
				SpAcsURL:        "http://example.com/acs",
				SpEntityID:      "my-sp",
			},
			expectedErrMsg: "",
		},
		{
			name: "valid URL-based config",
			saml: &SAML{
				IdpMetadataURL: "https://idp.example.com/metadata",
				SpAcsURL:       "http://example.com/acs",
				SpEntityID:     "my-sp",
			},
			expectedErrMsg: "",
		},
		{
			name: "missing both file and url",
			saml: &SAML{
				SpAcsURL:   "http://example.com/acs",
				SpEntityID: "my-sp",
			},
			expectedErrMsg: "one of idp_metadata_file or idp_metadata_url is required",
		},
		{
			name: "both file and url specified",
			saml: &SAML{
				IdpMetadataFile: "/path/to/metadata.xml",
				IdpMetadataURL:  "https://idp.example.com/metadata",
				SpAcsURL:        "http://example.com/acs",
				SpEntityID:      "my-sp",
			},
			expectedErrMsg: "idp_metadata_file and idp_metadata_url are mutually exclusive",
		},
		{
			name: "backend with file-based config",
			saml: &SAML{
				IdpMetadataFile: "/path/to/metadata.xml",
				BackendName:     "my-backend",
				SpAcsURL:        "http://example.com/acs",
				SpEntityID:      "my-sp",
			},
			expectedErrMsg: "backend is only valid with idp_metadata_url",
		},
		{
			name: "metadata_ttl with file-based config",
			saml: &SAML{
				IdpMetadataFile: "/path/to/metadata.xml",
				MetadataTTL:     "1h",
				SpAcsURL:        "http://example.com/acs",
				SpEntityID:      "my-sp",
			},
			expectedErrMsg: "metadata_ttl is only valid with idp_metadata_url",
		},
		{
			name: "metadata_max_stale with file-based config",
			saml: &SAML{
				IdpMetadataFile:  "/path/to/metadata.xml",
				MetadataMaxStale: "1h",
				SpAcsURL:         "http://example.com/acs",
				SpEntityID:       "my-sp",
			},
			expectedErrMsg: "metadata_max_stale is only valid with idp_metadata_url",
		},
		{
			name: "URL-based config with TTL and max_stale",
			saml: &SAML{
				IdpMetadataURL:   "https://idp.example.com/metadata",
				MetadataTTL:      "2h",
				MetadataMaxStale: "30m",
				SpAcsURL:         "http://example.com/acs",
				SpEntityID:       "my-sp",
			},
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.saml.check()
			if tt.expectedErrMsg == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedErrMsg)
				} else if err.Error() != tt.expectedErrMsg {
					t.Errorf("expected error %q, got %q", tt.expectedErrMsg, err.Error())
				}
			}
		})
	}
}
