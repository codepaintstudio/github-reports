package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// FeishuNotifier sends notifications to Feishu (Lark)
type FeishuNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewFeishuNotifier creates a new Feishu notifier
func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

type feishuMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Send sends a notification to Feishu
func (f *FeishuNotifier) Send(ctx context.Context, content string) error {
	msg := feishuMessage{
		MsgType: "text",
	}
	msg.Content.Text = content

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Feishu API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
