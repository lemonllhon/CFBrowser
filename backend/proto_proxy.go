package backend

import (
	"ant-chrome/backend/internal/transport/protoipc"
	"context"
	"encoding/json"
	"strings"
)

func registerProtoProxyHandlers(app *App, dispatcher *protoipc.Dispatcher) {
	if app == nil || dispatcher == nil {
		return
	}
	dispatcher.Register(protoipc.MethodBrowserProxyList, app.handleProtoBrowserProxyList)
	dispatcher.Register(protoipc.MethodBrowserProxyGroupList, app.handleProtoBrowserProxyGroupList)
	dispatcher.Register(protoipc.MethodBrowserProxySave, app.handleProtoBrowserProxySave)
	dispatcher.Register(protoipc.MethodBrowserProxyFetchClashByURL, app.handleProtoBrowserProxyFetchClashByURL)
	dispatcher.Register(protoipc.MethodBrowserProxyValidateConfig, app.handleProtoBrowserProxyValidateConfig)
	dispatcher.Register(protoipc.MethodBrowserProxyTestConnectivity, app.handleProtoBrowserProxyTestConnectivity)
	dispatcher.Register(protoipc.MethodBrowserProxyTestRealConnectivity, app.handleProtoBrowserProxyTestRealConnectivity)
	dispatcher.Register(protoipc.MethodBrowserProxyTestSpeed, app.handleProtoBrowserProxyTestSpeed)
	dispatcher.Register(protoipc.MethodBrowserProxyBatchTestSpeed, app.handleProtoBrowserProxyBatchTestSpeed)
	dispatcher.Register(protoipc.MethodBrowserProxyPreviewBatchTestSpeed, app.handleProtoBrowserProxyPreviewBatchTestSpeed)
	dispatcher.Register(protoipc.MethodBrowserProxyCheckIPHealth, app.handleProtoBrowserProxyCheckIPHealth)
	dispatcher.Register(protoipc.MethodBrowserProxyBatchCheckIPHealth, app.handleProtoBrowserProxyBatchCheckIPHealth)
	dispatcher.Register(protoipc.MethodBrowserProxyPreviewBatchCheckIPHealth, app.handleProtoBrowserProxyPreviewBatchCheckIPHealth)
}

func (a *App) handleProtoBrowserProxyList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyListRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyListRequest 解码失败",
			Details: err.Error(),
		}
	}

	groupName := strings.TrimSpace(input.GroupName)
	var proxies []BrowserProxy
	if groupName == "" {
		proxies = a.BrowserProxyList()
	} else {
		proxies = a.BrowserProxyListByGroup(groupName)
	}

	return protoipc.EncodeBrowserProxyListResponse(protoipc.BrowserProxyListResponse{
		Proxies: browserProxiesToProto(proxies),
	}), nil
}

func (a *App) handleProtoBrowserProxyGroupList(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	return protoipc.EncodeBrowserProxyGroupListResponse(protoipc.BrowserProxyGroupListResponse{
		Groups: a.BrowserProxyListGroups(),
	}), nil
}

func (a *App) handleProtoBrowserProxySave(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxySaveRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxySaveRequest 解码失败",
			Details: err.Error(),
		}
	}

	if err := a.SaveBrowserProxies(browserProxiesFromProto(input.Proxies)); err != nil {
		return nil, protoBrowserOperationError("保存浏览器代理失败", err)
	}
	return protoipc.EncodeBrowserActionResponse(protoipc.BrowserActionResponse{OK: true}), nil
}

func (a *App) handleProtoBrowserProxyFetchClashByURL(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyFetchClashByURLRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyFetchClashByURLRequest 解码失败",
			Details: err.Error(),
		}
	}

	result, err := a.BrowserProxyFetchClashByURL(input.URL)
	if err != nil {
		return nil, protoBrowserOperationError("拉取 Clash 订阅失败", err)
	}
	return protoipc.EncodeBrowserProxyFetchClashByURLResponse(protoipc.BrowserProxyFetchClashByURLResponse{
		URL:            mapString(result, "url"),
		Content:        mapString(result, "content"),
		ProxyCount:     int32(mapInt64(result, "proxyCount")),
		DNSServers:     mapString(result, "dnsServers"),
		SuggestedGroup: mapString(result, "suggestedGroup"),
	}), nil
}

func (a *App) handleProtoBrowserProxyValidateConfig(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyValidateConfigRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyValidateConfigRequest 解码失败",
			Details: err.Error(),
		}
	}

	result := a.ValidateProxyConfig(input.ProxyConfig, input.ProxyID)
	return protoipc.EncodeBrowserProxyValidateConfigResponse(protoipc.BrowserProxyValidateConfigResponse{
		Supported: result.Supported,
		ErrorMsg:  result.ErrorMsg,
	}), nil
}

func (a *App) handleProtoBrowserProxyTestConnectivity(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyTestResult(proxyTestResultToProto(a.TestProxyConnectivity(input.ProxyID, input.ProxyConfig))), nil
}

func (a *App) handleProtoBrowserProxyTestRealConnectivity(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyTestResult(proxyTestResultToProto(a.TestProxyRealConnectivity(input.ProxyID))), nil
}

func (a *App) handleProtoBrowserProxyTestSpeed(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyTestResult(proxyTestResultToProto(a.BrowserProxyTestSpeed(input.ProxyID))), nil
}

func (a *App) handleProtoBrowserProxyBatchTestSpeed(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyIDListRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyIDListRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyTestResultListResponse(protoipc.BrowserProxyTestResultListResponse{
		Results: proxyTestResultsToProto(a.BrowserProxyBatchTestSpeed(input.ProxyIDs, int(input.Concurrency))),
	}), nil
}

func (a *App) handleProtoBrowserProxyPreviewBatchTestSpeed(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyPreviewTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyPreviewTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyTestResultListResponse(protoipc.BrowserProxyTestResultListResponse{
		Results: proxyTestResultsToProto(a.BrowserProxyPreviewBatchTestSpeed(proxyPreviewTestInputsFromProto(input.Items), int(input.Concurrency))),
	}), nil
}

func (a *App) handleProtoBrowserProxyCheckIPHealth(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyIPHealthResult(proxyIPHealthResultToProto(a.BrowserProxyCheckIPHealth(input.ProxyID))), nil
}

func (a *App) handleProtoBrowserProxyBatchCheckIPHealth(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyIDListRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyIDListRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyIPHealthResultListResponse(protoipc.BrowserProxyIPHealthResultListResponse{
		Results: proxyIPHealthResultsToProto(a.BrowserProxyBatchCheckIPHealth(input.ProxyIDs, int(input.Concurrency))),
	}), nil
}

func (a *App) handleProtoBrowserProxyPreviewBatchCheckIPHealth(ctx context.Context, request protoipc.Envelope) ([]byte, *protoipc.RPCError) {
	if rpcErr := a.ensureProtoBrowserReady(ctx); rpcErr != nil {
		return nil, rpcErr
	}
	input, err := protoipc.DecodeBrowserProxyPreviewTestRequest(request.Payload)
	if err != nil {
		return nil, &protoipc.RPCError{
			Code:    protoipc.ErrorCodeInvalidPayload,
			Message: "BrowserProxyPreviewTestRequest 解码失败",
			Details: err.Error(),
		}
	}

	return protoipc.EncodeBrowserProxyIPHealthResultListResponse(protoipc.BrowserProxyIPHealthResultListResponse{
		Results: proxyIPHealthResultsToProto(a.BrowserProxyPreviewBatchCheckIPHealth(proxyPreviewTestInputsFromProto(input.Items), int(input.Concurrency))),
	}), nil
}

func browserProxiesToProto(proxies []BrowserProxy) []protoipc.BrowserProxy {
	out := make([]protoipc.BrowserProxy, 0, len(proxies))
	for _, item := range proxies {
		out = append(out, browserProxyToProto(item))
	}
	return out
}

func proxyPreviewTestInputsFromProto(items []protoipc.BrowserProxyPreviewTestInput) []ProxyPreviewTestInput {
	out := make([]ProxyPreviewTestInput, 0, len(items))
	for _, item := range items {
		out = append(out, ProxyPreviewTestInput{
			ProxyId:     item.ProxyID,
			ProxyConfig: item.ProxyConfig,
		})
	}
	return out
}

func proxyTestResultsToProto(results []ProxyTestResult) []protoipc.BrowserProxyTestResult {
	out := make([]protoipc.BrowserProxyTestResult, 0, len(results))
	for _, item := range results {
		out = append(out, proxyTestResultToProto(item))
	}
	return out
}

func proxyTestResultToProto(item ProxyTestResult) protoipc.BrowserProxyTestResult {
	return protoipc.BrowserProxyTestResult{
		ProxyID:   item.ProxyId,
		OK:        item.Ok,
		LatencyMS: item.LatencyMs,
		Error:     item.Error,
	}
}

func proxyIPHealthResultsToProto(results []ProxyIPHealthResult) []protoipc.BrowserProxyIPHealthResult {
	out := make([]protoipc.BrowserProxyIPHealthResult, 0, len(results))
	for _, item := range results {
		out = append(out, proxyIPHealthResultToProto(item))
	}
	return out
}

func proxyIPHealthResultToProto(item ProxyIPHealthResult) protoipc.BrowserProxyIPHealthResult {
	return protoipc.BrowserProxyIPHealthResult{
		ProxyID:        item.ProxyId,
		OK:             item.Ok,
		Source:         item.Source,
		Error:          item.Error,
		IP:             item.IP,
		FraudScore:     item.FraudScore,
		IsResidential:  item.IsResidential,
		IsBroadcast:    item.IsBroadcast,
		Country:        item.Country,
		Region:         item.Region,
		City:           item.City,
		AsOrganization: item.AsOrganization,
		RawDataJSON:    rawDataToJSON(item.RawData),
		UpdatedAt:      item.UpdatedAt,
	}
}

func rawDataToJSON(rawData map[string]interface{}) string {
	if rawData == nil {
		return "{}"
	}
	payload, err := json.Marshal(rawData)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func (a *App) emitProxyTestResultEvent(eventName string, result ProxyTestResult) {
	if a == nil {
		return
	}
	a.emitProtoEvent(eventName, protoipc.EncodeBrowserProxyTestResult(proxyTestResultToProto(result)))
}

func (a *App) emitProxyIPHealthResultEvent(eventName string, result ProxyIPHealthResult) {
	if a == nil {
		return
	}
	a.emitProtoEvent(eventName, protoipc.EncodeBrowserProxyIPHealthResult(proxyIPHealthResultToProto(result)))
}

func browserProxyToProto(item BrowserProxy) protoipc.BrowserProxy {
	return protoipc.BrowserProxy{
		ProxyID:                item.ProxyId,
		ProxyName:              item.ProxyName,
		ProxyConfig:            item.ProxyConfig,
		DNSServers:             item.DnsServers,
		GroupName:              item.GroupName,
		SortOrder:              int32(item.SortOrder),
		SourceID:               item.SourceID,
		SourceURL:              item.SourceURL,
		SourceNamePrefix:       item.SourceNamePrefix,
		SourceFilterJSON:       item.SourceFilterJSON,
		SourceAutoRefresh:      item.SourceAutoRefresh,
		SourceRefreshIntervalM: int32(item.SourceRefreshIntervalM),
		SourceLastRefreshAt:    item.SourceLastRefreshAt,
		LastLatencyMS:          item.LastLatencyMs,
		LastTestOK:             item.LastTestOk,
		LastTestedAt:           item.LastTestedAt,
		LastIPHealthJSON:       item.LastIPHealthJSON,
	}
}

func browserProxiesFromProto(proxies []protoipc.BrowserProxy) []BrowserProxy {
	out := make([]BrowserProxy, 0, len(proxies))
	for _, item := range proxies {
		out = append(out, BrowserProxy{
			ProxyId:                item.ProxyID,
			ProxyName:              item.ProxyName,
			ProxyConfig:            item.ProxyConfig,
			DnsServers:             item.DNSServers,
			GroupName:              item.GroupName,
			SortOrder:              int(item.SortOrder),
			SourceID:               item.SourceID,
			SourceURL:              item.SourceURL,
			SourceNamePrefix:       item.SourceNamePrefix,
			SourceFilterJSON:       item.SourceFilterJSON,
			SourceAutoRefresh:      item.SourceAutoRefresh,
			SourceRefreshIntervalM: int(item.SourceRefreshIntervalM),
			SourceLastRefreshAt:    item.SourceLastRefreshAt,
			LastLatencyMs:          item.LastLatencyMS,
			LastTestOk:             item.LastTestOK,
			LastTestedAt:           item.LastTestedAt,
			LastIPHealthJSON:       item.LastIPHealthJSON,
		})
	}
	return out
}
