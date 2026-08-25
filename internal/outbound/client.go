package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	TenantID   string
	APIToken   string
	SourceType string
	HTTP       *http.Client
}

func (c Client) Enabled() bool {
	return strings.TrimSpace(c.BaseURL) != "" && strings.TrimSpace(c.APIToken) != ""
}

func (c Client) SendText(ctx context.Context, req SendTextRequest) (SendResult, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return SendResult{}, fmt.Errorf("message text is required")
	}

	if convID := conversationTarget(req); convID != "" {
		return c.sendConversationTextV3(ctx, convID, text, req.IsBot)
	}

	channel := req.Channel
	if channel == "" {
		channel = ChannelWhatsApp
	}
	target := strings.TrimSpace(req.Target)
	if target == "" {
		return SendResult{}, fmt.Errorf("target is required")
	}

	return c.sendSessionMessage(ctx, req, channel, target, text)
}

func conversationTarget(req SendTextRequest) string {
	if convID := strings.TrimSpace(req.ConversationID); convID != "" {
		return convID
	}
	target := strings.TrimSpace(req.Target)
	if isMongoObjectID(target) {
		return target
	}
	return ""
}

func (c Client) sendSessionMessage(ctx context.Context, req SendTextRequest, channel Channel, target, text string) (SendResult, error) {
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = strings.TrimSpace(c.SourceType)
	}
	if sourceType == "" {
		sourceType = "API"
	}

	query := url.Values{}
	query.Set("messageText", text)
	query.Set("sourceType", sourceType)
	if reply := strings.TrimSpace(req.ReplyContextID); reply != "" {
		query.Set("replyContextId", reply)
	}

	switch channel {
	case ChannelWhatsApp:
		if phone := strings.TrimSpace(req.ChannelPhoneNumber); phone != "" {
			query.Set("channelPhoneNumber", phone)
		}
	case ChannelInstagram:
		if channelID := strings.TrimSpace(req.ChannelID); channelID != "" {
			query.Set("channelId", channelID)
		}
	default:
		return SendResult{}, fmt.Errorf("unsupported channel %q", channel)
	}

	endpoint := fmt.Sprintf("%s/api/v1/sendSessionMessage/%s?%s",
		c.apiPrefixV1(), url.PathEscape(target), query.Encode())

	body, status, err := c.doAuthorizedRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return SendResult{}, err
	}
	if status < 200 || status >= 300 {
		return SendResult{}, fmt.Errorf("sendSessionMessage http %d: %s", status, strings.TrimSpace(string(body)))
	}

	result, err := decodeSendSessionMessageResponse(body)
	if err != nil {
		return SendResult{}, fmt.Errorf("decode sendSessionMessage response: %w (body=%s)", err, truncateBody(body))
	}
	if !result.OK {
		if result.Info == "" {
			result.Info = truncateBody(body)
		}
		return result, fmt.Errorf("sendSessionMessage failed: %s", result.Info)
	}
	return result, nil
}

func (c Client) sendConversationTextV3(ctx context.Context, conversationID, text string, isBot bool) (SendResult, error) {
	payload, err := json.Marshal(map[string]any{
		"target": conversationID,
		"text":   text,
		"isBot":  isBot,
	})
	if err != nil {
		return SendResult{}, err
	}

	endpoint := c.apiPrefixV3() + "/api/ext/v3/conversations/messages/text"
	body, status, err := c.doAuthorizedRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return SendResult{}, err
	}
	if status < 200 || status >= 300 {
		info := decodeV3ErrorMessage(body)
		if info == "" {
			info = strings.TrimSpace(string(body))
		}
		return SendResult{}, fmt.Errorf("send conversation text http %d: %s", status, info)
	}

	result, err := decodeSendTextV3Response(body)
	if err != nil {
		return SendResult{}, fmt.Errorf("decode send conversation text response: %w (body=%s)", err, truncateBody(body))
	}
	return result, nil
}

func (c Client) apiPrefixV1() string {
	pathPrefix := strings.TrimRight(c.BaseURL, "/")
	if tenant := strings.Trim(c.TenantID, "/"); tenant != "" {
		pathPrefix += "/" + tenant
	}
	return pathPrefix
}

func (c Client) apiPrefixV3() string {
	return strings.TrimRight(c.BaseURL, "/")
}

func (c Client) doAuthorizedRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, 0, err
	}
	token := strings.TrimSpace(c.APIToken)
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = "Bearer " + token
	}
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func isMongoObjectID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 24 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
