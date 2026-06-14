package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://www.googleapis.com/youtube/v3"

type Client struct {
	apiKey     string
	channelID  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey, channelID string) *Client {
	return &Client{
		apiKey:    apiKey,
		channelID: channelID,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) ActiveLiveChatID(ctx context.Context) (string, error) {
	videoID, err := c.activeLiveVideoID(ctx)
	if err != nil {
		return "", err
	}
	if videoID == "" {
		return "", nil
	}

	params := url.Values{
		"part": {"liveStreamingDetails"},
		"id":   {videoID},
		"key":  {c.apiKey},
	}
	var resp VideoListResponse
	if err := c.getJSON(ctx, "/videos", params, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", nil
	}
	return resp.Items[0].LiveStreamingDetails.ActiveLiveChatID, nil
}

func (c *Client) activeLiveVideoID(ctx context.Context) (string, error) {
	params := url.Values{
		"part":       {"id"},
		"channelId":  {c.channelID},
		"type":       {"video"},
		"eventType":  {"live"},
		"maxResults": {"1"},
		"key":        {c.apiKey},
	}
	var resp SearchResponse
	if err := c.getJSON(ctx, "/search", params, &resp); err != nil {
		return "", err
	}
	if len(resp.Items) == 0 {
		return "", nil
	}
	return resp.Items[0].ID.VideoID, nil
}

func (c *Client) ListChatMessages(ctx context.Context, liveChatID, pageToken string) (ChatMessagesResponse, error) {
	params := url.Values{
		"liveChatId": {liveChatID},
		"part":       {"snippet,authorDetails"},
		"maxResults": {"200"},
		"key":        {c.apiKey},
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}

	var resp ChatMessagesResponse
	if err := c.getJSON(ctx, "/liveChatMessages", params, &resp); err != nil {
		return ChatMessagesResponse{}, err
	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, path string, params url.Values, out any) error {
	u := c.baseURL + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("youtube api %s: %s", res.Status, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode youtube response: %w", err)
	}
	return nil
}
