package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"strings"
)

func (a *App) handleProtoBrowserCookieList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserCookieProfileRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserCookieProfileRequest 解码失败",
			Details: err.Error(),
		}
	}
	cookies, err := a.BrowserGetCookies(strings.TrimSpace(input.ProfileID))
	if err != nil {
		return nil, protoBrowserOperationError("获取浏览器实例 Cookie 失败", err)
	}
	return protoipc.EncodeBrowserCookieListResponse(protoipc.BrowserCookieListResponse{
		Cookies: browserCookiesToProto(cookies),
	}), nil
}

func (a *App) handleProtoBrowserCookieClear(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserCookieProfileRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserCookieProfileRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.BrowserClearCookies(strings.TrimSpace(input.ProfileID)); err != nil {
		return nil, protoBrowserOperationError("清空浏览器实例 Cookie 失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserCookieExport(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserCookieProfileRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserCookieProfileRequest 解码失败",
			Details: err.Error(),
		}
	}
	content, err := a.BrowserExportCookies(strings.TrimSpace(input.ProfileID))
	if err != nil {
		return nil, protoBrowserOperationError("导出浏览器实例 Cookie 失败", err)
	}
	return protoipc.EncodeBrowserCookieExportResponse(protoipc.BrowserCookieExportResponse{
		Content: content,
	}), nil
}

func (a *App) handleProtoBrowserCookieImport(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserCookieImportRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserCookieImportRequest 解码失败",
			Details: err.Error(),
		}
	}
	result, err := a.BrowserImportCookies(strings.TrimSpace(input.ProfileID), input.Content)
	if err != nil {
		return nil, protoBrowserOperationError("导入浏览器实例 Cookie 失败", err)
	}
	return protoipc.EncodeBrowserCookieImportResult(protoipc.BrowserCookieImportResult{
		Imported: int32(result.Imported),
		Skipped:  int32(result.Skipped),
	}), nil
}

func browserCookiesToProto(cookies []CookieInfo) []protoipc.BrowserCookieInfo {
	out := make([]protoipc.BrowserCookieInfo, 0, len(cookies))
	for _, cookie := range cookies {
		out = append(out, protoipc.BrowserCookieInfo{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  int64(cookie.Expires),
			HTTPOnly: cookie.HttpOnly,
			Secure:   cookie.Secure,
			SameSite: cookie.SameSite,
		})
	}
	return out
}
