package handlers

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/meshery/meshery/server/meshes"
	"github.com/meshery/meshkit/logger"
	_events "github.com/meshery/meshkit/utils/events"
)

type testFlusher struct {
	flushes int
}

func (f *testFlusher) Flush() {
	f.flushes++
}

func TestSendStreamEvent(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func() (context.Context, context.CancelFunc)
		setupChan   func() chan []byte
		expectSent  bool
		expectValue []byte
	}{
		{
			name: "sends data when receiver is available",
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			setupChan: func() chan []byte {
				return make(chan []byte, 1)
			},
			expectSent:  true,
			expectValue: []byte("payload"),
		},
		{
			name: "returns false when context is cancelled",
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			setupChan: func() chan []byte {
				return make(chan []byte)
			},
			expectSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupCtx()
			defer cancel()

			respChan := tt.setupChan()
			sent := sendStreamEvent(ctx, respChan, []byte("payload"))
			if sent != tt.expectSent {
				t.Fatalf("expected sent=%v, got %v", tt.expectSent, sent)
			}

			if !tt.expectSent {
				return
			}

			select {
			case got := <-respChan:
				if string(got) != string(tt.expectValue) {
					t.Fatalf("expected %q, got %q", tt.expectValue, got)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for streamed payload")
			}
		})
	}
}

func TestWriteEventStream_StopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, err := logger.New("test", logger.Options{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	respChan := make(chan []byte)
	flusher := &testFlusher{}
	var body bytes.Buffer
	done := make(chan struct{})

	go func() {
		writeEventStream(ctx, &body, respChan, log, flusher)
		close(done)
	}()

	respChan <- []byte(`{"status":"ok"}`)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if body.String() == "data: {\"status\":\"ok\"}\n\n" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if body.String() != "data: {\"status\":\"ok\"}\n\n" {
		t.Fatalf("unexpected stream output: %q", body.String())
	}

	if flusher.flushes != 1 {
		t.Fatalf("expected flusher to be called once, got %d", flusher.flushes)
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("writeEventStream did not stop after context cancellation")
	}
}

func TestListenForCoreEvents_StopsBlockedSendOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log, err := logger.New("test", logger.Options{})
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	eb := _events.NewEventStreamer()
	respChan := make(chan []byte)
	done := make(chan struct{})

	go func() {
		listenForCoreEvents(ctx, eb, respChan, log, nil)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	eb.Publish(&meshes.EventsResponse{Summary: "stream event"})
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listenForCoreEvents remained blocked after cancellation")
	}
}
