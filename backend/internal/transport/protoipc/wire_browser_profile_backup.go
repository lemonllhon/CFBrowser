package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserProfileBackupExport       = "trace.browser.ProfileBackupExport"
	MethodBrowserProfileBackupChooseImport = "trace.browser.ProfileBackupChooseImport"
	MethodBrowserProfileBackupImport       = "trace.browser.ProfileBackupImport"
)

type BrowserProfileBackupExportRequest struct {
	Scope                          string
	ProfileIDs                     []string
	IncludeCookies                 bool
	IncludePlainCookiesWhenRunning bool
}

type BrowserProfileBackupImportRequest struct {
	ZipPath        string
	RestoreCookies bool
}

type BrowserProfileBackupSummary struct {
	ZipPath              string
	Format               string
	Version              int32
	AppName              string
	AppVersion           string
	CreatedAt            string
	SourceOS             string
	ProfileCount         int32
	CookieProfileCount   int32
	IncludesCookies      bool
	IncludesPlainCookies bool
	CookieNotice         string
	Warnings             []string
}

type BrowserProfileBackupWarning struct {
	ProfileID   string
	ProfileName string
	Message     string
}

type BrowserProfileBackupActionResult struct {
	Cancelled          bool
	Message            string
	ZipPath            string
	CreatedAt          string
	Exported           int32
	Imported           int32
	Skipped            int32
	Failed             int32
	ProfileCount       int32
	CookieProfileCount int32
	Summary            BrowserProfileBackupSummary
	Warnings           []BrowserProfileBackupWarning
}

func EncodeBrowserProfileBackupExportRequest(message BrowserProfileBackupExportRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Scope)
	out = appendRepeatedStringField(out, 2, message.ProfileIDs)
	out = appendBoolField(out, 3, message.IncludeCookies)
	out = appendBoolField(out, 4, message.IncludePlainCookiesWhenRunning)
	return out
}

func DecodeBrowserProfileBackupExportRequest(payload []byte) (BrowserProfileBackupExportRequest, error) {
	var result BrowserProfileBackupExportRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Scope = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 3:
			boolValue, err := consumeBoolValue(wireType, value)
			result.IncludeCookies = boolValue
			return err
		case 4:
			boolValue, err := consumeBoolValue(wireType, value)
			result.IncludePlainCookiesWhenRunning = boolValue
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBackupExportRequest{}, fmt.Errorf("decode browser profile backup export request: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfileBackupImportRequest(message BrowserProfileBackupImportRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ZipPath)
	out = appendBoolField(out, 2, message.RestoreCookies)
	return out
}

func DecodeBrowserProfileBackupImportRequest(payload []byte) (BrowserProfileBackupImportRequest, error) {
	var result BrowserProfileBackupImportRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ZipPath = text
			return err
		case 2:
			boolValue, err := consumeBoolValue(wireType, value)
			result.RestoreCookies = boolValue
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBackupImportRequest{}, fmt.Errorf("decode browser profile backup import request: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfileBackupActionResult(message BrowserProfileBackupActionResult) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Cancelled)
	out = appendStringField(out, 2, message.Message)
	out = appendStringField(out, 3, message.ZipPath)
	out = appendStringField(out, 4, message.CreatedAt)
	out = appendInt32Field(out, 5, message.Exported)
	out = appendInt32Field(out, 6, message.Imported)
	out = appendInt32Field(out, 7, message.Skipped)
	out = appendInt32Field(out, 8, message.Failed)
	out = appendInt32Field(out, 9, message.ProfileCount)
	out = appendInt32Field(out, 10, message.CookieProfileCount)
	out = appendBytesField(out, 11, EncodeBrowserProfileBackupSummary(message.Summary))
	for _, warning := range message.Warnings {
		out = appendBytesField(out, 12, EncodeBrowserProfileBackupWarning(warning))
	}
	return out
}

func DecodeBrowserProfileBackupActionResult(payload []byte) (BrowserProfileBackupActionResult, error) {
	var result BrowserProfileBackupActionResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			boolValue, err := consumeBoolValue(wireType, value)
			result.Cancelled = boolValue
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ZipPath = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 5:
			number, err := consumeVarintValue(wireType, value)
			result.Exported = int32(number)
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.Imported = int32(number)
			return err
		case 7:
			number, err := consumeVarintValue(wireType, value)
			result.Skipped = int32(number)
			return err
		case 8:
			number, err := consumeVarintValue(wireType, value)
			result.Failed = int32(number)
			return err
		case 9:
			number, err := consumeVarintValue(wireType, value)
			result.ProfileCount = int32(number)
			return err
		case 10:
			number, err := consumeVarintValue(wireType, value)
			result.CookieProfileCount = int32(number)
			return err
		case 11:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			summary, err := DecodeBrowserProfileBackupSummary(data)
			result.Summary = summary
			return err
		case 12:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			warning, err := DecodeBrowserProfileBackupWarning(data)
			if err != nil {
				return err
			}
			result.Warnings = append(result.Warnings, warning)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBackupActionResult{}, fmt.Errorf("decode browser profile backup action result: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfileBackupSummary(message BrowserProfileBackupSummary) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ZipPath)
	out = appendStringField(out, 2, message.Format)
	out = appendInt32Field(out, 3, message.Version)
	out = appendStringField(out, 4, message.AppName)
	out = appendStringField(out, 5, message.AppVersion)
	out = appendStringField(out, 6, message.CreatedAt)
	out = appendStringField(out, 7, message.SourceOS)
	out = appendInt32Field(out, 8, message.ProfileCount)
	out = appendInt32Field(out, 9, message.CookieProfileCount)
	out = appendBoolField(out, 10, message.IncludesCookies)
	out = appendBoolField(out, 11, message.IncludesPlainCookies)
	out = appendStringField(out, 12, message.CookieNotice)
	out = appendRepeatedStringField(out, 13, message.Warnings)
	return out
}

func DecodeBrowserProfileBackupSummary(payload []byte) (BrowserProfileBackupSummary, error) {
	var result BrowserProfileBackupSummary
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ZipPath = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Format = text
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.Version = int32(number)
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.AppName = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.AppVersion = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.SourceOS = text
			return err
		case 8:
			number, err := consumeVarintValue(wireType, value)
			result.ProfileCount = int32(number)
			return err
		case 9:
			number, err := consumeVarintValue(wireType, value)
			result.CookieProfileCount = int32(number)
			return err
		case 10:
			boolValue, err := consumeBoolValue(wireType, value)
			result.IncludesCookies = boolValue
			return err
		case 11:
			boolValue, err := consumeBoolValue(wireType, value)
			result.IncludesPlainCookies = boolValue
			return err
		case 12:
			text, err := consumeStringValue(wireType, value)
			result.CookieNotice = text
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.Warnings = append(result.Warnings, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBackupSummary{}, fmt.Errorf("decode browser profile backup summary: %w", err)
	}
	return result, nil
}

func EncodeBrowserProfileBackupWarning(message BrowserProfileBackupWarning) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.ProfileName)
	out = appendStringField(out, 3, message.Message)
	return out
}

func DecodeBrowserProfileBackupWarning(payload []byte) (BrowserProfileBackupWarning, error) {
	var result BrowserProfileBackupWarning
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
			result.Message = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProfileBackupWarning{}, fmt.Errorf("decode browser profile backup warning: %w", err)
	}
	return result, nil
}
