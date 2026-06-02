package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserCoreList           = "trace.browser.CoreList"
	MethodBrowserCoreSave           = "trace.browser.CoreSave"
	MethodBrowserCoreDelete         = "trace.browser.CoreDelete"
	MethodBrowserCoreSetDefault     = "trace.browser.CoreSetDefault"
	MethodBrowserCoreValidate       = "trace.browser.CoreValidate"
	MethodBrowserCoreRenamePath     = "trace.browser.CoreRenamePath"
	MethodBrowserCoreExtendedInfo   = "trace.browser.CoreExtendedInfo"
	MethodBrowserCoreScan           = "trace.browser.CoreScan"
	MethodBrowserCoreDownload       = "trace.browser.CoreDownload"
	MethodBrowserCoreCancelDownload = "trace.browser.CoreCancelDownload"
	MethodBrowserCoreOpenPath       = "trace.browser.CoreOpenPath"
)

type BrowserCore struct {
	CoreID    string
	CoreName  string
	CorePath  string
	IsDefault bool
}

type BrowserCoreListResponse struct {
	Cores []BrowserCore
}

type BrowserCoreSaveRequest struct {
	Core BrowserCore
}

type BrowserCoreIDRequest struct {
	CoreID string
}

type BrowserCorePathRequest struct {
	CorePath string
}

type BrowserCoreValidateResponse struct {
	Valid   bool
	Message string
}

type BrowserCoreRenamePathRequest struct {
	CorePath      string
	NewFolderName string
}

type BrowserCoreExtendedInfo struct {
	CoreID        string
	ChromeVersion string
	InstanceCount int32
}

type BrowserCoreExtendedInfoResponse struct {
	Items []BrowserCoreExtendedInfo
}

type BrowserCoreDownloadRequest struct {
	CoreName    string
	URL         string
	ProxyConfig string
}

type BrowserCoreDownloadProgress struct {
	Phase    string
	Progress int32
	Message  string
	CorePath string
}

func EncodeBrowserCoreListResponse(message BrowserCoreListResponse) []byte {
	var out []byte
	for _, item := range message.Cores {
		out = appendBytesField(out, 1, EncodeBrowserCore(item))
	}
	return out
}

func DecodeBrowserCoreListResponse(payload []byte) (BrowserCoreListResponse, error) {
	var result BrowserCoreListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserCore(data)
			if err != nil {
				return err
			}
			result.Cores = append(result.Cores, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserCoreSaveRequest(message BrowserCoreSaveRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserCore(message.Core))
	return out
}

func DecodeBrowserCoreSaveRequest(payload []byte) (BrowserCoreSaveRequest, error) {
	var result BrowserCoreSaveRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			core, err := DecodeBrowserCore(data)
			result.Core = core
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreSaveRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCoreIDRequest(message BrowserCoreIDRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CoreID)
	return out
}

func DecodeBrowserCoreIDRequest(payload []byte) (BrowserCoreIDRequest, error) {
	var result BrowserCoreIDRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CoreID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreIDRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCorePathRequest(message BrowserCorePathRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CorePath)
	return out
}

func DecodeBrowserCorePathRequest(payload []byte) (BrowserCorePathRequest, error) {
	var result BrowserCorePathRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CorePath = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCorePathRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCoreValidateResponse(message BrowserCoreValidateResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Valid)
	out = appendStringField(out, 2, message.Message)
	return out
}

func DecodeBrowserCoreValidateResponse(payload []byte) (BrowserCoreValidateResponse, error) {
	var result BrowserCoreValidateResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Valid = value
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreValidateResponse{}, err
	}
	return result, nil
}

func EncodeBrowserCoreRenamePathRequest(message BrowserCoreRenamePathRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CorePath)
	out = appendStringField(out, 2, message.NewFolderName)
	return out
}

func DecodeBrowserCoreRenamePathRequest(payload []byte) (BrowserCoreRenamePathRequest, error) {
	var result BrowserCoreRenamePathRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CorePath = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.NewFolderName = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreRenamePathRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCoreExtendedInfoResponse(message BrowserCoreExtendedInfoResponse) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserCoreExtendedInfo(item))
	}
	return out
}

func DecodeBrowserCoreExtendedInfoResponse(payload []byte) (BrowserCoreExtendedInfoResponse, error) {
	var result BrowserCoreExtendedInfoResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserCoreExtendedInfo(data)
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
		return BrowserCoreExtendedInfoResponse{}, err
	}
	return result, nil
}

func EncodeBrowserCoreDownloadRequest(message BrowserCoreDownloadRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CoreName)
	out = appendStringField(out, 2, message.URL)
	out = appendStringField(out, 3, message.ProxyConfig)
	return out
}

func DecodeBrowserCoreDownloadRequest(payload []byte) (BrowserCoreDownloadRequest, error) {
	var result BrowserCoreDownloadRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CoreName = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreDownloadRequest{}, err
	}
	return result, nil
}

func EncodeBrowserCoreDownloadProgress(message BrowserCoreDownloadProgress) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Phase)
	out = appendInt32Field(out, 2, message.Progress)
	out = appendStringField(out, 3, message.Message)
	out = appendStringField(out, 4, message.CorePath)
	return out
}

func DecodeBrowserCoreDownloadProgress(payload []byte) (BrowserCoreDownloadProgress, error) {
	var result BrowserCoreDownloadProgress
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Phase = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Progress = int32(number)
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.CorePath = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreDownloadProgress{}, err
	}
	return result, nil
}

func EncodeBrowserCore(message BrowserCore) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CoreID)
	out = appendStringField(out, 2, message.CoreName)
	out = appendStringField(out, 3, message.CorePath)
	out = appendBoolField(out, 4, message.IsDefault)
	return out
}

func DecodeBrowserCore(payload []byte) (BrowserCore, error) {
	var result BrowserCore
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CoreID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.CoreName = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.CorePath = text
			return err
		case 4:
			value, err := consumeBoolValue(wireType, value)
			result.IsDefault = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCore{}, fmt.Errorf("decode browser core: %w", err)
	}
	return result, nil
}

func EncodeBrowserCoreExtendedInfo(message BrowserCoreExtendedInfo) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CoreID)
	out = appendStringField(out, 2, message.ChromeVersion)
	out = appendInt32Field(out, 3, message.InstanceCount)
	return out
}

func DecodeBrowserCoreExtendedInfo(payload []byte) (BrowserCoreExtendedInfo, error) {
	var result BrowserCoreExtendedInfo
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CoreID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ChromeVersion = text
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.InstanceCount = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserCoreExtendedInfo{}, err
	}
	return result, nil
}
