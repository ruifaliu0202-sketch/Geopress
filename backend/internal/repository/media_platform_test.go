package repository

import (
	"testing"

	"geopress/backend/internal/domain"
)

func TestDecodeMediaPlatformCapabilitiesFallsBackForEmptyJSON(t *testing.T) {
	caps := decodeMediaPlatformCapabilities(`{}`, true, false, true, []string{"qrLogin"})

	if !caps.HasCapability(domain.ConnectorCapabilityAuthorization) {
		t.Fatalf("empty JSON should fall back to legacy authorization capability: %#v", caps)
	}
	if !caps.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("empty JSON should fall back to legacy publish capability: %#v", caps)
	}
	if len(caps.PublishModes) != 2 || caps.PublishModes[1] != domain.PublishModeAPI {
		t.Fatalf("publish modes = %#v, want manual and api", caps.PublishModes)
	}
}

func TestDecodeStringSliceNeverReturnsNil(t *testing.T) {
	if values := decodeStringSlice(""); values == nil || len(values) != 0 {
		t.Fatalf("empty credential field JSON should decode to empty slice: %#v", values)
	}
	if values := decodeStringSlice(`not-json`); values == nil || len(values) != 0 {
		t.Fatalf("invalid credential field JSON should decode to empty slice: %#v", values)
	}
}
