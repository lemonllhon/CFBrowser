package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Wails3ProtoIPC struct {
	dispatcher   *protoipc.Dispatcher
	binaryServer *protoipc.BinaryServer
	ctxProvider  func() context.Context
}

func NewWails3ProtoIPC(app *App, ctxProvider func() context.Context) (*Wails3ProtoIPC, error) {
	dispatcher := protoipc.NewDispatcher()
	registerProtoHandlers(app, dispatcher)
	binaryServer, err := protoipc.StartBinaryServer(context.Background(), dispatcher)
	if err != nil {
		return nil, err
	}
	if app != nil {
		app.setProtoEventSink(func(eventName string, payload []byte) {
			binaryServer.BroadcastEvent(protoipc.Event{
				EventID:     fmt.Sprintf("evt-%d", time.Now().UnixNano()),
				EventName:   eventName,
				Payload:     payload,
				TimestampMS: time.Now().UnixMilli(),
			})
		})
	}
	return &Wails3ProtoIPC{
		dispatcher:   dispatcher,
		binaryServer: binaryServer,
		ctxProvider:  ctxProvider,
	}, nil
}

func NewWails3ProtoRawMessageHandler(ctxProvider func() context.Context) func(application.Window, string, *application.OriginInfo) {
	ipc := &Wails3ProtoIPC{
		dispatcher:  protoipc.NewDispatcher(),
		ctxProvider: ctxProvider,
	}
	return ipc.RawMessageHandler()
}

func (ipc *Wails3ProtoIPC) RawMessageHandler() func(application.Window, string, *application.OriginInfo) {
	return func(window application.Window, message string, _ *application.OriginInfo) {
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
		emitWails3ProtoResponse(window, ipc.dispatcher.Dispatch(ctx, payload))
	}
}

func (ipc *Wails3ProtoIPC) InjectConfig(window application.Window) {
	if ipc == nil || ipc.binaryServer == nil || window == nil {
		return
	}
	script := ipc.ConfigScript()
	if strings.TrimSpace(script) == "" {
		return
	}
	window.ExecJS(script)
}

func (ipc *Wails3ProtoIPC) ConfigScript() string {
	if ipc == nil || ipc.binaryServer == nil {
		return ""
	}
	configJSON, err := json.Marshal(ipc.binaryServer.Config())
	if err != nil {
		log.Printf("Wails3 Proto IPC config marshal failed: %v", err)
		return ""
	}
	return fmt.Sprintf(
		`window.__TRACE_PROTO_IPC__=%s;window.dispatchEvent(new CustomEvent("trace-proto-config",{detail:%s}));`,
		configJSON,
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

func emitWails3ProtoResponse(window application.Window, payload []byte) {
	if window == nil {
		log.Printf("Wails3 Proto IPC response dropped: window is nil")
		return
	}
	window.EmitEvent(protoipc.ResponseEventName, protoipc.EncodeRawMessage(payload))
}
