package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserSettingsGet  = "trace.browser.SettingsGet"
	MethodBrowserSettingsSave = "trace.browser.SettingsSave"
)

type BrowserSettings struct {
	UserDataRoot           string
	DefaultFingerprintArgs []string
	DefaultLaunchArgs      []string
	DefaultProxy           string
	StartReadyTimeoutMs    int32
	StartStableWindowMs    int32
}

type BrowserSettingsResponse struct {
	Settings BrowserSettings
}

type BrowserSettingsSaveRequest struct {
	Settings BrowserSettings
}

func EncodeBrowserSettingsResponse(message BrowserSettingsResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserSettings(message.Settings))
	return out
}

func DecodeBrowserSettingsResponse(payload []byte) (BrowserSettingsResponse, error) {
	var result BrowserSettingsResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			settings, err := DecodeBrowserSettings(data)
			if err != nil {
				return err
			}
			result.Settings = settings
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSettingsResponse{}, err
	}
	return result, nil
}

func EncodeBrowserSettingsSaveRequest(message BrowserSettingsSaveRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserSettings(message.Settings))
	return out
}

func DecodeBrowserSettingsSaveRequest(payload []byte) (BrowserSettingsSaveRequest, error) {
	response, err := DecodeBrowserSettingsResponse(payload)
	if err != nil {
		return BrowserSettingsSaveRequest{}, err
	}
	return BrowserSettingsSaveRequest{Settings: response.Settings}, nil
}

func EncodeBrowserSettings(message BrowserSettings) []byte {
	var out []byte
	out = appendStringField(out, 1, message.UserDataRoot)
	out = appendRepeatedStringField(out, 2, message.DefaultFingerprintArgs)
	out = appendRepeatedStringField(out, 3, message.DefaultLaunchArgs)
	out = appendStringField(out, 4, message.DefaultProxy)
	out = appendInt32Field(out, 5, message.StartReadyTimeoutMs)
	out = appendInt32Field(out, 6, message.StartStableWindowMs)
	return out
}

func DecodeBrowserSettings(payload []byte) (BrowserSettings, error) {
	var result BrowserSettings
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.UserDataRoot = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.DefaultFingerprintArgs = append(result.DefaultFingerprintArgs, text)
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.DefaultLaunchArgs = append(result.DefaultLaunchArgs, text)
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.DefaultProxy = text
			return err
		case 5:
			number, err := consumeVarintValue(wireType, value)
			result.StartReadyTimeoutMs = int32(number)
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.StartStableWindowMs = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSettings{}, fmt.Errorf("decode browser settings: %w", err)
	}
	return result, nil
}
