package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"math"
	"strings"
)

func (a *App) handleProtoBrowserSnapshotList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserSnapshotProfileRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserSnapshotProfileRequest 解码失败",
			Details: err.Error(),
		}
	}
	snapshots, err := a.BrowserSnapshotList(strings.TrimSpace(input.ProfileID))
	if err != nil {
		return nil, protoBrowserOperationError("获取浏览器实例快照失败", err)
	}
	return protoipc.EncodeBrowserSnapshotListResponse(protoipc.BrowserSnapshotListResponse{
		Snapshots: browserSnapshotsToProto(snapshots),
	}), nil
}

func (a *App) handleProtoBrowserSnapshotCreate(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserSnapshotCreateRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserSnapshotCreateRequest 解码失败",
			Details: err.Error(),
		}
	}
	snapshot, err := a.BrowserSnapshotCreate(strings.TrimSpace(input.ProfileID), strings.TrimSpace(input.Name))
	if err != nil {
		return nil, protoBrowserOperationError("创建浏览器实例快照失败", err)
	}
	return protoipc.EncodeBrowserSnapshotResponse(protoipc.BrowserSnapshotResponse{
		Snapshot: browserSnapshotToProto(snapshot),
	}), nil
}

func (a *App) handleProtoBrowserSnapshotRestore(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserSnapshotActionRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserSnapshotActionRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.BrowserSnapshotRestore(strings.TrimSpace(input.ProfileID), strings.TrimSpace(input.SnapshotID)); err != nil {
		return nil, protoBrowserOperationError("恢复浏览器实例快照失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserSnapshotDelete(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserSnapshotActionRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserSnapshotActionRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.BrowserSnapshotDelete(strings.TrimSpace(input.ProfileID), strings.TrimSpace(input.SnapshotID)); err != nil {
		return nil, protoBrowserOperationError("删除浏览器实例快照失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func browserSnapshotsToProto(snapshots []SnapshotInfo) []protoipc.BrowserSnapshotInfo {
	out := make([]protoipc.BrowserSnapshotInfo, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, browserSnapshotToProto(snapshot))
	}
	return out
}

func browserSnapshotToProto(snapshot SnapshotInfo) protoipc.BrowserSnapshotInfo {
	return protoipc.BrowserSnapshotInfo{
		SnapshotID:  snapshot.SnapshotId,
		ProfileID:   snapshot.ProfileId,
		Name:        snapshot.Name,
		SizeMBMilli: int64(math.Round(snapshot.SizeMB * 1000)),
		CreatedAt:   snapshot.CreatedAt,
	}
}
