package protoipc

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	MethodBrowserProxyList                      = "trace.browser.ProxyList"
	MethodBrowserProxyGroupList                 = "trace.browser.ProxyGroupList"
	MethodBrowserProxySave                      = "trace.browser.ProxySave"
	MethodBrowserProxyFetchClashByURL           = "trace.browser.ProxyFetchClashByURL"
	MethodBrowserProxyValidateConfig            = "trace.browser.ProxyValidateConfig"
	MethodBrowserProxyTestConnectivity          = "trace.browser.ProxyTestConnectivity"
	MethodBrowserProxyTestRealConnectivity      = "trace.browser.ProxyTestRealConnectivity"
	MethodBrowserProxyTestSpeed                 = "trace.browser.ProxyTestSpeed"
	MethodBrowserProxyBatchTestSpeed            = "trace.browser.ProxyBatchTestSpeed"
	MethodBrowserProxyPreviewBatchTestSpeed     = "trace.browser.ProxyPreviewBatchTestSpeed"
	MethodBrowserProxyCheckIPHealth             = "trace.browser.ProxyCheckIPHealth"
	MethodBrowserProxyBatchCheckIPHealth        = "trace.browser.ProxyBatchCheckIPHealth"
	MethodBrowserProxyPreviewBatchCheckIPHealth = "trace.browser.ProxyPreviewBatchCheckIPHealth"
)

type BrowserProxy struct {
	ProxyID                string
	ProxyName              string
	ProxyConfig            string
	DNSServers             string
	GroupName              string
	SortOrder              int32
	SourceID               string
	SourceURL              string
	SourceNamePrefix       string
	SourceFilterJSON       string
	SourceAutoRefresh      bool
	SourceRefreshIntervalM int32
	SourceLastRefreshAt    string
	LastLatencyMS          int64
	LastTestOK             bool
	LastTestedAt           string
	LastIPHealthJSON       string
}

type BrowserProxyListRequest struct {
	GroupName string
}

type BrowserProxyListResponse struct {
	Proxies []BrowserProxy
}

type BrowserProxyGroupListResponse struct {
	Groups []string
}

type BrowserProxySaveRequest struct {
	Proxies []BrowserProxy
}

type BrowserProxyFetchClashByURLRequest struct {
	URL string
}

type BrowserProxyFetchClashByURLResponse struct {
	URL            string
	Content        string
	ProxyCount     int32
	DNSServers     string
	SuggestedGroup string
}

type BrowserProxyValidateConfigRequest struct {
	ProxyConfig string
	ProxyID     string
}

type BrowserProxyValidateConfigResponse struct {
	Supported bool
	ErrorMsg  string
}

type BrowserProxyTestRequest struct {
	ProxyID     string
	ProxyConfig string
}

type BrowserProxyIDListRequest struct {
	ProxyIDs    []string
	Concurrency int32
}

type BrowserProxyPreviewTestInput struct {
	ProxyID     string
	ProxyConfig string
}

type BrowserProxyPreviewTestRequest struct {
	Items       []BrowserProxyPreviewTestInput
	Concurrency int32
}

type BrowserProxyTestResult struct {
	ProxyID   string
	OK        bool
	LatencyMS int64
	Error     string
}

type BrowserProxyTestResultListResponse struct {
	Results []BrowserProxyTestResult
}

type BrowserProxyIPHealthResult struct {
	ProxyID        string
	OK             bool
	Source         string
	Error          string
	IP             string
	FraudScore     int64
	IsResidential  bool
	IsBroadcast    bool
	Country        string
	Region         string
	City           string
	AsOrganization string
	RawDataJSON    string
	UpdatedAt      string
}

type BrowserProxyIPHealthResultListResponse struct {
	Results []BrowserProxyIPHealthResult
}

func EncodeBrowserProxyListRequest(message BrowserProxyListRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.GroupName)
	return out
}

func DecodeBrowserProxyListRequest(payload []byte) (BrowserProxyListRequest, error) {
	var result BrowserProxyListRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.GroupName = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyListRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyListResponse(message BrowserProxyListResponse) []byte {
	var out []byte
	for _, item := range message.Proxies {
		out = appendBytesField(out, 1, EncodeBrowserProxy(item))
	}
	return out
}

func DecodeBrowserProxyListResponse(payload []byte) (BrowserProxyListResponse, error) {
	var result BrowserProxyListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserProxy(data)
			if err != nil {
				return err
			}
			result.Proxies = append(result.Proxies, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxyGroupListResponse(message BrowserProxyGroupListResponse) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.Groups)
	return out
}

func DecodeBrowserProxyGroupListResponse(payload []byte) (BrowserProxyGroupListResponse, error) {
	var result BrowserProxyGroupListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Groups = append(result.Groups, text)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyGroupListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxySaveRequest(message BrowserProxySaveRequest) []byte {
	var out []byte
	for _, item := range message.Proxies {
		out = appendBytesField(out, 1, EncodeBrowserProxy(item))
	}
	return out
}

func DecodeBrowserProxySaveRequest(payload []byte) (BrowserProxySaveRequest, error) {
	var result BrowserProxySaveRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserProxy(data)
			if err != nil {
				return err
			}
			result.Proxies = append(result.Proxies, item)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxySaveRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyFetchClashByURLRequest(message BrowserProxyFetchClashByURLRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.URL)
	return out
}

func DecodeBrowserProxyFetchClashByURLRequest(payload []byte) (BrowserProxyFetchClashByURLRequest, error) {
	var result BrowserProxyFetchClashByURLRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyFetchClashByURLRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyFetchClashByURLResponse(message BrowserProxyFetchClashByURLResponse) []byte {
	var out []byte
	out = appendStringField(out, 1, message.URL)
	out = appendStringField(out, 2, message.Content)
	out = appendInt32Field(out, 3, message.ProxyCount)
	out = appendStringField(out, 4, message.DNSServers)
	out = appendStringField(out, 5, message.SuggestedGroup)
	return out
}

func DecodeBrowserProxyFetchClashByURLResponse(payload []byte) (BrowserProxyFetchClashByURLResponse, error) {
	var result BrowserProxyFetchClashByURLResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.URL = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Content = text
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.ProxyCount = int32(number)
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.DNSServers = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.SuggestedGroup = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyFetchClashByURLResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxyValidateConfigRequest(message BrowserProxyValidateConfigRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyConfig)
	out = appendStringField(out, 2, message.ProxyID)
	return out
}

func DecodeBrowserProxyValidateConfigRequest(payload []byte) (BrowserProxyValidateConfigRequest, error) {
	var result BrowserProxyValidateConfigRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyValidateConfigRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyValidateConfigResponse(message BrowserProxyValidateConfigResponse) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Supported)
	out = appendStringField(out, 2, message.ErrorMsg)
	return out
}

func DecodeBrowserProxyValidateConfigResponse(payload []byte) (BrowserProxyValidateConfigResponse, error) {
	var result BrowserProxyValidateConfigResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			value, err := consumeBoolValue(wireType, value)
			result.Supported = value
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ErrorMsg = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyValidateConfigResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxyTestRequest(message BrowserProxyTestRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyID)
	out = appendStringField(out, 2, message.ProxyConfig)
	return out
}

func DecodeBrowserProxyTestRequest(payload []byte) (BrowserProxyTestRequest, error) {
	var result BrowserProxyTestRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyTestRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyIDListRequest(message BrowserProxyIDListRequest) []byte {
	var out []byte
	out = appendRepeatedStringField(out, 1, message.ProxyIDs)
	out = appendInt32Field(out, 2, message.Concurrency)
	return out
}

func DecodeBrowserProxyIDListRequest(payload []byte) (BrowserProxyIDListRequest, error) {
	var result BrowserProxyIDListRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyIDs = append(result.ProxyIDs, text)
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Concurrency = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyIDListRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyPreviewTestRequest(message BrowserProxyPreviewTestRequest) []byte {
	var out []byte
	for _, item := range message.Items {
		out = appendBytesField(out, 1, EncodeBrowserProxyPreviewTestInput(item))
	}
	out = appendInt32Field(out, 2, message.Concurrency)
	return out
}

func DecodeBrowserProxyPreviewTestRequest(payload []byte) (BrowserProxyPreviewTestRequest, error) {
	var result BrowserProxyPreviewTestRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserProxyPreviewTestInput(data)
			if err != nil {
				return err
			}
			result.Items = append(result.Items, item)
			return nil
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Concurrency = int32(number)
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyPreviewTestRequest{}, err
	}
	return result, nil
}

func EncodeBrowserProxyPreviewTestInput(message BrowserProxyPreviewTestInput) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyID)
	out = appendStringField(out, 2, message.ProxyConfig)
	return out
}

func DecodeBrowserProxyPreviewTestInput(payload []byte) (BrowserProxyPreviewTestInput, error) {
	var result BrowserProxyPreviewTestInput
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyPreviewTestInput{}, err
	}
	return result, nil
}

func EncodeBrowserProxyTestResult(message BrowserProxyTestResult) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyID)
	out = appendBoolField(out, 2, message.OK)
	out = appendInt64Field(out, 3, message.LatencyMS)
	out = appendStringField(out, 4, message.Error)
	return out
}

func DecodeBrowserProxyTestResult(payload []byte) (BrowserProxyTestResult, error) {
	var result BrowserProxyTestResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.OK = value
			return err
		case 3:
			number, err := consumeVarintValue(wireType, value)
			result.LatencyMS = int64(number)
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.Error = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyTestResult{}, err
	}
	return result, nil
}

func EncodeBrowserProxyTestResultListResponse(message BrowserProxyTestResultListResponse) []byte {
	var out []byte
	for _, item := range message.Results {
		out = appendBytesField(out, 1, EncodeBrowserProxyTestResult(item))
	}
	return out
}

func DecodeBrowserProxyTestResultListResponse(payload []byte) (BrowserProxyTestResultListResponse, error) {
	var result BrowserProxyTestResultListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserProxyTestResult(data)
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
		return BrowserProxyTestResultListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxyIPHealthResult(message BrowserProxyIPHealthResult) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyID)
	out = appendBoolField(out, 2, message.OK)
	out = appendStringField(out, 3, message.Source)
	out = appendStringField(out, 4, message.Error)
	out = appendStringField(out, 5, message.IP)
	out = appendInt64Field(out, 6, message.FraudScore)
	out = appendBoolField(out, 7, message.IsResidential)
	out = appendBoolField(out, 8, message.IsBroadcast)
	out = appendStringField(out, 9, message.Country)
	out = appendStringField(out, 10, message.Region)
	out = appendStringField(out, 11, message.City)
	out = appendStringField(out, 12, message.AsOrganization)
	out = appendStringField(out, 13, message.RawDataJSON)
	out = appendStringField(out, 14, message.UpdatedAt)
	return out
}

func DecodeBrowserProxyIPHealthResult(payload []byte) (BrowserProxyIPHealthResult, error) {
	var result BrowserProxyIPHealthResult
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 2:
			value, err := consumeBoolValue(wireType, value)
			result.OK = value
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.Source = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.Error = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.IP = text
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.FraudScore = int64(number)
			return err
		case 7:
			value, err := consumeBoolValue(wireType, value)
			result.IsResidential = value
			return err
		case 8:
			value, err := consumeBoolValue(wireType, value)
			result.IsBroadcast = value
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.Country = text
			return err
		case 10:
			text, err := consumeStringValue(wireType, value)
			result.Region = text
			return err
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.City = text
			return err
		case 12:
			text, err := consumeStringValue(wireType, value)
			result.AsOrganization = text
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.RawDataJSON = text
			return err
		case 14:
			text, err := consumeStringValue(wireType, value)
			result.UpdatedAt = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxyIPHealthResult{}, err
	}
	return result, nil
}

func EncodeBrowserProxyIPHealthResultListResponse(message BrowserProxyIPHealthResultListResponse) []byte {
	var out []byte
	for _, item := range message.Results {
		out = appendBytesField(out, 1, EncodeBrowserProxyIPHealthResult(item))
	}
	return out
}

func DecodeBrowserProxyIPHealthResultListResponse(payload []byte) (BrowserProxyIPHealthResultListResponse, error) {
	var result BrowserProxyIPHealthResultListResponse
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			item, err := DecodeBrowserProxyIPHealthResult(data)
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
		return BrowserProxyIPHealthResultListResponse{}, err
	}
	return result, nil
}

func EncodeBrowserProxy(message BrowserProxy) []byte {
	var out []byte
	out = appendStringField(out, 1, message.ProxyID)
	out = appendStringField(out, 2, message.ProxyName)
	out = appendStringField(out, 3, message.ProxyConfig)
	out = appendStringField(out, 4, message.DNSServers)
	out = appendStringField(out, 5, message.GroupName)
	out = appendInt32Field(out, 6, message.SortOrder)
	out = appendStringField(out, 7, message.SourceID)
	out = appendStringField(out, 8, message.SourceURL)
	out = appendStringField(out, 9, message.SourceNamePrefix)
	out = appendStringField(out, 10, message.SourceFilterJSON)
	out = appendBoolField(out, 11, message.SourceAutoRefresh)
	out = appendInt32Field(out, 12, message.SourceRefreshIntervalM)
	out = appendStringField(out, 13, message.SourceLastRefreshAt)
	out = appendInt64Field(out, 14, message.LastLatencyMS)
	out = appendBoolField(out, 15, message.LastTestOK)
	out = appendStringField(out, 16, message.LastTestedAt)
	out = appendStringField(out, 17, message.LastIPHealthJSON)
	return out
}

func DecodeBrowserProxy(payload []byte) (BrowserProxy, error) {
	var result BrowserProxy
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.ProxyID = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.ProxyName = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ProxyConfig = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.DNSServers = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.GroupName = text
			return err
		case 6:
			number, err := consumeVarintValue(wireType, value)
			result.SortOrder = int32(number)
			return err
		case 7:
			text, err := consumeStringValue(wireType, value)
			result.SourceID = text
			return err
		case 8:
			text, err := consumeStringValue(wireType, value)
			result.SourceURL = text
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.SourceNamePrefix = text
			return err
		case 10:
			text, err := consumeStringValue(wireType, value)
			result.SourceFilterJSON = text
			return err
		case 11:
			value, err := consumeBoolValue(wireType, value)
			result.SourceAutoRefresh = value
			return err
		case 12:
			number, err := consumeVarintValue(wireType, value)
			result.SourceRefreshIntervalM = int32(number)
			return err
		case 13:
			text, err := consumeStringValue(wireType, value)
			result.SourceLastRefreshAt = text
			return err
		case 14:
			number, err := consumeVarintValue(wireType, value)
			result.LastLatencyMS = int64(number)
			return err
		case 15:
			value, err := consumeBoolValue(wireType, value)
			result.LastTestOK = value
			return err
		case 16:
			text, err := consumeStringValue(wireType, value)
			result.LastTestedAt = text
			return err
		case 17:
			text, err := consumeStringValue(wireType, value)
			result.LastIPHealthJSON = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return BrowserProxy{}, fmt.Errorf("decode browser proxy: %w", err)
	}
	return result, nil
}
