package protoipc

import (
	"encoding/base64"
	"errors"
	"strings"
)

const (
	RawMessagePrefix  = "trace-proto:"
	ResponseEventName = "trace:proto:response"
)

var ErrRawMessagePrefix = errors.New("proto ipc raw message prefix mismatch")

func EncodeRawMessage(payload []byte) string {
	return RawMessagePrefix + base64.StdEncoding.EncodeToString(payload)
}

func DecodeRawMessage(message string) ([]byte, bool, error) {
	if !strings.HasPrefix(message, RawMessagePrefix) {
		return nil, false, ErrRawMessagePrefix
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(message, RawMessagePrefix))
	if err != nil {
		return nil, true, err
	}
	return payload, true, nil
}
