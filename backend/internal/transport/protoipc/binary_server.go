package protoipc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const WebSocketPath = "/trace/proto"

const (
	BinaryFrameResponse byte = 1
	BinaryFrameEvent    byte = 2
)

type BinaryServerConfig struct {
	URL string `json:"wsUrl"`
}

type BinaryServer struct {
	dispatcher *Dispatcher
	listener   net.Listener
	server     *http.Server
	token      string
	url        string
	clientsMu  sync.RWMutex
	clients    map[*binaryClient]struct{}
	closeOnce  sync.Once
}

type binaryClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func StartBinaryServer(ctx context.Context, dispatcher *Dispatcher) (*BinaryServer, error) {
	if dispatcher == nil {
		dispatcher = NewDispatcher()
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start proto ipc listener: %w", err)
	}

	result := &BinaryServer{
		dispatcher: dispatcher,
		listener:   listener,
		token:      token,
		url:        fmt.Sprintf("ws://%s%s?token=%s", listener.Addr().String(), WebSocketPath, token),
		clients:    map[*binaryClient]struct{}{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(WebSocketPath, result.handleWebSocket(ctx))
	result.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := result.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Proto IPC binary server stopped unexpectedly: %v", err)
		}
	}()

	return result, nil
}

func (s *BinaryServer) Config() BinaryServerConfig {
	if s == nil {
		return BinaryServerConfig{}
	}
	return BinaryServerConfig{URL: s.url}
}

func (s *BinaryServer) URL() string {
	if s == nil {
		return ""
	}
	return s.url
}

func (s *BinaryServer) BroadcastEvent(event Event) {
	if s == nil {
		return
	}
	payload := EncodeBinaryFrame(BinaryFrameEvent, EncodeEvent(event))
	s.clientsMu.RLock()
	clients := make([]*binaryClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	s.clientsMu.RUnlock()

	for _, client := range clients {
		if err := client.write(websocket.BinaryMessage, payload); err != nil {
			s.removeClient(client)
		}
	}
}

func (s *BinaryServer) Close(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	var err error
	s.closeOnce.Do(func() {
		err = s.server.Shutdown(ctx)
	})
	return err
}

func (s *BinaryServer) handleWebSocket(ctx context.Context) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return isLoopbackHost(r.Host)
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if s == nil || subtleStringMismatch(r.URL.Query().Get("token"), s.token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !isLoopbackRemote(r.RemoteAddr) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		client := &binaryClient{conn: conn}
		s.addClient(client)
		defer s.removeClient(client)

		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if messageType != websocket.BinaryMessage {
				if err := client.write(websocket.BinaryMessage, EncodeBinaryFrame(BinaryFrameResponse, EncodeResponse(Response{Error: &RPCError{
					Code:    ErrorCodeBadRequest,
					Message: "Proto IPC WebSocket 仅接受 binary frame",
				}}))); err != nil {
					return
				}
				continue
			}

			requestPayload := append([]byte(nil), payload...)
			go s.dispatchAndWrite(ctx, client, requestPayload)
		}
	}
}

func (s *BinaryServer) dispatchAndWrite(ctx context.Context, client *binaryClient, payload []byte) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Proto IPC binary dispatch panic: %v\n%s", recovered, debug.Stack())
			requestID := ""
			if request, err := DecodeEnvelope(payload); err == nil {
				requestID = request.RequestID
			}
			response := EncodeResponse(Response{Error: &RPCError{
				Code:    ErrorCodeInternal,
				Message: "Proto IPC handler panic",
				Details: fmt.Sprint(recovered),
			}, RequestID: requestID})
			if err := client.write(websocket.BinaryMessage, EncodeBinaryFrame(BinaryFrameResponse, response)); err != nil {
				s.removeClient(client)
			}
		}
	}()

	startedAt := time.Now()
	response := s.dispatcher.Dispatch(ctx, payload)
	if elapsed := time.Since(startedAt); elapsed > 5*time.Second {
		if request, err := DecodeEnvelope(payload); err == nil {
			log.Printf("Proto IPC slow request: method=%s request_id=%s elapsed=%s", request.Method, request.RequestID, elapsed.String())
		} else {
			log.Printf("Proto IPC slow invalid request: elapsed=%s error=%v", elapsed.String(), err)
		}
	}
	if err := client.write(websocket.BinaryMessage, EncodeBinaryFrame(BinaryFrameResponse, response)); err != nil {
		s.removeClient(client)
	}
}

func (s *BinaryServer) addClient(client *binaryClient) {
	if s == nil || client == nil {
		return
	}
	s.clientsMu.Lock()
	s.clients[client] = struct{}{}
	s.clientsMu.Unlock()
}

func (s *BinaryServer) removeClient(client *binaryClient) {
	if s == nil || client == nil {
		return
	}
	s.clientsMu.Lock()
	delete(s.clients, client)
	s.clientsMu.Unlock()
}

func (c *binaryClient) write(messageType int, payload []byte) error {
	if c == nil || c.conn == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(messageType, payload)
}

func EncodeBinaryFrame(frameType byte, payload []byte) []byte {
	out := make([]byte, 0, len(payload)+1)
	out = append(out, frameType)
	out = append(out, payload...)
	return out
}

func DecodeBinaryFrame(frame []byte) (byte, []byte, bool) {
	if len(frame) == 0 {
		return 0, nil, false
	}
	switch frame[0] {
	case BinaryFrameResponse, BinaryFrameEvent:
		return frame[0], frame[1:], true
	default:
		return 0, frame, false
	}
}

func generateToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate proto ipc token: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func isLoopbackHost(host string) bool {
	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}
	ip := net.ParseIP(strings.Trim(name, "[]"))
	return ip != nil && ip.IsLoopback()
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func subtleStringMismatch(left string, right string) bool {
	if len(left) != len(right) {
		return true
	}
	var diff byte
	for i := 0; i < len(left); i++ {
		diff |= left[i] ^ right[i]
	}
	return diff != 0
}
