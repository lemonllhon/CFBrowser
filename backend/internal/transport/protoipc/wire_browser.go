package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserProfileList            = "trace.browser.ProfileList"
	MethodBrowserProfileCreate          = "trace.browser.ProfileCreate"
	MethodBrowserProfileUpdate          = "trace.browser.ProfileUpdate"
	MethodBrowserProfileDelete          = "trace.browser.ProfileDelete"
	MethodBrowserProfileCopy            = "trace.browser.ProfileCopy"
	MethodBrowserInstanceStart          = "trace.browser.InstanceStart"
	MethodBrowserInstanceStop           = "trace.browser.InstanceStop"
	MethodBrowserInstanceRestart        = "trace.browser.InstanceRestart"
	MethodBrowserInstanceStartByCode    = "trace.browser.InstanceStartByCode"
	MethodBrowserTagList                = "trace.browser.TagList"
	MethodBrowserProfileSetKeywords     = "trace.browser.ProfileSetKeywords"
	MethodBrowserProfileBatchSetTags    = "trace.browser.ProfileBatchSetTags"
	MethodBrowserProfileBatchRemoveTags = "trace.browser.ProfileBatchRemoveTags"
	MethodBrowserTagRename              = "trace.browser.TagRename"
	MethodBrowserGroupList              = "trace.browser.GroupList"
	MethodBrowserGroupCreate            = "trace.browser.GroupCreate"
	MethodBrowserGroupUpdate            = "trace.browser.GroupUpdate"
	MethodBrowserGroupDelete            = "trace.browser.GroupDelete"
	MethodBrowserGroupMoveProfiles      = "trace.browser.GroupMoveProfiles"
	MethodBrowserInstancePinCenter      = "trace.browser.InstancePinCenter"
	MethodBrowserProfileSwitchProxyNow  = "trace.browser.ProfileSwitchProxyNow"
	MethodBrowserInstanceOpenURL        = "trace.browser.InstanceOpenURL"
	MethodBrowserInstanceTabList        = "trace.browser.InstanceTabList"
	MethodBrowserProfileCodeGet         = "trace.browser.ProfileCodeGet"
	MethodBrowserProfileCodeRegenerate  = "trace.browser.ProfileCodeRegenerate"
	MethodBrowserProfileCodeSet         = "trace.browser.ProfileCodeSet"
	MethodBrowserLaunchServerInfo       = "trace.browser.LaunchServerInfo"
)

type BrowserProfile struct {
	ProfileID                    string
	ProfileName                  string
	UserDataDir                  string
	CoreID                       string
	FingerprintArgs              []string
	ProxyID                      string
	ProxyConfig                  string
	ProxyBindSourceID            string
	ProxyBindSourceURL           string
	ProxyBindName                string
	ProxyBindUpdatedAt           string
	AutoProxySwitchEnabled       bool
	AutoProxySwitchGroupName     string
	AutoProxySwitchMode          string
	AutoProxySwitchIntervalM     int32
	AutoProxySwitchRotateByGroup bool
	AutoProxySwitchLastProxyID   string
	LaunchArgs                   []string
	Tags                         []string
	Keywords                     []string
	GroupID                      string
	LaunchCode                   string
	Running                      bool
	DebugPort                    int32
	DebugReady                   bool
	PID                          int32
	RuntimeWarning               string
	LastError                    string
	CreatedAt                    string
	UpdatedAt                    string
	LastStartAt                  string
	LastStopAt                   string
	InstanceMarkerIndex          int32
	InstanceMarker               string
}

type BrowserProfileListRequest struct {
	Tag string
}

type BrowserProfileListResponse struct {
	Profiles []BrowserProfile
}

type BrowserProfileInput struct {
	ProfileName                  string
	UserDataDir                  string
	CoreID                       string
	FingerprintArgs              []string
	ProxyID                      string
	ProxyConfig                  string
	AutoProxySwitchEnabled       bool
	AutoProxySwitchGroupName     string
	AutoProxySwitchMode          string
	AutoProxySwitchIntervalM     int32
	AutoProxySwitchRotateByGroup bool
	LaunchArgs                   []string
	Tags                         []string
	Keywords                     []string
	GroupID                      string
}

type BrowserProfileCreateRequest struct {
	Profile BrowserProfileInput
}

type BrowserProfileUpdateRequest struct {
	ProfileID string
	Profile   BrowserProfileInput
}

type BrowserProfileDeleteRequest struct {
	ProfileID string
}

type BrowserProfileDeleteResponse struct {
	Deleted bool
}

type BrowserProfileCopyRequest struct {
	ProfileID string
	NewName   string
}

type BrowserProfileResponse struct {
	Profile BrowserProfile
}

type BrowserInstanceProfileRequest struct {
	ProfileID string
}

type BrowserInstanceStartByCodeRequest struct {
	Code string
}

type BrowserTagListResponse struct {
	Tags []string
}

type BrowserProfileSetKeywordsRequest struct {
	ProfileID string
	Keywords  []string
}

type BrowserProfileBatchSetTagsRequest struct {
	ProfileIDs []string
	Tags       []string
	Replace    bool
}

type BrowserProfileBatchRemoveTagsRequest struct {
	ProfileIDs []string
	Tags       []string
}

type BrowserTagRenameRequest struct {
	OldName string
	NewName string
}

type BrowserActionResponse struct {
	OK bool
}

type BrowserGroup struct {
	GroupID       string
	GroupName     string
	ParentID      string
	SortOrder     int32
	CreatedAt     string
	UpdatedAt     string
	InstanceCount int32
}

type BrowserGroupInput struct {
	GroupName string
	ParentID  string
	SortOrder int32
}

type BrowserGroupListResponse struct {
	Groups []BrowserGroup
}

type BrowserGroupCreateRequest struct {
	Group BrowserGroupInput
}

type BrowserGroupUpdateRequest struct {
	GroupID string
	Group   BrowserGroupInput
}

type BrowserGroupDeleteRequest struct {
	GroupID string
}

type BrowserGroupMoveProfilesRequest struct {
	ProfileIDs []string
	GroupID    string
}

type BrowserGroupResponse struct {
	Group BrowserGroup
}

type BrowserTab struct {
	TabID  string
	Title  string
	URL    string
	Active bool
}

type BrowserInstanceOpenURLRequest struct {
	ProfileID string
	TargetURL string
}

type BrowserTabListResponse struct {
	Tabs []BrowserTab
}

type BrowserProfileCodeRequest struct {
	ProfileID string
}

type BrowserProfileSetCodeRequest struct {
	ProfileID string
	Code      string
}

type BrowserProfileCodeResponse struct {
	Code string
}

type BrowserLaunchServerAPIAuth struct {
	Requested  bool
	Configured bool
	Enabled    bool
	Header     string
}

type BrowserLaunchServerInfoResponse struct {
	Host            string
	Port            int32
	PreferredPort   int32
	BaseURL         string
	CDPURL          string
	ActiveDebugPort int32
	Ready           bool
	APIAuth         BrowserLaunchServerAPIAuth
}

func EncodeBrowserProfileListRequest(message BrowserProfileListRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Tag)
	return out
}

func DecodeBrowserProfileListRequest(payload []byte) (BrowserProfileListRequest, error) {
	var result BrowserProfileListRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Tag = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileListRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileListResponse(message BrowserProfileListResponse) []byte {
	var out []byte
	for _, profile := range message.Profiles {
		out = appendBytesField(out, 1, EncodeBrowserProfile(profile))
	}
	return out
}

func DecodeBrowserProfileListResponse(payload []byte) (BrowserProfileListResponse, error) {
	var result BrowserProfileListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			profile, err := DecodeBrowserProfile(data)
			if err != nil {
				return err
			}
			result.Profiles = append(result.Profiles, profile)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProfileCreateRequest(message BrowserProfileCreateRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserProfileInput(message.Profile))
	return out
}

func DecodeBrowserProfileCreateRequest(payload []byte) (BrowserProfileCreateRequest, error) {
	var result BrowserProfileCreateRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			profile, err := DecodeBrowserProfileInput(data)
			result.Profile = profile
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileCreateRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileUpdateRequest(message BrowserProfileUpdateRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendBytesField(out, 2, EncodeBrowserProfileInput(message.Profile))
	return out
}

func DecodeBrowserProfileUpdateRequest(payload []byte) (BrowserProfileUpdateRequest, error) {
	var result BrowserProfileUpdateRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			profile, err := DecodeBrowserProfileInput(data)
			result.Profile = profile
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileUpdateRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileDeleteRequest(message BrowserProfileDeleteRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserProfileDeleteRequest(payload []byte) (BrowserProfileDeleteRequest, error) {
	var result BrowserProfileDeleteRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileDeleteRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileDeleteResponse(message BrowserProfileDeleteResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Deleted)
	return out
}

func DecodeBrowserProfileDeleteResponse(payload []byte) (BrowserProfileDeleteResponse, error) {
	var result BrowserProfileDeleteResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Deleted = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileDeleteResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProfileCopyRequest(message BrowserProfileCopyRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.NewName)
	return out
}

func DecodeBrowserProfileCopyRequest(payload []byte) (BrowserProfileCopyRequest, error) {
	var result BrowserProfileCopyRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.NewName = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileCopyRequest{}, err
	}
	return result, nil
}

func EncodeBrowserInstanceProfileRequest(message BrowserInstanceProfileRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserInstanceProfileRequest(payload []byte) (BrowserInstanceProfileRequest, error) {
	var result BrowserInstanceProfileRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserInstanceProfileRequest{}, err
	}
	return result, nil
}

func EncodeBrowserInstanceStartByCodeRequest(message BrowserInstanceStartByCodeRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Code)
	return out
}

func DecodeBrowserInstanceStartByCodeRequest(payload []byte) (BrowserInstanceStartByCodeRequest, error) {
	var result BrowserInstanceStartByCodeRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Code = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserInstanceStartByCodeRequest{}, err
	}
	return result, nil
}

func EncodeBrowserTagListResponse(message BrowserTagListResponse) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.Tags)
	return out
}

func DecodeBrowserTagListResponse(payload []byte) (BrowserTagListResponse, error) {
	var result BrowserTagListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Tags = append(result.Tags, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserTagListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProfileSetKeywordsRequest(message BrowserProfileSetKeywordsRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendRepeatedStringField(out, 2, message.Keywords)
	return out
}

func DecodeBrowserProfileSetKeywordsRequest(payload []byte) (BrowserProfileSetKeywordsRequest, error) {
	var result BrowserProfileSetKeywordsRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Keywords = append(result.Keywords, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileSetKeywordsRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileBatchSetTagsRequest(message BrowserProfileBatchSetTagsRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.ProfileIDs)
	out = appendRepeatedStringField(out, 2, message.Tags)
	out = appendBoolField(out, 3, message.Replace)
	return out
}

func DecodeBrowserProfileBatchSetTagsRequest(payload []byte) (BrowserProfileBatchSetTagsRequest, error) {
	var result BrowserProfileBatchSetTagsRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Tags = append(result.Tags, text)
			return err
		case 3:
			value, err := consumeBoolValue(wireType, value)
			result.Replace = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBatchSetTagsRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileBatchRemoveTagsRequest(message BrowserProfileBatchRemoveTagsRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.ProfileIDs)
	out = appendRepeatedStringField(out, 2, message.Tags)
	return out
}

func DecodeBrowserProfileBatchRemoveTagsRequest(payload []byte) (BrowserProfileBatchRemoveTagsRequest, error) {
	var result BrowserProfileBatchRemoveTagsRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Tags = append(result.Tags, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBatchRemoveTagsRequest{}, err
	}
	return result, nil
}

func EncodeBrowserTagRenameRequest(message BrowserTagRenameRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.OldName)
	out = appendStringField(out, 2, message.NewName)
	return out
}

func DecodeBrowserTagRenameRequest(payload []byte) (BrowserTagRenameRequest, error) {
	var result BrowserTagRenameRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.OldName = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.NewName = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserTagRenameRequest{}, err
	}
	return result, nil
}

func EncodeBrowserActionResponse(message BrowserActionResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.OK)
	return out
}

func DecodeBrowserActionResponse(payload []byte) (BrowserActionResponse, error) {
	var result BrowserActionResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.OK = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserActionResponse{}, err
	}
	return result, nil
}

func EncodeBrowserGroupListResponse(message BrowserGroupListResponse) []byte {
	var out []byte
	for _, group := range message.Groups {
		out = appendBytesField(out, 1, EncodeBrowserGroup(group))
	}
	return out
}

func DecodeBrowserGroupListResponse(payload []byte) (BrowserGroupListResponse, error) {
	var result BrowserGroupListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			group, err := DecodeBrowserGroup(data)
			if err != nil {
				return err
			}
			result.Groups = append(result.Groups, group)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserGroupCreateRequest(message BrowserGroupCreateRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserGroupInput(message.Group))
	return out
}

func DecodeBrowserGroupCreateRequest(payload []byte) (BrowserGroupCreateRequest, error) {
	var result BrowserGroupCreateRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			group, err := DecodeBrowserGroupInput(data)
			result.Group = group
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupCreateRequest{}, err
	}
	return result, nil
}

func EncodeBrowserGroupUpdateRequest(message BrowserGroupUpdateRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.GroupID)
	out = appendBytesField(out, 2, EncodeBrowserGroupInput(message.Group))
	return out
}

func DecodeBrowserGroupUpdateRequest(payload []byte) (BrowserGroupUpdateRequest, error) {
	var result BrowserGroupUpdateRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		case 2:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			group, err := DecodeBrowserGroupInput(data)
			result.Group = group
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupUpdateRequest{}, err
	}
	return result, nil
}

func EncodeBrowserGroupDeleteRequest(message BrowserGroupDeleteRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.GroupID)
	return out
}

func DecodeBrowserGroupDeleteRequest(payload []byte) (BrowserGroupDeleteRequest, error) {
	var result BrowserGroupDeleteRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupDeleteRequest{}, err
	}
	return result, nil
}

func EncodeBrowserGroupMoveProfilesRequest(message BrowserGroupMoveProfilesRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.ProfileIDs)
	out = appendStringField(out, 2, message.GroupID)
	return out
}

func DecodeBrowserGroupMoveProfilesRequest(payload []byte) (BrowserGroupMoveProfilesRequest, error) {
	var result BrowserGroupMoveProfilesRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupMoveProfilesRequest{}, err
	}
	return result, nil
}

func EncodeBrowserGroupResponse(message BrowserGroupResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserGroup(message.Group))
	return out
}

func DecodeBrowserGroupResponse(payload []byte) (BrowserGroupResponse, error) {
	var result BrowserGroupResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			group, err := DecodeBrowserGroup(data)
			result.Group = group
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupResponse{}, err
	}
	return result, nil
}

func EncodeBrowserGroupInput(message BrowserGroupInput) []byte {
	var out []byte
	out = appendStringField(out, 1, message.GroupName)
	out = appendStringField(out, 2, message.ParentID)
	out = appendInt32Field(out, 3, message.SortOrder)
	return out
}

func DecodeBrowserGroupInput(payload []byte) (BrowserGroupInput, error) {
	var result BrowserGroupInput
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.GroupName = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ParentID = text
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.SortOrder = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroupInput{}, fmt.Errorf("decode browser group input: %w", err)
	}
	return result, nil
}

func EncodeBrowserGroup(message BrowserGroup) []byte {
	var out []byte
	out = appendStringField(out, 1, message.GroupID)
	out = appendStringField(out, 2, message.GroupName)
	out = appendStringField(out, 3, message.ParentID)
	out = appendInt32Field(out, 4, message.SortOrder)
	out = appendStringField(out, 5, message.CreatedAt)
	out = appendStringField(out, 6, message.UpdatedAt)
	out = appendInt32Field(out, 7, message.InstanceCount)
	return out
}

func DecodeBrowserGroup(payload []byte) (BrowserGroup, error) {
	var result BrowserGroup
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.GroupName = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ParentID = text
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.SortOrder = int32(number)
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		case 7:
			number, err := consumeVarintValue(wireType, value)
			result.InstanceCount = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserGroup{}, fmt.Errorf("decode browser group: %w", err)
	}
	return result, nil
}

func EncodeBrowserInstanceOpenURLRequest(message BrowserInstanceOpenURLRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.TargetURL)
	return out
}

func DecodeBrowserInstanceOpenURLRequest(payload []byte) (BrowserInstanceOpenURLRequest, error) {
	var result BrowserInstanceOpenURLRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.TargetURL = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserInstanceOpenURLRequest{}, err
	}
	return result, nil
}

func EncodeBrowserTabListResponse(message BrowserTabListResponse) []byte {
	var out []byte
	for _, tab := range message.Tabs {
		out = appendBytesField(out, 1, EncodeBrowserTab(tab))
	}
	return out
}

func DecodeBrowserTabListResponse(payload []byte) (BrowserTabListResponse, error) {
	var result BrowserTabListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			tab, err := DecodeBrowserTab(data)
			if err != nil {
				return err
			}
			result.Tabs = append(result.Tabs, tab)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserTabListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserTab(message BrowserTab) []byte {
	var out []byte
	out = appendStringField(out, 1, message.TabID)
	out = appendStringField(out, 2, message.Title)
	out = appendStringField(out, 3, message.URL)
	out = appendBoolField(out, 4, message.Active)
	return out
}

func DecodeBrowserTab(payload []byte) (BrowserTab, error) {
	var result BrowserTab
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.TabID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Title = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		case 4:
			value, err := consumeBoolValue(wireType, value)
			result.Active = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserTab{}, fmt.Errorf("decode browser tab: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfileCodeRequest(message BrowserProfileCodeRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserProfileCodeRequest(payload []byte) (BrowserProfileCodeRequest, error) {
	var result BrowserProfileCodeRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileCodeRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileSetCodeRequest(message BrowserProfileSetCodeRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.Code)
	return out
}

func DecodeBrowserProfileSetCodeRequest(payload []byte) (BrowserProfileSetCodeRequest, error) {
	var result BrowserProfileSetCodeRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Code = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileSetCodeRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileCodeResponse(message BrowserProfileCodeResponse) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Code)
	return out
}

func DecodeBrowserProfileCodeResponse(payload []byte) (BrowserProfileCodeResponse, error) {
	var result BrowserProfileCodeResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Code = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileCodeResponse{}, err
	}
	return result, nil
}

func EncodeBrowserLaunchServerInfoResponse(message BrowserLaunchServerInfoResponse) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Host)
	out = appendInt32Field(out, 2, message.Port)
	out = appendInt32Field(out, 3, message.PreferredPort)
	out = appendStringField(out, 4, message.BaseURL)
	out = appendStringField(out, 5, message.CDPURL)
	out = appendInt32Field(out, 6, message.ActiveDebugPort)
	out = appendBoolField(out, 7, message.Ready)
	out = appendBytesField(out, 8, encodeBrowserLaunchServerAPIAuth(message.APIAuth))
	return out
}

func DecodeBrowserLaunchServerInfoResponse(payload []byte) (BrowserLaunchServerInfoResponse, error) {
	var result BrowserLaunchServerInfoResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Host = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Port = int32(number)
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.PreferredPort = int32(number)
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.BaseURL = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.CDPURL = text
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.ActiveDebugPort = int32(number)
			return err
		case 7:
			value, err := consumeBoolValue(wireType, value)
			result.Ready = value
			return err
		case 8:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			auth, err := decodeBrowserLaunchServerAPIAuth(data)
			result.APIAuth = auth
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserLaunchServerInfoResponse{}, err
	}
	return result, nil
}

func encodeBrowserLaunchServerAPIAuth(message BrowserLaunchServerAPIAuth) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Requested)
	out = appendBoolField(out, 2, message.Configured)
	out = appendBoolField(out, 3, message.Enabled)
	out = appendStringField(out, 4, message.Header)
	return out
}

func decodeBrowserLaunchServerAPIAuth(payload []byte) (BrowserLaunchServerAPIAuth, error) {
	var result BrowserLaunchServerAPIAuth
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Requested = value
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.Configured = value
			return err
		case 3:
			value, err := consumeBoolValue(wireType, value)
			result.Enabled = value
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.Header = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserLaunchServerAPIAuth{}, err
	}
	return result, nil
}

func EncodeBrowserProfileResponse(message BrowserProfileResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserProfile(message.Profile))
	return out
}

func DecodeBrowserProfileResponse(payload []byte) (BrowserProfileResponse, error) {
	var result BrowserProfileResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			profile, err := DecodeBrowserProfile(data)
			result.Profile = profile
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProfileInput(message BrowserProfileInput) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileName)
	out = appendStringField(out, 2, message.UserDataDir)
	out = appendStringField(out, 3, message.CoreID)
	out = appendRepeatedStringField(out, 4, message.FingerprintArgs)
	out = appendStringField(out, 5, message.ProxyID)
	out = appendStringField(out, 6, message.ProxyConfig)
	out = appendBoolField(out, 7, message.AutoProxySwitchEnabled)
	out = appendStringField(out, 8, message.AutoProxySwitchGroupName)
	out = appendStringField(out, 9, message.AutoProxySwitchMode)
	out = appendInt32Field(out, 10, message.AutoProxySwitchIntervalM)
	out = appendBoolField(out, 15, message.AutoProxySwitchRotateByGroup)
	out = appendRepeatedStringField(out, 11, message.LaunchArgs)
	out = appendRepeatedStringField(out, 12, message.Tags)
	out = appendRepeatedStringField(out, 13, message.Keywords)
	out = appendStringField(out, 14, message.GroupID)
	return out
}

func DecodeBrowserProfileInput(payload []byte) (BrowserProfileInput, error) {
	var result BrowserProfileInput
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileName = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.UserDataDir = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.CoreID = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.FingerprintArgs = append(result.FingerprintArgs, text)
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		case 7:
			value, err := consumeBoolValue(wireType, value)
			result.AutoProxySwitchEnabled = value
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.AutoProxySwitchGroupName = text
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.AutoProxySwitchMode = text
			return err
		case 10:
			number, err := consumeVarintValue(wireType, value)
			result.AutoProxySwitchIntervalM = int32(number)
			return err
		case 15:
			value, err := consumeBoolValue(wireType, value)
			result.AutoProxySwitchRotateByGroup = value
			return err
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.LaunchArgs = append(result.LaunchArgs, text)
			return err
		case 12:
			text, err := consumeStringValue(wireType, value)
			result.Tags = append(result.Tags, text)
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.Keywords = append(result.Keywords, text)
			return err
		case 14:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileInput{}, fmt.Errorf("decode browser profile input: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfile(message BrowserProfile) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.ProfileName)
	out = appendStringField(out, 3, message.UserDataDir)
	out = appendStringField(out, 4, message.CoreID)
	out = appendRepeatedStringField(out, 5, message.FingerprintArgs)
	out = appendStringField(out, 6, message.ProxyID)
	out = appendStringField(out, 7, message.ProxyConfig)
	out = appendStringField(out, 8, message.ProxyBindSourceID)
	out = appendStringField(out, 9, message.ProxyBindSourceURL)
	out = appendStringField(out, 10, message.ProxyBindName)
	out = appendStringField(out, 11, message.ProxyBindUpdatedAt)
	out = appendBoolField(out, 12, message.AutoProxySwitchEnabled)
	out = appendStringField(out, 13, message.AutoProxySwitchGroupName)
	out = appendStringField(out, 14, message.AutoProxySwitchMode)
	out = appendInt32Field(out, 15, message.AutoProxySwitchIntervalM)
	out = appendStringField(out, 16, message.AutoProxySwitchLastProxyID)
	out = appendBoolField(out, 32, message.AutoProxySwitchRotateByGroup)
	out = appendRepeatedStringField(out, 17, message.LaunchArgs)
	out = appendRepeatedStringField(out, 18, message.Tags)
	out = appendRepeatedStringField(out, 19, message.Keywords)
	out = appendStringField(out, 20, message.GroupID)
	out = appendStringField(out, 21, message.LaunchCode)
	out = appendBoolField(out, 22, message.Running)
	out = appendInt32Field(out, 23, message.DebugPort)
	out = appendBoolField(out, 24, message.DebugReady)
	out = appendInt32Field(out, 25, message.PID)
	out = appendStringField(out, 26, message.RuntimeWarning)
	out = appendStringField(out, 27, message.LastError)
	out = appendStringField(out, 28, message.CreatedAt)
	out = appendStringField(out, 29, message.UpdatedAt)
	out = appendStringField(out, 30, message.LastStartAt)
	out = appendStringField(out, 31, message.LastStopAt)
	out = appendInt32Field(out, 33, message.InstanceMarkerIndex)
	out = appendStringField(out, 34, message.InstanceMarker)
	return out
}

func DecodeBrowserProfile(payload []byte) (BrowserProfile, error) {
	var result BrowserProfile
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileName = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.UserDataDir = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.CoreID = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.FingerprintArgs = append(result.FingerprintArgs, text)
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.ProxyBindSourceID = text
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.ProxyBindSourceURL = text
			return err
		case 10:
			text, err := consumeStringValue(wireType, value)
			result.ProxyBindName = text
			return err
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.ProxyBindUpdatedAt = text
			return err
		case 12:
			value, err := consumeBoolValue(wireType, value)
			result.AutoProxySwitchEnabled = value
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.AutoProxySwitchGroupName = text
			return err
		case 14:
			text, err := consumeStringValue(wireType, value)
			result.AutoProxySwitchMode = text
			return err
		case 15:
			number, err := consumeVarintValue(wireType, value)
			result.AutoProxySwitchIntervalM = int32(number)
			return err
		case 16:
			text, err := consumeStringValue(wireType, value)
			result.AutoProxySwitchLastProxyID = text
			return err
		case 32:
			value, err := consumeBoolValue(wireType, value)
			result.AutoProxySwitchRotateByGroup = value
			return err
		case 17:
			text, err := consumeStringValue(wireType, value)
			result.LaunchArgs = append(result.LaunchArgs, text)
			return err
		case 18:
			text, err := consumeStringValue(wireType, value)
			result.Tags = append(result.Tags, text)
			return err
		case 19:
			text, err := consumeStringValue(wireType, value)
			result.Keywords = append(result.Keywords, text)
			return err
		case 20:
			text, err := consumeStringValue(wireType, value)
			result.GroupID = text
			return err
		case 21:
			text, err := consumeStringValue(wireType, value)
			result.LaunchCode = text
			return err
		case 22:
			value, err := consumeBoolValue(wireType, value)
			result.Running = value
			return err
		case 23:
			number, err := consumeVarintValue(wireType, value)
			result.DebugPort = int32(number)
			return err
		case 24:
			value, err := consumeBoolValue(wireType, value)
			result.DebugReady = value
			return err
		case 25:
			number, err := consumeVarintValue(wireType, value)
			result.PID = int32(number)
			return err
		case 26:
			text, err := consumeStringValue(wireType, value)
			result.RuntimeWarning = text
			return err
		case 27:
			text, err := consumeStringValue(wireType, value)
			result.LastError = text
			return err
		case 28:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 29:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		case 30:
			text, err := consumeStringValue(wireType, value)
			result.LastStartAt = text
			return err
		case 31:
			text, err := consumeStringValue(wireType, value)
			result.LastStopAt = text
			return err
		case 33:
			number, err := consumeVarintValue(wireType, value)
			result.InstanceMarkerIndex = int32(number)
			return err
		case 34:
			text, err := consumeStringValue(wireType, value)
			result.InstanceMarker = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfile{}, fmt.Errorf("decode browser profile: %w", err)
	}
	return result, nil
}

func appendRepeatedStringField(out []byte, field protowire.Number, values []string) []byte {
	for _, value := range values {
		out = appendStringField(out, field, value)
	}
	return out
}

func appendBoolField(out []byte, field protowire.Number, value bool) []byte {
	if !value {
		return out
	}
	out = protowire.AppendTag(out, field, protowire.VarintType)
	return protowire.AppendVarint(out, 1)
}

func consumeBoolValue(wireType protowire.Type, payload []byte) (bool, error) {
	value, err := consumeVarintValue(wireType, payload)
	if err != nil {
		return false, err
	}
	return value != 0, nil
}
