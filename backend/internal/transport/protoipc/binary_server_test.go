package protoipc

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBinaryServerPing(t *testing.T) {
	server, err := StartBinaryServer(context.Background(), NewDispatcher())
	if err != nil {
		t.Fatalf("StartBinaryServer failed: %v", err)
	}
	defer server.Close(context.Background())

	conn, _, err := websocket.DefaultDialer.Dial(server.URL(), nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	requestPayload := EncodePingRequest(PingRequest{
		Message:      "websocket",
		SentAtUnixMS: time.Now().UnixMilli(),
	})
	envelope := EncodeEnvelope(Envelope{
		RequestID:     "ws-req-1",
		Method:        MethodDevPing,
		Payload:       requestPayload,
		SchemaVersion: SchemaVersion,
		TimestampMS:   time.Now().UnixMilli(),
	})
	if err := conn.WriteMessage(websocket.BinaryMessage, envelope); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	messageType, responsePayload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("unexpected message type: %d", messageType)
	}
	frameType, responseFrame, ok := DecodeBinaryFrame(responsePayload)
	if !ok || frameType != BinaryFrameResponse {
		t.Fatalf("unexpected response frame type: %d ok=%v", frameType, ok)
	}
	response, err := DecodeResponse(responseFrame)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected rpc error: %v", response.Error)
	}
	if response.RequestID != "ws-req-1" {
		t.Fatalf("unexpected request id: %s", response.RequestID)
	}
	pong, err := DecodePingResponse(response.Payload)
	if err != nil {
		t.Fatalf("DecodePingResponse failed: %v", err)
	}
	if pong.Message != "pong: websocket" {
		t.Fatalf("unexpected pong message: %s", pong.Message)
	}
}

func TestBinaryServerRejectsInvalidToken(t *testing.T) {
	server, err := StartBinaryServer(context.Background(), NewDispatcher())
	if err != nil {
		t.Fatalf("StartBinaryServer failed: %v", err)
	}
	defer server.Close(context.Background())

	badURL := server.URL() + "x"

	_, response, err := websocket.DefaultDialer.Dial(badURL, nil)
	if err == nil {
		t.Fatal("expected Dial to fail")
	}
	if response == nil {
		t.Fatalf("expected HTTP response for rejected dial: %v", err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
}

func TestBinaryServerBroadcastEvent(t *testing.T) {
	server, err := StartBinaryServer(context.Background(), NewDispatcher())
	if err != nil {
		t.Fatalf("StartBinaryServer failed: %v", err)
	}
	defer server.Close(context.Background())

	conn, _, err := websocket.DefaultDialer.Dial(server.URL(), nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.BinaryMessage, EncodeEnvelope(Envelope{
		RequestID:     "ready",
		Method:        MethodDevPing,
		Payload:       EncodePingRequest(PingRequest{Message: "ready"}),
		SchemaVersion: SchemaVersion,
		TimestampMS:   time.Now().UnixMilli(),
	})); err != nil {
		t.Fatalf("Write ready message failed: %v", err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Read ready response failed: %v", err)
	}

	eventPayload := EncodePingResponse(PingResponse{
		Message:          "event-payload",
		ServerTimeUnixMS: 1700000000000,
		PayloadSize:      13,
	})
	server.BroadcastEvent(Event{
		EventID:     "event-1",
		EventName:   "trace.dev.Event",
		Payload:     eventPayload,
		TimestampMS: 1700000000001,
	})

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if messageType != websocket.BinaryMessage {
		t.Fatalf("unexpected message type: %d", messageType)
	}

	frameType, eventFrame, ok := DecodeBinaryFrame(payload)
	if !ok || frameType != BinaryFrameEvent {
		t.Fatalf("unexpected event frame type: %d ok=%v", frameType, ok)
	}
	event, err := DecodeEvent(eventFrame)
	if err != nil {
		t.Fatalf("DecodeEvent failed: %v", err)
	}
	if event.EventID != "event-1" {
		t.Fatalf("unexpected event id: %s", event.EventID)
	}
	if event.EventName != "trace.dev.Event" {
		t.Fatalf("unexpected event name: %s", event.EventName)
	}
	decodedPayload, err := DecodePingResponse(event.Payload)
	if err != nil {
		t.Fatalf("DecodePingResponse failed: %v", err)
	}
	if decodedPayload.Message != "event-payload" {
		t.Fatalf("unexpected event payload: %s", decodedPayload.Message)
	}
}
