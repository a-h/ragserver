package models

type ChatPostRequest struct {
	Messages []ChatMessage `json:"msgs"`
}

type ChatMessageType string

const (
	ChatMessageTypeSystem ChatMessageType = "system"
	ChatMessageTypeHuman  ChatMessageType = "human"
	ChatMessageTypeAI     ChatMessageType = "ai"
)

type ChatMessage struct {
	Type    ChatMessageType `json:"type"`
	Content string          `json:"content"`
}
