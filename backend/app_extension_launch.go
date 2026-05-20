package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ant-chrome/backend/internal/browser"
)

func (a *App) buildExtensionLaunchArg(profileId string, profileArgs []string, extraArgs []string) (string, int, error) {
	dirs, err := a.extensionLaunchDirsForProfile(profileId)
	if err != nil {
		return "", 0, err
	}
	dirs = append(dirs, extractLoadExtensionDirs(profileArgs, a.resolveAppPath)...)
	dirs = append(dirs, extractLoadExtensionDirs(extraArgs, a.resolveAppPath)...)
	dirs = uniqueCleanPaths(dirs)
	if len(dirs) == 0 {
		return "", 0, nil
	}
	return "--load-extension=" + strings.Join(dirs, ","), len(dirs), nil
}

func (a *App) extensionLaunchDirsForProfile(profileId string) ([]string, error) {
	if a.browserMgr == nil || a.browserMgr.ExtensionDAO == nil {
		return nil, nil
	}
	dao, err := a.extensionDAO()
	if err != nil {
		return nil, err
	}
	bindings, err := dao.ListBindingsByProfile(profileId)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	dirs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}
		extension, err := dao.Get(binding.ExtensionId)
		if err != nil {
			return nil, err
		}
		dir, err := a.extensionDirForBinding(binding, extension)
		if err != nil {
			return nil, err
		}
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

func (a *App) extensionDirForBinding(binding browser.ExtensionBinding, extension *browser.Extension) (string, error) {
	if extension == nil {
		return "", fmt.Errorf("扩展插件不存在: %s", binding.ExtensionId)
	}
	mode := normalizeExtensionBindingMode(binding.Mode)
	rawDir := strings.TrimSpace(extension.InstallDir)
	if mode == "exclusive" {
		rawDir = strings.TrimSpace(binding.ExclusiveDir)
		if rawDir == "" {
			return "", fmt.Errorf("扩展插件 %s 的独享目录为空，请重新保存绑定", binding.ExtensionId)
		}
	}
	if rawDir == "" {
		return "", fmt.Errorf("扩展插件 %s 的安装目录为空", binding.ExtensionId)
	}
	absDir := ""
	var err error
	if mode == "exclusive" {
		absDir, err = a.safeExtensionExclusiveDir(rawDir)
	} else {
		absDir, err = a.safeExtensionLibraryDir(a.resolveAppPath(rawDir))
	}
	if err != nil {
		return "", err
	}
	absDir = filepath.Clean(absDir)
	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("扩展插件目录不可访问: %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("扩展插件路径不是目录: %s", absDir)
	}
	if _, err := os.Stat(filepath.Join(absDir, "manifest.json")); err != nil {
		return "", fmt.Errorf("扩展插件目录缺少 manifest.json: %s", absDir)
	}
	return absDir, nil
}

func extractLoadExtensionDirs(args []string, resolvePath func(string) string) []string {
	if len(args) == 0 {
		return nil
	}
	dirs := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		lower := strings.ToLower(arg)
		value := ""
		if lower == "--load-extension" {
			if i+1 < len(args) {
				value = strings.TrimSpace(args[i+1])
				i++
			}
		} else if strings.HasPrefix(lower, "--load-extension=") {
			value = strings.TrimSpace(arg[len("--load-extension="):])
		}
		if value == "" {
			continue
		}
		for _, item := range strings.Split(value, ",") {
			dir := strings.TrimSpace(strings.Trim(item, `"`))
			if dir == "" {
				continue
			}
			if resolvePath != nil {
				dir = resolvePath(dir)
			}
			dirs = append(dirs, filepath.Clean(dir))
		}
	}
	return dirs
}

func uniqueCleanPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.Clean(strings.TrimSpace(path))
		if cleaned == "." || cleaned == "" {
			continue
		}
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}
