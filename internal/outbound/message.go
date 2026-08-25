package outbound

type SendTextRequest struct {
	Channel            Channel
	Target             string
	Text               string
	ReplyContextID     string
	ChannelPhoneNumber string
	ChannelID          string
	ConversationID     string
	SourceType         string
	IsBot              bool
}

type SendResult struct {
	OK                bool
	Info              string
	PlatformMessageID string
}
