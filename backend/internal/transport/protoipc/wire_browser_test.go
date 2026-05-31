package protoipc

import "testing"

func TestBrowserProfileListRoundTrip(t *testing.T) {
	request := BrowserProfileListRequest{Tag: "工作"}
	decodedRequest, err := DecodeBrowserProfileListRequest(EncodeBrowserProfileListRequest(request))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileListRequest failed: %v", err)
	}
	if decodedRequest.Tag != request.Tag {
		t.Fatalf("unexpected request tag: %q", decodedRequest.Tag)
	}

	response := BrowserProfileListResponse{
		Profiles: []BrowserProfile{
			{
				ProfileID:                    "profile-1",
				ProfileName:                  "测试实例",
				UserDataDir:                  "data/profile-1",
				CoreID:                       "chrome",
				FingerprintArgs:              []string{"--a=1", "--b=2"},
				ProxyID:                      "proxy-1",
				ProxyConfig:                  "socks5://127.0.0.1:1080",
				ProxyBindSourceID:            "source-1",
				ProxyBindSourceURL:           "https://example.test/proxies",
				ProxyBindName:                "线路 1",
				ProxyBindUpdatedAt:           "2026-05-24T00:00:00Z",
				AutoProxySwitchEnabled:       true,
				AutoProxySwitchGroupName:     "default",
				AutoProxySwitchMode:          "interval",
				AutoProxySwitchIntervalM:     15,
				AutoProxySwitchRotateByGroup: true,
				AutoProxySwitchLastProxyID:   "proxy-0",
				LaunchArgs:                   []string{"--disable-gpu"},
				Tags:                         []string{"工作", "默认"},
				Keywords:                     []string{"alpha", "beta"},
				GroupID:                      "group-1",
				LaunchCode:                   "ABCD12",
				Running:                      true,
				DebugPort:                    9222,
				DebugReady:                   true,
				PID:                          12345,
				RuntimeWarning:               "warning",
				LastError:                    "error",
				CreatedAt:                    "2026-05-24T00:00:00Z",
				UpdatedAt:                    "2026-05-24T00:01:00Z",
				LastStartAt:                  "2026-05-24T00:02:00Z",
				LastStopAt:                   "2026-05-24T00:03:00Z",
				InstanceMarkerIndex:          2,
				InstanceMarker:               "[Trace #02] 测试实例",
			},
		},
	}

	decodedResponse, err := DecodeBrowserProfileListResponse(EncodeBrowserProfileListResponse(response))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileListResponse failed: %v", err)
	}
	if len(decodedResponse.Profiles) != 1 {
		t.Fatalf("unexpected profile count: %d", len(decodedResponse.Profiles))
	}
	profile := decodedResponse.Profiles[0]
	if profile.ProfileID != "profile-1" {
		t.Fatalf("unexpected profile id: %s", profile.ProfileID)
	}
	if len(profile.Tags) != 2 || profile.Tags[0] != "工作" || profile.Tags[1] != "默认" {
		t.Fatalf("unexpected tags: %#v", profile.Tags)
	}
	if !profile.AutoProxySwitchEnabled || !profile.AutoProxySwitchRotateByGroup || !profile.Running || !profile.DebugReady {
		t.Fatalf("bool fields were not preserved: %#v", profile)
	}
	if profile.DebugPort != 9222 || profile.PID != 12345 {
		t.Fatalf("numeric fields were not preserved: %#v", profile)
	}
	if profile.InstanceMarkerIndex != 2 || profile.InstanceMarker != "[Trace #02] 测试实例" {
		t.Fatalf("instance marker fields were not preserved: %#v", profile)
	}
}

func TestBrowserProfileMutationRoundTrip(t *testing.T) {
	input := BrowserProfileInput{
		ProfileName:                  "新实例",
		UserDataDir:                  "data/new",
		CoreID:                       "chrome",
		FingerprintArgs:              []string{"--fingerprint=1"},
		ProxyID:                      "proxy-1",
		ProxyConfig:                  "http://127.0.0.1:8080",
		AutoProxySwitchEnabled:       true,
		AutoProxySwitchGroupName:     "默认",
		AutoProxySwitchMode:          "interval",
		AutoProxySwitchIntervalM:     10,
		AutoProxySwitchRotateByGroup: true,
		LaunchArgs:                   []string{"--start-maximized"},
		Tags:                         []string{"工作"},
		Keywords:                     []string{"alpha"},
		GroupID:                      "group-1",
	}

	createRequest, err := DecodeBrowserProfileCreateRequest(EncodeBrowserProfileCreateRequest(BrowserProfileCreateRequest{
		Profile: input,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileCreateRequest failed: %v", err)
	}
	if createRequest.Profile.ProfileName != input.ProfileName || !createRequest.Profile.AutoProxySwitchEnabled {
		t.Fatalf("create request was not preserved: %#v", createRequest)
	}

	updateRequest, err := DecodeBrowserProfileUpdateRequest(EncodeBrowserProfileUpdateRequest(BrowserProfileUpdateRequest{
		ProfileID: "profile-1",
		Profile:   input,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileUpdateRequest failed: %v", err)
	}
	if updateRequest.ProfileID != "profile-1" || updateRequest.Profile.AutoProxySwitchIntervalM != 10 || !updateRequest.Profile.AutoProxySwitchRotateByGroup {
		t.Fatalf("update request was not preserved: %#v", updateRequest)
	}

	deleteRequest, err := DecodeBrowserProfileDeleteRequest(EncodeBrowserProfileDeleteRequest(BrowserProfileDeleteRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileDeleteRequest failed: %v", err)
	}
	if deleteRequest.ProfileID != "profile-1" {
		t.Fatalf("delete request was not preserved: %#v", deleteRequest)
	}

	deleteResponse, err := DecodeBrowserProfileDeleteResponse(EncodeBrowserProfileDeleteResponse(BrowserProfileDeleteResponse{Deleted: true}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileDeleteResponse failed: %v", err)
	}
	if !deleteResponse.Deleted {
		t.Fatalf("delete response was not preserved: %#v", deleteResponse)
	}

	copyRequest, err := DecodeBrowserProfileCopyRequest(EncodeBrowserProfileCopyRequest(BrowserProfileCopyRequest{
		ProfileID: "profile-1",
		NewName:   "副本",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileCopyRequest failed: %v", err)
	}
	if copyRequest.ProfileID != "profile-1" || copyRequest.NewName != "副本" {
		t.Fatalf("copy request was not preserved: %#v", copyRequest)
	}

	response, err := DecodeBrowserProfileResponse(EncodeBrowserProfileResponse(BrowserProfileResponse{
		Profile: BrowserProfile{
			ProfileID:   "profile-2",
			ProfileName: "新实例",
			Tags:        []string{"工作"},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileResponse failed: %v", err)
	}
	if response.Profile.ProfileID != "profile-2" || len(response.Profile.Tags) != 1 {
		t.Fatalf("profile response was not preserved: %#v", response)
	}
}

func TestBrowserInstanceRequestRoundTrip(t *testing.T) {
	profileRequest, err := DecodeBrowserInstanceProfileRequest(EncodeBrowserInstanceProfileRequest(BrowserInstanceProfileRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserInstanceProfileRequest failed: %v", err)
	}
	if profileRequest.ProfileID != "profile-1" {
		t.Fatalf("instance profile request was not preserved: %#v", profileRequest)
	}

	codeRequest, err := DecodeBrowserInstanceStartByCodeRequest(EncodeBrowserInstanceStartByCodeRequest(BrowserInstanceStartByCodeRequest{
		Code: "ABC123",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserInstanceStartByCodeRequest failed: %v", err)
	}
	if codeRequest.Code != "ABC123" {
		t.Fatalf("instance start by code request was not preserved: %#v", codeRequest)
	}
}

func TestBrowserTagAndKeywordRoundTrip(t *testing.T) {
	tagList, err := DecodeBrowserTagListResponse(EncodeBrowserTagListResponse(BrowserTagListResponse{
		Tags: []string{"工作", "默认"},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserTagListResponse failed: %v", err)
	}
	if len(tagList.Tags) != 2 || tagList.Tags[0] != "工作" || tagList.Tags[1] != "默认" {
		t.Fatalf("tag list was not preserved: %#v", tagList)
	}

	keywordsRequest, err := DecodeBrowserProfileSetKeywordsRequest(EncodeBrowserProfileSetKeywordsRequest(BrowserProfileSetKeywordsRequest{
		ProfileID: "profile-1",
		Keywords:  []string{"alpha", "beta"},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileSetKeywordsRequest failed: %v", err)
	}
	if keywordsRequest.ProfileID != "profile-1" || len(keywordsRequest.Keywords) != 2 {
		t.Fatalf("keywords request was not preserved: %#v", keywordsRequest)
	}

	batchSetRequest, err := DecodeBrowserProfileBatchSetTagsRequest(EncodeBrowserProfileBatchSetTagsRequest(BrowserProfileBatchSetTagsRequest{
		ProfileIDs: []string{"profile-1", "profile-2"},
		Tags:       []string{"工作"},
		Replace:    true,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileBatchSetTagsRequest failed: %v", err)
	}
	if len(batchSetRequest.ProfileIDs) != 2 || !batchSetRequest.Replace {
		t.Fatalf("batch set tags request was not preserved: %#v", batchSetRequest)
	}

	batchRemoveRequest, err := DecodeBrowserProfileBatchRemoveTagsRequest(EncodeBrowserProfileBatchRemoveTagsRequest(BrowserProfileBatchRemoveTagsRequest{
		ProfileIDs: []string{"profile-1"},
		Tags:       []string{"默认"},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileBatchRemoveTagsRequest failed: %v", err)
	}
	if len(batchRemoveRequest.ProfileIDs) != 1 || len(batchRemoveRequest.Tags) != 1 {
		t.Fatalf("batch remove tags request was not preserved: %#v", batchRemoveRequest)
	}

	renameRequest, err := DecodeBrowserTagRenameRequest(EncodeBrowserTagRenameRequest(BrowserTagRenameRequest{
		OldName: "旧标签",
		NewName: "新标签",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserTagRenameRequest failed: %v", err)
	}
	if renameRequest.OldName != "旧标签" || renameRequest.NewName != "新标签" {
		t.Fatalf("tag rename request was not preserved: %#v", renameRequest)
	}

	actionResponse, err := DecodeBrowserActionResponse(EncodeBrowserActionResponse(BrowserActionResponse{OK: true}))
	if err != nil {
		t.Fatalf("DecodeBrowserActionResponse failed: %v", err)
	}
	if !actionResponse.OK {
		t.Fatalf("action response was not preserved: %#v", actionResponse)
	}
}

func TestBrowserGroupRoundTrip(t *testing.T) {
	input := BrowserGroupInput{
		GroupName: "工作组",
		ParentID:  "root",
		SortOrder: 20,
	}

	createRequest, err := DecodeBrowserGroupCreateRequest(EncodeBrowserGroupCreateRequest(BrowserGroupCreateRequest{Group: input}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupCreateRequest failed: %v", err)
	}
	if createRequest.Group.GroupName != input.GroupName || createRequest.Group.SortOrder != 20 {
		t.Fatalf("group create request was not preserved: %#v", createRequest)
	}

	updateRequest, err := DecodeBrowserGroupUpdateRequest(EncodeBrowserGroupUpdateRequest(BrowserGroupUpdateRequest{
		GroupID: "group-1",
		Group:   input,
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupUpdateRequest failed: %v", err)
	}
	if updateRequest.GroupID != "group-1" || updateRequest.Group.ParentID != "root" {
		t.Fatalf("group update request was not preserved: %#v", updateRequest)
	}

	deleteRequest, err := DecodeBrowserGroupDeleteRequest(EncodeBrowserGroupDeleteRequest(BrowserGroupDeleteRequest{GroupID: "group-1"}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupDeleteRequest failed: %v", err)
	}
	if deleteRequest.GroupID != "group-1" {
		t.Fatalf("group delete request was not preserved: %#v", deleteRequest)
	}

	moveRequest, err := DecodeBrowserGroupMoveProfilesRequest(EncodeBrowserGroupMoveProfilesRequest(BrowserGroupMoveProfilesRequest{
		ProfileIDs: []string{"profile-1", "profile-2"},
		GroupID:    "group-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupMoveProfilesRequest failed: %v", err)
	}
	if len(moveRequest.ProfileIDs) != 2 || moveRequest.GroupID != "group-1" {
		t.Fatalf("group move profiles request was not preserved: %#v", moveRequest)
	}

	listResponse, err := DecodeBrowserGroupListResponse(EncodeBrowserGroupListResponse(BrowserGroupListResponse{
		Groups: []BrowserGroup{
			{
				GroupID:       "group-1",
				GroupName:     "工作组",
				ParentID:      "root",
				SortOrder:     20,
				CreatedAt:     "2026-05-24T00:00:00Z",
				UpdatedAt:     "2026-05-24T00:01:00Z",
				InstanceCount: 3,
			},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupListResponse failed: %v", err)
	}
	if len(listResponse.Groups) != 1 || listResponse.Groups[0].InstanceCount != 3 {
		t.Fatalf("group list response was not preserved: %#v", listResponse)
	}

	groupResponse, err := DecodeBrowserGroupResponse(EncodeBrowserGroupResponse(BrowserGroupResponse{
		Group: BrowserGroup{
			GroupID:   "group-2",
			GroupName: "新工作组",
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserGroupResponse failed: %v", err)
	}
	if groupResponse.Group.GroupID != "group-2" {
		t.Fatalf("group response was not preserved: %#v", groupResponse)
	}
}

func TestBrowserTabRoundTrip(t *testing.T) {
	openRequest, err := DecodeBrowserInstanceOpenURLRequest(EncodeBrowserInstanceOpenURLRequest(BrowserInstanceOpenURLRequest{
		ProfileID: "profile-1",
		TargetURL: "https://example.test/",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserInstanceOpenURLRequest failed: %v", err)
	}
	if openRequest.ProfileID != "profile-1" || openRequest.TargetURL != "https://example.test/" {
		t.Fatalf("open url request was not preserved: %#v", openRequest)
	}

	response, err := DecodeBrowserTabListResponse(EncodeBrowserTabListResponse(BrowserTabListResponse{
		Tabs: []BrowserTab{
			{
				TabID:  "tab-1",
				Title:  "新标签页",
				URL:    "about:blank",
				Active: true,
			},
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserTabListResponse failed: %v", err)
	}
	if len(response.Tabs) != 1 || response.Tabs[0].TabID != "tab-1" || !response.Tabs[0].Active {
		t.Fatalf("tab list response was not preserved: %#v", response)
	}
}

func TestBrowserLaunchCodeRoundTrip(t *testing.T) {
	codeRequest, err := DecodeBrowserProfileCodeRequest(EncodeBrowserProfileCodeRequest(BrowserProfileCodeRequest{
		ProfileID: "profile-1",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileCodeRequest failed: %v", err)
	}
	if codeRequest.ProfileID != "profile-1" {
		t.Fatalf("profile code request was not preserved: %#v", codeRequest)
	}

	setCodeRequest, err := DecodeBrowserProfileSetCodeRequest(EncodeBrowserProfileSetCodeRequest(BrowserProfileSetCodeRequest{
		ProfileID: "profile-1",
		Code:      "ABC123",
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileSetCodeRequest failed: %v", err)
	}
	if setCodeRequest.ProfileID != "profile-1" || setCodeRequest.Code != "ABC123" {
		t.Fatalf("profile set code request was not preserved: %#v", setCodeRequest)
	}

	codeResponse, err := DecodeBrowserProfileCodeResponse(EncodeBrowserProfileCodeResponse(BrowserProfileCodeResponse{Code: "ABC123"}))
	if err != nil {
		t.Fatalf("DecodeBrowserProfileCodeResponse failed: %v", err)
	}
	if codeResponse.Code != "ABC123" {
		t.Fatalf("profile code response was not preserved: %#v", codeResponse)
	}
}

func TestBrowserLaunchServerInfoRoundTrip(t *testing.T) {
	response, err := DecodeBrowserLaunchServerInfoResponse(EncodeBrowserLaunchServerInfoResponse(BrowserLaunchServerInfoResponse{
		Host:            "127.0.0.1",
		Port:            19876,
		PreferredPort:   19876,
		BaseURL:         "http://127.0.0.1:19876",
		CDPURL:          "http://127.0.0.1:19876",
		ActiveDebugPort: 9222,
		Ready:           true,
		APIAuth: BrowserLaunchServerAPIAuth{
			Requested:  true,
			Configured: true,
			Enabled:    true,
			Header:     "X-Ant-Api-Key",
		},
	}))
	if err != nil {
		t.Fatalf("DecodeBrowserLaunchServerInfoResponse failed: %v", err)
	}
	if response.Port != 19876 || response.ActiveDebugPort != 9222 || !response.Ready {
		t.Fatalf("launch server info was not preserved: %#v", response)
	}
	if !response.APIAuth.Enabled || response.APIAuth.Header != "X-Ant-Api-Key" {
		t.Fatalf("launch server auth was not preserved: %#v", response.APIAuth)
	}
}
