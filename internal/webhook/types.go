package webhook

type MessageReceived struct {
	ID                 string           `json:"id"`
	ConversationID     string           `json:"conversationId"`
	TicketID           string           `json:"ticketId"`
	Text               string           `json:"text"`
	Type               string           `json:"type"`
	Data               string           `json:"data"`
	WhatsappMessageID  string           `json:"whatsappMessageId"`
	WaID               string           `json:"waId"`
	SenderName         string           `json:"senderName"`
	ReplyContextID     string           `json:"replyContextId"`
	EventType          string           `json:"eventType"`
	Owner              bool             `json:"owner"`
	TenantID           string           `json:"tenantId"`
	ChannelType        ChannelTypeValue `json:"channelType"`
	ChannelID          string           `json:"channelId"`
	ChannelPhoneNumber string           `json:"channelPhoneNumber"`
}

type ChatAssigned struct {
	EventType          string           `json:"eventType"`
	ConversationID     string           `json:"conversationId"`
	TicketID           string           `json:"ticketId"`
	WaID               string           `json:"waId"`
	AssigneeName       string           `json:"assigneeName"`
	TenantID           string           `json:"tenantId"`
	ChannelType        ChannelTypeValue `json:"channelType"`
	ChannelID          string           `json:"channelId"`
	ChannelPhoneNumber string           `json:"channelPhoneNumber"`
	EventDescription   string           `json:"eventDescription"`
	IsChangedToOpen    bool             `json:"isChangedToOpen"`
}

type ConversationContext struct {
	ConversationID     string
	TicketID           string
	WaID               string
	ChannelType        ChannelTypeValue
	ChannelID          string
	ChannelPhoneNumber string
}

func (m MessageReceived) Context() ConversationContext {
	return ConversationContext{
		ConversationID:     m.ConversationID,
		TicketID:           m.TicketID,
		WaID:               m.WaID,
		ChannelType:        m.ChannelType,
		ChannelID:          m.ChannelID,
		ChannelPhoneNumber: m.ChannelPhoneNumber,
	}
}

func (c ChatAssigned) Context() ConversationContext {
	return ConversationContext{
		ConversationID:     c.ConversationID,
		TicketID:           c.TicketID,
		WaID:               c.WaID,
		ChannelType:        c.ChannelType,
		ChannelID:          c.ChannelID,
		ChannelPhoneNumber: c.ChannelPhoneNumber,
	}
}

func (m MessageReceived) UserText() string {
	if m.Text != "" {
		return m.Text
	}
	if m.Data != "" && m.Type != "text" {
		return "[" + m.Type + "] " + m.Data
	}
	return ""
}
