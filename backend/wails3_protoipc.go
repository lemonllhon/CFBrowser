package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const rawProtoEventName = "trace:proto:event"

type Wails3ProtoIPC struct {
	dispatcher   *protoipc.Dispatcher
	binaryServer *protoipc.BinaryServer
	ctxProvider  func() context.Context
	windowsMu    sync.RWMutex
	windows      map[application.Window]struct{}
}

func NewWails3ProtoIPC(app *App, ctxProvider func() context.Context) (*Wails3ProtoIPC, error) {
	dispatcher := protoipc.NewDispatcher()
	registerProtoHandlers(app, dispatcher)
	ipc := &Wails3ProtoIPC{
		dispatcher:  dispatcher,
		ctxProvider: ctxProvider,
		windows:     map[application.Window]struct{}{},
	}
	binaryServer, err := protoipc.StartBinaryServer(context.Background(), dispatcher)
	if err != nil {
		if app != nil {
			app.setProtoEventSink(ipc.broadcastEvent)
		}
		return ipc, err
	}
	ipc.binaryServer = binaryServer
	if app != nil {
		app.setProtoEventSink(ipc.broadcastEvent)
	}
	return ipc, nil
}

func NewWails3ProtoRawMessageHandler(app *App, ctxProvider func() context.Context) func(application.Window, string, *application.OriginInfo) {
	dispatcher := protoipc.NewDispatcher()
	registerProtoHandlers(app, dispatcher)
	ipc := &Wails3ProtoIPC{
		dispatcher:  dispatcher,
		ctxProvider: ctxProvider,
		windows:     map[application.Window]struct{}{},
	}
	return ipc.RawMessageHandler()
}

func (ipc *Wails3ProtoIPC) RawMessageHandler() func(application.Window, string, *application.OriginInfo) {
	return func(window application.Window, message string, _ *application.OriginInfo) {
		ipc.RegisterWindow(window)
		payload, matched, err := protoipc.DecodeRawMessage(message)
		if !matched {
			return
		}
		if err != nil {
			emitWails3ProtoResponse(window, protoipc.EncodeResponse(protoipc.Response{Error: &protoipc.RPCError{
				Code:    protoipc.ErrorCodeBadRequest,
				Message: "Protobuf raw message 解码失败",
				Details: err.Error(),
			}}))
			return
		}

		ctx := context.Background()
		if ipc != nil && ipc.ctxProvider != nil {
			if provided := ipc.ctxProvider(); provided != nil {
				ctx = provided
			}
		}
		dispatcher := ipc.dispatcher
		if dispatcher == nil {
			dispatcher = protoipc.NewDispatcher()
		}
		go ipc.dispatchRawMessage(window, ctx, dispatcher, payload)
	}
}

func (ipc *Wails3ProtoIPC) dispatchRawMessage(window application.Window, ctx context.Context, dispatcher *protoipc.Dispatcher, payload []byte) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Wails3 Proto IPC raw dispatch panic: %v\n%s", recovered, debug.Stack())
			emitWails3ProtoResponse(window, protoipc.EncodeResponse(protoipc.Response{
				RequestID: requestIDFromProtoPayload(payload),
				Error: &protoipc.RPCError{
					Code:    protoipc.ErrorCodeInternal,
					Message: "Proto IPC raw handler panic",
					Details: fmt.Sprint(recovered),
				},
			}))
		}
	}()

	startedAt := time.Now()
	response := dispatcher.Dispatch(ctx, payload)
	if elapsed := time.Since(startedAt); elapsed > 5*time.Second {
		if request, err := protoipc.DecodeEnvelope(payload); err == nil {
			log.Printf("Wails3 Proto IPC raw slow request: method=%s request_id=%s elapsed=%s", request.Method, request.RequestID, elapsed.String())
		} else {
			log.Printf("Wails3 Proto IPC raw slow invalid request: elapsed=%s error=%v", elapsed.String(), err)
		}
	}
	emitWails3ProtoResponse(window, response)
}

func requestIDFromProtoPayload(payload []byte) string {
	request, err := protoipc.DecodeEnvelope(payload)
	if err != nil {
		return ""
	}
	return request.RequestID
}

func (ipc *Wails3ProtoIPC) RegisterWindow(window application.Window) {
	if ipc == nil || window == nil {
		return
	}
	ipc.windowsMu.Lock()
	if ipc.windows == nil {
		ipc.windows = map[application.Window]struct{}{}
	}
	ipc.windows[window] = struct{}{}
	ipc.windowsMu.Unlock()
}

func (ipc *Wails3ProtoIPC) InjectConfig(window application.Window) {
	if ipc == nil || window == nil {
		return
	}
	ipc.RegisterWindow(window)
	script := ipc.ConfigScript()
	if strings.TrimSpace(script) == "" {
		return
	}
	window.ExecJS(script)
}

func (ipc *Wails3ProtoIPC) ConfigScript() string {
	if ipc == nil {
		return ""
	}
	config := struct {
		RawAvailable bool   `json:"rawAvailable"`
		WsURL        string `json:"wsUrl,omitempty"`
	}{
		RawAvailable: true,
	}
	if ipc.binaryServer != nil {
		config.WsURL = ipc.binaryServer.URL()
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		log.Printf("Wails3 Proto IPC config marshal failed: %v", err)
		return ""
	}
	return fmt.Sprintf(
		`window.__TRACE_PROTO_IPC__=Object.assign({},window.__TRACE_PROTO_IPC__||{},%s);window.dispatchEvent(new CustomEvent("trace-proto-config",{detail:window.__TRACE_PROTO_IPC__}));`,
		configJSON,
	)
}

func (ipc *Wails3ProtoIPC) Close() {
	if ipc == nil || ipc.binaryServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := ipc.binaryServer.Close(ctx); err != nil && !strings.Contains(err.Error(), "Server closed") {
		log.Printf("Wails3 Proto IPC binary server shutdown failed: %v", err)
	}
}

func (ipc *Wails3ProtoIPC) broadcastEvent(eventName string, payload []byte) {
	if ipc == nil {
		return
	}
	event := protoipc.Event{
		EventID:     fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		EventName:   eventName,
		Payload:     payload,
		TimestampMS: time.Now().UnixMilli(),
	}
	if ipc.binaryServer != nil {
		ipc.binaryServer.BroadcastEvent(event)
	}
	ipc.broadcastRawEvent(event)
}

func (ipc *Wails3ProtoIPC) broadcastRawEvent(event protoipc.Event) {
	if ipc == nil {
		return
	}
	payload := protoipc.EncodeRawMessage(protoipc.EncodeEvent(event))
	ipc.windowsMu.RLock()
	windows := make([]application.Window, 0, len(ipc.windows))
	for window := range ipc.windows {
		windows = append(windows, window)
	}
	ipc.windowsMu.RUnlock()

	for _, window := range windows {
		if window != nil {
			window.EmitEvent(rawProtoEventName, payload)
		}
	}
}

func emitWails3ProtoResponse(window application.Window, payload []byte) {
	if window == nil {
		log.Printf("Wails3 Proto IPC response dropped: window is nil")
		return
	}
	window.EmitEvent(protoipc.ResponseEventName, protoipc.EncodeRawMessage(payload))
}
