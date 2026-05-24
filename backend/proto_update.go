package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
)

func registerProtoUpdateHandlers(app *App, dispatcher *protoipc.Dispatcher) {
	if app == nil || dispatcher == nil {
		return
	}
	dispatcher.Register(protoipc.MethodAppUpdateCheck, app.handleProtoAppUpdateCheck)
	dispatcher.Register(protoipc.MethodAppUpdateDownload, app.handleProtoAppUpdateDownload)
	dispatcher.Register(protoipc.MethodAppUpdateInstallDownloaded, app.handleProtoAppUpdateInstallDownloaded)
	dispatcher.Register(protoipc.MethodAppUpdateDownloadPortable, app.handleProtoAppUpdateDownloadPortable)
}

func (a *App) handleProtoAppUpdateCheck(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	info, err := a.CheckAppUpdate()
	if err != nil {
		return nil, protoBrowserOperationError("检查更新失败", err)
	}
	return protoipc.EncodeAppUpdateInfo(appUpdateInfoToProto(info)), nil
}

func (a *App) handleProtoAppUpdateDownload(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	input, err := protoipc.DecodeAppUpdateDownloadRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "AppUpdateDownloadRequest 解码失败",
			Details: err.Error(),
		}
	}
	result, err := a.DownloadAppUpdate(appUpdateInfoFromProto(input.Info), input.InstallOnRestart)
	if err != nil {
		return nil, protoBrowserOperationError("下载更新失败", err)
	}
	return protoipc.EncodeAppUpdateDownloadResult(appUpdateDownloadResultToProto(result)), nil
}

func (a *App) handleProtoAppUpdateInstallDownloaded(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	input, err := protoipc.DecodeAppUpdateInstallDownloadedRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "AppUpdateInstallDownloadedRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.InstallDownloadedAppUpdate(input.InstallerPath); err != nil {
		return nil, protoBrowserOperationError("安装更新失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoAppUpdateDownloadPortable(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	input, err := protoipc.DecodeAppUpdateInfoRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "AppUpdateInfoRequest 解码失败",
			Details: err.Error(),
		}
	}
	result, err := a.DownloadAndExtractPortableUpdate(appUpdateInfoFromProto(input.Info))
	if err != nil {
		return nil, protoBrowserOperationError("下载 ZIP 便携包失败", err)
	}
	return protoipc.EncodeAppUpdateDownloadResult(appUpdateDownloadResultToProto(result)), nil
}

func appUpdateInfoToProto(info *AppUpdateInfo) protoipc.AppUpdateInfo {
	if info == nil {
		return protoipc.AppUpdateInfo{}
	}
	return protoipc.AppUpdateInfo{
		CurrentVersion:         info.CurrentVersion,
		LatestVersion:          info.LatestVersion,
		ReleaseName:            info.ReleaseName,
		ReleaseURL:             info.ReleaseURL,
		PublishedAt:            info.PublishedAt,
		Body:                   info.Body,
		HasUpdate:              info.HasUpdate,
		Asset:                  appUpdateAssetToProto(info.Asset),
		InstallerAsset:         appUpdateAssetToProto(info.InstallerAsset),
		PortableAsset:          appUpdateAssetToProto(info.PortableAsset),
		DistributionKind:       info.DistributionKind,
		RecommendedPackageKind: info.RecommendedPackageKind,
		CanSelfUpdatePortable:  info.CanSelfUpdatePortable,
		Message:                info.Message,
	}
}

func appUpdateAssetToProto(asset *AppUpdateAsset) *protoipc.AppUpdateAsset {
	if asset == nil {
		return nil
	}
	return &protoipc.AppUpdateAsset{
		Name:        asset.Name,
		Size:        asset.Size,
		DownloadURL: asset.DownloadURL,
		Checksum:    asset.Checksum,
	}
}

func appUpdateInfoFromProto(info protoipc.AppUpdateInfo) AppUpdateInfo {
	return AppUpdateInfo{
		CurrentVersion:         info.CurrentVersion,
		LatestVersion:          info.LatestVersion,
		ReleaseName:            info.ReleaseName,
		ReleaseURL:             info.ReleaseURL,
		PublishedAt:            info.PublishedAt,
		Body:                   info.Body,
		HasUpdate:              info.HasUpdate,
		Asset:                  appUpdateAssetFromProto(info.Asset),
		InstallerAsset:         appUpdateAssetFromProto(info.InstallerAsset),
		PortableAsset:          appUpdateAssetFromProto(info.PortableAsset),
		DistributionKind:       info.DistributionKind,
		RecommendedPackageKind: info.RecommendedPackageKind,
		CanSelfUpdatePortable:  info.CanSelfUpdatePortable,
		Message:                info.Message,
	}
}

func appUpdateAssetFromProto(asset *protoipc.AppUpdateAsset) *AppUpdateAsset {
	if asset == nil {
		return nil
	}
	return &AppUpdateAsset{
		Name:        asset.Name,
		Size:        asset.Size,
		DownloadURL: asset.DownloadURL,
		Checksum:    asset.Checksum,
	}
}

func appUpdateDownloadResultToProto(result *AppUpdateDownloadResult) protoipc.AppUpdateDownloadResult {
	if result == nil {
		return protoipc.AppUpdateDownloadResult{}
	}
	return protoipc.AppUpdateDownloadResult{
		Cancelled:        result.Cancelled,
		Message:          result.Message,
		Version:          result.Version,
		InstallerPath:    result.InstallerPath,
		PackagePath:      result.PackagePath,
		ExtractedPath:    result.ExtractedPath,
		InstallOnRestart: result.InstallOnRestart,
		RestartScheduled: result.RestartScheduled,
		PackageKind:      result.PackageKind,
	}
}

func (a *App) emitAppUpdateDownloadProgressEvent(eventName string, progress AppUpdateDownloadProgress) {
	if a == nil || eventName != "app:update:download:progress" {
		return
	}
	a.emitProtoEvent(eventName, protoipc.EncodeAppUpdateDownloadProgress(protoipc.AppUpdateDownloadProgress{
		Phase:    progress.Phase,
		Progress: int32(progress.Progress),
		Message:  progress.Message,
	}))
}

func (a *App) emitAppUpdatePendingEvent(eventName string, payload any) {
	if a == nil {
		return
	}
	switch eventName {
	case "app:update:pending:notification":
		data, ok := payload.(map[string]interface{})
		if !ok {
			return
		}
		a.emitProtoEvent(eventName, protoipc.EncodeAppUpdatePendingNotification(protoipc.AppUpdatePendingNotification{
			Version: mapString(data, "version"),
			Message: mapString(data, "message"),
		}))
	case "app:update:pending:install-failed":
		data, ok := payload.(map[string]interface{})
		if !ok {
			return
		}
		a.emitProtoEvent(eventName, protoipc.EncodeAppUpdatePendingInstallFailed(protoipc.AppUpdatePendingInstallFailed{
			Version: mapString(data, "version"),
			Error:   mapString(data, "error"),
		}))
	case "app:update:pending":
		switch pending := payload.(type) {
		case pendingAppUpdate:
			a.emitProtoEvent(eventName, protoipc.EncodeAppUpdatePendingUpdate(appUpdatePendingToProto(pending)))
		case *pendingAppUpdate:
			if pending != nil {
				a.emitProtoEvent(eventName, protoipc.EncodeAppUpdatePendingUpdate(appUpdatePendingToProto(*pending)))
			}
		}
	}
}

func appUpdatePendingToProto(pending pendingAppUpdate) protoipc.AppUpdatePendingUpdate {
	return protoipc.AppUpdatePendingUpdate{
		Version:            pending.Version,
		InstallerPath:      pending.InstallerPath,
		ReleaseURL:         pending.ReleaseURL,
		InstallOnNextStart: pending.InstallOnNextStart,
		CreatedAt:          pending.CreatedAt,
	}
}
