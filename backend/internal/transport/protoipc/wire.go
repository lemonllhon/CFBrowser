package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const SchemaVersion int32 = 1

type Envelope struct {
	RequestID     string
	Method        string
	Payload       []byte
	SchemaVersion int32
	TimestampMS   int64
}

type RPCError struct {
	Code    string
	Message string
	Details string
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

type Response struct {
	RequestID string
	Payload   []byte
	Error     *RPCError
}

type Event struct {
	EventID     string
	EventName   string
	Payload     []byte
	TimestampMS int64
}

type PingRequest struct {
	Message      string
	SentAtUnixMS int64
}

type PingResponse struct {
	Message          string
	ServerTimeUnixMS int64
	PayloadSize      int32
}

func EncodeEnvelope(message Envelope) []byte {
	var out []byte
	out = appendStringField(out, 1, message.RequestID)
	out = appendStringField(out, 2, message.Method)
	out = appendBytesField(out, 3, message.Payload)
	out = appendInt32Field(out, 4, message.SchemaVersion)
	out = appendInt64Field(out, 5, message.TimestampMS)
	return out
}

func DecodeEnvelope(payload []byte) (Envelope, error) {
	var result Envelope
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.RequestID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Method = text
			return err
		case 3:
			data, err := consumeBytesValue(wireType, value)
			result.Payload = data
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.SchemaVersion = int32(number)
			return err
		case 5:
			number, err := consumeVarintValue(wireType, value)
			result.TimestampMS = int64(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return Envelope{}, err
	}
	return result, nil
}

func EncodeResponse(message Response) []byte {
	var out []byte
	out = appendStringField(out, 1, message.RequestID)
	out = appendBytesField(out, 2, message.Payload)
	if message.Error != nil {
		out = appendBytesField(out, 3, encodeRPCError(*message.Error))
	}
	return out
}

func DecodeResponse(payload []byte) (Response, error) {
	var result Response
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.RequestID = text
			return err
		case 2:
			data, err := consumeBytesValue(wireType, value)
			result.Payload = data
			return err
		case 3:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			rpcErr, err := decodeRPCError(data)
			result.Error = &rpcErr
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return Response{}, err
	}
	return result, nil
}

func EncodeEvent(message Event) []byte {
	var out []byte
	out = appendStringField(out, 1, message.EventID)
	out = appendStringField(out, 2, message.EventName)
	out = appendBytesField(out, 3, message.Payload)
	out = appendInt64Field(out, 4, message.TimestampMS)
	return out
}

func DecodeEvent(payload []byte) (Event, error) {
	var result Event
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.EventID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.EventName = text
			return err
		case 3:
			data, err := consumeBytesValue(wireType, value)
			result.Payload = data
			return err
		case 4:
			number, err := consumeVarintValue(wireType, value)
			result.TimestampMS = int64(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return Event{}, err
	}
	return result, nil
}

func EncodePingRequest(message PingRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Message)
	out = appendInt64Field(out, 2, message.SentAtUnixMS)
	return out
}

func DecodePingRequest(payload []byte) (PingRequest, error) {
	var result PingRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.SentAtUnixMS = int64(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return PingRequest{}, err
	}
	return result, nil
}

func EncodePingResponse(message PingResponse) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Message)
	out = appendInt64Field(out, 2, message.ServerTimeUnixMS)
	out = appendInt32Field(out, 3, message.PayloadSize)
	return out
}

func DecodePingResponse(payload []byte) (PingResponse, error) {
	var result PingResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.ServerTimeUnixMS = int64(number)
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.PayloadSize = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return PingResponse{}, err
	}
	return result, nil
}

func encodeRPCError(message RPCError) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Code)
	out = appendStringField(out, 2, message.Message)
	out = appendStringField(out, 3, message.Details)
	return out
}

func decodeRPCError(payload []byte) (RPCError, error) {
	var result RPCError
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Code = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Details = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return RPCError{}, err
	}
	return result, nil
}

func appendStringField(out []byte, field protowire.Number, value string) []byte {
	if value == "" {
		return out
	}
	out = protowire.AppendTag(out, field, protowire.BytesType)
	return protowire.AppendString(out, value)
}

func appendBytesField(out []byte, field protowire.Number, value []byte) []byte {
	if len(value) == 0 {
		return out
	}
	out = protowire.AppendTag(out, field, protowire.BytesType)
	return protowire.AppendBytes(out, value)
}

func appendInt32Field(out []byte, field protowire.Number, value int32) []byte {
	if value == 0 {
		return out
	}
	out = protowire.AppendTag(out, field, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func appendInt64Field(out []byte, field protowire.Number, value int64) []byte {
	if value == 0 {
		return out
	}
	out = protowire.AppendTag(out, field, protowire.VarintType)
	return protowire.AppendVarint(out, uint64(value))
}

func consumeFields(payload []byte, visit func(protowire.Number, protowire.Type, []byte) error) error {
	for len(payload) > 0 {
		field, wireType, tagBytes := protowire.ConsumeTag(payload)
		if tagBytes < 0 {
			return protowire.ParseError(tagBytes)
		}
		payload = payload[tagBytes:]

		valueBytes := protowire.ConsumeFieldValue(field, wireType, payload)
		if valueBytes < 0 {
			return protowire.ParseError(valueBytes)
		}
		if err := visit(field, wireType, payload[:valueBytes]); err != nil {
			return fmt.Errorf("field %d: %w", field, err)
		}
		payload = payload[valueBytes:]
	}
	return nil
}

func consumeStringValue(wireType protowire.Type, payload []byte) (string, error) {
	if wireType != protowire.BytesType {
		return "", fmt.Errorf("expected bytes wire type, got %v", wireType)
	}
	value, bytesRead := protowire.ConsumeString(payload)
	if bytesRead < 0 {
		return "", protowire.ParseError(bytesRead)
	}
	return value, nil
}

func consumeBytesValue(wireType protowire.Type, payload []byte) ([]byte, error) {
	if wireType != protowire.BytesType {
		return nil, fmt.Errorf("expected bytes wire type, got %v", wireType)
	}
	value, bytesRead := protowire.ConsumeBytes(payload)
	if bytesRead < 0 {
		return nil, protowire.ParseError(bytesRead)
	}
	return value, nil
}

func consumeVarintValue(wireType protowire.Type, payload []byte) (uint64, error) {
	if wireType != protowire.VarintType {
		return 0, fmt.Errorf("expected varint wire type, got %v", wireType)
	}
	value, bytesRead := protowire.ConsumeVarint(payload)
	if bytesRead < 0 {
		return 0, protowire.ParseError(bytesRead)
	}
	return value, nil
}
