package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"strings"
)

func (a *App) handleProtoBrowserExtensionList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	extensions, err := a.BrowserExtensionList()
	if err != nil {
		return nil, protoBrowserOperationError("获取扩展插件列表失败", err)
	}
	return protoipc.EncodeBrowserExtensionListResponse(protoipc.BrowserExtensionListResponse{
		Extensions: browserExtensionsToProto(extensions),
	}), nil
}

func (a *App) handleProtoBrowserExtensionGet(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionIDRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionIDRequest 解码失败", err)
	}
	extension, err := a.BrowserExtensionGet(strings.TrimSpace(input.ExtensionID))
	if err != nil {
		return nil, protoBrowserOperationError("获取扩展插件详情失败", err)
	}
	return encodeProtoBrowserExtensionResponse(extension)
}

func (a *App) handleProtoBrowserExtensionDelete(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionIDRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionIDRequest 解码失败", err)
	}
	if err := a.BrowserExtensionDelete(strings.TrimSpace(input.ExtensionID)); err != nil {
		return nil, protoBrowserOperationError("删除扩展插件失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserExtensionChooseArchive(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := a.BrowserExtensionChooseArchive()
	if err != nil {
		return nil, protoBrowserOperationError("选择扩展压缩包失败", err)
	}
	return encodeProtoBrowserExtensionChoosePathResponse(result), nil
}

func (a *App) handleProtoBrowserExtensionChooseDirectory(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := a.BrowserExtensionChooseDirectory()
	if err != nil {
		return nil, protoBrowserOperationError("选择扩展目录失败", err)
	}
	return encodeProtoBrowserExtensionChoosePathResponse(result), nil
}

func (a *App) handleProtoBrowserExtensionImportArchive(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionImportRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionImportRequest 解码失败", err)
	}
	result, err := a.BrowserExtensionImportArchive(browserExtensionImportInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("导入扩展压缩包失败", err)
	}
	return protoipc.EncodeBrowserExtensionImportResult(browserExtensionImportResultToProto(result)), nil
}

func (a *App) handleProtoBrowserExtensionImportDirectory(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionImportRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionImportRequest 解码失败", err)
	}
	result, err := a.BrowserExtensionImportDirectory(browserExtensionImportInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("导入扩展目录失败", err)
	}
	return protoipc.EncodeBrowserExtensionImportResult(browserExtensionImportResultToProto(result)), nil
}

func (a *App) handleProtoBrowserExtensionListProfileBindings(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionIDRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionIDRequest 解码失败", err)
	}
	bindings, err := a.BrowserExtensionListProfileBindings(strings.TrimSpace(input.ExtensionID))
	if err != nil {
		return nil, protoBrowserOperationError("获取扩展绑定实例失败", err)
	}
	return protoipc.EncodeBrowserExtensionBindingListResponse(protoipc.BrowserExtensionBindingListResponse{
		Bindings: browserExtensionBindingsToProto(bindings),
	}), nil
}

func (a *App) handleProtoBrowserExtensionListForProfile(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionProfileRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionProfileRequest 解码失败", err)
	}
	bindings, err := a.BrowserExtensionListForProfile(strings.TrimSpace(input.ProfileID))
	if err != nil {
		return nil, protoBrowserOperationError("获取实例扩展绑定失败", err)
	}
	return protoipc.EncodeBrowserExtensionBindingListResponse(protoipc.BrowserExtensionBindingListResponse{
		Bindings: browserExtensionBindingsToProto(bindings),
	}), nil
}

func (a *App) handleProtoBrowserExtensionAssignProfiles(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionAssignRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionAssignRequest 解码失败", err)
	}
	bindings, err := a.BrowserExtensionAssignProfiles(browserExtensionAssignInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("保存扩展绑定失败", err)
	}
	return protoipc.EncodeBrowserExtensionBindingListResponse(protoipc.BrowserExtensionBindingListResponse{
		Bindings: browserExtensionBindingsToProto(bindings),
	}), nil
}

func (a *App) handleProtoBrowserExtensionSetAutoBind(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionAutoBindRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionAutoBindRequest 解码失败", err)
	}
	extension, err := a.BrowserExtensionSetAutoBind(browserExtensionAutoBindInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("保存扩展自动绑定失败", err)
	}
	return encodeProtoBrowserExtensionResponse(extension)
}

func (a *App) handleProtoBrowserExtensionUnassignProfiles(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionUnassignRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionUnassignRequest 解码失败", err)
	}
	bindings, err := a.BrowserExtensionUnassignProfiles(browserExtensionUnassignInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("移除扩展绑定失败", err)
	}
	return protoipc.EncodeBrowserExtensionBindingListResponse(protoipc.BrowserExtensionBindingListResponse{
		Bindings: browserExtensionBindingsToProto(bindings),
	}), nil
}

func (a *App) handleProtoBrowserExtensionSyncData(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserExtensionSyncDataRequest(request.Payload)
	if err != nil {
		return nil, protoDecodeError("BrowserExtensionSyncDataRequest 解码失败", err)
	}
	bindings, err := a.BrowserExtensionSyncProfileData(browserExtensionSyncDataInputFromProto(input))
	if err != nil {
		return nil, protoBrowserOperationError("同步扩展插件数据失败", err)
	}
	return protoipc.EncodeBrowserExtensionBindingListResponse(protoipc.BrowserExtensionBindingListResponse{
		Bindings: browserExtensionBindingsToProto(bindings),
	}), nil
}

func protoDecodeError(message string, err error) *protoipc.RPCError {
	return &protoipc.RPCError{
		Code:    protoipc.ErrorCodeInvalidPayload,
		Message: message,
		Details: err.Error(),
	}
}

func encodeProtoBrowserExtensionResponse(extension *BrowserExtension) ([]byte, *protoipc.RPCError) {
	if extension == nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInternal,
			Message: "扩展插件操作未返回数据",
		}
	}
	return protoipc.EncodeBrowserExtensionResponse(protoipc.BrowserExtensionResponse{
		Extension: browserExtensionToProto(*extension),
	}), nil
}

func encodeProtoBrowserExtensionChoosePathResponse(result map[string]interface{}) []byte {
	return protoipc.EncodeBrowserExtensionChoosePathResponse(protoipc.BrowserExtensionChoosePathResponse{
		Cancelled: boolFromInterface(result["cancelled"]),
		Path:      stringFromInterface(result["path"]),
	})
}

func browserExtensionImportInputFromProto(input protoipc.BrowserExtensionImportRequest) BrowserExtensionImportInput {
	return BrowserExtensionImportInput{
		Path:     input.Path,
		Mode:     input.Mode,
		Existing: input.Existing,
	}
}

func browserExtensionAssignInputFromProto(input protoipc.BrowserExtensionAssignRequest) BrowserExtensionAssignInput {
	return BrowserExtensionAssignInput{
		ExtensionId: input.ExtensionID,
		ProfileIds:  append([]string{}, input.ProfileIDs...),
		Mode:        input.Mode,
		Enabled:     input.Enabled,
	}
}

func browserExtensionAutoBindInputFromProto(input protoipc.BrowserExtensionAutoBindRequest) BrowserExtensionAutoBindInput {
	return BrowserExtensionAutoBindInput{
		ExtensionId: input.ExtensionID,
		Enabled:     input.Enabled,
		Mode:        input.Mode,
	}
}

func browserExtensionUnassignInputFromProto(input protoipc.BrowserExtensionUnassignRequest) BrowserExtensionUnassignInput {
	return BrowserExtensionUnassignInput{
		ExtensionId: input.ExtensionID,
		ProfileIds:  append([]string{}, input.ProfileIDs...),
	}
}

func browserExtensionSyncDataInputFromProto(input protoipc.BrowserExtensionSyncDataRequest) BrowserExtensionSyncDataInput {
	return BrowserExtensionSyncDataInput{
		ExtensionId:      input.ExtensionID,
		SourceProfileId:  input.SourceProfileID,
		TargetProfileIds: append([]string{}, input.TargetProfileIDs...),
	}
}

func browserExtensionImportResultToProto(result *BrowserExtensionImportResult) protoipc.BrowserExtensionImportResult {
	if result == nil {
		return protoipc.BrowserExtensionImportResult{}
	}
	var existing *protoipc.BrowserExtension
	if result.Existing != nil {
		value := browserExtensionToProto(*result.Existing)
		existing = &value
	}
	var extension *protoipc.BrowserExtension
	if result.Extension != nil {
		value := browserExtensionToProto(*result.Extension)
		extension = &value
	}
	return protoipc.BrowserExtensionImportResult{
		Cancelled: result.Cancelled,
		Duplicate: result.Duplicate,
		Message:   result.Message,
		Existing:  existing,
		Extension: extension,
	}
}

func browserExtensionsToProto(extensions []BrowserExtension) []protoipc.BrowserExtension {
	out := make([]protoipc.BrowserExtension, 0, len(extensions))
	for _, extension := range extensions {
		out = append(out, browserExtensionToProto(extension))
	}
	return out
}

func browserExtensionToProto(extension BrowserExtension) protoipc.BrowserExtension {
	return protoipc.BrowserExtension{
		ExtensionID:     extension.ExtensionId,
		Name:            extension.Name,
		Version:         extension.Version,
		ManifestVersion: int32(extension.ManifestVersion),
		Description:     extension.Description,
		SourceType:      extension.SourceType,
		SourceURL:       extension.SourceURL,
		InstallDir:      extension.InstallDir,
		PackagePath:     extension.PackagePath,
		ManifestJSON:    extension.ManifestJSON,
		BoundCount:      int32(extension.BoundCount),
		AutoBindEnabled: extension.AutoBindEnabled,
		AutoBindMode:    extension.AutoBindMode,
		CreatedAt:       extension.CreatedAt,
		UpdatedAt:       extension.UpdatedAt,
	}
}

func browserExtensionBindingsToProto(bindings []BrowserExtensionBinding) []protoipc.BrowserExtensionBinding {
	out := make([]protoipc.BrowserExtensionBinding, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, protoipc.BrowserExtensionBinding{
			ID:               binding.Id,
			ProfileID:        binding.ProfileId,
			ProfileName:      binding.ProfileName,
			ExtensionID:      binding.ExtensionId,
			ExtensionName:    binding.ExtensionName,
			ExtensionVersion: binding.ExtensionVersion,
			Mode:             binding.Mode,
			Enabled:          binding.Enabled,
			ExclusiveDir:     binding.ExclusiveDir,
			CreatedAt:        binding.CreatedAt,
			UpdatedAt:        binding.UpdatedAt,
		})
	}
	return out
}
