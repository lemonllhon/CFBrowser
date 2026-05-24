package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserBookmarkList           = "trace.browser.BookmarkList"
	MethodBrowserBookmarkSave           = "trace.browser.BookmarkSave"
	MethodBrowserBookmarkReset          = "trace.browser.BookmarkReset"
	MethodBrowserDefaultStartURLList    = "trace.browser.DefaultStartURLList"
	MethodBrowserDefaultStartURLSave    = "trace.browser.DefaultStartURLSave"
	MethodBrowserDefaultStartURLReset   = "trace.browser.DefaultStartURLReset"
	MethodBrowserDefaultContentRuleList = "trace.browser.DefaultContentRuleList"
	MethodBrowserDefaultContentRuleSave = "trace.browser.DefaultContentRuleSave"
)

type BrowserBookmark struct {
	Name string
	URL  string
}

type BrowserBookmarkListResponse struct {
	Items []BrowserBookmark
}

type BrowserBookmarkSaveRequest struct {
	Items []BrowserBookmark
}

type BrowserStartURL struct {
	Name string
	URL  string
}

type BrowserStartURLListResponse struct {
	Items []BrowserStartURL
}

type BrowserStartURLSaveRequest struct {
	Items []BrowserStartURL
}

type BrowserDefaultContentRule struct {
	RuleID                string
	Scope                 string
	TargetID              string
	TargetName            string
	StartURLs             []BrowserStartURL
	Bookmarks             []BrowserBookmark
	Enabled               bool
	ApplyToChilds         bool
	IncludeGlobalDefaults *bool
}

type BrowserDefaultContentRuleListResponse struct {
	Rules []BrowserDefaultContentRule
}

type BrowserDefaultContentRuleSaveRequest struct {
	Rules []BrowserDefaultContentRule
}

func EncodeBrowserBookmarkListResponse(message BrowserBookmarkListResponse) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserBookmark(item))
	}
	return out
}

func DecodeBrowserBookmarkListResponse(payload []byte) (BrowserBookmarkListResponse, error) {
	var result BrowserBookmarkListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserBookmark(data)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserBookmarkListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserBookmarkSaveRequest(message BrowserBookmarkSaveRequest) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserBookmark(item))
	}
	return out
}

func DecodeBrowserBookmarkSaveRequest(payload []byte) (BrowserBookmarkSaveRequest, error) {
	response, err := DecodeBrowserBookmarkListResponse(payload)
	if err != nil {
		return BrowserBookmarkSaveRequest{}, err
	}
	return BrowserBookmarkSaveRequest{Items: response.Items}, nil
}

func EncodeBrowserBookmark(message BrowserBookmark) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Name)
	out = appendStringField(out, 2, message.URL)
	return out
}

func DecodeBrowserBookmark(payload []byte) (BrowserBookmark, error) {
	var result BrowserBookmark
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserBookmark{}, fmt.Errorf("decode browser bookmark: %w", err)
	}
	return result, nil
}

func EncodeBrowserStartURLListResponse(message BrowserStartURLListResponse) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserStartURL(item))
	}
	return out
}

func DecodeBrowserStartURLListResponse(payload []byte) (BrowserStartURLListResponse, error) {
	var result BrowserStartURLListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserStartURL(data)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserStartURLListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserStartURLSaveRequest(message BrowserStartURLSaveRequest) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserStartURL(item))
	}
	return out
}

func DecodeBrowserStartURLSaveRequest(payload []byte) (BrowserStartURLSaveRequest, error) {
	response, err := DecodeBrowserStartURLListResponse(payload)
	if err != nil {
		return BrowserStartURLSaveRequest{}, err
	}
	return BrowserStartURLSaveRequest{Items: response.Items}, nil
}

func EncodeBrowserStartURL(message BrowserStartURL) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Name)
	out = appendStringField(out, 2, message.URL)
	return out
}

func DecodeBrowserStartURL(payload []byte) (BrowserStartURL, error) {
	var result BrowserStartURL
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserStartURL{}, fmt.Errorf("decode browser start url: %w", err)
	}
	return result, nil
}

func EncodeBrowserDefaultContentRuleListResponse(message BrowserDefaultContentRuleListResponse) []byte {
	var out []byte
	for _, rule := range message.Rules {
		out = appendBytesField(out, 1, EncodeBrowserDefaultContentRule(rule))
	}
	return out
}

func DecodeBrowserDefaultContentRuleListResponse(payload []byte) (BrowserDefaultContentRuleListResponse, error) {
	var result BrowserDefaultContentRuleListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			rule, err := DecodeBrowserDefaultContentRule(data)
			if err != nil {
				return err
			}
			result.Rules = append(result.Rules, rule)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserDefaultContentRuleListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserDefaultContentRuleSaveRequest(message BrowserDefaultContentRuleSaveRequest) []byte {
	var out []byte
	for _, rule := range message.Rules {
		out = appendBytesField(out, 1, EncodeBrowserDefaultContentRule(rule))
	}
	return out
}

func DecodeBrowserDefaultContentRuleSaveRequest(payload []byte) (BrowserDefaultContentRuleSaveRequest, error) {
	response, err := DecodeBrowserDefaultContentRuleListResponse(payload)
	if err != nil {
		return BrowserDefaultContentRuleSaveRequest{}, err
	}
	return BrowserDefaultContentRuleSaveRequest{Rules: response.Rules}, nil
}

func EncodeBrowserDefaultContentRule(message BrowserDefaultContentRule) []byte {
	var out []byte
	out = appendStringField(out, 1, message.RuleID)
	out = appendStringField(out, 2, message.Scope)
	out = appendStringField(out, 3, message.TargetID)
	out = appendStringField(out, 4, message.TargetName)
	for _, item := range message.StartURLs {
		out = appendBytesField(out, 5, EncodeBrowserStartURL(item))
	}
	for _, item := range message.Bookmarks {
		out = appendBytesField(out, 6, EncodeBrowserBookmark(item))
	}
	out = appendBoolField(out, 7, message.Enabled)
	out = appendBoolField(out, 8, message.ApplyToChilds)
	if message.IncludeGlobalDefaults != nil {
		out = appendBoolField(out, 9, *message.IncludeGlobalDefaults)
		out = appendBoolField(out, 10, true)
	}
	return out
}

func DecodeBrowserDefaultContentRule(payload []byte) (BrowserDefaultContentRule, error) {
	var result BrowserDefaultContentRule
	includeGlobalDefaults := false
	hasIncludeGlobalDefaults := false
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.RuleID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Scope = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.TargetID = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.TargetName = text
			return err
		case 5:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserStartURL(data)
			if err != nil {
				return err
			}
			result.StartURLs = append(result.StartURLs, item)
			return nil
		case 6:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserBookmark(data)
			if err != nil {
				return err
			}
			result.Bookmarks = append(result.Bookmarks, item)
			return nil
		case 7:
			value, err := consumeBoolValue(wireType, value)
			result.Enabled = value
			return err
		case 8:
			value, err := consumeBoolValue(wireType, value)
			result.ApplyToChilds = value
			return err
		case 9:
			value, err := consumeBoolValue(wireType, value)
			includeGlobalDefaults = value
			hasIncludeGlobalDefaults = true
			return err
		case 10:
			value, err := consumeBoolValue(wireType, value)
			hasIncludeGlobalDefaults = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserDefaultContentRule{}, fmt.Errorf("decode browser default content rule: %w", err)
	}
	if hasIncludeGlobalDefaults {
		result.IncludeGlobalDefaults = &includeGlobalDefaults
	}
	return result, nil
}
