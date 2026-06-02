package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"strings"
)

func (a *App) handleProtoBrowserProfileBackupExport(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProfileBackupExportRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProfileBackupExportRequest 解码失败",
			Details: err.Error(),
		}
	}
	result, err := a.BrowserProfilesBackupExport(ProfileBackupExportRequest{
		Scope:                          strings.TrimSpace(input.Scope),
		ProfileIDs:                     append([]string{}, input.ProfileIDs...),
		IncludeCookies:                 input.IncludeCookies,
		IncludePlainCookiesWhenRunning: input.IncludePlainCookiesWhenRunning,
	})
	if err != nil {
		return nil, protoBrowserOperationError("导出实例备份失败", err)
	}
	return protoipc.EncodeBrowserProfileBackupActionResult(profileBackupActionResultToProto(result)), nil
}

func (a *App) handleProtoBrowserProfileBackupChooseImport(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := a.BrowserProfilesBackupChooseImportPackage()
	if err != nil {
		return nil, protoBrowserOperationError("选择实例备份包失败", err)
	}
	return protoipc.EncodeBrowserProfileBackupActionResult(profileBackupActionResultToProto(result)), nil
}

func (a *App) handleProtoBrowserProfileBackupImport(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProfileBackupImportRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProfileBackupImportRequest 解码失败",
			Details: err.Error(),
		}
	}
	result, err := a.BrowserProfilesBackupImport(ProfileBackupImportRequest{
		ZipPath:        strings.TrimSpace(input.ZipPath),
		RestoreCookies: input.RestoreCookies,
	})
	if err != nil {
		return nil, protoBrowserOperationError("恢复实例备份失败", err)
	}
	return protoipc.EncodeBrowserProfileBackupActionResult(profileBackupActionResultToProto(result)), nil
}

func profileBackupActionResultToProto(result ProfileBackupActionResult) protoipc.BrowserProfileBackupActionResult {
	return protoipc.BrowserProfileBackupActionResult{
		Cancelled:          result.Cancelled,
		Message:            result.Message,
		ZipPath:            result.ZipPath,
		CreatedAt:          result.CreatedAt,
		Exported:           int32(result.Exported),
		Imported:           int32(result.Imported),
		Skipped:            int32(result.Skipped),
		Failed:             int32(result.Failed),
		ProfileCount:       int32(result.ProfileCount),
		CookieProfileCount: int32(result.CookieProfileCount),
		Summary:            profileBackupSummaryToProto(result.Summary),
		Warnings:           profileBackupWarningsToProto(result.Warnings),
	}
}

func profileBackupSummaryToProto(summary ProfileBackupSummary) protoipc.BrowserProfileBackupSummary {
	return protoipc.BrowserProfileBackupSummary{
		ZipPath:              summary.ZipPath,
		Format:               summary.Format,
		Version:              int32(summary.Version),
		AppName:              summary.AppName,
		AppVersion:           summary.AppVersion,
		CreatedAt:            summary.CreatedAt,
		SourceOS:             summary.SourceOS,
		ProfileCount:         int32(summary.ProfileCount),
		CookieProfileCount:   int32(summary.CookieProfileCount),
		IncludesCookies:      summary.IncludesCookies,
		IncludesPlainCookies: summary.IncludesPlainCookies,
		CookieNotice:         summary.CookieNotice,
		Warnings:             append([]string{}, summary.Warnings...),
	}
}

func profileBackupWarningsToProto(items []ProfileBackupWarning) []protoipc.BrowserProfileBackupWarning {
	out := make([]protoipc.BrowserProfileBackupWarning, 0, len(items))
	for _, item := range items {
		out = append(out, protoipc.BrowserProfileBackupWarning{
			ProfileID:   item.ProfileID,
			ProfileName: item.ProfileName,
			Message:     item.Message,
		})
	}
	return out
}
