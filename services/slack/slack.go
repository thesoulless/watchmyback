package slack

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

func SendToChanel(ctx context.Context, url string, msg string) error {
	// url := "https://hooks.slack.com/services/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"
	if !strings.HasPrefix(url, "https://hooks.slack.com/services/") {
		url = fmt.Sprintf("https://hooks.slack.com/services/%s", url)
	}
	body := strings.NewReader(msg)

	client := cleanhttp.DefaultClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
