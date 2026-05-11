package types

// Message represents a single message in a chat conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatMessage represents either a direct text prompt or structured chat messages
type ChatMessage struct {
	// Text is used for direct prompt input (completion mode)
	Text string `json:"text,omitempty"`

	// Messages is used for chat conversation input (chat mode)
	Messages []Message `json:"messages,omitempty"`
}


