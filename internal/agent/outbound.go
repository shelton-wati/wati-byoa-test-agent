package agent

import (
	"strings"

	"github.com/wati/wati-byoa-test-agent/internal/outbound"
	"github.com/wati/wati-byoa-test-agent/internal/webhook"
)

func sendRequestFor(ctx webhook.ConversationContext, text, replyContextID string) outbound.SendTextRequest {
	channel, _ := outbound.ParseChannel(ctx.ChannelType.String())
	if channel == "" {
		channel = outbound.ChannelWhatsApp
	}
	return outbound.SendTextRequest{
		Channel:            channel,
		Target:             ctx.WaID,
		Text:               text,
		ReplyContextID:     strings.TrimSpace(replyContextID),
		ChannelPhoneNumber: ctx.ChannelPhoneNumber,
		ChannelID:          ctx.ChannelID,
		ConversationID:     ctx.ConversationID,
		IsBot:              channel != outbound.ChannelInstagram,
	}
}
