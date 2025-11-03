package workflowrunner

import (
	"testing"

	"github.com/openai/openai-go/v2/responses"
)

func TestBuildInputItems_TextMessage(t *testing.T) {
	inputs := []WorkflowInput{
		{
			Type:    "message",
			Role:    "user",
			Content: "Hello there!",
		},
	}
	items, err := buildInputItems(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	msg := items[0].OfMessage
	if msg == nil {
		t.Fatalf("expected message input, got %#v", items[0])
	}
	if !msg.Content.OfString.Valid() || msg.Content.OfString.Value != "Hello there!" {
		t.Fatalf("unexpected message content: %#v", msg.Content)
	}
}

func TestBuildInputItems_MessageWithImageAndText(t *testing.T) {
	inputs := []WorkflowInput{
		{
			Type: "message",
			Role: "user",
			Content: []any{
				map[string]any{
					"type":      "input_image",
					"image_url": "https://example.com/image.png",
					"detail":    "high",
				},
				map[string]any{
					"type": "input_text",
					"text": "Describe this picture.",
				},
			},
		},
	}
	items, err := buildInputItems(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	msg := items[0].OfMessage
	if msg == nil {
		t.Fatalf("expected message input, got %#v", items[0])
	}
	list := msg.Content.OfInputItemContentList
	if len(list) != 2 {
		t.Fatalf("expected 2 content items, got %d", len(list))
	}
	if list[0].OfInputImage == nil {
		t.Fatalf("expected first item to be image, got %#v", list[0])
	}
	if !list[0].OfInputImage.ImageURL.Valid() || list[0].OfInputImage.ImageURL.Value != "https://example.com/image.png" {
		t.Fatalf("unexpected image url: %#v", list[0].OfInputImage.ImageURL)
	}
	if list[0].OfInputImage.Detail != responses.ResponseInputImageDetailHigh {
		t.Fatalf("unexpected image detail: %s", list[0].OfInputImage.Detail)
	}
	if list[1].OfInputText == nil || list[1].OfInputText.Text != "Describe this picture." {
		t.Fatalf("unexpected text content: %#v", list[1])
	}
}

func TestBuildInputItems_ImageShortcut(t *testing.T) {
	inputs := []WorkflowInput{
		{
			Type: "image",
			URI:  "https://example.com/photo.jpg",
		},
	}
	items, err := buildInputItems(inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	msg := items[0].OfMessage
	if msg == nil {
		t.Fatalf("expected message input, got %#v", items[0])
	}
	list := msg.Content.OfInputItemContentList
	if len(list) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(list))
	}
	if list[0].OfInputImage == nil {
		t.Fatalf("expected image content, got %#v", list[0])
	}
	if !list[0].OfInputImage.ImageURL.Valid() || list[0].OfInputImage.ImageURL.Value != "https://example.com/photo.jpg" {
		t.Fatalf("unexpected image url: %#v", list[0].OfInputImage.ImageURL)
	}
}
