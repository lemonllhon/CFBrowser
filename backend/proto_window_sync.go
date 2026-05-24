package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
)

func registerProtoWindowSyncHandlers(app *App, dispatcher *protoipc.Dispatcher) {
	if app == nil || dispatcher == nil {
		return
	}
	dispatcher.Register(protoipc.MethodWindowSyncCandidateList, app.handleProtoWindowSyncCandidateList)
	dispatcher.Register(protoipc.MethodWindowSyncStart, app.handleProtoWindowSyncStart)
	dispatcher.Register(protoipc.MethodWindowSyncStateGet, app.handleProtoWindowSyncStateGet)
	dispatcher.Register(protoipc.MethodWindowSyncStop, app.handleProtoWindowSyncStop)
	dispatcher.Register(protoipc.MethodWindowSyncPause, app.handleProtoWindowSyncPause)
	dispatcher.Register(protoipc.MethodWindowSyncResume, app.handleProtoWindowSyncResume)
	dispatcher.Register(protoipc.MethodWindowSyncShowAll, app.handleProtoWindowSyncShowAll)
	dispatcher.Register(protoipc.MethodWindowSyncSettingsGet, app.handleProtoWindowSyncSettingsGet)
	dispatcher.Register(protoipc.MethodWindowSyncSettingsSave, app.handleProtoWindowSyncSettingsSave)
	dispatcher.Register(protoipc.MethodWindowSyncLayoutSettingsGet, app.handleProtoWindowSyncLayoutSettingsGet)
	dispatcher.Register(protoipc.MethodWindowSyncLayoutSettingsSave, app.handleProtoWindowSyncLayoutSettingsSave)
	dispatcher.Register(protoipc.MethodWindowSyncLayoutApply, app.handleProtoWindowSyncLayoutApply)
	dispatcher.Register(protoipc.MethodWindowSyncBatchInputSame, app.handleProtoWindowSyncBatchInputSame)
	dispatcher.Register(protoipc.MethodWindowSyncBatchInputDifferent, app.handleProtoWindowSyncBatchInputDifferent)
	dispatcher.Register(protoipc.MethodWindowSyncCloseOtherTabs, app.handleProtoWindowSyncCloseOtherTabs)
	dispatcher.Register(protoipc.MethodWindowSyncCloseCurrentTab, app.handleProtoWindowSyncCloseCurrentTab)
	dispatcher.Register(protoipc.MethodWindowSyncCloseBlankTabs, app.handleProtoWindowSyncCloseBlankTabs)
	dispatcher.Register(protoipc.MethodWindowSyncOpenUrls, app.handleProtoWindowSyncOpenUrls)
	dispatcher.Register(protoipc.MethodWindowSyncToolbarResize, app.handleProtoWindowSyncToolbarResize)
}

func (a *App) handleProtoWindowSyncCandidateList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeWindowSyncCandidateListResponse(protoipc.WindowSyncCandidateListResponse{
		Candidates: windowSyncCandidatesToProto(a.WindowSyncListCandidates()),
	}), nil
}

func (a *App) handleProtoWindowSyncStart(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncStartRequest(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncStartRequest 解码失败", err)
	}
	state, err := a.WindowSyncStart(WindowSyncStartInput{
		ProfileIds:      append([]string{}, input.ProfileIDs...),
		MasterProfileId: input.MasterProfileID,
	})
	if err != nil {
		return nil, protoWindowSyncOperationError("启动窗口同步失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncStateGet(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	return encodeProtoWindowSyncStateResponse(a.WindowSyncGetState()), nil
}

func (a *App) handleProtoWindowSyncStop(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	state, err := a.WindowSyncStop()
	if err != nil {
		return nil, protoWindowSyncOperationError("停止窗口同步失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncPause(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	state, err := a.WindowSyncPause()
	if err != nil {
		return nil, protoWindowSyncOperationError("暂停窗口同步失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncResume(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	state, err := a.WindowSyncResume()
	if err != nil {
		return nil, protoWindowSyncOperationError("恢复窗口同步失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncShowAll(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	state, err := a.WindowSyncShowAll()
	if err != nil {
		return nil, protoWindowSyncOperationError("展示同步窗口失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncSettingsGet(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeWindowSyncSettingsResponse(protoipc.WindowSyncSettingsResponse{
		Settings: windowSyncSettingsToProto(a.WindowSyncGetSettings()),
	}), nil
}

func (a *App) handleProtoWindowSyncSettingsSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncSettings(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncSettings 解码失败", err)
	}
	state, err := a.WindowSyncSaveSettings(windowSyncSettingsFromProto(input))
	if err != nil {
		return nil, protoWindowSyncOperationError("保存窗口同步设置失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncLayoutSettingsGet(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeWindowSyncLayoutSettingsResponse(protoipc.WindowSyncLayoutSettingsResponse{
		Layout: windowSyncLayoutSettingsToProto(a.WindowSyncGetLayoutSettings()),
	}), nil
}

func (a *App) handleProtoWindowSyncLayoutSettingsSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncLayoutSettings(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncLayoutSettings 解码失败", err)
	}
	layout, err := a.WindowSyncSaveLayoutSettings(windowSyncLayoutSettingsFromProto(input))
	if err != nil {
		return nil, protoWindowSyncOperationError("保存窗口同步布局失败", err)
	}
	return protoipc.EncodeWindowSyncLayoutSettingsResponse(protoipc.WindowSyncLayoutSettingsResponse{
		Layout: windowSyncLayoutSettingsToProtoValue(layout),
	}), nil
}

func (a *App) handleProtoWindowSyncLayoutApply(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncLayoutSettings(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncLayoutSettings 解码失败", err)
	}
	state, err := a.WindowSyncApplyLayout(windowSyncLayoutSettingsFromProto(input))
	if err != nil {
		return nil, protoWindowSyncOperationError("应用窗口同步布局失败", err)
	}
	return encodeProtoWindowSyncStateResponse(state), nil
}

func (a *App) handleProtoWindowSyncBatchInputSame(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncBatchInputSameRequest(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncBatchInputSameRequest 解码失败", err)
	}
	result, err := a.WindowSyncBatchInputSame(WindowSyncBatchInputSameInput{Text: input.Text})
	if err != nil {
		return nil, protoWindowSyncOperationError("窗口同步批量输入失败", err)
	}
	return protoipc.EncodeWindowSyncBatchInputResult(windowSyncBatchInputResultToProtoValue(result)), nil
}

func (a *App) handleProtoWindowSyncBatchInputDifferent(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncBatchInputDifferentRequest(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncBatchInputDifferentRequest 解码失败", err)
	}
	result, err := a.WindowSyncBatchInputDifferent(WindowSyncBatchInputDifferentInput{Items: windowSyncBatchInputDifferentItemsFromProto(input.Items)})
	if err != nil {
		return nil, protoWindowSyncOperationError("窗口同步差异批量输入失败", err)
	}
	return protoipc.EncodeWindowSyncBatchInputResult(windowSyncBatchInputResultToProtoValue(result)), nil
}

func (a *App) handleProtoWindowSyncCloseOtherTabs(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	return a.handleProtoWindowSyncAction(ctx, a.WindowSyncCloseOtherTabs, "关闭其他标签页失败")
}

func (a *App) handleProtoWindowSyncCloseCurrentTab(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	return a.handleProtoWindowSyncAction(ctx, a.WindowSyncCloseCurrentTab, "关闭当前标签页失败")
}

func (a *App) handleProtoWindowSyncCloseBlankTabs(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	return a.handleProtoWindowSyncAction(ctx, a.WindowSyncCloseBlankTabs, "关闭空白标签页失败")
}

func (a *App) handleProtoWindowSyncOpenUrls(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncOpenUrlsRequest(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncOpenUrlsRequest 解码失败", err)
	}
	result, err := a.WindowSyncOpenUrls(WindowSyncOpenUrlsInput{Urls: input.URLs})
	if err != nil {
		return nil, protoWindowSyncOperationError("同步打开网站失败", err)
	}
	return protoipc.EncodeWindowSyncActionResult(windowSyncActionResultToProtoValue(result)), nil
}

func (a *App) handleProtoWindowSyncToolbarResize(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeWindowSyncToolbarResizeRequest(request.Payload)
	if err != nil {
		return nil, protoWindowSyncInvalidPayload("WindowSyncToolbarResizeRequest 解码失败", err)
	}
	if err := a.WindowSyncToolbarSetSize(int(input.Width), int(input.Height)); err != nil {
		return nil, protoWindowSyncOperationError("调整窗口同步工具栏尺寸失败", err)
	}
	return protoipc.EncodeWindowSyncToolbarResizeResponse(protoipc.WindowSyncToolbarResizeResponse{OK: true}), nil
}

func (a *App) handleProtoWindowSyncAction(ctx context.Context, action func() (*WindowSyncActionResult, error), errorMessage string) ([]byte, *protoipc.RPCError) {
	if rpcErr := ensureProtoWindowSyncReady(ctx, a); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := action()
	if err != nil {
		return nil, protoWindowSyncOperationError(errorMessage, err)
	}
	return protoipc.EncodeWindowSyncActionResult(windowSyncActionResultToProtoValue(result)), nil
}

func ensureProtoWindowSyncReady(ctx context.Context, app *App) *protoipc.RPCError {
	if err := ctx.Err(); err != nil {
		return &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInternal,
			Message: "请求上下文已取消",
			Details: err.Error(),
		}
	}
	if app == nil {
		return &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInternal,
			Message: "窗口同步服务尚未初始化",
		}
	}
	return nil
}

func protoWindowSyncInvalidPayload(message string, err error) *protoipc.RPCError {
	return &protoipc.RPCError{
		Code:    protoipc.ErrorCodeInvalidPayload,
		Message: message,
		Details: err.Error(),
	}
}

func protoWindowSyncOperationError(message string, err error) *protoipc.RPCError {
	return &protoipc.RPCError{
		Code:    protoipc.ErrorCodeBadRequest,
		Message: message,
		Details: err.Error(),
	}
}

func encodeProtoWindowSyncStateResponse(state *WindowSyncState) []byte {
	return protoipc.EncodeWindowSyncStateResponse(protoipc.WindowSyncStateResponse{
		State: windowSyncStateToProtoPointer(state),
	})
}

func windowSyncCandidatesToProto(items []WindowSyncCandidate) []protoipc.WindowSyncCandidate {
	out := make([]protoipc.WindowSyncCandidate, 0, len(items))
	for _, item := range items {
		out = append(out, windowSyncCandidateToProto(item))
	}
	return out
}

func windowSyncCandidateToProto(item WindowSyncCandidate) protoipc.WindowSyncCandidate {
	return protoipc.WindowSyncCandidate{
		ProfileID:    item.ProfileId,
		ProfileName:  item.ProfileName,
		DebugPort:    int32(item.DebugPort),
		PID:          int32(item.Pid),
		Running:      item.Running,
		DebugReady:   item.DebugReady,
		Role:         item.Role,
		Master:       item.Master,
		CanSync:      item.CanSync,
		CanAutoStart: item.CanAutoStart,
		Unavailable:  item.Unavailable,
	}
}

func windowSyncCandidateFromProto(item protoipc.WindowSyncCandidate) WindowSyncCandidate {
	return WindowSyncCandidate{
		ProfileId:    item.ProfileID,
		ProfileName:  item.ProfileName,
		DebugPort:    int(item.DebugPort),
		Pid:          int(item.PID),
		Running:      item.Running,
		DebugReady:   item.DebugReady,
		Role:         item.Role,
		Master:       item.Master,
		CanSync:      item.CanSync,
		CanAutoStart: item.CanAutoStart,
		Unavailable:  item.Unavailable,
	}
}

func windowSyncLayoutSettingsToProto(input WindowSyncLayoutSettings) protoipc.WindowSyncLayoutSettings {
	return protoipc.WindowSyncLayoutSettings{
		Mode:      input.Mode,
		Width:     int32(input.Width),
		Height:    int32(input.Height),
		GapX:      int32(input.GapX),
		GapY:      int32(input.GapY),
		PerRow:    int32(input.PerRow),
		UpdatedAt: input.UpdatedAt,
	}
}

func windowSyncLayoutSettingsToProtoValue(input *WindowSyncLayoutSettings) protoipc.WindowSyncLayoutSettings {
	if input == nil {
		return protoipc.WindowSyncLayoutSettings{}
	}
	return windowSyncLayoutSettingsToProto(*input)
}

func windowSyncLayoutSettingsFromProto(input protoipc.WindowSyncLayoutSettings) WindowSyncLayoutSettings {
	return WindowSyncLayoutSettings{
		Mode:      input.Mode,
		Width:     int(input.Width),
		Height:    int(input.Height),
		GapX:      int(input.GapX),
		GapY:      int(input.GapY),
		PerRow:    int(input.PerRow),
		UpdatedAt: input.UpdatedAt,
	}
}

func windowSyncSettingsToProto(input WindowSyncSettings) protoipc.WindowSyncSettings {
	return protoipc.WindowSyncSettings{
		MasterColor:  input.MasterColor,
		SyncKeyboard: input.SyncKeyboard,
		SyncMouse:    input.SyncMouse,
	}
}

func windowSyncSettingsFromProto(input protoipc.WindowSyncSettings) WindowSyncSettings {
	return WindowSyncSettings{
		MasterColor:  input.MasterColor,
		SyncKeyboard: input.SyncKeyboard,
		SyncMouse:    input.SyncMouse,
	}
}

func windowSyncStateToProtoPointer(state *WindowSyncState) *protoipc.WindowSyncState {
	if state == nil {
		return nil
	}
	out := windowSyncStateToProto(*state)
	return &out
}

func windowSyncStateToProto(state WindowSyncState) protoipc.WindowSyncState {
	return protoipc.WindowSyncState{
		SessionID:       state.SessionId,
		Active:          state.Active,
		Paused:          state.Paused,
		MasterProfileID: state.MasterProfileId,
		ProfileIDs:      append([]string{}, state.ProfileIds...),
		Windows:         windowSyncCandidatesToProto(state.Windows),
		MasterColor:     state.MasterColor,
		SyncKeyboard:    state.SyncKeyboard,
		SyncMouse:       state.SyncMouse,
		Layout:          windowSyncLayoutSettingsToProto(state.Layout),
		StartedAt:       state.StartedAt,
		UpdatedAt:       state.UpdatedAt,
	}
}

func windowSyncBatchInputDifferentItemsFromProto(items []protoipc.WindowSyncBatchInputDifferentItem) []WindowSyncBatchInputDifferentItem {
	out := make([]WindowSyncBatchInputDifferentItem, 0, len(items))
	for _, item := range items {
		out = append(out, WindowSyncBatchInputDifferentItem{
			ProfileId: item.ProfileID,
			Text:      item.Text,
		})
	}
	return out
}

func windowSyncBatchInputResultToProtoValue(input *WindowSyncBatchInputResult) protoipc.WindowSyncBatchInputResult {
	if input == nil {
		return protoipc.WindowSyncBatchInputResult{}
	}
	out := protoipc.WindowSyncBatchInputResult{
		Total:   int32(input.Total),
		Success: int32(input.Success),
		Failed:  int32(input.Failed),
		Results: make([]protoipc.WindowSyncBatchInputResultItem, 0, len(input.Results)),
	}
	for _, item := range input.Results {
		out.Results = append(out.Results, protoipc.WindowSyncBatchInputResultItem{
			ProfileID:   item.ProfileId,
			ProfileName: item.ProfileName,
			Master:      item.Master,
			Success:     item.Success,
			Error:       item.Error,
		})
	}
	return out
}

func windowSyncActionResultToProtoValue(input *WindowSyncActionResult) protoipc.WindowSyncActionResult {
	if input == nil {
		return protoipc.WindowSyncActionResult{}
	}
	out := protoipc.WindowSyncActionResult{
		Total:   int32(input.Total),
		Success: int32(input.Success),
		Failed:  int32(input.Failed),
		Results: make([]protoipc.WindowSyncActionResultItem, 0, len(input.Results)),
	}
	for _, item := range input.Results {
		out.Results = append(out.Results, protoipc.WindowSyncActionResultItem{
			ProfileID:   item.ProfileId,
			ProfileName: item.ProfileName,
			Master:      item.Master,
			Success:     item.Success,
			Error:       item.Error,
		})
	}
	return out
}
