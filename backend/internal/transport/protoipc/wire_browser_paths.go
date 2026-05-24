package protoipc

import "google.golang.org/protobuf/encoding/protowire"

const (
	MethodBrowserUserDataDirOpen        = "trace.browser.UserDataDirOpen"
	MethodBrowserProfileUserDataDirOpen = "trace.browser.ProfileUserDataDirOpen"
)

type BrowserUserDataDirOpenRequest struct {
	UserDataDir string
}

type BrowserProfileUserDataDirOpenRequest struct {
	ProfileID string
}

func EncodeBrowserUserDataDirOpenRequest(message BrowserUserDataDirOpenRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.UserDataDir)
	return out
}

func DecodeBrowserUserDataDirOpenRequest(payload []byte) (BrowserUserDataDirOpenRequest, error) {
	var result BrowserUserDataDirOpenRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.UserDataDir = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserUserDataDirOpenRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProfileUserDataDirOpenRequest(message BrowserProfileUserDataDirOpenRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProfileID)
	return out
}

func DecodeBrowserProfileUserDataDirOpenRequest(payload []byte) (BrowserProfileUserDataDirOpenRequest, error) {
	var result BrowserProfileUserDataDirOpenRequest
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
		return BrowserProfileUserDataDirOpenRequest{}, err
	}
	return result, nil
}
