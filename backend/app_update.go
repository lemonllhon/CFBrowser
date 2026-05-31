package backend

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	updatergithub "github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const PROJECT_GITHUB_URL = "https://github.com/lemon-casino/trace-browser-release/releases"

const defaultWailsChecksumAssetName = "SHA256SUMS"

const (
	appUpdatePackageSelfUpdate = "selfupdate"
	appUpdatePackageManual     = "manual"
)

type AppUpdateAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
	Checksum    string `json:"checksum,omitempty"`
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

type wailsUpdateRuntime struct {
	app              *application.App
	owner            string
	repo             string
	checksumAsset    string
	initialized      bool
	eventsRegistered bool
}

func (a *App) ConfigureOfficialWailsUpdater(wailsApp *application.App) {
	if a == nil || wailsApp == nil {
		return
	}
	a.updateRuntimeMu.Lock()
	defer a.updateRuntimeMu.Unlock()
	if a.updateRuntime == nil {
		a.updateRuntime = &wailsUpdateRuntime{}
	}
	a.updateRuntime.app = wailsApp
}

func (a *App) CheckAppUpdate() (*AppUpdateInfo, error) {
	currentVersion := a.appVersion()
	info, err := a.checkAppUpdateFromOfficialWailsRuntime(context.Background(), currentVersion)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (a *App) checkAppUpdateFromOfficialWailsRuntime(ctx context.Context, currentVersion string) (*AppUpdateInfo, error) {
	runtime, err := a.officialWailsUpdateRuntime(ctx)
	if err != nil {
		return nil, err
	}
	result, err := runtime.app.Updater.Check(ctx)
	if err != nil {
		return nil, fmt.Errorf("Wails3 官方 updater 检查失败: %w", err)
	}
	distributionKind := a.currentUpdateDistributionKind()
	recommendedPackageKind := appUpdatePackageSelfUpdate
	if shouldUseManualPackageUpdate(distributionKind) {
		recommendedPackageKind = appUpdatePackageManual
	}
	if result == nil {
		return &AppUpdateInfo{
			CurrentVersion:         currentVersion,
			LatestVersion:          currentVersion,
			ReleaseURL:             PROJECT_GITHUB_URL + "/latest",
			HasUpdate:              false,
			DistributionKind:       distributionKind,
			RecommendedPackageKind: recommendedPackageKind,
			CanSelfUpdatePortable:  false,
			Message:                "当前已是最新版本",
		}, nil
	}

	latestVersion := normalizeVersion(result.Version)
	if latestVersion == "" {
		latestVersion = normalizeVersion(currentVersion)
	}
	asset := appUpdateAssetFromOfficialRelease(result)
	if shouldUseManualPackageUpdate(distributionKind) {
		asset = nil
	}
	info := &AppUpdateInfo{
		CurrentVersion:         currentVersion,
		LatestVersion:          latestVersion,
		ReleaseName:            firstNonEmpty(result.Name, "v"+latestVersion, result.Version),
		ReleaseURL:             firstNonEmpty(officialReleaseURL(result), PROJECT_GITHUB_URL+"/latest"),
		PublishedAt:            formatOfficialReleaseTime(result.PublishedAt),
		Body:                   strings.TrimSpace(result.Notes),
		HasUpdate:              appReleaseVersionsDiffer(latestVersion, currentVersion),
		Asset:                  asset,
		InstallerAsset:         asset,
		DistributionKind:       distributionKind,
		RecommendedPackageKind: recommendedPackageKind,
		CanSelfUpdatePortable:  false,
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
	} else if shouldUseManualPackageUpdate(distributionKind) {
		info.Message = manualPackageUpdateMessage(distributionKind)
	} else if asset == nil {
		info.Message = "检测到新版本，但没有找到官方 Wails3 自更新资产"
	} else {
		info.Message = "检测到新版本，可使用 Wails3 官方 updater 直接更新"
	}
	return info, nil
}

func appUpdateAssetFromOfficialRelease(release *updater.Release) *AppUpdateAsset {
	if release == nil {
		return nil
	}
	downloadURL := officialReleaseDownloadURL(release)
	if strings.TrimSpace(downloadURL) == "" {
		return nil
	}
	return &AppUpdateAsset{
		Name:        firstNonEmpty(release.Artifact.Filename, fileNameFromURL(downloadURL), "trace-browser-selfupdate"),
		Size:        release.Artifact.Size,
		DownloadURL: downloadURL,
		Checksum:    officialReleaseChecksum(release.Verification),
	}
}

func officialReleaseURL(release *updater.Release) string {
	if release == nil || release.Metadata == nil {
		return ""
	}
	if value, ok := release.Metadata["github.release.htmlURL"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func officialReleaseDownloadURL(release *updater.Release) string {
	if release == nil || release.Metadata == nil {
		return ""
	}
	if value, ok := release.Metadata["github.asset.url"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func officialReleaseChecksum(verification *updater.Verification) string {
	if verification == nil || len(verification.Digest) == 0 {
		return ""
	}
	if !strings.EqualFold(strings.TrimSpace(verification.DigestAlgo), "sha256") {
		return ""
	}
	return hex.EncodeToString(verification.Digest)
}

func formatOfficialReleaseTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (a *App) officialWailsUpdateRuntime(ctx context.Context) (*wailsUpdateRuntime, error) {
	if a == nil {
		return nil, fmt.Errorf("app is nil")
	}
	owner, repo, err := resolveOfficialUpdateRepository()
	if err != nil {
		return nil, err
	}
	checksumAsset := strings.TrimSpace(os.Getenv("TRACE_BROWSER_UPDATE_CHECKSUM_ASSET"))
	if checksumAsset == "" {
		checksumAsset = defaultWailsChecksumAssetName
	}

	a.updateRuntimeMu.Lock()
	defer a.updateRuntimeMu.Unlock()
	if a.updateRuntime == nil || a.updateRuntime.app == nil {
		return nil, fmt.Errorf("Wails3 官方 updater 尚未初始化")
	}
	if a.updateRuntime.initialized &&
		a.updateRuntime.owner == owner && a.updateRuntime.repo == repo && a.updateRuntime.checksumAsset == checksumAsset {
		return a.updateRuntime, nil
	}

	provider, err := updatergithub.New(updatergithub.Config{
		Repository:    owner + "/" + repo,
		Token:         strings.TrimSpace(os.Getenv("TRACE_BROWSER_UPDATE_TOKEN")),
		ChecksumAsset: checksumAsset,
		AssetMatcher:  matchOfficialSelfUpdateAsset,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 Wails3 GitHub updater provider 失败: %w", err)
	}
	if err := a.updateRuntime.app.Updater.Init(updater.Config{
		Providers:      []updater.Provider{provider},
		CurrentVersion: normalizeVersion(a.appVersion()),
		Window:         updater.WindowNone,
	}); err != nil {
		return nil, fmt.Errorf("初始化 Wails3 官方 updater 失败: %w", err)
	}
	if !a.updateRuntime.eventsRegistered {
		a.registerOfficialWailsUpdateEvents(a.updateRuntime.app)
		a.updateRuntime.eventsRegistered = true
	}
	a.updateRuntime.owner = owner
	a.updateRuntime.repo = repo
	a.updateRuntime.checksumAsset = checksumAsset
	a.updateRuntime.initialized = true
	return a.updateRuntime, nil
}

func resolveOfficialUpdateRepository() (string, string, error) {
	value := firstNonEmpty(
		os.Getenv("TRACE_BROWSER_UPDATE_REPOSITORY"),
		os.Getenv("PUBLIC_RELEASE_REPOSITORY"),
		"lemon-casino/trace-browser-release",
	)
	value = strings.TrimSpace(strings.TrimSuffix(value, ".git"))
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimPrefix(value, "http://github.com/")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("官方 updater 仓库格式无效: %s", value)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func (a *App) OpenAppReleasePage(url string) error {
	target := strings.TrimSpace(url)
	if target == "" {
		target = PROJECT_GITHUB_URL + "/latest"
	}
	if a.ctx != nil {
		a.appRuntime().OpenExternalURL(a.ctx, target)
		return nil
	}
	return openURLWithSystem(target)
}

func (a *App) DownloadAppUpdate(info AppUpdateInfo, installOnRestart bool) (*AppUpdateDownloadResult, error) {
	if shouldUseManualPackageUpdate(a.currentUpdateDistributionKind()) {
		return nil, fmt.Errorf("当前为解压版或开发版，请打开官方下载页手动下载并解压")
	}
	if !strings.EqualFold(info.RecommendedPackageKind, appUpdatePackageSelfUpdate) {
		return nil, fmt.Errorf("当前版本不走应用内更新，请打开官方下载页手动下载并解压/安装")
	}
	return a.downloadOfficialWailsUpdate(info, installOnRestart)
}

func (a *App) downloadOfficialWailsUpdate(info AppUpdateInfo, installOnRestart bool) (*AppUpdateDownloadResult, error) {
	if shouldUseManualPackageUpdate(a.currentUpdateDistributionKind()) {
		return nil, fmt.Errorf("当前为解压版或开发版，请打开官方下载页手动下载并解压")
	}
	if installOnRestart {
		return nil, fmt.Errorf("Wails3 官方 updater 不支持跨进程保留下次启动安装，请选择立即下载并应用更新")
	}
	runtime, err := a.officialWailsUpdateRuntime(context.Background())
	if err != nil {
		return nil, err
	}
	if normalizeVersion(info.LatestVersion) != "" {
		if _, err := runtime.app.Updater.Check(context.Background()); err != nil {
			return nil, fmt.Errorf("Wails3 官方 updater 刷新版本信息失败: %w", err)
		}
	}
	a.emitAppUpdateDownloadProgress("starting", 0, "准备通过 Wails3 官方 updater 下载并准备更新...")
	if err := runtime.app.Updater.DownloadAndInstall(context.Background()); err != nil {
		a.emitAppUpdateDownloadProgress("error", 0, err.Error())
		return nil, fmt.Errorf("Wails3 官方 updater 下载或准备更新失败: %w", err)
	}
	a.emitAppUpdateDownloadProgress("done", 100, "官方更新包已准备完成，等待重启应用")
	return &AppUpdateDownloadResult{
		Message:     "官方更新包已准备完成，正在准备重启应用",
		Version:     resolveUpdateVersion(info),
		PackageKind: "selfupdate",
	}, nil
}

func (a *App) DownloadAndExtractPortableUpdate(info AppUpdateInfo) (*AppUpdateDownloadResult, error) {
	if err := a.OpenAppReleasePage(info.ReleaseURL); err != nil {
		return nil, err
	}
	return &AppUpdateDownloadResult{
		Message:     "已打开官方下载页，请下载 ZIP 解压版并自行解压",
		Version:     resolveUpdateVersion(info),
		PackageKind: appUpdatePackageManual,
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

func (a *App) InstallDownloadedAppUpdate(installerPath string) error {
	target := strings.TrimSpace(installerPath)
	if target != "" {
		return fmt.Errorf("安装包直装已停用，请使用 Wails3 官方 updater 或打开官方下载页手动安装")
	}
	return a.installOfficialWailsUpdate()
}

func (a *App) installOfficialWailsUpdate() error {
	if shouldUseManualPackageUpdate(a.currentUpdateDistributionKind()) {
		return fmt.Errorf("当前为解压版或开发版，请打开官方下载页手动下载并解压")
	}
	a.updateRuntimeMu.Lock()
	runtime := a.updateRuntime
	a.updateRuntimeMu.Unlock()
	if runtime == nil || runtime.app == nil || !runtime.initialized {
		return fmt.Errorf("Wails3 官方 updater 尚未下载更新")
	}
	a.clearPendingAppUpdate()
	a.emitAppUpdateDownloadProgress("done", 100, "更新已应用，应用即将重启")
	go func(wailsApp *application.App) {
		time.Sleep(600 * time.Millisecond)
		if err := wailsApp.Updater.Restart(context.Background()); err != nil {
			a.emitAppUpdatePendingEvent("app:update:pending:install-failed", map[string]interface{}{
				"version": "",
				"error":   err.Error(),
			})
			a.ForceQuit()
		}
	}(runtime.app)
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
	a.emitAppUpdateDownloadProgressEvent("app:update:download:progress", AppUpdateDownloadProgress{
		Phase:    phase,
		Progress: progress,
		Message:  strings.TrimSpace(message),
	})
}

func (a *App) registerOfficialWailsUpdateEvents(wailsApp *application.App) {
	if a == nil || wailsApp == nil {
		return
	}
	wailsApp.Event.On(updater.EventDownloadStarted, func(event *application.CustomEvent) {
		a.emitAppUpdateDownloadProgress("started", 0, "Wails3 官方 updater 已开始下载")
	})
	wailsApp.Event.On(updater.EventDownloadProgress, func(event *application.CustomEvent) {
		written, total := officialProgressBytes(event)
		progress := 0
		if total > 0 {
			progress = int(float64(written) * 100 / float64(total))
		}
		a.emitAppUpdateDownloadProgress("downloading", progress, formatDownloadProgress(written, total))
	})
	wailsApp.Event.On(updater.EventDownloadComplete, func(event *application.CustomEvent) {
		a.emitAppUpdateDownloadProgress("downloaded", 100, "Wails3 官方 updater 下载完成")
	})
	wailsApp.Event.On(updater.EventError, func(event *application.CustomEvent) {
		a.emitAppUpdateDownloadProgress("error", 0, firstNonEmpty(officialEventMessage(event), "Wails3 官方 updater 更新失败"))
	})
}

func matchOfficialSelfUpdateAsset(req updater.CheckRequest, assets []updatergithub.ReleaseAsset) int {
	if len(assets) == 0 {
		return -1
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	arch := strings.ToLower(strings.TrimSpace(req.Arch))
	if platform == "" {
		platform = goruntime.GOOS
	}
	if arch == "" {
		arch = goruntime.GOARCH
	}
	platformAliases := []string{platform}
	if platform == "windows" {
		platformAliases = append(platformAliases, "win")
	} else if platform == "darwin" {
		platformAliases = append(platformAliases, "macos", "mac")
	}
	archAliases := []string{arch}
	if arch == "amd64" {
		archAliases = append(archAliases, "x64", "x86_64")
	}

	bestIndex := -1
	bestScore := 0
	for i, asset := range assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		if name == "" || strings.Contains(name, "sha256sums") || strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".txt") {
			continue
		}
		score := 0
		if strings.Contains(name, "selfupdate") || strings.Contains(name, "self-update") {
			score += 100
		}
		for _, alias := range platformAliases {
			if alias != "" && strings.Contains(name, alias) {
				score += 25
				break
			}
		}
		for _, alias := range archAliases {
			if alias != "" && strings.Contains(name, alias) {
				score += 25
				break
			}
		}
		if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".tar.gz") {
			score += 10
		}
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	if bestScore < 125 {
		return -1
	}
	return bestIndex
}

func officialProgressBytes(event *application.CustomEvent) (int64, int64) {
	if event == nil {
		return 0, 0
	}
	if progress, ok := event.Data.(updater.Progress); ok {
		return progress.Written, progress.Total
	}
	if progress, ok := event.Data.(*updater.Progress); ok && progress != nil {
		return progress.Written, progress.Total
	}
	var progress updater.Progress
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return 0, 0
	}
	if err := json.Unmarshal(payload, &progress); err != nil {
		return 0, 0
	}
	return progress.Written, progress.Total
}

func officialEventMessage(event *application.CustomEvent) string {
	if event == nil {
		return ""
	}
	if info, ok := event.Data.(updater.ErrorInfo); ok {
		return strings.TrimSpace(info.Message)
	}
	if info, ok := event.Data.(*updater.ErrorInfo); ok && info != nil {
		return strings.TrimSpace(info.Message)
	}
	var info updater.ErrorInfo
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return ""
	}
	if err := json.Unmarshal(payload, &info); err != nil {
		return ""
	}
	return strings.TrimSpace(info.Message)
}

func (a *App) emitPendingAppUpdateIfNeeded() {
	a.clearPendingAppUpdate()
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

func (a *App) clearPendingAppUpdate() {
	_ = os.Remove(a.pendingAppUpdatePath())
	_ = os.Remove(a.legacyPendingAppUpdatePath())
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

func shouldUseManualPackageUpdate(distributionKind string) bool {
	switch strings.ToLower(strings.TrimSpace(distributionKind)) {
	case "dev", "portable":
		return true
	default:
		return false
	}
}

func manualPackageUpdateMessage(distributionKind string) string {
	if strings.EqualFold(strings.TrimSpace(distributionKind), "portable") {
		return "检测到新版本，当前为解压版，请打开官方下载页下载 ZIP 后自行解压"
	}
	return "检测到新版本，请打开官方下载页手动下载发布包"
}

func (a *App) isSourceCheckoutRoot() bool {
	root := a.appRootAbs()
	if root == "" {
		return false
	}
	requiredFiles := []string{"go.mod", filepath.Join("build", "config.yml")}
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func fileNameFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err == nil && strings.TrimSpace(parsed.Path) != "" {
		return filepath.Base(parsed.Path)
	}
	if rawURL = strings.TrimSpace(rawURL); rawURL != "" {
		return filepath.Base(rawURL)
	}
	return ""
}
