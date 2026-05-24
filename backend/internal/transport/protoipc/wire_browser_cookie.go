package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserCookieList   = "trace.browser.CookieList"
	MethodBrowserCookieClear  = "trace.browser.CookieClear"
	MethodBrowserCookieExport = "trace.browser.CookieExport"
	MethodBrowserCookieImport = "trace.browser.CookieImport"
)

type BrowserCookieInfo struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  int64
	HTTPOnly bool
	Secure   bool
	SameSite string
}

type BrowserCookieProfileRequest struct {
	ProfileID string
}

type BrowserCookieImportRequest struct {
	ProfileID string
	Content   string
}

type BrowserCookieListResponse struct {
	Cookies []BrowserCookieInfo
}

type BrowserCookieExportResponse struct {
	Content string
}

type BrowserCookieImportResult struct {
	Imported int32
	Skipped  int32
}

func EncodeBrowserCookieProfileRequest(message BrowserCookieProfileRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserCookieProfileRequest(payload []byte) (BrowserCookieProfileRequest, error) {
	var result BrowserCookieProfileRequest
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
		return BrowserCookieProfileRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCookieImportRequest(message BrowserCookieImportRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.Content)
	return out
}

func DecodeBrowserCookieImportRequest(payload []byte) (BrowserCookieImportRequest, error) {
	var result BrowserCookieImportRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Content = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCookieImportRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCookieListResponse(message BrowserCookieListResponse) []byte {
	var out []byte
	for _, cookie := range message.Cookies {
		out = appendBytesField(out, 1, EncodeBrowserCookieInfo(cookie))
	}
	return out
}

func DecodeBrowserCookieListResponse(payload []byte) (BrowserCookieListResponse, error) {
	var result BrowserCookieListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			cookie, err := DecodeBrowserCookieInfo(data)
			if err != nil {
				return err
			}
			result.Cookies = append(result.Cookies, cookie)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCookieListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserCookieExportResponse(message BrowserCookieExportResponse) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Content)
	return out
}

func DecodeBrowserCookieExportResponse(payload []byte) (BrowserCookieExportResponse, error) {
	var result BrowserCookieExportResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Content = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCookieExportResponse{}, err
	}
	return result, nil
}

func EncodeBrowserCookieImportResult(message BrowserCookieImportResult) []byte {
	var out []byte
	out = appendInt32Field(out, 1, message.Imported)
	out = appendInt32Field(out, 2, message.Skipped)
	return out
}

func DecodeBrowserCookieImportResult(payload []byte) (BrowserCookieImportResult, error) {
	var result BrowserCookieImportResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			number, err := consumeVarintValue(wireType, value)
			result.Imported = int32(number)
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Skipped = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCookieImportResult{}, err
	}
	return result, nil
}

func EncodeBrowserCookieInfo(message BrowserCookieInfo) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Name)
	out = appendStringField(out, 2, message.Value)
	out = appendStringField(out, 3, message.Domain)
	out = appendStringField(out, 4, message.Path)
	out = appendInt64Field(out, 5, message.Expires)
	out = appendBoolField(out, 6, message.HTTPOnly)
	out = appendBoolField(out, 7, message.Secure)
	out = appendStringField(out, 8, message.SameSite)
	return out
}

func DecodeBrowserCookieInfo(payload []byte) (BrowserCookieInfo, error) {
	var result BrowserCookieInfo
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Value = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Domain = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.Path = text
			return err
		case 5:
			number, err := consumeVarintValue(wireType, value)
			result.Expires = int64(number)
			return err
		case 6:
			value, err := consumeBoolValue(wireType, value)
			result.HTTPOnly = value
			return err
		case 7:
			value, err := consumeBoolValue(wireType, value)
			result.Secure = value
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.SameSite = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCookieInfo{}, fmt.Errorf("decode browser cookie info: %w", err)
	}
	return result, nil
}
