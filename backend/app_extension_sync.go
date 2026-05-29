package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ant-chrome/backend/internal/browser"
)

// BrowserExtensionSyncDataInput 扩展插件数据同步输入。
type BrowserExtensionSyncDataInput struct {
	ExtensionId      string   `json:"extensionId"`
	SourceProfileId  string   `json:"sourceProfileId"`
	TargetProfileIds []string `json:"targetProfileIds"`
}

// BrowserExtensionSyncProfileData 将主实例中的扩展数据直接同步到副实例用户数据目录。
func (a *App) BrowserExtensionSyncProfileData(input BrowserExtensionSyncDataInput) ([]BrowserExtensionBinding, error) {
	dao, err := a.extensionDAO()
	if err != nil {
		return nil, err
	}
	extensionId := strings.TrimSpace(input.ExtensionId)
	if extensionId == "" {
		return nil, fmt.Errorf("扩展插件 ID 不能为空")
	}
	sourceProfileId := strings.TrimSpace(input.SourceProfileId)
	if sourceProfileId == "" {
		return nil, fmt.Errorf("请选择主实例")
	}
	targetProfileIds := normalizeStringList(input.TargetProfileIds)
	if len(targetProfileIds) == 0 {
		return nil, fmt.Errorf("请选择副实例")
	}
	for _, targetProfileId := range targetProfileIds {
		if strings.EqualFold(targetProfileId, sourceProfileId) {
			return nil, fmt.Errorf("主实例不能同时作为副实例")
		}
	}

	extension, err := dao.Get(extensionId)
	if err != nil {
		return nil, err
	}
	sourceProfile, err := a.requireStoppedProfileForExtensionSync(sourceProfileId, "主实例")
	if err != nil {
		return nil, err
	}
	sourceBinding, err := a.requireExtensionBindingForProfile(dao, sourceProfileId, extensionId, "主实例")
	if err != nil {
		return nil, err
	}
	sourceChromeID, err := a.chromeExtensionIDForProfileBinding(sourceBinding, extension)
	if err != nil {
		return nil, err
	}

	targets := make([]extensionDataSyncTarget, 0, len(targetProfileIds))
	for _, targetProfileId := range targetProfileIds {
		targetProfile, err := a.requireStoppedProfileForExtensionSync(targetProfileId, "副实例")
		if err != nil {
			return nil, err
		}
		targetBinding, err := a.requireExtensionBindingForProfile(dao, targetProfileId, extensionId, "副实例")
		if err != nil {
			return nil, err
		}
		if normalizeExtensionBindingMode(targetBinding.Mode) != "shared" {
			return nil, fmt.Errorf("副实例「%s」不是共享绑定，请先将绑定模式改为共享", profileDisplayName(targetProfile))
		}
		targetChromeID, err := a.chromeExtensionIDForProfileBinding(targetBinding, extension)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(sourceChromeID, targetChromeID) {
			return nil, fmt.Errorf("主实例与副实例「%s」的扩展 ID 不一致，无法安全同步数据", profileDisplayName(targetProfile))
		}
		targets = append(targets, extensionDataSyncTarget{
			profile:  targetProfile,
			chromeID: targetChromeID,
		})
	}

	sourceUserDataDir := a.browserMgr.ResolveUserDataDir(sourceProfile)
	for _, target := range targets {
		targetUserDataDir := a.browserMgr.ResolveUserDataDir(target.profile)
		if err := a.syncExtensionDataFromProfileToProfile(extension.ExtensionId, sourceUserDataDir, targetUserDataDir, sourceChromeID, target.chromeID); err != nil {
			return nil, fmt.Errorf("同步到副实例「%s」失败: %w", profileDisplayName(target.profile), err)
		}
	}
	return dao.ListBindings(extensionId)
}

type extensionDataSyncTarget struct {
	profile  *browser.Profile
	chromeID string
}

type extensionDataSyncPath struct {
	profileRel              string
	sharedRel               string
	kind                    extensionSharedDataKind
	syncSharedBacking       bool
	removeWhenSourceMissing bool
}

func (a *App) requireStoppedProfileForExtensionSync(profileId string, role string) (*browser.Profile, error) {
	profile, err := a.requireProfile(profileId)
	if err != nil {
		return nil, err
	}
	if a.isProfileRunning(profileId) {
		return nil, fmt.Errorf("%s「%s」正在运行，请先停止后再同步插件数据", role, profileDisplayName(profile))
	}
	return profile, nil
}

func (a *App) requireExtensionBindingForProfile(dao browser.ExtensionDAO, profileId string, extensionId string, role string) (browser.ExtensionBinding, error) {
	bindings, err := dao.ListBindingsByProfile(profileId)
	if err != nil {
		return browser.ExtensionBinding{}, err
	}
	for _, binding := range bindings {
		if binding.ExtensionId != extensionId {
			continue
		}
		if !binding.Enabled {
			return browser.ExtensionBinding{}, fmt.Errorf("%s未启用当前扩展插件", role)
		}
		return binding, nil
	}
	return browser.ExtensionBinding{}, fmt.Errorf("%s未绑定当前扩展插件", role)
}

func (a *App) chromeExtensionIDForProfileBinding(binding browser.ExtensionBinding, extension *browser.Extension) (string, error) {
	extensionDir, err := a.extensionDirForBinding(binding, extension)
	if err != nil {
		return "", err
	}
	return chromeExtensionIDForDirectory(extensionDir)
}

func (a *App) syncExtensionDataFromProfileToProfile(extensionID string, sourceUserDataDir string, targetUserDataDir string, sourceChromeID string, targetChromeID string) error {
	sourceBase := filepath.Join(sourceUserDataDir, "Default")
	targetBase := filepath.Join(targetUserDataDir, "Default")
	if err := os.MkdirAll(targetBase, 0755); err != nil {
		return fmt.Errorf("创建副实例默认配置目录失败: %w", err)
	}

	sourcePaths := extensionDataSyncPaths(sourceChromeID)
	targetPaths := extensionDataSyncPaths(targetChromeID)
	for i, sourceItem := range sourcePaths {
		targetItem := targetPaths[i]
		sourcePath := filepath.Join(sourceBase, filepath.FromSlash(sourceItem.profileRel))
		targetPath := filepath.Join(targetBase, filepath.FromSlash(targetItem.profileRel))
		if err := a.syncExtensionDataPath(sourceUserDataDir, targetUserDataDir, sourcePath, targetPath, sourceItem.kind, sourceItem.removeWhenSourceMissing); err != nil {
			return fmt.Errorf("%s: %w", sourceItem.profileRel, err)
		}
		if targetItem.syncSharedBacking && strings.TrimSpace(targetItem.sharedRel) != "" {
			targetSharedPath := filepath.Join(a.extensionSharedDataDir(extensionID), targetChromeID, filepath.FromSlash(targetItem.sharedRel))
			if err := a.syncExtensionDataPath(sourceUserDataDir, a.extensionSharedDataRoot(), sourcePath, targetSharedPath, sourceItem.kind, sourceItem.removeWhenSourceMissing); err != nil {
				return fmt.Errorf("%s: %w", sourceItem.profileRel, err)
			}
		}
	}
	return nil
}

func extensionDataSyncPaths(chromeID string) []extensionDataSyncPath {
	paths := make([]extensionDataSyncPath, 0, len(sharedExtensionDataPaths(chromeID))+8)
	for _, item := range sharedExtensionDataPaths(chromeID) {
		paths = append(paths, extensionDataSyncPath{
			profileRel:              item.profileRel,
			sharedRel:               item.sharedRel,
			kind:                    item.kind,
			syncSharedBacking:       true,
			removeWhenSourceMissing: true,
		})
	}
	paths = append(paths,
		extensionDataSyncPath{profileRel: "Storage/ext/" + chromeID, kind: extensionSharedDataDir, removeWhenSourceMissing: true},
		extensionDataSyncPath{profileRel: "Storage/ext/chrome-extension_" + chromeID, kind: extensionSharedDataDir, removeWhenSourceMissing: true},
		extensionDataSyncPath{profileRel: "Storage/ext/chrome-extension_" + chromeID + "_0", kind: extensionSharedDataDir, removeWhenSourceMissing: true},
		extensionDataSyncPath{profileRel: "Local Storage/leveldb", kind: extensionSharedDataDir},
		extensionDataSyncPath{profileRel: "Session Storage", kind: extensionSharedDataDir},
		extensionDataSyncPath{profileRel: "Service Worker", kind: extensionSharedDataDir},
		extensionDataSyncPath{profileRel: "Extension State", kind: extensionSharedDataDir},
	)
	return paths
}

func (a *App) syncExtensionDataPath(sourceUserDataDir string, targetUserDataDir string, sourcePath string, targetPath string, kind extensionSharedDataKind, removeWhenSourceMissing bool) error {
	sourceRoot, err := filepath.Abs(sourceUserDataDir)
	if err != nil {
		return err
	}
	targetRoot, err := filepath.Abs(targetUserDataDir)
	if err != nil {
		return err
	}
	sharedRoot, err := filepath.Abs(a.extensionSharedDataRoot())
	if err != nil {
		return err
	}
	sourceAllowedRoots := []string{filepath.Clean(sourceRoot), filepath.Clean(sharedRoot)}
	targetAllowedRoots := []string{filepath.Clean(targetRoot), filepath.Clean(sharedRoot)}

	cleanSource, sourceInfo, sourceExists, err := resolveExtensionDataSyncSource(sourcePath, sourceAllowedRoots)
	if err != nil {
		return err
	}
	cleanTarget, err := resolveExtensionDataSyncTarget(targetPath, targetAllowedRoots)
	if err != nil {
		return err
	}
	if !sourceExists {
		if !removeWhenSourceMissing {
			return nil
		}
		return removeExtensionProfileDataPath(cleanTarget)
	}
	if strings.EqualFold(filepath.Clean(cleanSource), filepath.Clean(cleanTarget)) {
		return nil
	}
	if kind == extensionSharedDataFile {
		if sourceInfo.IsDir() {
			return fmt.Errorf("主实例扩展数据应为文件，实际是目录: %s", cleanSource)
		}
		return replaceExtensionProfileDataFile(cleanSource, cleanTarget)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("主实例扩展数据应为目录，实际是文件: %s", cleanSource)
	}
	return replaceExtensionProfileDataDir(cleanSource, cleanTarget)
}

func resolveExtensionDataSyncSource(sourcePath string, allowedRoots []string) (string, os.FileInfo, bool, error) {
	cleanSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", nil, false, err
	}
	cleanSource = filepath.Clean(cleanSource)
	if !isPathInsideAny(cleanSource, allowedRoots) {
		return "", nil, false, fmt.Errorf("主实例扩展数据路径不在允许范围内: %s", cleanSource)
	}
	sourceInfo, err := os.Stat(cleanSource)
	if err != nil {
		if os.IsNotExist(err) {
			return cleanSource, nil, false, nil
		}
		return "", nil, false, err
	}
	resolvedSource := cleanSource
	if resolved, err := filepath.EvalSymlinks(cleanSource); err == nil && strings.TrimSpace(resolved) != "" {
		resolvedAbs, absErr := filepath.Abs(resolved)
		if absErr != nil {
			return "", nil, false, absErr
		}
		resolvedSource = filepath.Clean(resolvedAbs)
	}
	if !isPathInsideAny(resolvedSource, allowedRoots) {
		return "", nil, false, fmt.Errorf("主实例扩展数据链接不在允许范围内: %s", resolvedSource)
	}
	return resolvedSource, sourceInfo, true, nil
}

func resolveExtensionDataSyncTarget(targetPath string, allowedRoots []string) (string, error) {
	cleanTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}
	cleanTarget = filepath.Clean(cleanTarget)
	if !isPathInsideAny(cleanTarget, allowedRoots) {
		return "", fmt.Errorf("副实例扩展数据路径不在允许范围内: %s", cleanTarget)
	}
	if _, err := os.Lstat(cleanTarget); err != nil {
		if os.IsNotExist(err) {
			return cleanTarget, nil
		}
		return "", err
	}
	resolvedTarget := cleanTarget
	if resolved, err := filepath.EvalSymlinks(cleanTarget); err == nil && strings.TrimSpace(resolved) != "" {
		resolvedAbs, absErr := filepath.Abs(resolved)
		if absErr != nil {
			return "", absErr
		}
		resolvedTarget = filepath.Clean(resolvedAbs)
	}
	if !isPathInsideAny(resolvedTarget, allowedRoots) {
		return "", fmt.Errorf("副实例扩展数据链接不在允许范围内: %s", resolvedTarget)
	}
	return resolvedTarget, nil
}

func isPathInsideAny(child string, parents []string) bool {
	cleanChild := filepath.Clean(child)
	for _, parent := range parents {
		parent = strings.TrimSpace(parent)
		if parent == "" {
			continue
		}
		if isPathInside(cleanChild, filepath.Clean(parent)) {
			return true
		}
	}
	return false
}

func replaceExtensionProfileDataDir(sourceDir string, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp(filepath.Dir(targetDir), ".extension-sync-")
	if err != nil {
		return err
	}
	_ = os.RemoveAll(tmpDir)
	if err := copyDirContents(sourceDir, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := removeExtensionProfileDataPath(targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}
	return nil
}

func replaceExtensionProfileDataFile(sourceFile string, targetFile string) error {
	if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(targetFile), ".extension-sync-")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if closeErr := tmpFile.Close(); closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := copyFileContents(sourceFile, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := removeExtensionProfileDataPath(targetFile); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, targetFile); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func removeExtensionProfileDataPath(target string) error {
	if strings.TrimSpace(target) == "" {
		return nil
	}
	if _, err := os.Lstat(target); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(target)
}

func profileDisplayName(profile *browser.Profile) string {
	if profile == nil {
		return ""
	}
	if name := strings.TrimSpace(profile.ProfileName); name != "" {
		return name
	}
	return strings.TrimSpace(profile.ProfileId)
}
