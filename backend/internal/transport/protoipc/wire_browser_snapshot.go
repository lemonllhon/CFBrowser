package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserSnapshotList    = "trace.browser.SnapshotList"
	MethodBrowserSnapshotCreate  = "trace.browser.SnapshotCreate"
	MethodBrowserSnapshotRestore = "trace.browser.SnapshotRestore"
	MethodBrowserSnapshotDelete  = "trace.browser.SnapshotDelete"
)

type BrowserSnapshotInfo struct {
	SnapshotID  string
	ProfileID   string
	Name        string
	SizeMBMilli int64
	CreatedAt   string
}

type BrowserSnapshotProfileRequest struct {
	ProfileID string
}

type BrowserSnapshotCreateRequest struct {
	ProfileID string
	Name      string
}

type BrowserSnapshotActionRequest struct {
	ProfileID  string
	SnapshotID string
}

type BrowserSnapshotListResponse struct {
	Snapshots []BrowserSnapshotInfo
}

type BrowserSnapshotResponse struct {
	Snapshot BrowserSnapshotInfo
}

func EncodeBrowserSnapshotProfileRequest(message BrowserSnapshotProfileRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserSnapshotProfileRequest(payload []byte) (BrowserSnapshotProfileRequest, error) {
	var result BrowserSnapshotProfileRequest
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
		return BrowserSnapshotProfileRequest{}, err
	}
	return result, nil
}

func EncodeBrowserSnapshotCreateRequest(message BrowserSnapshotCreateRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.Name)
	return out
}

func DecodeBrowserSnapshotCreateRequest(payload []byte) (BrowserSnapshotCreateRequest, error) {
	var result BrowserSnapshotCreateRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSnapshotCreateRequest{}, err
	}
	return result, nil
}

func EncodeBrowserSnapshotActionRequest(message BrowserSnapshotActionRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	out = appendStringField(out, 2, message.SnapshotID)
	return out
}

func DecodeBrowserSnapshotActionRequest(payload []byte) (BrowserSnapshotActionRequest, error) {
	var result BrowserSnapshotActionRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.SnapshotID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSnapshotActionRequest{}, err
	}
	return result, nil
}

func EncodeBrowserSnapshotListResponse(message BrowserSnapshotListResponse) []byte {
	var out []byte
	for _, snapshot := range message.Snapshots {
		out = appendBytesField(out, 1, EncodeBrowserSnapshotInfo(snapshot))
	}
	return out
}

func DecodeBrowserSnapshotListResponse(payload []byte) (BrowserSnapshotListResponse, error) {
	var result BrowserSnapshotListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			snapshot, err := DecodeBrowserSnapshotInfo(data)
			if err != nil {
				return err
			}
			result.Snapshots = append(result.Snapshots, snapshot)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSnapshotListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserSnapshotResponse(message BrowserSnapshotResponse) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeBrowserSnapshotInfo(message.Snapshot))
	return out
}

func DecodeBrowserSnapshotResponse(payload []byte) (BrowserSnapshotResponse, error) {
	var result BrowserSnapshotResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			snapshot, err := DecodeBrowserSnapshotInfo(data)
			result.Snapshot = snapshot
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSnapshotResponse{}, err
	}
	return result, nil
}

func EncodeBrowserSnapshotInfo(message BrowserSnapshotInfo) []byte {
	var out []byte
	out = appendStringField(out, 1, message.SnapshotID)
	out = appendStringField(out, 2, message.ProfileID)
	out = appendStringField(out, 3, message.Name)
	out = appendInt64Field(out, 4, message.SizeMBMilli)
	out = appendStringField(out, 5, message.CreatedAt)
	return out
}

func DecodeBrowserSnapshotInfo(payload []byte) (BrowserSnapshotInfo, error) {
	var result BrowserSnapshotInfo
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.SnapshotID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProfileID = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.SizeMBMilli = int64(number)
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.CreatedAt = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserSnapshotInfo{}, fmt.Errorf("decode browser snapshot info: %w", err)
	}
	return result, nil
}
