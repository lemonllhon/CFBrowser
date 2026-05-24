package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"strings"
)

func (a *App) handleProtoBrowserUserDataDirOpen(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserUserDataDirOpenRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserUserDataDirOpenRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.OpenUserDataDir(strings.TrimSpace(input.UserDataDir)); err != nil {
		return nil, protoBrowserOperationError("打开用户数据目录失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserProfileUserDataDirOpen(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProfileUserDataDirOpenRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProfileUserDataDirOpenRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.OpenProfileUserDataDir(strings.TrimSpace(input.ProfileID)); err != nil {
		return nil, protoBrowserOperationError("打开实例用户数据目录失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}
