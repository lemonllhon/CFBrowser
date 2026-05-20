package backend

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const PROJECT_GITHUB_URL = "https://github.com/lemon-casino/trace-browser-release/releases"

const githubLatestReleaseAPI = "https://api.github.com/repos/lemon-casino/trace-browser-release/releases/latest"

type AppUpdateAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
}

type AppUpdateInfo struct {
	CurrentVersion         string          `json:"currentVersion"`
	LatestVersion          string          `json:"latestVersion"`
	ReleaseName            string          `json:"releaseName"`
	ReleaseURL             string          `json:"releaseUrl"`
	PublishedAt            string          `json:"publishedAt"`
	Body                   string          `json:"body"`
	HasUpdate              bool            `json:"hasUpdate"`
	Asset                  *AppUpdateAsset `json:"asset,omitempty"`
	InstallerAsset         *AppUpdateAsset `json:"installerAsset,omitempty"`
	PortableAsset          *AppUpdateAsset `json:"portableAsset,omitempty"`
	DistributionKind       string          `json:"distributionKind"`
	RecommendedPackageKind string          `json:"recommendedPackageKind"`
	CanSelfUpdatePortable  bool            `json:"canSelfUpdatePortable"`
	Message                string          `json:"message"`
}

type AppUpdateDownloadResult struct {
	Cancelled        bool   `json:"cancelled"`
	Message          string `json:"message"`
	Version          string `json:"version"`
	InstallerPath    string `json:"installerPath"`
	PackagePath      string `json:"packagePath"`
	ExtractedPath    string `json:"extractedPath"`
	InstallOnRestart bool   `json:"installOnRestart"`
	RestartScheduled bool   `json:"restartScheduled"`
	PackageKind      string `json:"packageKind"`
}

type AppUpdateDownloadProgress struct {
	Phase    string `json:"phase"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

type pendingAppUpdate struct {
	Version            string `json:"version"`
	InstallerPath      string `json:"installerPath"`
	ReleaseURL         string `json:"releaseUrl"`
	InstallOnNextStart bool   `json:"installOnNextStart"`
	CreatedAt          string `json:"createdAt"`
}

type githubRelease struct {
	TagName     string               `json:"tag_name"`
	Name        string               `json:"name"`
	HTMLURL     string               `json:"html_url"`
	PublishedAt string               `json:"published_at"`
	Body        string               `json:"body"`
	Prerelease  bool                 `json:"prerelease"`
	Draft       bool                 `json:"draft"`
	Assets      []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (a *App) CheckAppUpdate() (*AppUpdateInfo, error) {
	currentVersion := a.appVersion()
	release, err := fetchLatestGithubRelease(context.Background())
	if err != nil {
		return nil, err
	}
	latestVersion := normalizeVersion(release.TagName)
	if latestVersion == "" {
		latestVersion = normalizeVersion(release.Name)
	}
	if latestVersion == "" {
		return nil, fmt.Errorf("最新版本信息缺少 tag")
	}

	installerAsset, portableAsset := pickReleaseAssets(release.Assets)
	distributionKind := a.currentUpdateDistributionKind()
	canSelfUpdatePortable := a.canSelfUpdatePortable()
	recommendedPackageKind := "installer"
	displayAsset := firstAsset(installerAsset, portableAsset)
	if distributionKind == "portable" && portableAsset != nil {
		recommendedPackageKind = "portable"
		displayAsset = portableAsset
	} else if installerAsset == nil && portableAsset != nil {
		recommendedPackageKind = "portable"
		displayAsset = portableAsset
	}
	info := &AppUpdateInfo{
		CurrentVersion:         currentVersion,
		LatestVersion:          latestVersion,
		ReleaseName:            firstNonEmpty(release.Name, release.TagName),
		ReleaseURL:             firstNonEmpty(release.HTMLURL, PROJECT_GITHUB_URL+"/latest"),
		PublishedAt:            release.PublishedAt,
		Body:                   strings.TrimSpace(release.Body),
		HasUpdate:              appReleaseVersionsDiffer(latestVersion, currentVersion),
		Asset:                  displayAsset,
		InstallerAsset:         installerAsset,
		PortableAsset:          portableAsset,
		DistributionKind:       distributionKind,
		RecommendedPackageKind: recommendedPackageKind,
		CanSelfUpdatePortable:  canSelfUpdatePortable,
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
	} else if installerAsset == nil && portableAsset == nil {
		info.Message = "检测到新版本，但没有找到适合当前系统的安装包"
	} else if recommendedPackageKind == "portable" && canSelfUpdatePortable {
		info.Message = "检测到新版本，可使用 ZIP 便携包自动更新"
	} else {
		info.Message = "检测到新版本"
	}
	return info, nil
}

func (a *App) OpenAppReleasePage(url string) error {
	target := strings.TrimSpace(url)
	if target == "" {
		target = PROJECT_GITHUB_URL + "/latest"
	}
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, target)
		return nil
	}
	return openURLWithSystem(target)
}

func (a *App) DownloadAppUpdate(info AppUpdateInfo, installOnRestart bool) (*AppUpdateDownloadResult, error) {
	asset := firstAsset(info.InstallerAsset, info.Asset)
	if asset == nil || strings.TrimSpace(asset.DownloadURL) == "" {
		return nil, fmt.Errorf("没有可下载的安装包")
	}
	version := resolveUpdateVersion(info)
	targetPath, err := a.downloadUpdateAsset(asset, version)
	if err != nil {
		return nil, err
	}
	if !isInstallableUpdatePackage(targetPath) {
		return nil, fmt.Errorf("下载的文件不是可直接安装文件: %s", filepath.Base(targetPath))
	}

	pending := pendingAppUpdate{
		Version:            version,
		InstallerPath:      targetPath,
		ReleaseURL:         info.ReleaseURL,
		InstallOnNextStart: installOnRestart,
		CreatedAt:          time.Now().Format(time.RFC3339),
	}
	if err := a.savePendingAppUpdate(pending); err != nil {
		return nil, err
	}

	message := "更新安装包已下载"
	if installOnRestart {
		message = "更新安装包已下载，下次启动将自动安装"
	}
	return &AppUpdateDownloadResult{
		Message:          message,
		Version:          version,
		InstallerPath:    targetPath,
		PackagePath:      targetPath,
		InstallOnRestart: installOnRestart,
		PackageKind:      "installer",
	}, nil
}

func (a *App) DownloadAndExtractPortableUpdate(info AppUpdateInfo) (*AppUpdateDownloadResult, error) {
	if info.PortableAsset == nil || strings.TrimSpace(info.PortableAsset.DownloadURL) == "" {
		return nil, fmt.Errorf("没有可下载的 ZIP 便携包")
	}
	version := resolveUpdateVersion(info)
	targetPath, err := a.downloadUpdateAsset(info.PortableAsset, version)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(filepath.Ext(targetPath), ".zip") {
		return nil, fmt.Errorf("下载的文件不是 ZIP 便携包: %s", filepath.Base(targetPath))
	}
	safeVersion := sanitizeFileName(version)
	if safeVersion == "" {
		safeVersion = "latest"
	}
	extractDir := filepath.Join(a.appUpdateWorkDir(), "portable-"+safeVersion)
	_ = os.RemoveAll(extractDir)
	a.emitAppUpdateDownloadProgress("extracting", 100, "正在解压 ZIP 便携包...")
	if err := unzipFileTo(targetPath, extractDir); err != nil {
		a.emitAppUpdateDownloadProgress("error", 100, "解压 ZIP 便携包失败")
		return nil, fmt.Errorf("解压 ZIP 便携包失败: %w", err)
	}
	sourceRoot, err := resolvePortablePackageRoot(extractDir)
	if err != nil {
		a.emitAppUpdateDownloadProgress("error", 100, "ZIP 便携包结构无效")
		return nil, err
	}

	message := "ZIP 便携包已下载并解压"
	restartScheduled := false
	if a.canSelfUpdatePortable() {
		a.emitAppUpdateDownloadProgress("installing", 100, "正在准备 ZIP 便携版自更新...")
		if err := a.schedulePortableSelfUpdate(sourceRoot, version); err != nil {
			a.emitAppUpdateDownloadProgress("error", 100, "准备 ZIP 便携版自更新失败")
			return nil, fmt.Errorf("准备 ZIP 便携版自更新失败: %w", err)
		}
		restartScheduled = true
		message = "ZIP 便携包已下载，应用即将退出并完成更新"
		go func() {
			time.Sleep(700 * time.Millisecond)
			a.ForceQuit()
		}()
	}
	a.emitAppUpdateDownloadProgress("done", 100, message)
	return &AppUpdateDownloadResult{
		Message:          message,
		Version:          version,
		PackagePath:      targetPath,
		ExtractedPath:    sourceRoot,
		RestartScheduled: restartScheduled,
		PackageKind:      "portable",
	}, nil
}

func (a *App) OpenPath(path string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return fmt.Errorf("路径不能为空")
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		target = filepath.Dir(target)
	}
	return openPathInFileManager(target)
}

func (a *App) downloadUpdateAsset(asset *AppUpdateAsset, version string) (string, error) {
	safeVersion := sanitizeFileName(version)
	if safeVersion == "" {
		safeVersion = "latest"
	}
	updatesDir := filepath.Join(a.appUpdateWorkDir(), safeVersion)
	if err := os.MkdirAll(updatesDir, 0755); err != nil {
		return "", fmt.Errorf("创建更新目录失败: %w", err)
	}
	fileName := sanitizeFileName(asset.Name)
	if fileName == "" {
		fileName = "trace-browser-update-" + version
	}
	targetPath := filepath.Join(updatesDir, fileName)
	a.emitAppUpdateDownloadProgress("starting", 0, "准备下载更新包...")
	if err := downloadFileWithRetry(asset.DownloadURL, targetPath, func(progress int, message string) {
		a.emitAppUpdateDownloadProgress("downloading", progress, message)
	}, func(attempt int, maxAttempts int, err error) {
		a.emitAppUpdateDownloadProgress(
			"retrying",
			0,
			fmt.Sprintf("下载失败，正在重试 %d/%d：%s", attempt, maxAttempts, err.Error()),
		)
	}); err != nil {
		a.emitAppUpdateDownloadProgress("error", 0, err.Error())
		return "", err
	}
	a.emitAppUpdateDownloadProgress("done", 100, "更新包下载完成")
	return targetPath, nil
}

func (a *App) InstallDownloadedAppUpdate(installerPath string) error {
	target := strings.TrimSpace(installerPath)
	if target == "" {
		pending, err := a.loadPendingAppUpdate()
		if err != nil {
			return err
		}
		target = pending.InstallerPath
	}
	if target == "" {
		return fmt.Errorf("没有待安装的更新包")
	}
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("更新包不存在: %w", err)
	}
	if !isInstallableUpdatePackage(target) {
		return fmt.Errorf("更新包不是可直接安装文件，请打开下载页手动下载 Setup 安装包: %s", filepath.Base(target))
	}
	launchPath, err := a.prepareUpdateInstallerLaunchPath(target)
	if err != nil {
		return fmt.Errorf("准备安装程序失败: %w", err)
	}
	if err := a.startUpdateInstaller(launchPath); err != nil {
		return fmt.Errorf("启动安装程序失败: %w", err)
	}
	a.clearPendingAppUpdate()
	go func() {
		time.Sleep(600 * time.Millisecond)
		a.ForceQuit()
	}()
	return nil
}

func (a *App) emitAppUpdateDownloadProgress(phase string, progress int, message string) {
	if a.ctx == nil {
		return
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	runtime.EventsEmit(a.ctx, "app:update:download:progress", AppUpdateDownloadProgress{
		Phase:    phase,
		Progress: progress,
		Message:  strings.TrimSpace(message),
	})
}

func (a *App) emitPendingAppUpdateIfNeeded() {
	pending, err := a.loadPendingAppUpdate()
	if err != nil || strings.TrimSpace(pending.InstallerPath) == "" {
		return
	}
	if _, err := os.Stat(pending.InstallerPath); err != nil {
		a.clearPendingAppUpdate()
		return
	}
	if a.ctx == nil {
		return
	}
	go func() {
		time.Sleep(1200 * time.Millisecond)
		if pending.InstallOnNextStart {
			runtime.EventsEmit(a.ctx, "app:update:pending:notification", map[string]interface{}{
				"version": pending.Version,
				"message": "更新安装包已下载，正在启动安装程序",
			})
			if err := a.InstallDownloadedAppUpdate(pending.InstallerPath); err != nil {
				runtime.EventsEmit(a.ctx, "app:update:pending:install-failed", map[string]interface{}{
					"version": pending.Version,
					"error":   err.Error(),
				})
				runtime.EventsEmit(a.ctx, "app:update:pending", pending)
			}
			return
		}
		runtime.EventsEmit(a.ctx, "app:update:pending", pending)
	}()
}

func fetchLatestGithubRelease(ctx context.Context) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Trace-Browser-Updater")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("检查更新失败：HTTP %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}
	if release.Draft {
		return nil, fmt.Errorf("最新版本仍处于草稿状态")
	}
	return &release, nil
}

func pickReleaseAssets(assets []githubReleaseAsset) (*AppUpdateAsset, *AppUpdateAsset) {
	candidates := make([]githubReleaseAsset, 0, len(assets))
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		url := strings.TrimSpace(asset.BrowserDownloadURL)
		if url == "" {
			continue
		}
		if strings.Contains(name, "sha256") || strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".json") {
			continue
		}
		candidates = append(candidates, asset)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return assetScore(candidates[i].Name) > assetScore(candidates[j].Name)
	})
	if len(candidates) == 0 || assetScore(candidates[0].Name) <= 0 {
		return nil, nil
	}
	var installer *AppUpdateAsset
	var portable *AppUpdateAsset
	for _, candidate := range candidates {
		asset := &AppUpdateAsset{Name: candidate.Name, Size: candidate.Size, DownloadURL: candidate.BrowserDownloadURL}
		if installer == nil && isInstallerAssetName(candidate.Name) {
			installer = asset
		}
		if portable == nil && isPortableAssetName(candidate.Name) {
			portable = asset
		}
	}
	return installer, portable
}

func assetScore(name string) int {
	lower := strings.ToLower(name)
	score := 0
	if goruntime.GOOS == "windows" && (strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".msi")) {
		score += 100
	}
	if strings.Contains(lower, "setup") || strings.Contains(lower, "installer") {
		score += 30
	}
	if strings.Contains(lower, "portable") || strings.HasSuffix(lower, ".zip") {
		score -= 25
	}
	if strings.Contains(lower, goruntime.GOOS) {
		score += 20
	}
	if goruntime.GOOS == "windows" && (strings.Contains(lower, "win") || strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".msi")) {
		score += 30
	}
	if goruntime.GOOS == "darwin" && (strings.Contains(lower, "mac") || strings.HasSuffix(lower, ".dmg")) {
		score += 30
	}
	if goruntime.GOOS == "linux" && (strings.HasSuffix(lower, ".deb") || strings.HasSuffix(lower, ".appimage") || strings.HasSuffix(lower, ".tar.gz")) {
		score += 30
	}
	if strings.Contains(lower, goruntime.GOARCH) {
		score += 10
	}
	if goruntime.GOARCH == "amd64" && (strings.Contains(lower, "x64") || strings.Contains(lower, "x86_64")) {
		score += 10
	}
	return score
}

func firstAsset(values ...*AppUpdateAsset) *AppUpdateAsset {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func isInstallerAssetName(name string) bool {
	lower := strings.ToLower(name)
	switch goruntime.GOOS {
	case "windows":
		return strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".msi")
	case "darwin":
		return strings.HasSuffix(lower, ".dmg") || strings.HasSuffix(lower, ".pkg")
	default:
		return strings.HasSuffix(lower, ".deb") || strings.HasSuffix(lower, ".rpm") || strings.HasSuffix(lower, ".appimage")
	}
}

func isPortableAssetName(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".zip")
}

func downloadFile(url string, targetPath string, onProgress func(progress int, message string)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Trace-Browser-Updater")
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载更新失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("下载更新失败：HTTP %d", resp.StatusCode)
	}
	totalSize := resp.ContentLength
	tmp := targetPath + "." + shortHash(url) + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("创建下载文件失败: %w", err)
	}
	_, copyErr := copyWithProgress(out, resp.Body, totalSize, onProgress)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写入更新包失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存更新包失败: %w", closeErr)
	}
	_ = os.Remove(targetPath)
	if err := os.Rename(tmp, targetPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存更新包失败: %w", err)
	}
	return nil
}

func copyWithProgress(dst io.Writer, src io.Reader, totalSize int64, onProgress func(progress int, message string)) (int64, error) {
	buf := make([]byte, 256*1024)
	var written int64
	lastProgress := -1
	lastEmit := time.Now().Add(-time.Second)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
			if onProgress != nil {
				progress := 0
				if totalSize > 0 {
					progress = int(float64(written) / float64(totalSize) * 100)
					if progress > 99 {
						progress = 99
					}
				}
				if progress != lastProgress || time.Since(lastEmit) >= 500*time.Millisecond {
					onProgress(progress, formatDownloadProgress(written, totalSize))
					lastProgress = progress
					lastEmit = time.Now()
				}
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return written, er
		}
	}
	if onProgress != nil {
		onProgress(100, "下载完成")
	}
	return written, nil
}

func (a *App) pendingAppUpdatePath() string {
	return filepath.Join(a.appUpdateWorkDir(), "pending-update.json")
}

func (a *App) legacyPendingAppUpdatePath() string {
	return a.resolveAppPath(filepath.Join("data", "updates", "pending-update.json"))
}

func (a *App) appUpdateWorkDir() string {
	if base, err := os.UserCacheDir(); err == nil && strings.TrimSpace(base) != "" {
		return filepath.Join(base, "Trace Browser", "updates")
	}
	if tmp := strings.TrimSpace(os.TempDir()); tmp != "" {
		return filepath.Join(tmp, "trace-browser-updates")
	}
	return a.resolveAppPath(filepath.Join("data", "updates"))
}

func (a *App) savePendingAppUpdate(pending pendingAppUpdate) error {
	path := a.pendingAppUpdatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *App) loadPendingAppUpdate() (*pendingAppUpdate, error) {
	data, err := os.ReadFile(a.pendingAppUpdatePath())
	if err != nil {
		data, err = os.ReadFile(a.legacyPendingAppUpdatePath())
	}
	if err != nil {
		return nil, err
	}
	var pending pendingAppUpdate
	if err := json.Unmarshal(data, &pending); err != nil {
		return nil, err
	}
	return &pending, nil
}

func (a *App) clearPendingAppUpdate() {
	_ = os.Remove(a.pendingAppUpdatePath())
	_ = os.Remove(a.legacyPendingAppUpdatePath())
}

func (a *App) prepareUpdateInstallerLaunchPath(path string) (string, error) {
	if goruntime.GOOS != "windows" {
		return path, nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	appRoot := a.appRootAbs()
	if appRoot == "" || !pathInsideDir(absPath, appRoot) {
		return absPath, nil
	}

	launchDir := filepath.Join(a.appUpdateWorkDir(), "installer-launch")
	if err := os.MkdirAll(launchDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(launchDir, filepath.Base(absPath))
	if strings.EqualFold(filepath.Clean(absPath), filepath.Clean(target)) {
		return absPath, nil
	}
	if err := copyUpdateFile(absPath, target); err != nil {
		return "", err
	}
	return target, nil
}

func (a *App) startUpdateInstaller(path string) error {
	if goruntime.GOOS != "windows" {
		return startInstaller(path)
	}
	scriptDir := filepath.Join(a.appUpdateWorkDir(), "helpers")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptDir, "start-update-installer.ps1")
	logPath := filepath.Join(scriptDir, "start-update-installer.log")
	if err := os.WriteFile(scriptPath, []byte(updateInstallerPowerShellScript()), 0600); err != nil {
		return err
	}

	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-WindowStyle",
		"Hidden",
		"-File",
		scriptPath,
		"-Installer",
		path,
		"-Pid",
		fmt.Sprintf("%d", os.Getpid()),
		"-LogPath",
		logPath,
	)
	hideWindow(cmd)
	return cmd.Start()
}

func pathInsideDir(path string, dir string) bool {
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func copyUpdateFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + "." + shortHash(src+"|"+dst) + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func startInstaller(path string) error {
	switch goruntime.GOOS {
	case "windows":
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".msi" {
			return exec.Command("msiexec.exe", "/i", path).Start()
		}
		return exec.Command(
			"powershell",
			"-NoProfile",
			"-ExecutionPolicy",
			"Bypass",
			"-Command",
			"Start-Process -LiteralPath $args[0]",
			path,
		).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		if strings.HasSuffix(strings.ToLower(path), ".appimage") {
			_ = os.Chmod(path, 0755)
			return exec.Command(path).Start()
		}
		return exec.Command("xdg-open", path).Start()
	}
}

func updateInstallerPowerShellScript() string {
	return `param(
  [Parameter(Mandatory = $true)][string]$Installer,
  [Parameter(Mandatory = $true)][int]$Pid,
  [Parameter(Mandatory = $true)][string]$LogPath
)

$ErrorActionPreference = 'Stop'

function Write-TraceLog {
  param([string]$Message)
  $parent = Split-Path -Parent $LogPath
  if ($parent -and -not (Test-Path -LiteralPath $parent)) {
    New-Item -ItemType Directory -Path $parent -Force | Out-Null
  }
  Add-Content -LiteralPath $LogPath -Value ("{0} {1}" -f (Get-Date -Format o), $Message)
}

try {
  Write-TraceLog "installer helper started"
  if ($Pid -gt 0) {
    $deadline = (Get-Date).AddSeconds(45)
    while ((Get-Date) -lt $deadline) {
      $proc = Get-Process -Id $Pid -ErrorAction SilentlyContinue
      if (-not $proc) {
        break
      }
      Start-Sleep -Milliseconds 250
    }
  }
  Start-Sleep -Milliseconds 500

  $installerPath = [System.IO.Path]::GetFullPath($Installer)
  if (-not (Test-Path -LiteralPath $installerPath -PathType Leaf)) {
    throw "installer missing: $installerPath"
  }
  $workDir = Split-Path -Parent $installerPath
  $ext = [System.IO.Path]::GetExtension($installerPath).ToLowerInvariant()
  Write-TraceLog "starting installer: $installerPath"
  if ($ext -eq '.msi') {
    Start-Process -FilePath 'msiexec.exe' -ArgumentList @('/i', $installerPath) -WorkingDirectory $workDir
  } else {
    Start-Process -FilePath $installerPath -WorkingDirectory $workDir
  }
  Write-TraceLog "installer helper completed"
  exit 0
} catch {
  Write-TraceLog ("installer helper failed: " + $_.Exception.Message)
  exit 1
}
`
}

func isInstallableUpdatePackage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch goruntime.GOOS {
	case "windows":
		return ext == ".exe" || ext == ".msi"
	case "darwin":
		return ext == ".dmg" || ext == ".pkg"
	default:
		lower := strings.ToLower(path)
		return ext == ".deb" || strings.HasSuffix(lower, ".appimage") || strings.HasSuffix(lower, ".rpm")
	}
}

func unzipFileTo(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	cleanDest := filepath.Clean(dest)
	for _, f := range r.File {
		target := filepath.Join(dest, filepath.FromSlash(f.Name))
		cleanTarget := filepath.Clean(target)
		if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func formatDownloadProgress(written int64, totalSize int64) string {
	if totalSize > 0 {
		return fmt.Sprintf("正在下载 %.1f/%.1f MB", float64(written)/1024/1024, float64(totalSize)/1024/1024)
	}
	return fmt.Sprintf("正在下载 %.1f MB", float64(written)/1024/1024)
}

func resolveUpdateVersion(info AppUpdateInfo) string {
	version := normalizeVersion(firstNonEmpty(info.LatestVersion, info.ReleaseName))
	if version == "" {
		return "latest"
	}
	return version
}

func openURLWithSystem(url string) error {
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V")
	return strings.TrimSpace(value)
}

func compareVersions(a string, b string) int {
	ap := versionParts(a)
	bp := versionParts(b)
	maxLen := len(ap)
	if len(bp) > maxLen {
		maxLen = len(bp)
	}
	for i := 0; i < maxLen; i++ {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func appReleaseVersionsDiffer(latestVersion string, currentVersion string) bool {
	latest := normalizeVersion(latestVersion)
	current := normalizeVersion(currentVersion)
	if latest == "" || current == "" || strings.EqualFold(current, "unknown") {
		return latest != ""
	}
	return compareVersions(latest, current) > 0
}

func downloadFileWithRetry(
	url string,
	targetPath string,
	onProgress func(progress int, message string),
	onRetry func(attempt int, maxAttempts int, err error),
) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * 800 * time.Millisecond)
		}
		if err := downloadFile(url, targetPath, onProgress); err != nil {
			lastErr = err
			if attempt < maxAttempts && onRetry != nil {
				onRetry(attempt+1, maxAttempts, err)
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (a *App) currentUpdateDistributionKind() string {
	if a.isSourceCheckoutRoot() {
		return "dev"
	}
	if goruntime.GOOS == "windows" {
		if _, err := os.Stat(filepath.Join(a.appRootAbs(), "Uninstall.exe")); err == nil {
			return "installer"
		}
		return "portable"
	}
	return "installer"
}

func (a *App) canSelfUpdatePortable() bool {
	return goruntime.GOOS == "windows" && a.currentUpdateDistributionKind() == "portable"
}

func (a *App) isSourceCheckoutRoot() bool {
	root := a.appRootAbs()
	if root == "" {
		return false
	}
	requiredFiles := []string{"go.mod", "wails.json"}
	for _, name := range requiredFiles {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			return false
		}
	}
	if info, err := os.Stat(filepath.Join(root, "frontend")); err != nil || !info.IsDir() {
		return false
	}
	return true
}

func resolvePortablePackageRoot(extractDir string) (string, error) {
	extractDir = filepath.Clean(extractDir)
	if _, err := os.Stat(filepath.Join(extractDir, "trace-browser.exe")); err == nil {
		return extractDir, nil
	}

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(extractDir, entry.Name())
		if _, err := os.Stat(filepath.Join(candidate, "trace-browser.exe")); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("ZIP 便携包中未找到 trace-browser.exe")
}

func (a *App) schedulePortableSelfUpdate(sourceRoot string, version string) error {
	if goruntime.GOOS != "windows" {
		return fmt.Errorf("ZIP 便携版自更新仅支持 Windows")
	}
	sourceRoot = filepath.Clean(sourceRoot)
	targetRoot := filepath.Clean(a.appRootAbs())
	if sourceRoot == "" || targetRoot == "" {
		return fmt.Errorf("更新源目录或目标目录为空")
	}
	if strings.EqualFold(sourceRoot, targetRoot) {
		return fmt.Errorf("更新源目录不能与当前应用目录相同")
	}
	if _, err := os.Stat(filepath.Join(sourceRoot, "trace-browser.exe")); err != nil {
		return fmt.Errorf("更新源缺少 trace-browser.exe: %w", err)
	}
	if _, err := os.Stat(filepath.Join(targetRoot, "trace-browser.exe")); err != nil {
		return fmt.Errorf("当前应用目录缺少 trace-browser.exe: %w", err)
	}

	safeVersion := sanitizeFileName(version)
	if safeVersion == "" {
		safeVersion = "latest"
	}
	scriptDir := filepath.Join(os.TempDir(), "trace-browser-updates", safeVersion+"-"+shortHash(sourceRoot+"|"+targetRoot))
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return err
	}
	scriptPath := filepath.Join(scriptDir, "apply-portable-update.ps1")
	logPath := filepath.Join(scriptDir, "apply-portable-update.log")
	if err := os.WriteFile(scriptPath, []byte(portableUpdatePowerShellScript()), 0600); err != nil {
		return err
	}

	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy",
		"Bypass",
		"-WindowStyle",
		"Hidden",
		"-File",
		scriptPath,
		"-Source",
		sourceRoot,
		"-Target",
		targetRoot,
		"-Pid",
		fmt.Sprintf("%d", os.Getpid()),
		"-ExeName",
		"trace-browser.exe",
		"-LogPath",
		logPath,
	)
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func portableUpdatePowerShellScript() string {
	return `param(
  [Parameter(Mandatory = $true)][string]$Source,
  [Parameter(Mandatory = $true)][string]$Target,
  [Parameter(Mandatory = $true)][int]$Pid,
  [Parameter(Mandatory = $true)][string]$ExeName,
  [Parameter(Mandatory = $true)][string]$LogPath
)

$ErrorActionPreference = 'Stop'

function Write-TraceLog {
  param([string]$Message)
  $parent = Split-Path -Parent $LogPath
  if ($parent -and -not (Test-Path -LiteralPath $parent)) {
    New-Item -ItemType Directory -Path $parent -Force | Out-Null
  }
  Add-Content -LiteralPath $LogPath -Value ("{0} {1}" -f (Get-Date -Format o), $Message)
}

function Copy-DirectoryContents {
  param([string]$Src, [string]$Dst)
  if (-not (Test-Path -LiteralPath $Dst)) {
    New-Item -ItemType Directory -Path $Dst -Force | Out-Null
  }
  & robocopy.exe $Src $Dst /E /R:2 /W:1 /NFL /NDL /NJH /NJS /NP | Out-Null
  if ($LASTEXITCODE -ge 8) {
    throw "robocopy failed for $Src -> $Dst with exit code $LASTEXITCODE"
  }
}

try {
  Write-TraceLog "portable update helper started"
  if ($Pid -gt 0) {
    $deadline = (Get-Date).AddSeconds(60)
    while ((Get-Date) -lt $deadline) {
      $proc = Get-Process -Id $Pid -ErrorAction SilentlyContinue
      if (-not $proc) {
        break
      }
      Start-Sleep -Milliseconds 300
    }
  }
  Start-Sleep -Milliseconds 800

  $sourceRoot = [System.IO.Path]::GetFullPath($Source).TrimEnd('\')
  $targetRoot = [System.IO.Path]::GetFullPath($Target).TrimEnd('\')
  $sourceExe = Join-Path $sourceRoot $ExeName
  $targetExe = Join-Path $targetRoot $ExeName

  if (-not (Test-Path -LiteralPath $sourceExe -PathType Leaf)) {
    throw "source executable missing: $sourceExe"
  }
  if (-not (Test-Path -LiteralPath $targetRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $targetRoot -Force | Out-Null
  }

  Get-ChildItem -LiteralPath $sourceRoot -Force | ForEach-Object {
    $name = $_.Name
    $dest = Join-Path $targetRoot $name
    if ($_.PSIsContainer) {
      if ($name -ieq 'data' -or $name -ieq 'logs') {
        if (-not (Test-Path -LiteralPath $dest)) {
          New-Item -ItemType Directory -Path $dest -Force | Out-Null
        }
      } else {
        Copy-DirectoryContents -Src $_.FullName -Dst $dest
      }
    } else {
      if ($name -ieq 'config.yaml' -or $name -ieq 'proxies.yaml') {
        if (-not (Test-Path -LiteralPath $dest)) {
          Copy-Item -LiteralPath $_.FullName -Destination $dest -Force
        }
      } else {
        Copy-Item -LiteralPath $_.FullName -Destination $dest -Force
      }
    }
  }

  if (-not (Test-Path -LiteralPath $targetExe -PathType Leaf)) {
    throw "updated executable missing: $targetExe"
  }
  Write-TraceLog "starting updated application"
  Start-Process -FilePath $targetExe -WorkingDirectory $targetRoot
  Write-TraceLog "portable update helper completed"
  exit 0
} catch {
  Write-TraceLog ("portable update helper failed: " + $_.Exception.Message)
  exit 1
}
`
}

func versionParts(value string) []int {
	value = normalizeVersion(value)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}
