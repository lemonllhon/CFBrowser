package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodWindowSyncCandidateList       = "trace.windowSync.CandidateList"
	MethodWindowSyncStart               = "trace.windowSync.Start"
	MethodWindowSyncStateGet            = "trace.windowSync.StateGet"
	MethodWindowSyncStop                = "trace.windowSync.Stop"
	MethodWindowSyncPause               = "trace.windowSync.Pause"
	MethodWindowSyncResume              = "trace.windowSync.Resume"
	MethodWindowSyncShowAll             = "trace.windowSync.ShowAll"
	MethodWindowSyncSettingsGet         = "trace.windowSync.SettingsGet"
	MethodWindowSyncSettingsSave        = "trace.windowSync.SettingsSave"
	MethodWindowSyncLayoutSettingsGet   = "trace.windowSync.LayoutSettingsGet"
	MethodWindowSyncLayoutSettingsSave  = "trace.windowSync.LayoutSettingsSave"
	MethodWindowSyncLayoutApply         = "trace.windowSync.LayoutApply"
	MethodWindowSyncBatchInputSame      = "trace.windowSync.BatchInputSame"
	MethodWindowSyncBatchInputDifferent = "trace.windowSync.BatchInputDifferent"
	MethodWindowSyncCloseOtherTabs      = "trace.windowSync.CloseOtherTabs"
	MethodWindowSyncCloseCurrentTab     = "trace.windowSync.CloseCurrentTab"
	MethodWindowSyncCloseBlankTabs      = "trace.windowSync.CloseBlankTabs"
	MethodWindowSyncOpenUrls            = "trace.windowSync.OpenUrls"
	MethodWindowSyncToolbarResize       = "trace.windowSync.ToolbarResize"
)

const EventWindowSyncStateChanged = "window-sync:state-changed"

type WindowSyncCandidate struct {
	ProfileID    string
	ProfileName  string
	DebugPort    int32
	PID          int32
	Running      bool
	DebugReady   bool
	Role         string
	Master       bool
	CanSync      bool
	CanAutoStart bool
	Unavailable  string
}

type WindowSyncCandidateListResponse struct {
	Candidates []WindowSyncCandidate
}

type WindowSyncStartRequest struct {
	ProfileIDs      []string
	MasterProfileID string
}

type WindowSyncLayoutSettings struct {
	Mode      string
	Scope     string
	Width     int32
	Height    int32
	GapX      int32
	GapY      int32
	PerRow    int32
	UpdatedAt string
}

type WindowSyncSettings struct {
	MasterColor  string
	SyncKeyboard bool
	SyncMouse    bool
}

type WindowSyncState struct {
	SessionID       string
	Active          bool
	Paused          bool
	MasterProfileID string
	ProfileIDs      []string
	Windows         []WindowSyncCandidate
	MasterColor     string
	SyncKeyboard    bool
	SyncMouse       bool
	Layout          WindowSyncLayoutSettings
	StartedAt       string
	UpdatedAt       string
}

type WindowSyncStateResponse struct {
	State *WindowSyncState
}

type WindowSyncLayoutSettingsResponse struct {
	Layout WindowSyncLayoutSettings
}

type WindowSyncSettingsResponse struct {
	Settings WindowSyncSettings
}

type WindowSyncBatchInputSameRequest struct {
	Text string
}

type WindowSyncBatchInputDifferentItem struct {
	ProfileID string
	Text      string
}

type WindowSyncBatchInputDifferentRequest struct {
	Items []WindowSyncBatchInputDifferentItem
}

type WindowSyncBatchInputResultItem struct {
	ProfileID   string
	ProfileName string
	Master      bool
	Success     bool
	Error       string
}

type WindowSyncBatchInputResult struct {
	Total   int32
	Success int32
	Failed  int32
	Results []WindowSyncBatchInputResultItem
}

type WindowSyncOpenUrlsRequest struct {
	URLs []string
}

type WindowSyncActionResultItem struct {
	ProfileID   string
	ProfileName string
	Master      bool
	Success     bool
	Error       string
}

type WindowSyncActionResult struct {
	Total   int32
	Success int32
	Failed  int32
	Results []WindowSyncActionResultItem
}

type WindowSyncToolbarResizeRequest struct {
	Width  int32
	Height int32
}

type WindowSyncToolbarResizeResponse struct {
	OK bool
}

func EncodeWindowSyncCandidateListResponse(message WindowSyncCandidateListResponse) []byte {
	var out []byte
	for _, candidate := range message.Candidates {
		out = appendBytesField(out, 1, EncodeWindowSyncCandidate(candidate))
	}
	return out
}

func DecodeWindowSyncCandidateListResponse(payload []byte) (WindowSyncCandidateListResponse, error) {
	var result WindowSyncCandidateListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			candidate, err := DecodeWindowSyncCandidate(data)
			if err != nil {
				return err
			}
			result.Candidates = append(result.Candidates, candidate)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncCandidateListResponse{}, err
	}
	return result, nil
}

func EncodeWindowSyncStartRequest(message WindowSyncStartRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.ProfileIDs)
	out = appendStringField(out, 2, message.MasterProfileID)
	return out
}

func DecodeWindowSyncStartRequest(payload []byte) (WindowSyncStartRequest, error) {
	var result WindowSyncStartRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.MasterProfileID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncStartRequest{}, fmt.Errorf("decode window sync start request: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncLayoutSettingsResponse(message WindowSyncLayoutSettingsResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeWindowSyncLayoutSettings(message.Layout))
	return out
}

func DecodeWindowSyncLayoutSettingsResponse(payload []byte) (WindowSyncLayoutSettingsResponse, error) {
	var result WindowSyncLayoutSettingsResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			layout, err := DecodeWindowSyncLayoutSettings(data)
			if err != nil {
				return err
			}
			result.Layout = layout
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncLayoutSettingsResponse{}, err
	}
	return result, nil
}

func EncodeWindowSyncSettingsResponse(message WindowSyncSettingsResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeWindowSyncSettings(message.Settings))
	return out
}

func DecodeWindowSyncSettingsResponse(payload []byte) (WindowSyncSettingsResponse, error) {
	var result WindowSyncSettingsResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			settings, err := DecodeWindowSyncSettings(data)
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
		return WindowSyncSettingsResponse{}, err
	}
	return result, nil
}

func EncodeWindowSyncStateResponse(message WindowSyncStateResponse) []byte {
	var out []byte
	if message.State != nil {
		out = appendBoolField(out, 1, true)
		out = appendBytesField(out, 2, EncodeWindowSyncState(*message.State))
	}
	return out
}

func DecodeWindowSyncStateResponse(payload []byte) (WindowSyncStateResponse, error) {
	hasState := false
	var state *WindowSyncState
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			hasState = value
			return err
		case 2:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			decoded, err := DecodeWindowSyncState(data)
			if err != nil {
				return err
			}
			state = &decoded
			hasState = true
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncStateResponse{}, err
	}
	if hasState && state == nil {
		state = &WindowSyncState{}
	}
	return WindowSyncStateResponse{State: state}, nil
}

func EncodeWindowSyncBatchInputSameRequest(message WindowSyncBatchInputSameRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Text)
	return out
}

func DecodeWindowSyncBatchInputSameRequest(payload []byte) (WindowSyncBatchInputSameRequest, error) {
	var result WindowSyncBatchInputSameRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Text = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncBatchInputSameRequest{}, fmt.Errorf("decode window sync batch input same request: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncBatchInputDifferentRequest(message WindowSyncBatchInputDifferentRequest) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeWindowSyncBatchInputDifferentItem(item))
	}
	return out
}

func DecodeWindowSyncBatchInputDifferentRequest(payload []byte) (WindowSyncBatchInputDifferentRequest, error) {
	var result WindowSyncBatchInputDifferentRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeWindowSyncBatchInputDifferentItem(data)
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
		return WindowSyncBatchInputDifferentRequest{}, err
	}
	return result, nil
}

func EncodeWindowSyncOpenUrlsRequest(message WindowSyncOpenUrlsRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.URLs)
	return out
}

func DecodeWindowSyncOpenUrlsRequest(payload []byte) (WindowSyncOpenUrlsRequest, error) {
	var result WindowSyncOpenUrlsRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.URLs = append(result.URLs, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncOpenUrlsRequest{}, fmt.Errorf("decode window sync open urls request: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncBatchInputResult(message WindowSyncBatchInputResult) []byte {
	var out []byte
	out = appendInt32Field(out, 1, message.Total)
	out = appendInt32Field(out, 2, message.Success)
	out = appendInt32Field(out, 3, message.Failed)
	for _, item := range message.Results {
		out = appendBytesField(out, 4, EncodeWindowSyncBatchInputResultItem(item))
	}
	return out
}

func DecodeWindowSyncBatchInputResult(payload []byte) (WindowSyncBatchInputResult, error) {
	var result WindowSyncBatchInputResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			number, err := consumeVarintValue(wireType, value)
			result.Total = int32(number)
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Success = int32(number)
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.Failed = int32(number)
			return err
		case 4:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeWindowSyncBatchInputResultItem(data)
			if err != nil {
				return err
			}
			result.Results = append(result.Results, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncBatchInputResult{}, err
	}
	return result, nil
}

func EncodeWindowSyncActionResult(message WindowSyncActionResult) []byte {
	var out []byte
	out = appendInt32Field(out, 1, message.Total)
	out = appendInt32Field(out, 2, message.Success)
	out = appendInt32Field(out, 3, message.Failed)
	for _, item := range message.Results {
		out = appendBytesField(out, 4, EncodeWindowSyncActionResultItem(item))
	}
	return out
}

func DecodeWindowSyncActionResult(payload []byte) (WindowSyncActionResult, error) {
	var result WindowSyncActionResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			number, err := consumeVarintValue(wireType, value)
			result.Total = int32(number)
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Success = int32(number)
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.Failed = int32(number)
			return err
		case 4:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeWindowSyncActionResultItem(data)
			if err != nil {
				return err
			}
			result.Results = append(result.Results, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncActionResult{}, err
	}
	return result, nil
}

func EncodeWindowSyncToolbarResizeRequest(message WindowSyncToolbarResizeRequest) []byte {
	var out []byte
	out = appendInt32Field(out, 1, message.Width)
	out = appendInt32Field(out, 2, message.Height)
	return out
}

func DecodeWindowSyncToolbarResizeRequest(payload []byte) (WindowSyncToolbarResizeRequest, error) {
	var result WindowSyncToolbarResizeRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			number, err := consumeVarintValue(wireType, value)
			result.Width = int32(number)
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Height = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncToolbarResizeRequest{}, fmt.Errorf("decode window sync toolbar resize request: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncToolbarResizeResponse(message WindowSyncToolbarResizeResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.OK)
	return out
}

func DecodeWindowSyncToolbarResizeResponse(payload []byte) (WindowSyncToolbarResizeResponse, error) {
	var result WindowSyncToolbarResizeResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			ok, err := consumeBoolValue(wireType, value)
			result.OK = ok
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncToolbarResizeResponse{}, fmt.Errorf("decode window sync toolbar resize response: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncCandidate(message WindowSyncCandidate) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.ProfileName)
	out = appendInt32Field(out, 3, message.DebugPort)
	out = appendInt32Field(out, 4, message.PID)
	out = appendBoolField(out, 5, message.Running)
	out = appendBoolField(out, 6, message.DebugReady)
	out = appendStringField(out, 7, message.Role)
	out = appendBoolField(out, 8, message.Master)
	out = appendBoolField(out, 9, message.CanSync)
	out = appendBoolField(out, 10, message.CanAutoStart)
	out = appendStringField(out, 11, message.Unavailable)
	return out
}

func DecodeWindowSyncCandidate(payload []byte) (WindowSyncCandidate, error) {
	var result WindowSyncCandidate
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
			number, err := consumeVarintValue(wireType, value)
			result.DebugPort = int32(number)
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.PID = int32(number)
			return err
		case 5:
			value, err := consumeBoolValue(wireType, value)
			result.Running = value
			return err
		case 6:
			value, err := consumeBoolValue(wireType, value)
			result.DebugReady = value
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.Role = text
			return err
		case 8:
			value, err := consumeBoolValue(wireType, value)
			result.Master = value
			return err
		case 9:
			value, err := consumeBoolValue(wireType, value)
			result.CanSync = value
			return err
		case 10:
			value, err := consumeBoolValue(wireType, value)
			result.CanAutoStart = value
			return err
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.Unavailable = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncCandidate{}, fmt.Errorf("decode window sync candidate: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncLayoutSettings(message WindowSyncLayoutSettings) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Mode)
	out = appendInt32Field(out, 2, message.Width)
	out = appendInt32Field(out, 3, message.Height)
	out = appendInt32Field(out, 4, message.GapX)
	out = appendInt32Field(out, 5, message.GapY)
	out = appendInt32Field(out, 6, message.PerRow)
	out = appendStringField(out, 7, message.UpdatedAt)
	out = appendStringField(out, 8, message.Scope)
	return out
}

func DecodeWindowSyncLayoutSettings(payload []byte) (WindowSyncLayoutSettings, error) {
	var result WindowSyncLayoutSettings
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Mode = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Width = int32(number)
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.Height = int32(number)
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.GapX = int32(number)
			return err
		case 5:
			number, err := consumeVarintValue(wireType, value)
			result.GapY = int32(number)
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.PerRow = int32(number)
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.Scope = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncLayoutSettings{}, fmt.Errorf("decode window sync layout settings: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncSettings(message WindowSyncSettings) []byte {
	var out []byte
	out = appendStringField(out, 1, message.MasterColor)
	out = appendBoolField(out, 2, message.SyncKeyboard)
	out = appendBoolField(out, 3, message.SyncMouse)
	return out
}

func DecodeWindowSyncSettings(payload []byte) (WindowSyncSettings, error) {
	var result WindowSyncSettings
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.MasterColor = text
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.SyncKeyboard = value
			return err
		case 3:
			value, err := consumeBoolValue(wireType, value)
			result.SyncMouse = value
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncSettings{}, fmt.Errorf("decode window sync settings: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncState(message WindowSyncState) []byte {
	var out []byte
	out = appendStringField(out, 1, message.SessionID)
	out = appendBoolField(out, 2, message.Active)
	out = appendBoolField(out, 3, message.Paused)
	out = appendStringField(out, 4, message.MasterProfileID)
	out = appendRepeatedStringField(out, 5, message.ProfileIDs)
	for _, window := range message.Windows {
		out = appendBytesField(out, 6, EncodeWindowSyncCandidate(window))
	}
	out = appendStringField(out, 7, message.MasterColor)
	out = appendBoolField(out, 8, message.SyncKeyboard)
	out = appendBoolField(out, 9, message.SyncMouse)
	out = appendBytesField(out, 10, EncodeWindowSyncLayoutSettings(message.Layout))
	out = appendStringField(out, 11, message.StartedAt)
	out = appendStringField(out, 12, message.UpdatedAt)
	return out
}

func DecodeWindowSyncState(payload []byte) (WindowSyncState, error) {
	var result WindowSyncState
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.SessionID = text
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.Active = value
			return err
		case 3:
			value, err := consumeBoolValue(wireType, value)
			result.Paused = value
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.MasterProfileID = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.ProfileIDs = append(result.ProfileIDs, text)
			return err
		case 6:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			window, err := DecodeWindowSyncCandidate(data)
			if err != nil {
				return err
			}
			result.Windows = append(result.Windows, window)
			return nil
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.MasterColor = text
			return err
		case 8:
			value, err := consumeBoolValue(wireType, value)
			result.SyncKeyboard = value
			return err
		case 9:
			value, err := consumeBoolValue(wireType, value)
			result.SyncMouse = value
			return err
		case 10:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			layout, err := DecodeWindowSyncLayoutSettings(data)
			if err != nil {
				return err
			}
			result.Layout = layout
			return nil
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.StartedAt = text
			return err
		case 12:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncState{}, fmt.Errorf("decode window sync state: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncBatchInputDifferentItem(message WindowSyncBatchInputDifferentItem) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.Text)
	return out
}

func DecodeWindowSyncBatchInputDifferentItem(payload []byte) (WindowSyncBatchInputDifferentItem, error) {
	var result WindowSyncBatchInputDifferentItem
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Text = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return WindowSyncBatchInputDifferentItem{}, fmt.Errorf("decode window sync batch input different item: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncBatchInputResultItem(message WindowSyncBatchInputResultItem) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.ProfileName)
	out = appendBoolField(out, 3, message.Master)
	out = appendBoolField(out, 4, message.Success)
	out = appendStringField(out, 5, message.Error)
	return out
}

func DecodeWindowSyncBatchInputResultItem(payload []byte) (WindowSyncBatchInputResultItem, error) {
	var result WindowSyncBatchInputResultItem
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		return decodeWindowSyncResultItemField(field, wireType, value, &result.ProfileID, &result.ProfileName, &result.Master, &result.Success, &result.Error)
	})
	if err != nil {
		return WindowSyncBatchInputResultItem{}, fmt.Errorf("decode window sync batch input result item: %w", err)
	}
	return result, nil
}

func EncodeWindowSyncActionResultItem(message WindowSyncActionResultItem) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.ProfileName)
	out = appendBoolField(out, 3, message.Master)
	out = appendBoolField(out, 4, message.Success)
	out = appendStringField(out, 5, message.Error)
	return out
}

func DecodeWindowSyncActionResultItem(payload []byte) (WindowSyncActionResultItem, error) {
	var result WindowSyncActionResultItem
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		return decodeWindowSyncResultItemField(field, wireType, value, &result.ProfileID, &result.ProfileName, &result.Master, &result.Success, &result.Error)
	})
	if err != nil {
		return WindowSyncActionResultItem{}, fmt.Errorf("decode window sync action result item: %w", err)
	}
	return result, nil
}

func decodeWindowSyncResultItemField(field protowire.Number, wireType protowire.Type, value []byte, profileID *string, profileName *string, master *bool, success *bool, errorMessage *string) error {
	switch field {
	case 1:
		text, err := consumeStringValue(wireType, value)
		*profileID = text
		return err
	case 2:
		text, err := consumeStringValue(wireType, value)
		*profileName = text
		return err
	case 3:
		value, err := consumeBoolValue(wireType, value)
		*master = value
		return err
	case 4:
		value, err := consumeBoolValue(wireType, value)
		*success = value
		return err
	case 5:
		text, err := consumeStringValue(wireType, value)
		*errorMessage = text
		return err
	default:
		return nil
	}
}
