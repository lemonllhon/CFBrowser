package backend

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"ant-chrome/backend/internal/browser"
)

type extensionSharedDataPath struct {
	profileRel string
	sharedRel  string
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
		if !binding.Enabled || normalizeExtensionBindingMode(binding.Mode) != "shared" {
			continue
		}
		extension, err := dao.Get(binding.ExtensionId)
		if err != nil {
			return err
		}
		if err := a.prepareSharedExtensionDataBinding(userDataDir, binding, extension); err != nil {
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
	chromeID, err := chromeExtensionIDForDirectory(extensionDir)
	if err != nil {
		return err
	}

	for _, item := range sharedExtensionDataPaths(chromeID) {
		profilePath := filepath.Join(userDataDir, "Default", filepath.FromSlash(item.profileRel))
		sharedPath := filepath.Join(a.extensionSharedDataRoot(), sanitizeExtensionPackageName(extension.ExtensionId), chromeID, filepath.FromSlash(item.sharedRel))
		if err := a.ensureSharedExtensionDataLink(userDataDir, profilePath, sharedPath); err != nil {
			return fmt.Errorf("共享扩展数据目录失败（%s/%s）: %w", extension.Name, item.profileRel, err)
		}
	}
	return nil
}

func sharedExtensionDataPaths(chromeID string) []extensionSharedDataPath {
	return []extensionSharedDataPath{
		{profileRel: "Local Extension Settings/" + chromeID, sharedRel: "local-extension-settings"},
		{profileRel: "Sync Extension Settings/" + chromeID, sharedRel: "sync-extension-settings"},
		{profileRel: "IndexedDB/chrome-extension_" + chromeID + "_0.indexeddb.leveldb", sharedRel: "indexeddb"},
		{profileRel: "File System/chrome-extension_" + chromeID + "_0", sharedRel: "file-system"},
		{profileRel: "databases/chrome-extension_" + chromeID + "_0", sharedRel: "databases"},
		{profileRel: "Extension Rules/" + chromeID, sharedRel: "extension-rules"},
		{profileRel: "Extension Scripts/" + chromeID, sharedRel: "extension-scripts"},
	}
}

func (a *App) ensureSharedExtensionDataLink(userDataDir string, profilePath string, sharedPath string) error {
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

func linkedToTarget(linkPath string, targetPath string) bool {
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
