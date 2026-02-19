package onebot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chat-assist-backend/internal/config"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(cfg config.Config) *Client {
	base := strings.TrimSpace(cfg.OneBot.APIURL)
	if base == "" {
		return nil
	}
	return &Client{
		baseURL: strings.TrimRight(base, "/"),
		token:   strings.TrimSpace(cfg.OneBot.AccessToken),
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *Client) GetSelfID(ctx context.Context) (int64, error) {
	var data struct {
		UserID int64 `json:"user_id"`
	}
	if err := c.call(ctx, "get_login_info", map[string]interface{}{}, &data); err != nil {
		return 0, err
	}
	if data.UserID == 0 {
		return 0, fmt.Errorf("empty self user_id from onebot")
	}
	return data.UserID, nil
}

func (c *Client) GetGroupName(ctx context.Context, groupID string) (string, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(groupID), 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid group id: %s", groupID)
	}
	var data struct {
		GroupName string `json:"group_name"`
	}
	if err := c.call(ctx, "get_group_info", map[string]interface{}{"group_id": id}, &data); err != nil {
		return "", err
	}
	return strings.TrimSpace(data.GroupName), nil
}

func (c *Client) GetGroupMemberName(ctx context.Context, groupID string, userID int64) (string, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(groupID), 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid group id: %s", groupID)
	}
	var data struct {
		Card     string `json:"card"`
		Nickname string `json:"nickname"`
	}
	payload := map[string]interface{}{
		"group_id": id,
		"user_id":  userID,
	}
	if err := c.call(ctx, "get_group_member_info", payload, &data); err != nil {
		return "", err
	}
	if name := strings.TrimSpace(data.Card); name != "" {
		return name, nil
	}
	return strings.TrimSpace(data.Nickname), nil
}

func (c *Client) GetStrangerName(ctx context.Context, userID int64) (string, error) {
	var data struct {
		Nickname string `json:"nickname"`
		Remark   string `json:"remark"`
	}
	payload := map[string]interface{}{
		"user_id": userID,
	}
	if err := c.call(ctx, "get_stranger_info", payload, &data); err != nil {
		return "", err
	}
	if name := strings.TrimSpace(data.Remark); name != "" {
		return name, nil
	}
	return strings.TrimSpace(data.Nickname), nil
}

func (c *Client) call(ctx context.Context, action string, payload map[string]interface{}, out interface{}) error {
	if c == nil {
		return fmt.Errorf("onebot api not configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := c.baseURL + "/" + strings.TrimLeft(action, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("onebot status %d: %s", resp.StatusCode, msg)
	}

	var envelope struct {
		Status  string          `json:"status"`
		Retcode int             `json:"retcode"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Status != "ok" || envelope.Retcode != 0 {
		msg := strings.TrimSpace(envelope.Message)
		if msg == "" {
			msg = "onebot api error"
		}
		return fmt.Errorf("%s (retcode=%d)", msg, envelope.Retcode)
	}
	if out == nil {
		return nil
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return fmt.Errorf("empty onebot data")
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return err
	}
	return nil
}
