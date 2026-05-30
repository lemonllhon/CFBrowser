package backend

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"ant-chrome/backend/internal/browser"
)

type extensionSharedDataKind string

const (
	extensionSharedDataDir  extensionSharedDataKind = "dir"
	extensionSharedDataFile extensionSharedDataKind = "file"
)

type extensionSharedDataPath struct {
	profileRel string
	sharedRel  string
	kind       extensionSharedDataKind
}

func (a *App) prepareSharedExtensionDataForProfile(profile *BrowserProfile) error {
	if a == nil || a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil || profile == nil {
		return nil
	}
	dao, err := a.extensionDAO()
	if err != nil {
		return err
	}
	bindings, err := dao.ListBindingsByProfile(profile.ProfileId)
	if err != nil {
		return err
	}
	if len(bindings) == 0 {
		return nil
	}

	userDataDir := a.browserMgr.ResolveUserDataDir(profile)
	if err := os.MkdirAll(filepath.Join(userDataDir, "Default"), 0755); err != nil {
		return fmt.Errorf("创建浏览器默认配置目录失败: %w", err)
	}

	for _, binding := range bindings {
		extension, err := dao.Get(binding.ExtensionId)
		if err != nil {
			return err
		}
		if binding.Enabled && normalizeExtensionBindingMode(binding.Mode) == "shared" {
			if err := a.prepareSharedExtensionDataBinding(userDataDir, binding, extension); err != nil {
				return err
			}
			continue
		}
		if err := a.materializeSharedExtensionDataBinding(userDataDir, binding, extension); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) prepareSharedExtensionDataBinding(userDataDir string, binding browser.ExtensionBinding, extension *browser.Extension) error {
	if extension == nil {
		return fmt.Errorf("扩展插件不存在: %s", binding.ExtensionId)
	}
	extensionDir, err := a.extensionDirForBinding(binding, extension)
	if err != nil {
		return err
	}
	chromeID := ""
	if id, ok, prefErr := chromeExtensionIDFromProfilePreferences(userDataDir, extensionDir); prefErr != nil {
		return prefErr
	} else if ok {
		chromeID = id
	} else {
		id, idErr := chromeExtensionIDForDirectory(extensionDir)
		if idErr != nil {
			return idErr
		}
		chromeID = id
	}

	for _, item := range sharedExtensionDataPaths(chromeID) {
		profilePath := filepath.Join(userDataDir, "Default", filepath.FromSlash(item.profileRel))
		sharedPath := filepath.Join(a.extensionSharedDataDir(extension.ExtensionId), chromeID, filepath.FromSlash(item.sharedRel))
		if err := a.ensureSharedExtensionDataPath(userDataDir, profilePath, sharedPath, item.kind); err != nil {
			return fmt.Errorf("共享扩展数据目录失败（%s/%s）: %w", extension.Name, item.profileRel, err)
		}
	}
	return nil
}

func (a *App) materializeSharedExtensionDataBinding(userDataDir string, binding browser.ExtensionBinding, extension *browser.Extension) error {
	if extension == nil {
		return fmt.Errorf("扩展插件不存在: %s", binding.ExtensionId)
	}
	chromeIDs, err := a.chromeExtensionIDsForSharedData(binding, extension)
	if err != nil {
		return err
	}
	for _, chromeID := range chromeIDs {
		for _, item := range sharedExtensionDataPaths(chromeID) {
			profilePath := filepath.Join(userDataDir, "Default", filepath.FromSlash(item.profileRel))
			sharedPath := filepath.Join(a.extensionSharedDataDir(extension.ExtensionId), chromeID, filepath.FromSlash(item.sharedRel))
			if err := a.materializeSharedExtensionDataPath(userDataDir, profilePath, sharedPath, item.kind); err != nil {
				return fmt.Errorf("恢复扩展数据目录失败（%s/%s）: %w", extension.Name, item.profileRel, err)
			}
		}
	}
	return nil
}

func (a *App) chromeExtensionIDsForSharedData(binding browser.ExtensionBinding, extension *browser.Extension) ([]string, error) {
	ids := make([]string, 0, 2)
	if libraryDir, err := a.extensionContentDir(extension); err == nil {
		id, idErr := chromeExtensionIDForDirectory(libraryDir)
		if idErr != nil {
			return nil, idErr
		}
		ids = appendUniqueString(ids, id)
	}

	if normalizeExtensionBindingMode(binding.Mode) == "exclusive" && strings.TrimSpace(binding.ExclusiveDir) != "" {
		exclusiveDir, err := a.safeExtensionExclusiveDir(binding.ExclusiveDir)
		if err != nil {
			if binding.Enabled {
				return nil, err
			}
			return ids, nil
		}
		if id, err := chromeExtensionIDForDirectory(exclusiveDir); err == nil {
			ids = appendUniqueString(ids, id)
		}
	}
	if entries, err := os.ReadDir(a.extensionSharedDataDir(binding.ExtensionId)); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && isChromeExtensionID(entry.Name()) {
				ids = appendUniqueString(ids, entry.Name())
			}
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return ids, nil
}

func sharedExtensionDataPaths(chromeID string) []extensionSharedDataPath {
	return []extensionSharedDataPath{
		{profileRel: "Local Extension Settings/" + chromeID, sharedRel: "local-extension-settings", kind: extensionSharedDataDir},
		{profileRel: "Sync Extension Settings/" + chromeID, sharedRel: "sync-extension-settings", kind: extensionSharedDataDir},
		{profileRel: "IndexedDB/chrome-extension_" + chromeID + "_0.indexeddb.leveldb", sharedRel: "indexeddb", kind: extensionSharedDataDir},
		{profileRel: "File System/chrome-extension_" + chromeID + "_0", sharedRel: "file-system", kind: extensionSharedDataDir},
		{profileRel: "databases/chrome-extension_" + chromeID + "_0", sharedRel: "databases", kind: extensionSharedDataDir},
		{profileRel: "Extension Rules/" + chromeID, sharedRel: "extension-rules", kind: extensionSharedDataDir},
		{profileRel: "Extension Scripts/" + chromeID, sharedRel: "extension-scripts", kind: extensionSharedDataDir},
		{profileRel: "DNR Extension Rules/" + chromeID, sharedRel: "dnr-extension-rules", kind: extensionSharedDataDir},
		{profileRel: "Managed Extension Settings/" + chromeID, sharedRel: "managed-extension-settings", kind: extensionSharedDataDir},
		{profileRel: "Local Storage/chrome-extension_" + chromeID + "_0.localstorage", sharedRel: "local-storage/chrome-extension_" + chromeID + "_0.localstorage", kind: extensionSharedDataFile},
		{profileRel: "Local Storage/chrome-extension_" + chromeID + "_0.localstorage-journal", sharedRel: "local-storage/chrome-extension_" + chromeID + "_0.localstorage-journal", kind: extensionSharedDataFile},
	}
}

func (a *App) ensureSharedExtensionDataPath(userDataDir string, profilePath string, sharedPath string, kind extensionSharedDataKind) error {
	if kind == extensionSharedDataFile {
		return a.ensureSharedExtensionDataFileLink(userDataDir, profilePath, sharedPath)
	}
	return a.ensureSharedExtensionDataDirLink(userDataDir, profilePath, sharedPath)
}

func (a *App) ensureSharedExtensionDataDirLink(userDataDir string, profilePath string, sharedPath string) error {
	cleanProfile, err := filepath.Abs(profilePath)
	if err != nil {
		return err
	}
	cleanUserData, err := filepath.Abs(userDataDir)
	if err != nil {
		return err
	}
	if !isPathInside(filepath.Clean(cleanProfile), filepath.Clean(cleanUserData)) {
		return fmt.Errorf("实例扩展数据目录不在用户数据目录下: %s", cleanProfile)
	}

	cleanShared, err := filepath.Abs(sharedPath)
	if err != nil {
		return err
	}
	sharedRoot, err := filepath.Abs(a.extensionSharedDataRoot())
	if err != nil {
		return err
	}
	if !isPathInside(filepath.Clean(cleanShared), filepath.Clean(sharedRoot)) {
		return fmt.Errorf("共享扩展数据目录不在允许范围内: %s", cleanShared)
	}

	if err := os.MkdirAll(cleanShared, 0755); err != nil {
		return err
	}
	if linkedToTarget(cleanProfile, cleanShared) {
		return nil
	}

	if info, err := os.Lstat(cleanProfile); err == nil {
		if info.IsDir() {
			if empty, _ := isDirEmpty(cleanShared); empty {
				if err := copyDirContents(cleanProfile, cleanShared); err != nil {
					return err
				}
			}
		}
		if err := os.RemoveAll(cleanProfile); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cleanProfile), 0755); err != nil {
		return err
	}
	if err := os.Symlink(cleanShared, cleanProfile); err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		if err := createWindowsJunction(cleanProfile, cleanShared); err == nil {
			return nil
		}
	}
	return os.Symlink(cleanShared, cleanProfile)
}

func (a *App) ensureSharedExtensionDataFileLink(userDataDir string, profilePath string, sharedPath string) error {
	cleanProfile, cleanShared, err := a.cleanSharedExtensionDataPaths(userDataDir, profilePath, sharedPath)
	if err != nil {
		return err
	}
	if linkedToTarget(cleanProfile, cleanShared) {
		return nil
	}

	profileInfo, profileErr := os.Lstat(cleanProfile)
	sharedInfo, sharedErr := os.Stat(cleanShared)
	if os.IsNotExist(profileErr) && os.IsNotExist(sharedErr) {
		return nil
	}
	if sharedErr != nil && !os.IsNotExist(sharedErr) {
		return sharedErr
	}

	if profileErr == nil {
		if profileInfo.IsDir() {
			return fmt.Errorf("实例扩展数据文件位置已存在目录: %s", cleanProfile)
		}
		if sharedErr != nil || profileFileShouldReplaceShared(profileInfo, sharedInfo) {
			if err := os.MkdirAll(filepath.Dir(cleanShared), 0755); err != nil {
				return err
			}
			if err := copyFileContents(cleanProfile, cleanShared); err != nil {
				return err
			}
		}
		if err := os.RemoveAll(cleanProfile); err != nil {
			return err
		}
	} else if !os.IsNotExist(profileErr) {
		return profileErr
	}

	if _, err := os.Stat(cleanShared); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cleanProfile), 0755); err != nil {
		return err
	}
	if err := os.Symlink(cleanShared, cleanProfile); err == nil {
		return nil
	}
	if err := os.Link(cleanShared, cleanProfile); err == nil {
		return nil
	}
	return copyFileContents(cleanShared, cleanProfile)
}

func (a *App) materializeSharedExtensionDataPath(userDataDir string, profilePath string, sharedPath string, kind extensionSharedDataKind) error {
	cleanProfile, cleanShared, err := a.cleanSharedExtensionDataPaths(userDataDir, profilePath, sharedPath)
	if err != nil {
		return err
	}
	if !linkedToTarget(cleanProfile, cleanShared) {
		return nil
	}
	if kind == extensionSharedDataFile {
		return materializeSharedExtensionDataFile(cleanProfile, cleanShared)
	}
	return materializeSharedExtensionDataDir(cleanProfile, cleanShared)
}

func (a *App) cleanSharedExtensionDataPaths(userDataDir string, profilePath string, sharedPath string) (string, string, error) {
	cleanProfile, err := filepath.Abs(profilePath)
	if err != nil {
		return "", "", err
	}
	cleanUserData, err := filepath.Abs(userDataDir)
	if err != nil {
		return "", "", err
	}
	if !isPathInside(filepath.Clean(cleanProfile), filepath.Clean(cleanUserData)) {
		return "", "", fmt.Errorf("实例扩展数据目录不在用户数据目录下: %s", cleanProfile)
	}

	cleanShared, err := filepath.Abs(sharedPath)
	if err != nil {
		return "", "", err
	}
	sharedRoot, err := filepath.Abs(a.extensionSharedDataRoot())
	if err != nil {
		return "", "", err
	}
	if !isPathInside(filepath.Clean(cleanShared), filepath.Clean(sharedRoot)) {
		return "", "", fmt.Errorf("共享扩展数据目录不在允许范围内: %s", cleanShared)
	}
	return filepath.Clean(cleanProfile), filepath.Clean(cleanShared), nil
}

func materializeSharedExtensionDataDir(profilePath string, sharedPath string) error {
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp(filepath.Dir(profilePath), ".shared-extension-")
	if err != nil {
		return err
	}
	_ = os.RemoveAll(tmpDir)
	if _, err := os.Stat(sharedPath); err == nil {
		if err := copyDirContents(sharedPath, tmpDir); err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			_ = os.RemoveAll(tmpDir)
			return err
		}
	} else {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := os.RemoveAll(profilePath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := os.Rename(tmpDir, profilePath); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return nil
}

func materializeSharedExtensionDataFile(profilePath string, sharedPath string) error {
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(profilePath), ".shared-extension-")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if closeErr := tmpFile.Close(); closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := copyFileContents(sharedPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.RemoveAll(profilePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, profilePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func linkedToTarget(linkPath string, targetPath string) bool {
	if linkInfo, err := os.Stat(linkPath); err == nil {
		if targetInfo, targetErr := os.Stat(targetPath); targetErr == nil && os.SameFile(linkInfo, targetInfo) {
			return true
		}
	}
	resolvedLink, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		return false
	}
	resolvedTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return false
	}
	linkAbs, err := filepath.Abs(resolvedLink)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(resolvedTarget)
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(linkAbs), filepath.Clean(targetAbs))
}

func profileFileShouldReplaceShared(profileInfo os.FileInfo, sharedInfo os.FileInfo) bool {
	if sharedInfo == nil {
		return true
	}
	return profileInfo.ModTime().After(sharedInfo.ModTime())
}

func isDirEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func createWindowsJunction(linkPath string, targetPath string) error {
	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (a *App) extensionSharedDataRoot() string {
	return a.resolveAppPath(filepath.Join("data", "extensions", "shared-data"))
}

func (a *App) extensionSharedDataDir(extensionId string) string {
	return filepath.Join(a.extensionSharedDataRoot(), sanitizeExtensionPackageName(extensionId))
}

func copyFileContents(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func chromeExtensionIDForDirectory(extensionDir string) (string, error) {
	manifestPath := filepath.Join(extensionDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("读取扩展 manifest 失败: %w", err)
	}
	var manifest struct {
		Key string `json:"key"`
	}
	_ = json.Unmarshal(data, &manifest)
	key := strings.TrimSpace(manifest.Key)
	if key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return "", fmt.Errorf("解析扩展 manifest key 失败: %w", err)
		}
		return chromeExtensionIDFromBytes(decoded), nil
	}

	absDir, err := filepath.Abs(extensionDir)
	if err != nil {
		return "", err
	}
	return chromeExtensionIDFromBytes([]byte(filepath.Clean(absDir))), nil
}

func chromeExtensionIDFromBytes(data []byte) string {
	sum := sha256.Sum256(data)
	var builder strings.Builder
	builder.Grow(32)
	for _, b := range sum[:16] {
		builder.WriteByte('a' + (b >> 4))
		builder.WriteByte('a' + (b & 0x0f))
	}
	return builder.String()
}
