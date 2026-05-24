package protoipc

import "testing"

func TestBrowserBookmarkRoundTrip(t *testing.T) {
	request := BrowserBookmarkSaveRequest{
		Items: []BrowserBookmark{
			{Name: "Google", URL: "https://www.google.com/"},
			{Name: "ChatGPT", URL: "https://chatgpt.com/"},
		},
	}
	decodedRequest, err := DecodeBrowserBookmarkSaveRequest(EncodeBrowserBookmarkSaveRequest(request))
	if err != nil {
		t.Fatalf("DecodeBrowserBookmarkSaveRequest failed: %v", err)
	}
	if len(decodedRequest.Items) != 2 || decodedRequest.Items[1].Name != "ChatGPT" {
		t.Fatalf("bookmark save request was not preserved: %#v", decodedRequest)
	}

	response := BrowserBookmarkListResponse{Items: request.Items}
	decodedResponse, err := DecodeBrowserBookmarkListResponse(EncodeBrowserBookmarkListResponse(response))
	if err != nil {
		t.Fatalf("DecodeBrowserBookmarkListResponse failed: %v", err)
	}
	if len(decodedResponse.Items) != 2 || decodedResponse.Items[0].URL != "https://www.google.com/" {
		t.Fatalf("bookmark list response was not preserved: %#v", decodedResponse)
	}
}

func TestBrowserStartURLRoundTrip(t *testing.T) {
	request := BrowserStartURLSaveRequest{
		Items: []BrowserStartURL{
			{Name: "IPPure", URL: "https://ippure.com/"},
			{Name: "Ping0", URL: "https://ping0.cc/"},
		},
	}
	decodedRequest, err := DecodeBrowserStartURLSaveRequest(EncodeBrowserStartURLSaveRequest(request))
	if err != nil {
		t.Fatalf("DecodeBrowserStartURLSaveRequest failed: %v", err)
	}
	if len(decodedRequest.Items) != 2 || decodedRequest.Items[1].Name != "Ping0" {
		t.Fatalf("start url save request was not preserved: %#v", decodedRequest)
	}

	response := BrowserStartURLListResponse{Items: request.Items}
	decodedResponse, err := DecodeBrowserStartURLListResponse(EncodeBrowserStartURLListResponse(response))
	if err != nil {
		t.Fatalf("DecodeBrowserStartURLListResponse failed: %v", err)
	}
	if len(decodedResponse.Items) != 2 || decodedResponse.Items[0].URL != "https://ippure.com/" {
		t.Fatalf("start url list response was not preserved: %#v", decodedResponse)
	}
}

func TestBrowserDefaultContentRuleRoundTrip(t *testing.T) {
	includeGlobalDefaults := false
	request := BrowserDefaultContentRuleSaveRequest{
		Rules: []BrowserDefaultContentRule{
			{
				RuleID:                "tag:work",
				Scope:                 "tag",
				TargetName:            "work",
				StartURLs:             []BrowserStartURL{{Name: "Docs", URL: "https://docs.example.test/"}},
				Bookmarks:             []BrowserBookmark{{Name: "CRM", URL: "https://crm.example.test/"}},
				Enabled:               true,
				ApplyToChilds:         true,
				IncludeGlobalDefaults: &includeGlobalDefaults,
			},
		},
	}

	decodedRequest, err := DecodeBrowserDefaultContentRuleSaveRequest(EncodeBrowserDefaultContentRuleSaveRequest(request))
	if err != nil {
		t.Fatalf("DecodeBrowserDefaultContentRuleSaveRequest failed: %v", err)
	}
	if len(decodedRequest.Rules) != 1 {
		t.Fatalf("unexpected rule count: %#v", decodedRequest)
	}
	rule := decodedRequest.Rules[0]
	if rule.RuleID != "tag:work" || !rule.Enabled || !rule.ApplyToChilds {
		t.Fatalf("rule scalar fields were not preserved: %#v", rule)
	}
	if len(rule.StartURLs) != 1 || rule.StartURLs[0].Name != "Docs" {
		t.Fatalf("rule start urls were not preserved: %#v", rule.StartURLs)
	}
	if len(rule.Bookmarks) != 1 || rule.Bookmarks[0].URL != "https://crm.example.test/" {
		t.Fatalf("rule bookmarks were not preserved: %#v", rule.Bookmarks)
	}
	if rule.IncludeGlobalDefaults == nil || *rule.IncludeGlobalDefaults {
		t.Fatalf("false include_global_defaults presence was not preserved: %#v", rule.IncludeGlobalDefaults)
	}

	response := BrowserDefaultContentRuleListResponse{Rules: request.Rules}
	decodedResponse, err := DecodeBrowserDefaultContentRuleListResponse(EncodeBrowserDefaultContentRuleListResponse(response))
	if err != nil {
		t.Fatalf("DecodeBrowserDefaultContentRuleListResponse failed: %v", err)
	}
	if len(decodedResponse.Rules) != 1 || decodedResponse.Rules[0].RuleID != "tag:work" {
		t.Fatalf("default content rule list response was not preserved: %#v", decodedResponse)
	}
}
