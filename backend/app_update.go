package backend

import (
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
	Size       int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
}

type AppUpdateInfo struct {
	CurrentVersion string          `json:"currentVersion"`
	LatestVersion  string          `json:"latestVersion"`
	ReleaseName    string          `json:"releaseName"`
	ReleaseURL     string          `json:"releaseUrl"`
	PublishedAt    string          `json:"publishedAt"`
	Body           string          `json:"body"`
	HasUpdate      bool            `json:"hasUpdate"`
	Asset          *AppUpdateAsset `json:"asset,omitempty"`
	Message        string          `json:"message"`
}

type AppUpdateDownloadResult struct {
	Cancelled       bool   `json:"cancelled"`
	Message         string `json:"message"`
	Version         string `json:"version"`
	InstallerPath   string `json:"installerPath"`
	InstallOnRestart bool   `json:"installOnRestart"`
}

type pendingAppUpdate struct {
	Version           string `json:"version"`
	InstallerPath     string `json:"installerPath"`
	ReleaseURL        string `json:"releaseUrl"`
	InstallOnNextStart bool   `json:"installOnNextStart"`
	CreatedAt         string `json:"createdAt"`
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

	asset := pickReleaseAsset(release.Assets)
	info := &AppUpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseName:    firstNonEmpty(release.Name, release.TagName),
		ReleaseURL:     firstNonEmpty(release.HTMLURL, PROJECT_GITHUB_URL+"/latest"),
		PublishedAt:    release.PublishedAt,
		Body:           strings.TrimSpace(release.Body),
		HasUpdate:      appReleaseVersionsDiffer(latestVersion, currentVersion),
		Asset:          asset,
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
	} else if asset == nil {
		info.Message = "检测到新版本，但没有找到适合当前系统的安装包"
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
	if info.Asset == nil || strings.TrimSpace(info.Asset.DownloadURL) == "" {
		return nil, fmt.Errorf("没有可下载的更新安装包")
	}
	version := normalizeVersion(firstNonEmpty(info.LatestVersion, info.ReleaseName))
	if version == "" {
		version = "latest"
	}
	updatesDir := a.resolveAppPath(filepath.Join("data", "updates"))
	if err := os.MkdirAll(updatesDir, 0755); err != nil {
		return nil, fmt.Errorf("创建更新目录失败: %w", err)
	}
	fileName := sanitizeFileName(info.Asset.Name)
	if fileName == "" {
		fileName = "trace-browser-update-" + version
	}
	targetPath := filepath.Join(updatesDir, fileName)
	if err := downloadFile(info.Asset.DownloadURL, targetPath); err != nil {
		return nil, err
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

	return &AppUpdateDownloadResult{
		Message:          "更新安装包已下载",
		Version:          version,
		InstallerPath:    targetPath,
		InstallOnRestart: installOnRestart,
	}, nil
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
	if err := startInstaller(target); err != nil {
		return err
	}
	_ = os.Remove(a.pendingAppUpdatePath())
	go func() {
		time.Sleep(600 * time.Millisecond)
		a.ForceQuit()
	}()
	return nil
}

func (a *App) emitPendingAppUpdateIfNeeded() {
	pending, err := a.loadPendingAppUpdate()
	if err != nil || strings.TrimSpace(pending.InstallerPath) == "" {
		return
	}
	if _, err := os.Stat(pending.InstallerPath); err != nil {
		_ = os.Remove(a.pendingAppUpdatePath())
		return
	}
	if a.ctx == nil {
		return
	}
	go func() {
		time.Sleep(1200 * time.Millisecond)
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

func pickReleaseAsset(assets []githubReleaseAsset) *AppUpdateAsset {
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
		return nil
	}
	best := candidates[0]
	return &AppUpdateAsset{Name: best.Name, Size: best.Size, DownloadURL: best.BrowserDownloadURL}
}

func assetScore(name string) int {
	lower := strings.ToLower(name)
	score := 0
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
	if strings.Contains(lower, "installer") || strings.Contains(lower, "setup") {
		score += 5
	}
	return score
}

func downloadFile(url string, targetPath string) error {
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
	tmp := targetPath + "." + shortHash(url) + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("创建下载文件失败: %w", err)
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("写入更新包失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存更新包失败: %w", closeErr)
	}
	if err := os.Rename(tmp, targetPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("保存更新包失败: %w", err)
	}
	return nil
}

func (a *App) pendingAppUpdatePath() string {
	return a.resolveAppPath(filepath.Join("data", "updates", "pending-update.json"))
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
		return nil, err
	}
	var pending pendingAppUpdate
	if err := json.Unmarshal(data, &pending); err != nil {
		return nil, err
	}
	return &pending, nil
}

func startInstaller(path string) error {
	switch goruntime.GOOS {
	case "windows":
		return exec.Command(path).Start()
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
	if compareVersions(latest, current) > 0 {
		return true
	}
	return !strings.EqualFold(latest, current)
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
