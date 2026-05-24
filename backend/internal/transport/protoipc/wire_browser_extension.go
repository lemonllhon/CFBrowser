package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserExtensionList                = "trace.browser.ExtensionList"
	MethodBrowserExtensionGet                 = "trace.browser.ExtensionGet"
	MethodBrowserExtensionDelete              = "trace.browser.ExtensionDelete"
	MethodBrowserExtensionChooseArchive       = "trace.browser.ExtensionChooseArchive"
	MethodBrowserExtensionChooseDirectory     = "trace.browser.ExtensionChooseDirectory"
	MethodBrowserExtensionImportArchive       = "trace.browser.ExtensionImportArchive"
	MethodBrowserExtensionImportDirectory     = "trace.browser.ExtensionImportDirectory"
	MethodBrowserExtensionListProfileBindings = "trace.browser.ExtensionListProfileBindings"
	MethodBrowserExtensionListForProfile      = "trace.browser.ExtensionListForProfile"
	MethodBrowserExtensionAssignProfiles      = "trace.browser.ExtensionAssignProfiles"
	MethodBrowserExtensionSetAutoBind         = "trace.browser.ExtensionSetAutoBind"
	MethodBrowserExtensionUnassignProfiles    = "trace.browser.ExtensionUnassignProfiles"
)

type BrowserExtension struct {
	ExtensionID     string
	Name            string
	Version         string
	ManifestVersion int32
	Description     string
	SourceType      string
	SourceURL       string
	InstallDir      string
	PackagePath     string
	ManifestJSON    string
	BoundCount      int32
	AutoBindEnabled bool
	AutoBindMode    string
	CreatedAt       string
	UpdatedAt       string
}

type BrowserExtensionBinding struct {
	ID               int64
	ProfileID        string
	ProfileName      string
	ExtensionID      string
	ExtensionName    string
	ExtensionVersion string
	Mode             string
	Enabled          bool
	ExclusiveDir     string
	CreatedAt        string
	UpdatedAt        string
}

type BrowserExtensionIDRequest struct {
	ExtensionID string
}

type BrowserExtensionProfileRequest struct {
	ProfileID string
}

type BrowserExtensionListResponse struct {
	Extensions []BrowserExtension
}

type BrowserExtensionResponse struct {
	Extension BrowserExtension
}

type BrowserExtensionChoosePathResponse struct {
	Cancelled bool
	Path      string
}

type BrowserExtensionImportRequest struct {
	Path     string
	Mode     string
	Existing string
}

type BrowserExtensionImportResult struct {
	Cancelled bool
	Duplicate bool
	Message   string
	Existing  *BrowserExtension
	Extension *BrowserExtension
}

type BrowserExtensionAssignRequest struct {
	ExtensionID string
	ProfileIDs  []string
	Mode        string
	Enabled     bool
}

type BrowserExtensionAutoBindRequest struct {
	ExtensionID string
	Enabled     bool
	Mode        string
}

type BrowserExtensionUnassignRequest struct {
	ExtensionID string
	ProfileIDs  []string
}

type BrowserExtensionBindingListResponse struct {
	Bindings []BrowserExtensionBinding
}

func EncodeBrowserExtensionIDRequest(message BrowserExtensionIDRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ExtensionID)
	return out
}

func DecodeBrowserExtensionIDRequest(payload []byte) (BrowserExtensionIDRequest, error) {
	var result BrowserExtensionIDRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionIDRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionProfileRequest(message BrowserExtensionProfileRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserExtensionProfileRequest(payload []byte) (BrowserExtensionProfileRequest, error) {
	var result BrowserExtensionProfileRequest
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
		return BrowserExtensionProfileRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionListResponse(message BrowserExtensionListResponse) []byte {
	var out []byte
	for _, extension := range message.Extensions {
		out = appendBytesField(out, 1, EncodeBrowserExtension(extension))
	}
	return out
}

func DecodeBrowserExtensionListResponse(payload []byte) (BrowserExtensionListResponse, error) {
	var result BrowserExtensionListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			extension, err := DecodeBrowserExtension(data)
			if err != nil {
				return err
			}
			result.Extensions = append(result.Extensions, extension)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionResponse(message BrowserExtensionResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserExtension(message.Extension))
	return out
}

func DecodeBrowserExtensionResponse(payload []byte) (BrowserExtensionResponse, error) {
	var result BrowserExtensionResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			extension, err := DecodeBrowserExtension(data)
			result.Extension = extension
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionResponse{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionChoosePathResponse(message BrowserExtensionChoosePathResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Cancelled)
	out = appendStringField(out, 2, message.Path)
	return out
}

func DecodeBrowserExtensionChoosePathResponse(payload []byte) (BrowserExtensionChoosePathResponse, error) {
	var result BrowserExtensionChoosePathResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Cancelled = value
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Path = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionChoosePathResponse{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionImportRequest(message BrowserExtensionImportRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Path)
	out = appendStringField(out, 2, message.Mode)
	out = appendStringField(out, 3, message.Existing)
	return out
}

func DecodeBrowserExtensionImportRequest(payload []byte) (BrowserExtensionImportRequest, error) {
	var result BrowserExtensionImportRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Path = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Mode = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Existing = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionImportRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionImportResult(message BrowserExtensionImportResult) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Cancelled)
	out = appendBoolField(out, 2, message.Duplicate)
	out = appendStringField(out, 3, message.Message)
	if message.Existing != nil {
		out = appendBytesField(out, 4, EncodeBrowserExtension(*message.Existing))
	}
	if message.Extension != nil {
		out = appendBytesField(out, 5, EncodeBrowserExtension(*message.Extension))
	}
	return out
}

func DecodeBrowserExtensionImportResult(payload []byte) (BrowserExtensionImportResult, error) {
	var result BrowserExtensionImportResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Cancelled = value
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.Duplicate = value
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 4:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			extension, err := DecodeBrowserExtension(data)
			if err != nil {
				return err
			}
			result.Existing = &extension
			return nil
		case 5:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			extension, err := DecodeBrowserExtension(data)
			if err != nil {
				return err
			}
			result.Extension = &extension
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionImportResult{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionAssignRequest(message BrowserExtensionAssignRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ExtensionID)
	out = appendRepeatedStringField(out, 2, message.ProfileIDs)
	out = appendStringField(out, 3, message.Mode)
	out = appendBoolField(out, 4, message.Enabled)
	return out
}

func DecodeBrowserExtensionAssignRequest(payload []byte) (BrowserExtensionAssignRequest, error) {
	var result BrowserExtensionAssignRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Mode = text
			return err
		case 4:
			value, err := consumeBoolValue(wireType, value)
			result.Enabled = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionAssignRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionAutoBindRequest(message BrowserExtensionAutoBindRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ExtensionID)
	out = appendBoolField(out, 2, message.Enabled)
	out = appendStringField(out, 3, message.Mode)
	return out
}

func DecodeBrowserExtensionAutoBindRequest(payload []byte) (BrowserExtensionAutoBindRequest, error) {
	var result BrowserExtensionAutoBindRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.Enabled = value
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Mode = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionAutoBindRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionUnassignRequest(message BrowserExtensionUnassignRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ExtensionID)
	out = appendRepeatedStringField(out, 2, message.ProfileIDs)
	return out
}

func DecodeBrowserExtensionUnassignRequest(payload []byte) (BrowserExtensionUnassignRequest, error) {
	var result BrowserExtensionUnassignRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionUnassignRequest{}, err
	}
	return result, nil
}

func EncodeBrowserExtensionBindingListResponse(message BrowserExtensionBindingListResponse) []byte {
	var out []byte
	for _, binding := range message.Bindings {
		out = appendBytesField(out, 1, EncodeBrowserExtensionBinding(binding))
	}
	return out
}

func DecodeBrowserExtensionBindingListResponse(payload []byte) (BrowserExtensionBindingListResponse, error) {
	var result BrowserExtensionBindingListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			binding, err := DecodeBrowserExtensionBinding(data)
			if err != nil {
				return err
			}
			result.Bindings = append(result.Bindings, binding)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionBindingListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserExtension(message BrowserExtension) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ExtensionID)
	out = appendStringField(out, 2, message.Name)
	out = appendStringField(out, 3, message.Version)
	out = appendInt32Field(out, 4, message.ManifestVersion)
	out = appendStringField(out, 5, message.Description)
	out = appendStringField(out, 6, message.SourceType)
	out = appendStringField(out, 7, message.SourceURL)
	out = appendStringField(out, 8, message.InstallDir)
	out = appendStringField(out, 9, message.PackagePath)
	out = appendStringField(out, 10, message.ManifestJSON)
	out = appendInt32Field(out, 11, message.BoundCount)
	out = appendBoolField(out, 12, message.AutoBindEnabled)
	out = appendStringField(out, 13, message.AutoBindMode)
	out = appendStringField(out, 14, message.CreatedAt)
	out = appendStringField(out, 15, message.UpdatedAt)
	return out
}

func DecodeBrowserExtension(payload []byte) (BrowserExtension, error) {
	var result BrowserExtension
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Version = text
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.ManifestVersion = int32(number)
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.Description = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.SourceType = text
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.SourceURL = text
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.InstallDir = text
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.PackagePath = text
			return err
		case 10:
			text, err := consumeStringValue(wireType, value)
			result.ManifestJSON = text
			return err
		case 11:
			number, err := consumeVarintValue(wireType, value)
			result.BoundCount = int32(number)
			return err
		case 12:
			value, err := consumeBoolValue(wireType, value)
			result.AutoBindEnabled = value
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.AutoBindMode = text
			return err
		case 14:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 15:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtension{}, fmt.Errorf("decode browser extension: %w", err)
	}
	return result, nil
}

func EncodeBrowserExtensionBinding(message BrowserExtensionBinding) []byte {
	var out []byte
	out = appendInt64Field(out, 1, message.ID)
	out = appendStringField(out, 2, message.ProfileID)
	out = appendStringField(out, 3, message.ProfileName)
	out = appendStringField(out, 4, message.ExtensionID)
	out = appendStringField(out, 5, message.ExtensionName)
	out = appendStringField(out, 6, message.ExtensionVersion)
	out = appendStringField(out, 7, message.Mode)
	out = appendBoolField(out, 8, message.Enabled)
	out = appendStringField(out, 9, message.ExclusiveDir)
	out = appendStringField(out, 10, message.CreatedAt)
	out = appendStringField(out, 11, message.UpdatedAt)
	return out
}

func DecodeBrowserExtensionBinding(payload []byte) (BrowserExtensionBinding, error) {
	var result BrowserExtensionBinding
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			number, err := consumeVarintValue(wireType, value)
			result.ID = int64(number)
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ProfileName = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionID = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionName = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.ExtensionVersion = text
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.Mode = text
			return err
		case 8:
			value, err := consumeBoolValue(wireType, value)
			result.Enabled = value
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.ExclusiveDir = text
			return err
		case 10:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserExtensionBinding{}, fmt.Errorf("decode browser extension binding: %w", err)
	}
	return result, nil
}
