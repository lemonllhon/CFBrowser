package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
)

func (a *App) handleProtoBrowserSettingsGet(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserSettingsReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeBrowserSettingsResponse(protoipc.BrowserSettingsResponse{
		Settings: browserSettingsToProto(a.GetBrowserSettings()),
	}), nil
}

func (a *App) handleProtoBrowserSettingsSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserSettingsReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserSettingsSaveRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserSettingsSaveRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.SaveBrowserSettings(browserSettingsFromProto(input.Settings)); err != nil {
		return nil, protoBrowserOperationError("保存浏览器设置失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) ensureProtoBrowserSettingsReady(ctx context.Context) *protoipc.RPCError {
	if err := ctx.Err(); err != nil {
		return &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInternal,
			Message: "请求上下文已取消",
			Details: err.Error(),
		}
	}
	if a == nil || a.config == nil {
		return &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInternal,
			Message: "浏览器设置服务尚未初始化",
		}
	}
	return nil
}

func browserSettingsToProto(settings BrowserSettings) protoipc.BrowserSettings {
	return protoipc.BrowserSettings{
		UserDataRoot:           settings.UserDataRoot,
		DefaultFingerprintArgs: append([]string{}, settings.DefaultFingerprintArgs...),
		DefaultLaunchArgs:      append([]string{}, settings.DefaultLaunchArgs...),
		DefaultProxy:           settings.DefaultProxy,
		StartReadyTimeoutMs:    int32(settings.StartReadyTimeoutMs),
		StartStableWindowMs:    int32(settings.StartStableWindowMs),
	}
}

func browserSettingsFromProto(settings protoipc.BrowserSettings) BrowserSettings {
	return BrowserSettings{
		UserDataRoot:           settings.UserDataRoot,
		DefaultFingerprintArgs: append([]string{}, settings.DefaultFingerprintArgs...),
		DefaultLaunchArgs:      append([]string{}, settings.DefaultLaunchArgs...),
		DefaultProxy:           settings.DefaultProxy,
		StartReadyTimeoutMs:    int(settings.StartReadyTimeoutMs),
		StartStableWindowMs:    int(settings.StartStableWindowMs),
	}
}
