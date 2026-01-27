package saml

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/russellhaering/gosaml2/types"

	"github.com/coupergateway/couper/config"
	jsn "github.com/coupergateway/couper/json"
)

// MetadataProvider abstracts static (file-based) and dynamic (URL-based) metadata sources.
type MetadataProvider interface {
	Metadata() (*types.EntityDescriptor, error)
}

// StaticMetadata provides metadata from a static source (file or inline bytes).
type StaticMetadata struct {
	descriptor *types.EntityDescriptor
}

// NewStaticMetadata creates a MetadataProvider from raw XML bytes.
func NewStaticMetadata(raw []byte) (*StaticMetadata, error) {
	descriptor := &types.EntityDescriptor{}
	if err := xml.Unmarshal(raw, descriptor); err != nil {
		return nil, err
	}
	return &StaticMetadata{descriptor: descriptor}, nil
}

// Metadata returns the pre-parsed entity descriptor.
func (s *StaticMetadata) Metadata() (*types.EntityDescriptor, error) {
	return s.descriptor, nil
}

// SyncedMetadata provides metadata from a URL with automatic refresh.
type SyncedMetadata struct {
	syncedJSON *jsn.SyncedJSON
}

// NewSyncedMetadata creates a MetadataProvider that fetches and caches metadata from a URL.
func NewSyncedMetadata(ctx context.Context, uri string, ttl string, maxStale string, transport http.RoundTripper) (*SyncedMetadata, error) {
	timetolive, err := config.ParseDuration("metadata_ttl", ttl, time.Hour)
	if err != nil {
		return nil, err
	}
	maxStaleTime, err := config.ParseDuration("metadata_max_stale", maxStale, time.Hour)
	if err != nil {
		return nil, err
	}

	sm := &SyncedMetadata{}
	sm.syncedJSON, err = jsn.NewSyncedJSON(ctx, "", "idp_metadata_url", uri, transport, "saml_metadata", timetolive, maxStaleTime, sm)
	return sm, err
}

// Metadata returns the current cached entity descriptor.
func (s *SyncedMetadata) Metadata() (*types.EntityDescriptor, error) {
	data, err := s.syncedJSON.Data()
	if err != nil {
		return nil, err
	}
	descriptor, ok := data.(*types.EntityDescriptor)
	if !ok {
		return nil, fmt.Errorf("unexpected metadata type: %T", data)
	}
	return descriptor, nil
}

// Unmarshal implements jsn.SyncedJSONUnmarshaller for XML metadata.
func (s *SyncedMetadata) Unmarshal(rawXML []byte) (interface{}, error) {
	descriptor := &types.EntityDescriptor{}
	err := xml.Unmarshal(rawXML, descriptor)
	return descriptor, err
}
