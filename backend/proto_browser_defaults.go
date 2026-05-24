package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
)

func (a *App) handleProtoBrowserBookmarkList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeBrowserBookmarkListResponse(protoipc.BrowserBookmarkListResponse{
		Items: browserBookmarksToProto(a.BookmarkList()),
	}), nil
}

func (a *App) handleProtoBrowserBookmarkSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserBookmarkSaveRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserBookmarkSaveRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.BookmarkSave(browserBookmarksFromProto(input.Items)); err != nil {
		return nil, protoBrowserOperationError("保存默认书签失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserBookmarkReset(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	if err := a.BookmarkReset(); err != nil {
		return nil, protoBrowserOperationError("恢复默认书签失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserDefaultStartURLList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeBrowserStartURLListResponse(protoipc.BrowserStartURLListResponse{
		Items: browserStartURLsToProto(a.DefaultStartURLList()),
	}), nil
}

func (a *App) handleProtoBrowserDefaultStartURLSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserStartURLSaveRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserStartURLSaveRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.DefaultStartURLSave(browserStartURLsFromProto(input.Items)); err != nil {
		return nil, protoBrowserOperationError("保存默认打开页失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserDefaultStartURLReset(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	if err := a.DefaultStartURLReset(); err != nil {
		return nil, protoBrowserOperationError("恢复默认打开页失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserDefaultContentRuleList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeBrowserDefaultContentRuleListResponse(protoipc.BrowserDefaultContentRuleListResponse{
		Rules: browserDefaultContentRulesToProto(a.DefaultContentRuleList()),
	}), nil
}

func (a *App) handleProtoBrowserDefaultContentRuleSave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserDefaultContentRuleSaveRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserDefaultContentRuleSaveRequest 解码失败",
			Details: err.Error(),
		}
	}
	if err := a.DefaultContentRuleSave(browserDefaultContentRulesFromProto(input.Rules)); err != nil {
		return nil, protoBrowserOperationError("保存默认内容联动规则失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func browserBookmarksToProto(items []BrowserBookmark) []protoipc.BrowserBookmark {
	out := make([]protoipc.BrowserBookmark, 0, len(items))
	for _, item := range items {
		out = append(out, protoipc.BrowserBookmark{
			Name: item.Name,
			URL:  item.URL,
		})
	}
	return out
}

func browserBookmarksFromProto(items []protoipc.BrowserBookmark) []BrowserBookmark {
	out := make([]BrowserBookmark, 0, len(items))
	for _, item := range items {
		out = append(out, BrowserBookmark{
			Name: item.Name,
			URL:  item.URL,
		})
	}
	return out
}

func browserStartURLsToProto(items []BrowserStartURL) []protoipc.BrowserStartURL {
	out := make([]protoipc.BrowserStartURL, 0, len(items))
	for _, item := range items {
		out = append(out, protoipc.BrowserStartURL{
			Name: item.Name,
			URL:  item.URL,
		})
	}
	return out
}

func browserStartURLsFromProto(items []protoipc.BrowserStartURL) []BrowserStartURL {
	out := make([]BrowserStartURL, 0, len(items))
	for _, item := range items {
		out = append(out, BrowserStartURL{
			Name: item.Name,
			URL:  item.URL,
		})
	}
	return out
}

func browserDefaultContentRulesToProto(items []BrowserDefaultContentRule) []protoipc.BrowserDefaultContentRule {
	out := make([]protoipc.BrowserDefaultContentRule, 0, len(items))
	for _, item := range items {
		out = append(out, protoipc.BrowserDefaultContentRule{
			RuleID:                item.RuleId,
			Scope:                 item.Scope,
			TargetID:              item.TargetId,
			TargetName:            item.TargetName,
			StartURLs:             browserStartURLsToProto(item.StartURLs),
			Bookmarks:             browserBookmarksToProto(item.Bookmarks),
			Enabled:               item.Enabled,
			ApplyToChilds:         item.ApplyToChilds,
			IncludeGlobalDefaults: cloneBoolPointer(item.IncludeGlobalDefaults),
		})
	}
	return out
}

func browserDefaultContentRulesFromProto(items []protoipc.BrowserDefaultContentRule) []BrowserDefaultContentRule {
	out := make([]BrowserDefaultContentRule, 0, len(items))
	for _, item := range items {
		out = append(out, BrowserDefaultContentRule{
			RuleId:                item.RuleID,
			Scope:                 item.Scope,
			TargetId:              item.TargetID,
			TargetName:            item.TargetName,
			StartURLs:             browserStartURLsFromProto(item.StartURLs),
			Bookmarks:             browserBookmarksFromProto(item.Bookmarks),
			Enabled:               item.Enabled,
			ApplyToChilds:         item.ApplyToChilds,
			IncludeGlobalDefaults: cloneBoolPointer(item.IncludeGlobalDefaults),
		})
	}
	return out
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
