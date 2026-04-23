package handlers

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meshery/meshery/server/meshes"
	"github.com/meshery/meshkit/logger"
	_events "github.com/meshery/meshkit/utils/events"
)

const testTimeout = time.Second

type testFlusher struct {
	flushes atomic.Int32
	flushCh chan struct{}
}

func (f *testFlusher) Flush() {
	f.flushes.Add(1)
	if f.flushCh != nil {
		select {
		case f.flushCh <- struct{}{}:
		default:
		}
	}
}

type safeBuffer struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	writeCh chan struct{}
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	n, err := b.buf.Write(p)
	if b.writeCh != nil {
		select {
		case b.writeCh <- struct{}{}:
		default:
		}
	}

	return n, err
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func waitForSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", name)
	}
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
			case <-time.After(testTimeout):
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

	respChan := make(chan []byte, 1)
	flusher := &testFlusher{flushCh: make(chan struct{}, 1)}
	body := &safeBuffer{writeCh: make(chan struct{}, 1)}
	done := make(chan struct{})

	go func() {
		writeEventStream(ctx, body, respChan, log, flusher)
		close(done)
	}()

	respChan <- []byte(`{"status":"ok"}`)

	waitForSignal(t, body.writeCh, "stream write")
	waitForSignal(t, flusher.flushCh, "flush")

	if body.String() != "data: {\"status\":\"ok\"}\n\n" {
		t.Fatalf("unexpected stream output: %q", body.String())
	}

	if flusher.flushes.Load() != 1 {
		t.Fatalf("expected flusher to be called once, got %d", flusher.flushes.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(testTimeout):
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
	respChan := make(chan []byte, 1)
	respChan <- []byte("pre-filled to force a blocked send")
	done := make(chan struct{})
	subscribed := make(chan struct{}, 1)
	originalSubscribe := subscribeToEventStream
	subscribeToEventStream = func(eb *_events.EventStreamer, ch chan interface{}) {
		originalSubscribe(eb, ch)
		subscribed <- struct{}{}
	}
	defer func() {
		subscribeToEventStream = originalSubscribe
	}()

	go func() {
		listenForCoreEvents(ctx, eb, respChan, log, nil)
		close(done)
	}()

	waitForSignal(t, subscribed, "event streamer subscription")
	eb.Publish(&meshes.EventsResponse{Summary: "stream event"})
	cancel()

	select {
	case <-done:
	case <-time.After(2 * testTimeout):
		t.Fatal("listenForCoreEvents remained blocked after cancellation")
	}
}
