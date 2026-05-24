package protoipc

import (
	"context"
	"strings"
	"time"
)

const (
	MethodDevPing = "trace.dev.Ping"

	ErrorCodeBadRequest     = "bad_request"
	ErrorCodeUnknownMethod  = "unknown_method"
	ErrorCodeInvalidPayload = "invalid_payload"
	ErrorCodeInternal       = "internal"
)

type Handler func(ctx context.Context, request Envelope) ([]byte, *RPCError)

type Dispatcher struct {
	handlers map[string]Handler
	now      func() time.Time
}

func NewDispatcher() *Dispatcher {
	dispatcher := &Dispatcher{
		handlers: map[string]Handler{},
		now:      time.Now,
	}
	dispatcher.Register(MethodDevPing, dispatcher.handlePing)
	return dispatcher
}

func (d *Dispatcher) Register(method string, handler Handler) {
	if d == nil || strings.TrimSpace(method) == "" || handler == nil {
		return
	}
	if d.handlers == nil {
		d.handlers = map[string]Handler{}
	}
	d.handlers[method] = handler
}

func (d *Dispatcher) SetClock(now func() time.Time) {
	if d == nil || now == nil {
		return
	}
	d.now = now
}

func (d *Dispatcher) Dispatch(ctx context.Context, payload []byte) []byte {
	request, err := DecodeEnvelope(payload)
	if err != nil {
		return EncodeResponse(Response{Error: &RPCError{
			Code:    ErrorCodeBadRequest,
			Message: "请求包不是有效的 Protobuf envelope",
			Details: err.Error(),
		}})
	}

	if request.RequestID == "" {
		return EncodeResponse(Response{Error: &RPCError{
			Code:    ErrorCodeBadRequest,
			Message: "请求缺少 request_id",
		}})
	}
	if request.SchemaVersion != SchemaVersion {
		return EncodeResponse(Response{RequestID: request.RequestID, Error: &RPCError{
			Code:    ErrorCodeBadRequest,
			Message: "不支持的 Protobuf schema_version",
		}})
	}

	handler := d.handlerFor(request.Method)
	if handler == nil {
		return EncodeResponse(Response{RequestID: request.RequestID, Error: &RPCError{
			Code:    ErrorCodeUnknownMethod,
			Message: "未知的 Protobuf RPC 方法",
			Details: request.Method,
		}})
	}

	responsePayload, rpcErr := handler(ctx, request)
	if rpcErr != nil {
		return EncodeResponse(Response{RequestID: request.RequestID, Error: rpcErr})
	}
	return EncodeResponse(Response{
		RequestID: request.RequestID,
		Payload:   responsePayload,
	})
}

func (d *Dispatcher) handlerFor(method string) Handler {
	if d == nil {
		return nil
	}
	return d.handlers[method]
}

func (d *Dispatcher) handlePing(ctx context.Context, request Envelope) ([]byte, *RPCError) {
	if err := ctx.Err(); err != nil {
		return nil, &RPCError{
			Code:    ErrorCodeInternal,
			Message: "请求上下文已取消",
			Details: err.Error(),
		}
	}
	ping, err := DecodePingRequest(request.Payload)
	if err != nil {
		return nil, &RPCError{
			Code:    ErrorCodeInvalidPayload,
			Message: "PingRequest 解码失败",
			Details: err.Error(),
		}
	}

	message := strings.TrimSpace(ping.Message)
	if message == "" {
		message = "pong"
	} else {
		message = "pong: " + message
	}

	now := time.Now
	if d != nil && d.now != nil {
		now = d.now
	}

	return EncodePingResponse(PingResponse{
		Message:          message,
		ServerTimeUnixMS: now().UnixMilli(),
		PayloadSize:      int32(len(request.Payload)),
	}), nil
}
