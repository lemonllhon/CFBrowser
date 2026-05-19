package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ant-chrome/backend/internal/browser"

	"github.com/google/uuid"
)

type extensionManifestInfo struct {
	Name            string
	Version         string
	Description     string
	ManifestVersion int
	ManifestJSON    string
}

// BrowserExtensionList 获取扩展插件列表。
func (a *App) BrowserExtensionList() ([]BrowserExtension, error) {
	if err := a.syncExtensionLibraryDirectories(); err != nil {
		return nil, err
	}
	dao, err := a.extensionDAO()
	if err != nil {
		return nil, err
	}
	return dao.List()
}

// BrowserExtensionGet 获取扩展插件详情。
func (a *App) BrowserExtensionGet(extensionId string) (*BrowserExtension, error) {
	if err := a.syncExtensionLibraryDirectories(); err != nil {
		return nil, err
	}
	dao, err := a.extensionDAO()
	if err != nil {
		return nil, err
	}
	extensionId = strings.TrimSpace(extensionId)
	if extensionId == "" {
		return nil, fmt.Errorf("扩展插件 ID 不能为空")
	}
	return dao.Get(extensionId)
}

// BrowserExtensionDelete 删除未被实例使用的扩展插件。
func (a *App) BrowserExtensionDelete(extensionId string) error {
	dao, err := a.extensionDAO()
	if err != nil {
		return err
	}
	extensionId = strings.TrimSpace(extensionId)
	if extensionId == "" {
		return fmt.Errorf("扩展插件 ID 不能为空")
	}

	extension, err := dao.Get(extensionId)
	if err != nil {
		return err
	}
	count, err := dao.CountBindings(extensionId)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("扩展插件已绑定 %d 个实例，阶段 1 暂不允许直接删除", count)
	}

	installDir := strings.TrimSpace(extension.InstallDir)
	safeDir := ""
	if installDir != "" {
		absInstallDir := a.resolveAppPath(installDir)
		safeDir, err = a.safeExtensionLibraryDir(absInstallDir)
		if err != nil {
			return err
		}
	}

	if err := dao.Delete(extensionId); err != nil {
		return err
	}
	if safeDir != "" {
		if err := os.RemoveAll(safeDir); err != nil {
			return fmt.Errorf("扩展记录已删除，但插件目录删除失败: %w", err)
		}
	}
	return nil
}

func (a *App) extensionDAO() (browser.ExtensionDAO, error) {
	if a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil {
		return nil, fmt.Errorf("扩展插件服务未初始化")
	}
	return a.browserMgr.ExtensionDAO, nil
}

func (a *App) syncExtensionLibraryDirectories() error {
	dao, err := a.extensionDAO()
	if err != nil {
		return err
	}
	libraryRoot := a.extensionLibraryRoot()
	if err := os.MkdirAll(libraryRoot, 0755); err != nil {
		return fmt.Errorf("创建扩展插件库目录失败: %w", err)
	}

	existing, err := dao.List()
	if err != nil {
		return err
	}
	knownDirs := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		if dir := strings.TrimSpace(item.InstallDir); dir != "" {
			if abs, err := filepath.Abs(a.resolveAppPath(dir)); err == nil {
				knownDirs[filepath.Clean(abs)] = struct{}{}
			}
		}
	}

	entries, err := os.ReadDir(libraryRoot)
	if err != nil {
		return fmt.Errorf("读取扩展插件库目录失败: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(libraryRoot, entry.Name())
		cleanDir := filepath.Clean(dir)
		if _, ok := knownDirs[cleanDir]; ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(cleanDir, "manifest.json")); err != nil {
			continue
		}
		info, err := readExtensionManifest(cleanDir)
		if err != nil {
			continue
		}
		now := time.Now().Format(time.RFC3339)
		extensionId := strings.TrimSpace(entry.Name())
		if extensionId == "" {
			extensionId = uuid.NewString()
		}
		if _, err := dao.Get(extensionId); err == nil {
			extensionId = uuid.NewString()
		}
		installDir, err := filepath.Rel(a.appStateRootAbs(), cleanDir)
		if err != nil || strings.HasPrefix(installDir, "..") {
			installDir = cleanDir
		}
		extension := browser.Extension{
			ExtensionId:     extensionId,
			Name:            info.Name,
			Version:         info.Version,
			ManifestVersion: info.ManifestVersion,
			Description:     info.Description,
			SourceType:      "directory",
			InstallDir:      installDir,
			ManifestJSON:    info.ManifestJSON,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := dao.Upsert(extension); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) extensionLibraryRoot() string {
	return a.resolveAppPath(filepath.Join("data", "extensions", "library"))
}

func (a *App) safeExtensionLibraryDir(target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", nil
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("解析扩展目录失败: %w", err)
	}
	absRoot, err := filepath.Abs(a.extensionLibraryRoot())
	if err != nil {
		return "", fmt.Errorf("解析扩展库目录失败: %w", err)
	}
	cleanTarget := filepath.Clean(absTarget)
	cleanRoot := filepath.Clean(absRoot)
	if strings.EqualFold(cleanTarget, cleanRoot) {
		return "", fmt.Errorf("拒绝删除扩展库根目录: %s", cleanTarget)
	}
	if !isPathInside(cleanTarget, cleanRoot) {
		return "", fmt.Errorf("扩展目录不在扩展库目录下: %s", cleanTarget)
	}
	return cleanTarget, nil
}

func readExtensionManifest(extensionDir string) (extensionManifestInfo, error) {
	manifestPath := filepath.Join(extensionDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return extensionManifestInfo{}, fmt.Errorf("读取 manifest.json 失败: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return extensionManifestInfo{}, fmt.Errorf("解析 manifest.json 失败: %w", err)
	}

	return extensionManifestInfo{
		Name:            manifestString(payload["name"], filepath.Base(extensionDir)),
		Version:         manifestString(payload["version"], ""),
		Description:     manifestString(payload["description"], ""),
		ManifestVersion: manifestInt(payload["manifest_version"]),
		ManifestJSON:    string(data),
	}, nil
}

func manifestString(value interface{}, fallback string) string {
	switch v := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func manifestInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}
