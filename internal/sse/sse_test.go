package sse

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestReadNormalStream(t *testing.T) {
	stream := `data: {"choices":[{"text":"Today "}]}

data: {"choices":[{"text":"is "}]}

data: {"choices":[{"text":"sunny"}]}

data: {"choices":[{"text":"","finish_reason":"stop"}]}

data: [DONE]

`
	res, err := Read(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if res.TextChunks != 3 {
		t.Errorf("TextChunks = %d, want 3", res.TextChunks)
	}
	if res.FirstText.IsZero() || res.Done.IsZero() {
		t.Errorf("FirstText or Done is zero, want both set")
	}
	if res.FirstText.After(res.Done) {
		t.Errorf("FirstText %v is after Done %v", res.FirstText, res.Done)
	}
}

func TestReadEmptyFirstChunk(t *testing.T) {
	pr, pw := io.Pipe()
	textSent := make(chan time.Time, 1)

	go func() {
		defer pw.Close()
		io.WriteString(pw, "data: {\"choices\":[{\"text\":\"\"}]}\n\n")
		time.Sleep(20 * time.Millisecond)
		textSent <- time.Now() // the first real text is sent after this moment
		io.WriteString(pw, "data: {\"choices\":[{\"text\":\"Today \"}]}\n\n")
		io.WriteString(pw, "data: {\"choices\":[{\"text\":\"is\"}]}\n\n")
		io.WriteString(pw, "data: [DONE]\n\n")
	}()

	res, err := Read(pr)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if res.TextChunks != 2 {
		t.Errorf("TextChunks = %d, want 2", res.TextChunks)
	}
	if res.FirstText.Before(<-textSent) {
		t.Errorf("FirstText was stamped on the empty chunk, before any text was sent")
	}
	if res.Done.IsZero() {
		t.Errorf("Done is zero, want set")
	}
	if res.FirstText.After(res.Done) {
		t.Errorf("FirstText %v is after Done %v", res.FirstText, res.Done)
	}
}

func TestReadMissingDone(t *testing.T) {
	stream := `data: {"choices":[{"text":"Today "}]}

data: {"choices":[{"text":"is"}]}

`
	_, err := Read(strings.NewReader(stream))
	if err == nil {
		t.Fatal("Read returned nil error for a stream without [DONE]")
	}
}

func TestReadNoSpaceAfterColon(t *testing.T) {
	stream := `data:{"choices":[{"text":"Today "}]}

data:{"choices":[{"text":"is "}]}

data:{"choices":[{"text":"sunny"}]}

data:[DONE]

`
	res, err := Read(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if res.TextChunks != 3 {
		t.Errorf("TextChunks = %d, want 3", res.TextChunks)
	}
	if res.FirstText.IsZero() || res.Done.IsZero() {
		t.Errorf("FirstText or Done is zero, want both set")
	}
	if res.FirstText.After(res.Done) {
		t.Errorf("FirstText %v is after Done %v", res.FirstText, res.Done)
	}
}

func TestReadNoText(t *testing.T) {
	stream := `data: {"choices":[{"text":"","finish_reason":"stop"}]}

data: [DONE]

`
	res, err := Read(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if res.TextChunks != 0 {
		t.Errorf("TextChunks = %d, want 0", res.TextChunks)
	}
	if !res.FirstText.IsZero() {
		t.Errorf("FirstText = %v, want zero (caller treats this as a failure)", res.FirstText)
	}
	if res.Done.IsZero() {
		t.Errorf("Done is zero, want set")
	}
}
