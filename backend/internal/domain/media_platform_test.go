package domain

import "testing"

func TestDefaultXiaohongshuCapabilities(t *testing.T) {
	caps := DefaultXiaohongshuCapabilities()

	if !caps.HasCapability(ConnectorCapabilityAuthorization) {
		t.Fatalf("default xiaohongshu capabilities should include enabled authorization")
	}
	if !caps.HasCapability(ConnectorCapabilityContentPublish) {
		t.Fatalf("default xiaohongshu capabilities should include enabled content publishing")
	}
	if caps.HasCapability(ConnectorCapabilityCommentIngestion) {
		t.Fatalf("comment ingestion should remain disabled by default")
	}
	if len(caps.AuthorizationMethods) != 1 || caps.AuthorizationMethods[0] != AuthorizationMethodQRLogin {
		t.Fatalf("authorization methods = %#v, want qr_login", caps.AuthorizationMethods)
	}
}

func TestLegacyCapabilitiesFromOldMediaPlatformFields(t *testing.T) {
	caps := LegacyCapabilities(true, false, true, []string{"qrLogin", "ignored"})

	if !caps.HasCapability(ConnectorCapabilityAuthorization) {
		t.Fatalf("legacy capabilities should enable authorization when credentials exist")
	}
	if !caps.HasCapability(ConnectorCapabilityContentPublish) {
		t.Fatalf("legacy capabilities should enable content publishing when article/image is supported")
	}
	if len(caps.PublishModes) != 2 || caps.PublishModes[1] != PublishModeAPI {
		t.Fatalf("publish modes = %#v, want manual and api", caps.PublishModes)
	}
	if len(caps.ContentFormats) != 1 || caps.ContentFormats[0] != "article" {
		t.Fatalf("content formats = %#v, want article", caps.ContentFormats)
	}
}

func TestCapabilitiesWithDefaultsFiltersUnknownValues(t *testing.T) {
	caps := MediaPlatformCapabilities{
		AuthorizationMethods: []AuthorizationMethod{AuthorizationMethodQRLogin, AuthorizationMethod("unknown"), AuthorizationMethodQRLogin},
		PublishModes:         []PublishMode{PublishModeManual, PublishMode("unknown"), PublishModeManual},
		ContentFormats:       []string{"article", "", "article"},
		Capabilities: []ConnectorCapabilityContract{
			{Name: ConnectorCapabilityAuthorization, Enabled: true},
			{Name: ConnectorCapabilityAuthorization, Enabled: true, Mode: ConnectorCapabilityModeAPI},
			{Name: ConnectorCapability("unknown"), Enabled: true},
		},
	}.WithDefaults()

	if len(caps.AuthorizationMethods) != 1 || caps.AuthorizationMethods[0] != AuthorizationMethodQRLogin {
		t.Fatalf("authorization methods = %#v, want one qr_login", caps.AuthorizationMethods)
	}
	if len(caps.PublishModes) != 1 || caps.PublishModes[0] != PublishModeManual {
		t.Fatalf("publish modes = %#v, want one manual", caps.PublishModes)
	}
	if len(caps.ContentFormats) != 1 || caps.ContentFormats[0] != "article" {
		t.Fatalf("content formats = %#v, want one article", caps.ContentFormats)
	}
	if len(caps.Capabilities) != 1 || caps.Capabilities[0].Mode != ConnectorCapabilityModeManual {
		t.Fatalf("capability contracts = %#v, want one manual authorization", caps.Capabilities)
	}
	if caps.RateLimits == nil {
		t.Fatalf("rate limits should default to an empty map")
	}
}
