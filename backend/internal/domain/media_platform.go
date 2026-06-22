package domain

type ConnectorCapability string

const (
	ConnectorCapabilityAuthorization     ConnectorCapability = "authorization"
	ConnectorCapabilityProfileSync       ConnectorCapability = "profile_sync"
	ConnectorCapabilityContentPublish    ConnectorCapability = "content_publish"
	ConnectorCapabilityMetricIngestion   ConnectorCapability = "metric_ingestion"
	ConnectorCapabilityCommentIngestion  ConnectorCapability = "comment_ingestion"
	ConnectorCapabilityManualPublish     ConnectorCapability = "manual_publish"
	ConnectorCapabilityManualMetricInput ConnectorCapability = "manual_metric_input"
)

var supportedConnectorCapabilities = map[ConnectorCapability]bool{
	ConnectorCapabilityAuthorization:     true,
	ConnectorCapabilityProfileSync:       true,
	ConnectorCapabilityContentPublish:    true,
	ConnectorCapabilityMetricIngestion:   true,
	ConnectorCapabilityCommentIngestion:  true,
	ConnectorCapabilityManualPublish:     true,
	ConnectorCapabilityManualMetricInput: true,
}

type ConnectorCapabilityMode string

const (
	ConnectorCapabilityModeDisabled ConnectorCapabilityMode = "disabled"
	ConnectorCapabilityModeManual   ConnectorCapabilityMode = "manual"
	ConnectorCapabilityModeBrowser  ConnectorCapabilityMode = "browser"
	ConnectorCapabilityModeAPI      ConnectorCapabilityMode = "api"
)

var supportedConnectorCapabilityModes = map[ConnectorCapabilityMode]bool{
	ConnectorCapabilityModeDisabled: true,
	ConnectorCapabilityModeManual:   true,
	ConnectorCapabilityModeBrowser:  true,
	ConnectorCapabilityModeAPI:      true,
}

type AuthorizationMethod string

const (
	AuthorizationMethodNone       AuthorizationMethod = "none"
	AuthorizationMethodQRLogin    AuthorizationMethod = "qr_login"
	AuthorizationMethodOAuth      AuthorizationMethod = "oauth"
	AuthorizationMethodAPIToken   AuthorizationMethod = "api_token"
	AuthorizationMethodManualOnly AuthorizationMethod = "manual_only"
)

var supportedAuthorizationMethods = map[AuthorizationMethod]bool{
	AuthorizationMethodNone:       true,
	AuthorizationMethodQRLogin:    true,
	AuthorizationMethodOAuth:      true,
	AuthorizationMethodAPIToken:   true,
	AuthorizationMethodManualOnly: true,
}

type PublishMode string

const (
	PublishModeDisabled PublishMode = "disabled"
	PublishModeManual   PublishMode = "manual"
	PublishModeBrowser  PublishMode = "browser"
	PublishModeAPI      PublishMode = "api"
)

var supportedPublishModes = map[PublishMode]bool{
	PublishModeDisabled: true,
	PublishModeManual:   true,
	PublishModeBrowser:  true,
	PublishModeAPI:      true,
}

type ConnectorRateLimit struct {
	WindowSeconds int `json:"windowSeconds"`
	MaxRequests   int `json:"maxRequests"`
}

type ConnectorCapabilityContract struct {
	Name           ConnectorCapability     `json:"name"`
	Mode           ConnectorCapabilityMode `json:"mode"`
	Enabled        bool                    `json:"enabled"`
	ManualFallback bool                    `json:"manualFallback"`
	RateLimit      ConnectorRateLimit      `json:"rateLimit,omitempty"`
	Notes          string                  `json:"notes,omitempty"`
}

type MediaPlatformCapabilities struct {
	AuthorizationMethods []AuthorizationMethod         `json:"authorizationMethods"`
	PublishModes         []PublishMode                 `json:"publishModes"`
	ContentFormats       []string                      `json:"contentFormats"`
	Capabilities         []ConnectorCapabilityContract `json:"capabilities"`
	RateLimits           map[string]ConnectorRateLimit `json:"rateLimits,omitempty"`
}

func (caps MediaPlatformCapabilities) WithDefaults() MediaPlatformCapabilities {
	caps.AuthorizationMethods = uniqueAuthorizationMethods(caps.AuthorizationMethods)
	caps.PublishModes = uniquePublishModes(caps.PublishModes)
	caps.ContentFormats = uniqueStrings(caps.ContentFormats)
	caps.Capabilities = normalizeCapabilityContracts(caps.Capabilities)
	if caps.RateLimits == nil {
		caps.RateLimits = map[string]ConnectorRateLimit{}
	}
	return caps
}

func (caps MediaPlatformCapabilities) IsZero() bool {
	return len(caps.AuthorizationMethods) == 0 &&
		len(caps.PublishModes) == 0 &&
		len(caps.ContentFormats) == 0 &&
		len(caps.Capabilities) == 0 &&
		len(caps.RateLimits) == 0
}

func (caps MediaPlatformCapabilities) HasCapability(name ConnectorCapability) bool {
	for _, item := range caps.Capabilities {
		if item.Name == name && item.Enabled && item.Mode != ConnectorCapabilityModeDisabled {
			return true
		}
	}
	return false
}

func DefaultXiaohongshuCapabilities() MediaPlatformCapabilities {
	return MediaPlatformCapabilities{
		AuthorizationMethods: []AuthorizationMethod{AuthorizationMethodQRLogin},
		PublishModes:         []PublishMode{PublishModeManual, PublishModeBrowser},
		ContentFormats:       []string{"article", "image"},
		Capabilities: []ConnectorCapabilityContract{
			{
				Name:           ConnectorCapabilityAuthorization,
				Mode:           ConnectorCapabilityModeBrowser,
				Enabled:        true,
				ManualFallback: true,
				Notes:          "通过服务端托管浏览器完成二维码登录；不保存动态平台请求头。",
			},
			{
				Name:           ConnectorCapabilityProfileSync,
				Mode:           ConnectorCapabilityModeManual,
				Enabled:        false,
				ManualFallback: true,
				Notes:          "首轮仅声明边界，后续账号矩阵模块实现资料同步。",
			},
			{
				Name:           ConnectorCapabilityContentPublish,
				Mode:           ConnectorCapabilityModeBrowser,
				Enabled:        true,
				ManualFallback: true,
				Notes:          "浏览器发布只走已登录页面能力，失败时保留人工确认路径。",
			},
			{
				Name:           ConnectorCapabilityMetricIngestion,
				Mode:           ConnectorCapabilityModeManual,
				Enabled:        false,
				ManualFallback: true,
				Notes:          "未接入稳定合规的数据采集能力前，不默认抓取指标。",
			},
			{
				Name:           ConnectorCapabilityCommentIngestion,
				Mode:           ConnectorCapabilityModeDisabled,
				Enabled:        false,
				ManualFallback: false,
				Notes:          "评论采集涉及平台权限和合规边界，默认关闭。",
			},
		},
		RateLimits: map[string]ConnectorRateLimit{},
	}.WithDefaults()
}

func LegacyCapabilities(supportsArticle, supportsImage, supportsScheduling bool, credentialFields []string) MediaPlatformCapabilities {
	authorizationMethods := []AuthorizationMethod{AuthorizationMethodManualOnly}
	for _, field := range credentialFields {
		if field == "qrLogin" {
			authorizationMethods = []AuthorizationMethod{AuthorizationMethodQRLogin}
			break
		}
	}

	contentFormats := make([]string, 0, 2)
	if supportsArticle {
		contentFormats = append(contentFormats, "article")
	}
	if supportsImage {
		contentFormats = append(contentFormats, "image")
	}

	publishModes := []PublishMode{PublishModeManual}
	publishCapabilityMode := ConnectorCapabilityModeManual
	if supportsScheduling {
		publishModes = append(publishModes, PublishModeAPI)
		publishCapabilityMode = ConnectorCapabilityModeAPI
	}

	return MediaPlatformCapabilities{
		AuthorizationMethods: authorizationMethods,
		PublishModes:         publishModes,
		ContentFormats:       contentFormats,
		Capabilities: []ConnectorCapabilityContract{
			{
				Name:           ConnectorCapabilityAuthorization,
				Mode:           connectorModeForAuthorization(authorizationMethods),
				Enabled:        len(authorizationMethods) > 0,
				ManualFallback: true,
			},
			{
				Name:           ConnectorCapabilityContentPublish,
				Mode:           publishCapabilityMode,
				Enabled:        supportsArticle || supportsImage,
				ManualFallback: true,
			},
		},
		RateLimits: map[string]ConnectorRateLimit{},
	}.WithDefaults()
}

func connectorModeForAuthorization(methods []AuthorizationMethod) ConnectorCapabilityMode {
	for _, method := range methods {
		switch method {
		case AuthorizationMethodQRLogin:
			return ConnectorCapabilityModeBrowser
		case AuthorizationMethodOAuth, AuthorizationMethodAPIToken:
			return ConnectorCapabilityModeAPI
		}
	}
	return ConnectorCapabilityModeManual
}

func normalizeCapabilityContracts(items []ConnectorCapabilityContract) []ConnectorCapabilityContract {
	if len(items) == 0 {
		return []ConnectorCapabilityContract{}
	}
	seen := map[ConnectorCapability]bool{}
	normalized := make([]ConnectorCapabilityContract, 0, len(items))
	for _, item := range items {
		if item.Name == "" || seen[item.Name] || !supportedConnectorCapabilities[item.Name] {
			continue
		}
		if !supportedConnectorCapabilityModes[item.Mode] {
			item.Mode = ""
		}
		if item.Mode == "" {
			if item.Enabled {
				item.Mode = ConnectorCapabilityModeManual
			} else {
				item.Mode = ConnectorCapabilityModeDisabled
			}
		}
		if item.Mode == ConnectorCapabilityModeDisabled {
			item.Enabled = false
		}
		seen[item.Name] = true
		normalized = append(normalized, item)
	}
	return normalized
}

func uniqueAuthorizationMethods(values []AuthorizationMethod) []AuthorizationMethod {
	if len(values) == 0 {
		return []AuthorizationMethod{}
	}
	seen := map[AuthorizationMethod]bool{}
	result := make([]AuthorizationMethod, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] || !supportedAuthorizationMethods[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func uniquePublishModes(values []PublishMode) []PublishMode {
	if len(values) == 0 {
		return []PublishMode{}
	}
	seen := map[PublishMode]bool{}
	result := make([]PublishMode, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] || !supportedPublishModes[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
