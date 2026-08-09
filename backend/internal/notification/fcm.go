package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// FCMScope is the OAuth2 scope required to call the FCM HTTP v1 API.
const FCMScope = "https://www.googleapis.com/auth/firebase.messaging"

// FCMEndpoint formats the messages:send URL for a Firebase project.
const FCMEndpoint = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// fcmMessage is the FCM HTTP v1 request body (subset we need).
type fcmMessage struct {
	Message struct {
		Token        string `json:"token"`
		Notification struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"notification"`
	} `json:"message"`
}

// FCMConfig configures the Firebase Cloud Messaging sender.
type FCMConfig struct {
	// CredentialsFile is the path to a Google service-account JSON file.
	// Falls back to GOOGLE_APPLICATION_CREDENTIALS. Empty disables the sender.
	CredentialsFile string
	// ProjectID is the Firebase project ID. If empty, it is read from the
	// service-account JSON.
	ProjectID string
}

// FCMSender delivers push notifications via the FCM HTTP v1 API. When no
// credentials are configured it becomes a no-op so the pipeline stays healthy.
type FCMSender struct {
	projectID string
	client    *http.Client
	enabled   bool
}

// NewFCMSender reads service-account credentials and builds an authenticated
// HTTP client. Returns a no-op sender when credentials are absent.
func NewFCMSender(ctx context.Context, cfg FCMConfig) (*FCMSender, error) {
	credsFile := cfg.CredentialsFile
	if credsFile == "" {
		credsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	if credsFile == "" {
		return &FCMSender{enabled: false}, nil
	}

	raw, err := os.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("fcm: read credentials: %w", err)
	}
	// Validate that the credentials are a service account before use (the
	// non-validating CredentialsFromJSON* helpers are deprecated).
	creds, err := google.CredentialsFromJSONWithTypeAndParams(ctx, raw, google.ServiceAccount, google.CredentialsParams{
		Scopes: []string{FCMScope},
	})
	if err != nil {
		return nil, fmt.Errorf("fcm: parse credentials: %w", err)
	}

	projectID := cfg.ProjectID
	if projectID == "" {
		var meta struct {
			ProjectID string `json:"project_id"`
		}
		if err := json.Unmarshal(raw, &meta); err == nil {
			projectID = meta.ProjectID
		}
	}
	if projectID == "" {
		return nil, fmt.Errorf("fcm: project_id not configured and not present in credentials")
	}

	return &FCMSender{
		projectID: projectID,
		client:    oauth2.NewClient(ctx, creds.TokenSource),
		enabled:   true,
	}, nil
}

// SendPush sends a push notification to a device registration token.
func (f *FCMSender) SendPush(ctx context.Context, token, title, body string) error {
	if !f.enabled {
		return nil
	}

	var msg fcmMessage
	msg.Message.Token = token
	msg.Message.Notification.Title = title
	msg.Message.Notification.Body = body

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("fcm: marshal: %w", err)
	}

	url := fmt.Sprintf(FCMEndpoint, f.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("fcm: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fcm: send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var bodyBuf bytes.Buffer
		_, _ = bodyBuf.ReadFrom(resp.Body)
		return fmt.Errorf("fcm: unexpected status %d: %s", resp.StatusCode, bodyBuf.String())
	}
	return nil
}
