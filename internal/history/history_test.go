package history_test

import (
	"testing"

	"github.com/bkohler93/myhelper/internal/history"
)

// Test 1: Add appends a Message to the internal slice
func TestHistory_Add(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "hello")
	msgs := h.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("expected Message{Role:\"user\", Content:\"hello\"}, got %+v", msgs[0])
	}
}

// Test 2: TokenCount returns 0 for empty history
func TestHistory_TokenCount_Empty(t *testing.T) {
	h := history.New(4100, nil)
	if h.TokenCount() != 0 {
		t.Errorf("expected 0 tokens for empty history, got %d", h.TokenCount())
	}
}

// Test 3: TokenCount returns positive count for a single message
func TestHistory_TokenCount_NonEmpty(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "hello world")
	count := h.TokenCount()
	if count <= 0 {
		t.Errorf("expected positive token count, got %d", count)
	}
}

// Test 4: TokenCount accumulates across multiple messages
func TestHistory_TokenCount_Accumulates(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "hello world")
	countOne := h.TokenCount()
	h.Add("assistant", "how are you")
	countTwo := h.TokenCount()
	if countTwo <= countOne {
		t.Errorf("expected token count to increase after adding second message: was %d, now %d", countOne, countTwo)
	}
}

// Test 5: ExceedsLimit returns false when token count is within threshold
func TestHistory_ExceedsLimit_False(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "hi")
	if h.ExceedsLimit() {
		t.Errorf("expected ExceedsLimit() == false for small message within threshold")
	}
}

// Test 6: ExceedsLimit returns true when token count exceeds threshold
func TestHistory_ExceedsLimit_True(t *testing.T) {
	// Set a very small threshold so any content exceeds it
	h := history.New(1, nil)
	h.Add("user", "hello world this is a longer message to exceed the threshold")
	if !h.ExceedsLimit() {
		t.Errorf("expected ExceedsLimit() == true when token count exceeds threshold of 1")
	}
}

// Test 7: ExceedsLimit returns false when token count equals threshold exactly (strictly greater than)
func TestHistory_ExceedsLimit_Boundary(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "hi")
	exactCount := h.TokenCount()
	// Create a history with threshold exactly equal to token count
	hExact := history.New(exactCount, nil)
	hExact.Add("user", "hi")
	if hExact.ExceedsLimit() {
		t.Errorf("expected ExceedsLimit() == false when token count equals threshold exactly")
	}
}

// Test 8: Messages returns all messages in insertion order
func TestHistory_Messages_Order(t *testing.T) {
	h := history.New(4100, nil)
	h.Add("user", "a")
	h.Add("assistant", "b")
	msgs := h.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "a" {
		t.Errorf("expected first message {user, a}, got %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "b" {
		t.Errorf("expected second message {assistant, b}, got %+v", msgs[1])
	}
}

func TestHistory_Replace(t *testing.T) {
	t.Run("replaces all messages with provided slice", func(t *testing.T) {
		h := history.New(4100, nil)
		h.Add("user", "original")
		h.Replace([]history.Message{
			{Role: "system", Content: "Summary of previous conversation: xyz"},
		})
		msgs := h.Messages()
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message after Replace, got %d", len(msgs))
		}
		if msgs[0].Role != "system" || msgs[0].Content != "Summary of previous conversation: xyz" {
			t.Errorf("unexpected message after Replace: %+v", msgs[0])
		}
	})

	t.Run("replace with three messages preserves order", func(t *testing.T) {
		h := history.New(4100, nil)
		h.Replace([]history.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "u"},
			{Role: "assistant", Content: "a"},
		})
		msgs := h.Messages()
		if len(msgs) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(msgs))
		}
	})

	t.Run("replace with empty slice yields empty messages", func(t *testing.T) {
		h := history.New(4100, nil)
		h.Add("user", "something")
		h.Replace([]history.Message{})
		if len(h.Messages()) != 0 {
			t.Fatalf("expected 0 messages after Replace with empty, got %d", len(h.Messages()))
		}
	})

	t.Run("messages after replace are a copy (mutation-safe)", func(t *testing.T) {
		h := history.New(4100, nil)
		h.Replace([]history.Message{{Role: "user", Content: "original"}})
		msgs := h.Messages()
		msgs[0].Content = "mutated"
		if h.Messages()[0].Content != "original" {
			t.Error("Replace internal state is not copy-safe: mutation via Messages() affected internal state")
		}
	})
}
