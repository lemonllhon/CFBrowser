package protoipc

import "google.golang.org/protobuf/encoding/protowire"

const (
	MethodAppUpdateCheck             = "trace.app.UpdateCheck"
	MethodAppUpdateDownload          = "trace.app.UpdateDownload"
	MethodAppUpdateInstallDownloaded = "trace.app.UpdateInstallDownloaded"
	MethodAppUpdateDownloadPortable  = "trace.app.UpdateDownloadPortable"
)

type AppUpdateAsset struct {
	Name        string
	Size        int64
	DownloadURL string
	Checksum    string
}

type AppUpdateInfo struct {
	CurrentVersion         string
	LatestVersion          string
	ReleaseName            string
	ReleaseURL             string
	PublishedAt            string
	Body                   string
	HasUpdate              bool
	Asset                  *AppUpdateAsset
	InstallerAsset         *AppUpdateAsset
	PortableAsset          *AppUpdateAsset
	DistributionKind       string
	RecommendedPackageKind string
	CanSelfUpdatePortable  bool
	Message                string
}

type AppUpdateDownloadRequest struct {
	Info             AppUpdateInfo
	InstallOnRestart bool
}

type AppUpdateInfoRequest struct {
	Info AppUpdateInfo
}

type AppUpdateInstallDownloadedRequest struct {
	InstallerPath string
}

type AppUpdateDownloadResult struct {
	Cancelled        bool
	Message          string
	Version          string
	InstallerPath    string
	PackagePath      string
	ExtractedPath    string
	InstallOnRestart bool
	RestartScheduled bool
	PackageKind      string
}

type AppUpdateDownloadProgress struct {
	Phase    string
	Progress int32
	Message  string
}

type AppUpdatePendingUpdate struct {
	Version            string
	InstallerPath      string
	ReleaseURL         string
	InstallOnNextStart bool
	CreatedAt          string
}

type AppUpdatePendingNotification struct {
	Version string
	Message string
}

type AppUpdatePendingInstallFailed struct {
	Version string
	Error   string
}

func EncodeAppUpdateAsset(message AppUpdateAsset) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Name)
	out = appendInt64Field(out, 2, message.Size)
	out = appendStringField(out, 3, message.DownloadURL)
	out = appendStringField(out, 4, message.Checksum)
	return out
}

func DecodeAppUpdateAsset(payload []byte) (AppUpdateAsset, error) {
	var result AppUpdateAsset
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Name = text
			return err
		case 2:
			number, err := consumeVarintValue(wireType, value)
			result.Size = int64(number)
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.DownloadURL = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.Checksum = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdateAsset{}, err
	}
	return result, nil
}

func EncodeAppUpdateInfo(message AppUpdateInfo) []byte {
	var out []byte
	out = appendStringField(out, 1, message.CurrentVersion)
	out = appendStringField(out, 2, message.LatestVersion)
	out = appendStringField(out, 3, message.ReleaseName)
	out = appendStringField(out, 4, message.ReleaseURL)
	out = appendStringField(out, 5, message.PublishedAt)
	out = appendStringField(out, 6, message.Body)
	out = appendBoolField(out, 7, message.HasUpdate)
	if message.Asset != nil {
		out = appendBytesField(out, 8, EncodeAppUpdateAsset(*message.Asset))
	}
	if message.InstallerAsset != nil {
		out = appendBytesField(out, 9, EncodeAppUpdateAsset(*message.InstallerAsset))
	}
	if message.PortableAsset != nil {
		out = appendBytesField(out, 10, EncodeAppUpdateAsset(*message.PortableAsset))
	}
	out = appendStringField(out, 11, message.DistributionKind)
	out = appendStringField(out, 12, message.RecommendedPackageKind)
	out = appendBoolField(out, 13, message.CanSelfUpdatePortable)
	out = appendStringField(out, 14, message.Message)
	return out
}

func DecodeAppUpdateInfo(payload []byte) (AppUpdateInfo, error) {
	var result AppUpdateInfo
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.CurrentVersion = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.LatestVersion = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ReleaseName = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.ReleaseURL = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.PublishedAt = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.Body = text
			return err
		case 7:
			boolValue, err := consumeBoolValue(wireType, value)
			result.HasUpdate = boolValue
			return err
		case 8, 9, 10:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			asset, err := DecodeAppUpdateAsset(data)
			if err != nil {
				return err
			}
			if field == 8 {
				result.Asset = &asset
			} else if field == 9 {
				result.InstallerAsset = &asset
			} else {
				result.PortableAsset = &asset
			}
			return nil
		case 11:
			text, err := consumeStringValue(wireType, value)
			result.DistributionKind = text
			return err
		case 12:
			text, err := consumeStringValue(wireType, value)
			result.RecommendedPackageKind = text
			return err
		case 13:
			boolValue, err := consumeBoolValue(wireType, value)
			result.CanSelfUpdatePortable = boolValue
			return err
		case 14:
			text, err := consumeStringValue(wireType, value)
			result.Message = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdateInfo{}, err
	}
	return result, nil
}

func EncodeAppUpdateDownloadRequest(message AppUpdateDownloadRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeAppUpdateInfo(message.Info))
	out = appendBoolField(out, 2, message.InstallOnRestart)
	return out
}

func DecodeAppUpdateDownloadRequest(payload []byte) (AppUpdateDownloadRequest, error) {
	var result AppUpdateDownloadRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			data, err := consumeBytesValue(wireType, value)
			if err != nil {
				return err
			}
			info, err := DecodeAppUpdateInfo(data)
			result.Info = info
			return err
		case 2:
			boolValue, err := consumeBoolValue(wireType, value)
			result.InstallOnRestart = boolValue
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdateDownloadRequest{}, err
	}
	return result, nil
}

func EncodeAppUpdateInfoRequest(message AppUpdateInfoRequest) []byte {
	var out []byte
	out = appendBytesField(out, 1, EncodeAppUpdateInfo(message.Info))
	return out
}

func DecodeAppUpdateInfoRequest(payload []byte) (AppUpdateInfoRequest, error) {
	var result AppUpdateInfoRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		if field != 1 {
			return nil
		}
		data, err := consumeBytesValue(wireType, value)
		if err != nil {
			return err
		}
		info, err := DecodeAppUpdateInfo(data)
		result.Info = info
		return err
	})
	if err != nil {
		return AppUpdateInfoRequest{}, err
	}
	return result, nil
}

func EncodeAppUpdateInstallDownloadedRequest(message AppUpdateInstallDownloadedRequest) []byte {
	var out []byte
	out = appendStringField(out, 1, message.InstallerPath)
	return out
}

func DecodeAppUpdateInstallDownloadedRequest(payload []byte) (AppUpdateInstallDownloadedRequest, error) {
	var result AppUpdateInstallDownloadedRequest
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		if field != 1 {
			return nil
		}
		text, err := consumeStringValue(wireType, value)
		result.InstallerPath = text
		return err
	})
	if err != nil {
		return AppUpdateInstallDownloadedRequest{}, err
	}
	return result, nil
}

func EncodeAppUpdateDownloadResult(message AppUpdateDownloadResult) []byte {
	var out []byte
	out = appendBoolField(out, 1, message.Cancelled)
	out = appendStringField(out, 2, message.Message)
	out = appendStringField(out, 3, message.Version)
	out = appendStringField(out, 4, message.InstallerPath)
	out = appendStringField(out, 5, message.PackagePath)
	out = appendStringField(out, 6, message.ExtractedPath)
	out = appendBoolField(out, 7, message.InstallOnRestart)
	out = appendBoolField(out, 8, message.RestartScheduled)
	out = appendStringField(out, 9, message.PackageKind)
	return out
}

func DecodeAppUpdateDownloadResult(payload []byte) (AppUpdateDownloadResult, error) {
	var result AppUpdateDownloadResult
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
			result.Version = text
			return err
		case 4:
			text, err := consumeStringValue(wireType, value)
			result.InstallerPath = text
			return err
		case 5:
			text, err := consumeStringValue(wireType, value)
			result.PackagePath = text
			return err
		case 6:
			text, err := consumeStringValue(wireType, value)
			result.ExtractedPath = text
			return err
		case 7:
			boolValue, err := consumeBoolValue(wireType, value)
			result.InstallOnRestart = boolValue
			return err
		case 8:
			boolValue, err := consumeBoolValue(wireType, value)
			result.RestartScheduled = boolValue
			return err
		case 9:
			text, err := consumeStringValue(wireType, value)
			result.PackageKind = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdateDownloadResult{}, err
	}
	return result, nil
}

func EncodeAppUpdateDownloadProgress(message AppUpdateDownloadProgress) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Phase)
	out = appendInt32Field(out, 2, message.Progress)
	out = appendStringField(out, 3, message.Message)
	return out
}

func DecodeAppUpdateDownloadProgress(payload []byte) (AppUpdateDownloadProgress, error) {
	var result AppUpdateDownloadProgress
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
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdateDownloadProgress{}, err
	}
	return result, nil
}

func EncodeAppUpdatePendingUpdate(message AppUpdatePendingUpdate) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Version)
	out = appendStringField(out, 2, message.InstallerPath)
	out = appendStringField(out, 3, message.ReleaseURL)
	out = appendBoolField(out, 4, message.InstallOnNextStart)
	out = appendStringField(out, 5, message.CreatedAt)
	return out
}

func DecodeAppUpdatePendingUpdate(payload []byte) (AppUpdatePendingUpdate, error) {
	var result AppUpdatePendingUpdate
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Version = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.InstallerPath = text
			return err
		case 3:
			text, err := consumeStringValue(wireType, value)
			result.ReleaseURL = text
			return err
		case 4:
			boolValue, err := consumeBoolValue(wireType, value)
			result.InstallOnNextStart = boolValue
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
		return AppUpdatePendingUpdate{}, err
	}
	return result, nil
}

func EncodeAppUpdatePendingNotification(message AppUpdatePendingNotification) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Version)
	out = appendStringField(out, 2, message.Message)
	return out
}

func DecodeAppUpdatePendingNotification(payload []byte) (AppUpdatePendingNotification, error) {
	var result AppUpdatePendingNotification
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Version = text
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
		return AppUpdatePendingNotification{}, err
	}
	return result, nil
}

func EncodeAppUpdatePendingInstallFailed(message AppUpdatePendingInstallFailed) []byte {
	var out []byte
	out = appendStringField(out, 1, message.Version)
	out = appendStringField(out, 2, message.Error)
	return out
}

func DecodeAppUpdatePendingInstallFailed(payload []byte) (AppUpdatePendingInstallFailed, error) {
	var result AppUpdatePendingInstallFailed
	err := consumeFields(payload, func(field protowire.Number, wireType protowire.Type, value []byte) error {
		switch field {
		case 1:
			text, err := consumeStringValue(wireType, value)
			result.Version = text
			return err
		case 2:
			text, err := consumeStringValue(wireType, value)
			result.Error = text
			return err
		default:
			return nil
		}
	})
	if err != nil {
		return AppUpdatePendingInstallFailed{}, err
	}
	return result, nil
}
