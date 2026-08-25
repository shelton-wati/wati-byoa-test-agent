package outbound

import (
	"fmt"
	"strings"
)

type Channel string

const (
	ChannelWhatsApp  Channel = "whatsapp"
	ChannelInstagram Channel = "instagram"
)

func ParseChannel(raw string) (Channel, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "whatsapp", "wa", "whatsappconversation":
		return ChannelWhatsApp, nil
	case "instagram", "ig", "instagramconversation":
		return ChannelInstagram, nil
	default:
		return "", fmt.Errorf("unsupported channel %q (use whatsapp or instagram)", raw)
	}
}

func (c Channel) String() string {
	if c == "" {
		return string(ChannelWhatsApp)
	}
	return string(c)
}
