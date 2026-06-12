package publisher

import (
	"context"
	"time"

	"geopress/backend/internal/model"
)

const ModeManualAssist = "manual_assist"
const ModeMockHumanAPI = "mock_human_api"
const ModeBrowserAutomation = "browser_automation"

type PrepareRequest struct {
	Workspace       model.Workspace
	Content         model.Content
	Account         model.MediaAccount
	Platform        model.MediaPlatform
	PublishFormatID string
	RequestedAt     time.Time
}

type CopyBlock struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type PreparedPost struct {
	Mode            string      `json:"mode"`
	PlatformType    string      `json:"platformType"`
	PlatformName    string      `json:"platformName"`
	PublishFormatID string      `json:"publishFormatId"`
	PublishMode     string      `json:"publishMode"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	Hashtags        []string    `json:"hashtags"`
	CopyBlocks      []CopyBlock `json:"copyBlocks"`
	Checklist       []string    `json:"checklist"`
	Warnings        []string    `json:"warnings"`
	CharacterCount  int         `json:"characterCount"`
	PreparedAt      time.Time   `json:"preparedAt"`
}

type PublishRequest struct {
	Workspace    model.Workspace
	Account      model.MediaAccount
	Platform     model.MediaPlatform
	PreparedPost PreparedPost
	AssetPaths   []string
	ProfileDir   string
	StateFile    string
}

type PublishResult struct {
	Status      string         `json:"status"`
	Message     string         `json:"message"`
	ExternalID  string         `json:"externalId"`
	ExternalURL string         `json:"externalUrl"`
	RawResponse map[string]any `json:"rawResponse"`
}

type Publisher interface {
	PlatformType() string
	Prepare(ctx context.Context, req PrepareRequest) (PreparedPost, error)
	Publish(ctx context.Context, req PublishRequest) (PublishResult, error)
}
