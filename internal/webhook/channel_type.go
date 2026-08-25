package webhook

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ChannelTypeValue string

func (v *ChannelTypeValue) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*v = ""
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = ChannelTypeValue(normalizeChannelTypeString(s))
		return nil
	}

	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*v = ChannelTypeValue(channelTypeFromEnum(n))
		return nil
	}

	return fmt.Errorf("channelType: unsupported json value %s", string(data))
}

func (v ChannelTypeValue) String() string {
	return string(v)
}

func normalizeChannelTypeString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return channelTypeFromEnum(n)
	}
	return strings.ToUpper(raw)
}

func channelTypeFromEnum(n int) string {
	switch n {
	case 0:
		return "WHATSAPP"
	case 1:
		return "INSTAGRAM"
	case 2:
		return "MESSENGER"
	case 6:
		return "RCS"
	case 10:
		return "TIKTOK"
	default:
		return fmt.Sprintf("CHANNEL_%d", n)
	}
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
