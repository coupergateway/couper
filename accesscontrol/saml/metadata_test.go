package saml

import (
	"testing"
)

func TestStaticMetadata(t *testing.T) {
	validMetadata := []byte(`<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example.com">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIICpDCCAYwCCQDU+pQ4P2HlGjANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjMwMTAxMDAwMDAwWhcNMjQwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDH
xt2q8wq1Gq9QWLLe7fZHlDx8r2ABTaS9R7z9j7i8l5xbM1+QoB8HA0P8Jq2Cy3l3
tG4rJzPTWbQlOQmPO7WzNxFw/yBPv4J9a4f6gqE5ZEWwvYXnJz7fJf5fJf5fJf5f
Jf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5f
Jf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5f
Jf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5fJf5f
AgMBAAEwDQYJKoZIhvcNAQELBQADggEBAH0aH9zgLYs3cNvBsaHyLMQ9cJqE9G2t
V9P0l8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8
w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8
w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8
w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8w8
w8w8w8w8</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
  </IDPSSODescriptor>
</EntityDescriptor>`)

	t.Run("valid metadata", func(t *testing.T) {
		provider, err := NewStaticMetadata(validMetadata)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		metadata, err := provider.Metadata()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if metadata.EntityID != "https://idp.example.com" {
			t.Errorf("expected entityID %q, got %q", "https://idp.example.com", metadata.EntityID)
		}
	})

	t.Run("invalid XML", func(t *testing.T) {
		_, err := NewStaticMetadata([]byte("not xml"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns same metadata on multiple calls", func(t *testing.T) {
		provider, err := NewStaticMetadata(validMetadata)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		m1, _ := provider.Metadata()
		m2, _ := provider.Metadata()

		if m1 != m2 {
			t.Error("expected same metadata instance on multiple calls")
		}
	})
}

func TestStaticMetadata_EmptyMetadata(t *testing.T) {
	_, err := NewStaticMetadata([]byte(""))
	if err == nil {
		t.Error("expected error for empty metadata")
	}
}
