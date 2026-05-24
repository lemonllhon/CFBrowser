package protoipc

import "testing"

func TestBrowserProxyRoundTrip(t *testing.T) {
	proxyItem := BrowserProxy{
		ProxyID:                "proxy-1",
		ProxyName:              "香港 01",
		ProxyConfig:            "socks5://127.0.0.1:1080",
		DNSServers:             "1.1.1.1",
		GroupName:              "HK",
		SortOrder:              10,
		SourceID:               "source-1",
		SourceURL:              "https://example.test/sub",
		SourceNamePrefix:       "HK",
		SourceFilterJSON:       `{"keyword":"hk"}`,
		SourceAutoRefresh:      true,
		SourceRefreshIntervalM: 60,
		SourceLastRefreshAt:    "2026-05-24T00:00:00Z",
		LastLatencyMS:          123,
		LastTestOK:             true,
		LastTestedAt:           "2026-05-24T00:01:00Z",
		LastIPHealthJSON:       `{"ip":"127.0.0.1"}`,
	}

	decodedProxy, err := DecodeBrowserProxy(EncodeBrowserProxy(proxyItem))
	if err != nil {
		t.Fatalf("DecodeBrowserProxy failed: %v", err)
	}
	if decodedProxy.ProxyID != proxyItem.ProxyID || decodedProxy.ProxyConfig != proxyItem.ProxyConfig {
		t.Fatalf("proxy was not preserved: %#v", decodedProxy)
	}
	if !decodedProxy.SourceAutoRefresh || decodedProxy.SourceRefreshIntervalM != 60 {
		t.Fatalf("proxy source refresh fields were not preserved: %#v", decodedProxy)
	}
	if !decodedProxy.LastTestOK || decodedProxy.LastLatencyMS != 123 {
		t.Fatalf("proxy runtime fields were not preserved: %#v", decodedProxy)
	}

	listRequest, err := DecodeBrowserProxyListRequest(EncodeBrowserProxyListRequest(BrowserProxyListRequest{GroupName: "HK"}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyListRequest failed: %v", err)
	}
	if listRequest.GroupName != "HK" {
		t.Fatalf("list request was not preserved: %#v", listRequest)
	}

	listResponse, err := DecodeBrowserProxyListResponse(EncodeBrowserProxyListResponse(BrowserProxyListResponse{
		Proxies: []BrowserProxy{proxyItem},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyListResponse failed: %v", err)
	}
	if len(listResponse.Proxies) != 1 || listResponse.Proxies[0].ProxyID != proxyItem.ProxyID {
		t.Fatalf("list response was not preserved: %#v", listResponse)
	}

	groupResponse, err := DecodeBrowserProxyGroupListResponse(EncodeBrowserProxyGroupListResponse(BrowserProxyGroupListResponse{
		Groups: []string{"HK", "SG"},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyGroupListResponse failed: %v", err)
	}
	if len(groupResponse.Groups) != 2 || groupResponse.Groups[0] != "HK" {
		t.Fatalf("group response was not preserved: %#v", groupResponse)
	}

	saveRequest, err := DecodeBrowserProxySaveRequest(EncodeBrowserProxySaveRequest(BrowserProxySaveRequest{
		Proxies: []BrowserProxy{proxyItem},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxySaveRequest failed: %v", err)
	}
	if len(saveRequest.Proxies) != 1 || saveRequest.Proxies[0].ProxyName != proxyItem.ProxyName {
		t.Fatalf("save request was not preserved: %#v", saveRequest)
	}
}

func TestBrowserProxyOperationRoundTrip(t *testing.T) {
	clashRequest, err := DecodeBrowserProxyFetchClashByURLRequest(EncodeBrowserProxyFetchClashByURLRequest(BrowserProxyFetchClashByURLRequest{
		URL: "https://example.test/sub",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyFetchClashByURLRequest failed: %v", err)
	}
	if clashRequest.URL != "https://example.test/sub" {
		t.Fatalf("clash request was not preserved: %#v", clashRequest)
	}

	clashResponse, err := DecodeBrowserProxyFetchClashByURLResponse(EncodeBrowserProxyFetchClashByURLResponse(BrowserProxyFetchClashByURLResponse{
		URL:            "https://example.test/sub",
		Content:        "proxies: []",
		ProxyCount:     3,
		DNSServers:     "1.1.1.1",
		SuggestedGroup: "example",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyFetchClashByURLResponse failed: %v", err)
	}
	if clashResponse.ProxyCount != 3 || clashResponse.SuggestedGroup != "example" {
		t.Fatalf("clash response was not preserved: %#v", clashResponse)
	}

	validationRequest, err := DecodeBrowserProxyValidateConfigRequest(EncodeBrowserProxyValidateConfigRequest(BrowserProxyValidateConfigRequest{
		ProxyConfig: "socks5://127.0.0.1:1080",
		ProxyID:     "proxy-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyValidateConfigRequest failed: %v", err)
	}
	if validationRequest.ProxyID != "proxy-1" || validationRequest.ProxyConfig == "" {
		t.Fatalf("validation request was not preserved: %#v", validationRequest)
	}

	validationResponse, err := DecodeBrowserProxyValidateConfigResponse(EncodeBrowserProxyValidateConfigResponse(BrowserProxyValidateConfigResponse{
		Supported: true,
		ErrorMsg:  "",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyValidateConfigResponse failed: %v", err)
	}
	if !validationResponse.Supported {
		t.Fatalf("validation response was not preserved: %#v", validationResponse)
	}

	testRequest, err := DecodeBrowserProxyTestRequest(EncodeBrowserProxyTestRequest(BrowserProxyTestRequest{
		ProxyID:     "proxy-1",
		ProxyConfig: "socks5://127.0.0.1:1080",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyTestRequest failed: %v", err)
	}
	if testRequest.ProxyID != "proxy-1" || testRequest.ProxyConfig == "" {
		t.Fatalf("test request was not preserved: %#v", testRequest)
	}

	idListRequest, err := DecodeBrowserProxyIDListRequest(EncodeBrowserProxyIDListRequest(BrowserProxyIDListRequest{
		ProxyIDs:    []string{"proxy-1", "proxy-2"},
		Concurrency: 8,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyIDListRequest failed: %v", err)
	}
	if len(idListRequest.ProxyIDs) != 2 || idListRequest.Concurrency != 8 {
		t.Fatalf("id list request was not preserved: %#v", idListRequest)
	}

	previewRequest, err := DecodeBrowserProxyPreviewTestRequest(EncodeBrowserProxyPreviewTestRequest(BrowserProxyPreviewTestRequest{
		Items: []BrowserProxyPreviewTestInput{{
			ProxyID:     "preview-1",
			ProxyConfig: "socks5://127.0.0.1:1081",
		}},
		Concurrency: 4,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyPreviewTestRequest failed: %v", err)
	}
	if len(previewRequest.Items) != 1 || previewRequest.Items[0].ProxyID != "preview-1" || previewRequest.Concurrency != 4 {
		t.Fatalf("preview request was not preserved: %#v", previewRequest)
	}
}

func TestBrowserProxyResultRoundTrip(t *testing.T) {
	testResult := BrowserProxyTestResult{
		ProxyID:   "proxy-1",
		OK:        true,
		LatencyMS: 123,
		Error:     "",
	}
	decodedTestResult, err := DecodeBrowserProxyTestResult(EncodeBrowserProxyTestResult(testResult))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyTestResult failed: %v", err)
	}
	if decodedTestResult.ProxyID != testResult.ProxyID || !decodedTestResult.OK || decodedTestResult.LatencyMS != 123 {
		t.Fatalf("test result was not preserved: %#v", decodedTestResult)
	}

	testList, err := DecodeBrowserProxyTestResultListResponse(EncodeBrowserProxyTestResultListResponse(BrowserProxyTestResultListResponse{
		Results: []BrowserProxyTestResult{testResult},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyTestResultListResponse failed: %v", err)
	}
	if len(testList.Results) != 1 || testList.Results[0].ProxyID != testResult.ProxyID {
		t.Fatalf("test result list was not preserved: %#v", testList)
	}

	healthResult := BrowserProxyIPHealthResult{
		ProxyID:        "proxy-1",
		OK:             true,
		Source:         "ippure",
		IP:             "127.0.0.1",
		FraudScore:     8,
		IsResidential:  true,
		IsBroadcast:    false,
		Country:        "CN",
		Region:         "GD",
		City:           "Shenzhen",
		AsOrganization: "Example ISP",
		RawDataJSON:    `{"ip":"127.0.0.1"}`,
		UpdatedAt:      "2026-05-24T00:00:00Z",
	}
	decodedHealthResult, err := DecodeBrowserProxyIPHealthResult(EncodeBrowserProxyIPHealthResult(healthResult))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyIPHealthResult failed: %v", err)
	}
	if decodedHealthResult.ProxyID != healthResult.ProxyID || !decodedHealthResult.OK || decodedHealthResult.RawDataJSON == "" {
		t.Fatalf("health result was not preserved: %#v", decodedHealthResult)
	}

	healthList, err := DecodeBrowserProxyIPHealthResultListResponse(EncodeBrowserProxyIPHealthResultListResponse(BrowserProxyIPHealthResultListResponse{
		Results: []BrowserProxyIPHealthResult{healthResult},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProxyIPHealthResultListResponse failed: %v", err)
	}
	if len(healthList.Results) != 1 || healthList.Results[0].City != "Shenzhen" {
		t.Fatalf("health result list was not preserved: %#v", healthList)
	}
}
