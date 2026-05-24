package protoipc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDispatcherPing(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.SetClock(func() time.Time {
		return time.UnixMilli(1700000000123)
	})

	requestPayload := EncodePingRequest(PingRequest{
		Message:      "frontend",
		SentAtUnixMS: 1700000000000,
	})
	envelope := EncodeEnvelope(Envelope{
		RequestID:     "req-1",
		Method:        MethodDevPing,
		Payload:       requestPayload,
		SchemaVersion: SchemaVersion,
		TimestampMS:   1700000000001,
	})

	response, err := DecodeResponse(dispatcher.Dispatch(context.Background(), envelope))
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected rpc error: %v", response.Error)
	}
	if response.RequestID != "req-1" {
		t.Fatalf("request id mismatch: %q", response.RequestID)
	}
	pong, err := DecodePingResponse(response.Payload)
	if err != nil {
		t.Fatalf("DecodePingResponse failed: %v", err)
	}
	if pong.Message != "pong: frontend" {
		t.Fatalf("unexpected ping message: %q", pong.Message)
	}
	if pong.ServerTimeUnixMS != 1700000000123 {
		t.Fatalf("unexpected server time: %d", pong.ServerTimeUnixMS)
	}
	if pong.PayloadSize != int32(len(requestPayload)) {
		t.Fatalf("unexpected payload size: %d", pong.PayloadSize)
	}
}

func TestDispatcherUnknownMethod(t *testing.T) {
	dispatcher := NewDispatcher()
	envelope := EncodeEnvelope(Envelope{
		RequestID:     "req-unknown",
		Method:        "trace.dev.Missing",
		SchemaVersion: SchemaVersion,
	})

	response, err := DecodeResponse(dispatcher.Dispatch(context.Background(), envelope))
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if response.Error == nil {
		t.Fatal("expected rpc error")
	}
	if response.Error.Code != ErrorCodeUnknownMethod {
		t.Fatalf("unexpected error code: %q", response.Error.Code)
	}
}

func TestDispatcherInvalidEnvelope(t *testing.T) {
	dispatcher := NewDispatcher()

	response, err := DecodeResponse(dispatcher.Dispatch(context.Background(), []byte{0xff, 0xff, 0xff}))
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if response.Error == nil {
		t.Fatal("expected rpc error")
	}
	if response.Error.Code != ErrorCodeBadRequest {
		t.Fatalf("unexpected error code: %q", response.Error.Code)
	}
}

func TestDispatcherConcurrentPing(t *testing.T) {
	dispatcher := NewDispatcher()
	const requestCount = 100

	var wg sync.WaitGroup
	errCh := make(chan error, requestCount)
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			requestID := fmt.Sprintf("req-%03d", index)
			message := fmt.Sprintf("frontend-%03d", index)
			envelope := EncodeEnvelope(Envelope{
				RequestID: requestID,
				Method:    MethodDevPing,
				Payload: EncodePingRequest(PingRequest{
					Message:      message,
					SentAtUnixMS: 1700000000000 + int64(index),
				}),
				SchemaVersion: SchemaVersion,
				TimestampMS:   1700000001000 + int64(index),
			})

			response, err := DecodeResponse(dispatcher.Dispatch(context.Background(), envelope))
			if err != nil {
				errCh <- fmt.Errorf("%s: decode response: %w", requestID, err)
				return
			}
			if response.Error != nil {
				errCh <- fmt.Errorf("%s: unexpected rpc error: %v", requestID, response.Error)
				return
			}
			if response.RequestID != requestID {
				errCh <- fmt.Errorf("%s: response request id mismatch: %s", requestID, response.RequestID)
				return
			}
			pong, err := DecodePingResponse(response.Payload)
			if err != nil {
				errCh <- fmt.Errorf("%s: decode ping response: %w", requestID, err)
				return
			}
			if pong.Message != "pong: "+message {
				errCh <- fmt.Errorf("%s: unexpected pong: %s", requestID, pong.Message)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestDispatcherLargePingPayload(t *testing.T) {
	dispatcher := NewDispatcher()
	message := strings.Repeat("a", 1024*1024)
	requestPayload := EncodePingRequest(PingRequest{Message: message})
	envelope := EncodeEnvelope(Envelope{
		RequestID:     "req-large",
		Method:        MethodDevPing,
		Payload:       requestPayload,
		SchemaVersion: SchemaVersion,
		TimestampMS:   1700000000000,
	})

	response, err := DecodeResponse(dispatcher.Dispatch(context.Background(), envelope))
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected rpc error: %v", response.Error)
	}
	pong, err := DecodePingResponse(response.Payload)
	if err != nil {
		t.Fatalf("DecodePingResponse failed: %v", err)
	}
	if pong.Message != "pong: "+message {
		t.Fatalf("large payload response mismatch")
	}
	if pong.PayloadSize != int32(len(requestPayload)) {
		t.Fatalf("unexpected payload size: %d", pong.PayloadSize)
	}
}

func TestRawMessageRoundTrip(t *testing.T) {
	encoded := EncodeRawMessage([]byte{1, 2, 3, 4})
	decoded, matched, err := DecodeRawMessage(encoded)
	if err != nil {
		t.Fatalf("DecodeRawMessage failed: %v", err)
	}
	if !matched {
		t.Fatal("expected raw message to match")
	}
	if string(decoded) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected decoded payload: %v", decoded)
	}

	if _, matched, err := DecodeRawMessage("other"); matched || err == nil {
		t.Fatalf("expected prefix mismatch, matched=%v err=%v", matched, err)
	}
}
