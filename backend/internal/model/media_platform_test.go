package model

import (
	"testing"

	"geopress/backend/internal/domain"
)

func TestMediaPlatformEnsureCapabilitiesUsesXiaohongshuDefaults(t *testing.T) {
	platform := MediaPlatform{
		ID:                 "plt_xiaohongshu",
		Type:               "xiaohongshu",
		SupportsArticle:    true,
		SupportsImage:      true,
		SupportsScheduling: false,
		CredentialFields:   []string{"qrLogin"},
	}

	platform.EnsureCapabilities()

	if !platform.Capabilities.HasCapability(domain.ConnectorCapabilityAuthorization) {
		t.Fatalf("xiaohongshu default should retain browser authorization: %#v", platform.Capabilities)
	}
	if !platform.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("xiaohongshu default should retain browser publishing: %#v", platform.Capabilities)
	}
	if len(platform.Capabilities.PublishModes) != 2 || platform.Capabilities.PublishModes[1] != domain.PublishModeBrowser {
		t.Fatalf("publish modes = %#v, want manual and browser", platform.Capabilities.PublishModes)
	}
}

func TestMediaPlatformEnsureCapabilitiesUsesLegacyDefaultsForOtherPlatforms(t *testing.T) {
	platform := MediaPlatform{
		ID:                 "plt_manual",
		Type:               "manual",
		SupportsArticle:    true,
		SupportsImage:      false,
		SupportsScheduling: true,
		CredentialFields:   nil,
	}

	platform.EnsureCapabilities()

	if platform.CredentialFields == nil {
		t.Fatalf("credential fields should be an empty slice, not nil")
	}
	if !platform.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("legacy platform should expose content publish when old support flags allow it")
	}
	if len(platform.Capabilities.PublishModes) != 2 || platform.Capabilities.PublishModes[1] != domain.PublishModeAPI {
		t.Fatalf("publish modes = %#v, want manual and api", platform.Capabilities.PublishModes)
	}
}
