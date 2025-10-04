package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WeChatNotifier sends notifications to WeChat Work
type WeChatNotifier struct {
	webhookURL string
	client     *http.Client
}

// NewWeChatNotifier creates a new WeChat notifier
func NewWeChatNotifier(webhookURL string) *WeChatNotifier {
	return &WeChatNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

type wechatMarkdownMessage struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

// Send sends a notification to WeChat Work
func (w *WeChatNotifier) Send(ctx context.Context, content string) error {
	msg := wechatMarkdownMessage{
		MsgType: "markdown",
	}
	msg.Markdown.Content = content

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
