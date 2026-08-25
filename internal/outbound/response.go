package outbound

import (
	"encoding/json"
	"strings"
)

func decodeSendSessionMessageResponse(body []byte) (SendResult, error) {
	var payload struct {
		OK      bool            `json:"ok"`
		Result  json.RawMessage `json:"result"`
		Info    string          `json:"info"`
		Message struct {
			PlatformMessageID string `json:"whatsappMessageId"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SendResult{}, err
	}

	ok, resultText := parseAPIResultField(payload.Result)
	success := payload.OK || ok || strings.EqualFold(resultText, "success")
	info := payload.Info
	if !success && info == "" {
		info = resultText
	}
	return SendResult{
		OK:                success,
		Info:              info,
		PlatformMessageID: payload.Message.PlatformMessageID,
	}, nil
}

func decodeSendTextV3Response(body []byte) (SendResult, error) {
	var payload struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SendResult{}, err
	}
	return SendResult{
		OK:                true,
		PlatformMessageID: payload.Message.ID,
	}, nil
}

func decodeV3ErrorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
		Info    string `json:"info"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if msg := strings.TrimSpace(payload.Message); msg != "" {
		return msg
	}
	return strings.TrimSpace(payload.Info)
}

func parseAPIResultField(raw json.RawMessage) (bool, string) {
	raw = bytesTrimSpace(raw)
	if len(raw) == 0 {
		return false, ""
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b, ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.EqualFold(s, "success") || strings.EqualFold(s, "true"), s
	}
	return false, string(raw)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func truncateBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 240 {
		return text[:240] + "..."
	}
	return text
}
