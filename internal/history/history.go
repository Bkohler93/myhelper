package history

import (
	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Message represents a single conversation turn in Ollama /api/chat format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// History holds an ordered sequence of conversation messages and
// tracks a token threshold for summarization decisions.
type History struct {
	messages  []Message
	threshold int
	enc       *tiktoken.Tiktoken
}

// New creates a History with the given token threshold.
// Panics if the tiktoken encoder cannot be loaded (indicates bad installation).
func New(threshold int, initialMessages []Message) *History {
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		panic("history: failed to load tiktoken encoder: " + err.Error())
	}
	h := &History{
		messages:  []Message{},
		threshold: threshold,
		enc:       enc,
	}
	for _, m := range initialMessages {
		h.Add(m.Role, m.Content)
	}
	return h
}

// Add appends a message to the history.
func (h *History) Add(role, content string) {
	h.messages = append(h.messages, Message{Role: role, Content: content})
}

// Messages returns a copy of the current message slice.
// Callers may iterate without mutating internal state.
func (h *History) Messages() []Message {
	out := make([]Message, len(h.messages))
	copy(out, h.messages)
	return out
}

// Replace discards the current message slice and replaces it with a copy of
// the provided messages. The caller is responsible for ensuring the system
// message at index [0] is preserved in the new slice when required.
func (h *History) Replace(messages []Message) {
	h.messages = make([]Message, len(messages))
	copy(h.messages, messages)
}

// TokenCount returns the total number of tokens across all message contents
// using the cl100k_base tiktoken encoder.
func (h *History) TokenCount() int {
	total := 0
	for _, m := range h.messages {
		tokens := h.enc.Encode(m.Content, nil, nil)
		total += len(tokens)
	}
	return total
}

// ExceedsLimit reports whether the current token count strictly exceeds
// the threshold. Returns false when count equals the threshold.
func (h *History) ExceedsLimit() bool {
	return h.TokenCount() > h.threshold
}
