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
	CurrentVersion string          `json:"currentVersion"`
	LatestVersion  string          `json:"latestVersion"`
	ReleaseName    string          `json:"releaseName"`
	ReleaseURL     string          `json:"releaseUrl"`
	PublishedAt    string          `json:"publishedAt"`
	Body           string          `json:"body"`
	HasUpdate      bool            `json:"hasUpdate"`
	Asset          *AppUpdateAsset `json:"asset,omitempty"`
	InstallerAsset *AppUpdateAsset `json:"installerAsset,omitempty"`
	PortableAsset  *AppUpdateAsset `json:"portableAsset,omitempty"`
	Message        string          `json:"message"`
}

type AppUpdateDownloadResult struct {
	Cancelled        bool   `json:"cancelled"`
	Message          string `json:"message"`
	Version          string `json:"version"`
	InstallerPath    string `json:"installerPath"`
	PackagePath      string `json:"packagePath"`
	ExtractedPath    string `json:"extractedPath"`
	InstallOnRestart bool   `json:"installOnRestart"`
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
	info := &AppUpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		ReleaseName:    firstNonEmpty(release.Name, release.TagName),
		ReleaseURL:     firstNonEmpty(release.HTMLURL, PROJECT_GITHUB_URL+"/latest"),
		PublishedAt:    release.PublishedAt,
		Body:           strings.TrimSpace(release.Body),
		HasUpdate:      appReleaseVersionsDiffer(latestVersion, currentVersion),
		Asset:          firstAsset(installerAsset, portableAsset),
		InstallerAsset: installerAsset,
		PortableAsset:  portableAsset,
	}
	if !info.HasUpdate {
		info.Message = "当前已是最新版本"
	} else if installerAsset == nil && portableAsset == nil {
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

	return &AppUpdateDownloadResult{
		Message:          "更新安装包已下载",
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
	extractDir := a.resolveAppPath(filepath.Join("data", "updates", "portable-"+version))
	_ = os.RemoveAll(extractDir)
	a.emitAppUpdateDownloadProgress("extracting", 100, "正在解压 ZIP 便携包...")
	if err := unzipFileTo(targetPath, extractDir); err != nil {
		a.emitAppUpdateDownloadProgress("error", 100, "解压 ZIP 便携包失败")
		return nil, fmt.Errorf("解压 ZIP 便携包失败: %w", err)
	}
	a.emitAppUpdateDownloadProgress("done", 100, "ZIP 便携包已解压完成")
	return &AppUpdateDownloadResult{
		Message:       "ZIP 便携包已下载并解压",
		Version:       version,
		PackagePath:   targetPath,
		ExtractedPath: extractDir,
		PackageKind:   "portable",
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
	updatesDir := a.resolveAppPath(filepath.Join("data", "updates"))
	if err := os.MkdirAll(updatesDir, 0755); err != nil {
		return "", fmt.Errorf("创建更新目录失败: %w", err)
	}
	fileName := sanitizeFileName(asset.Name)
	if fileName == "" {
		fileName = "trace-browser-update-" + version
	}
	targetPath := filepath.Join(updatesDir, fileName)
	a.emitAppUpdateDownloadProgress("starting", 0, "准备下载更新包...")
	if err := downloadFile(asset.DownloadURL, targetPath, func(progress int, message string) {
		a.emitAppUpdateDownloadProgress("downloading", progress, message)
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
	if err := startInstaller(target); err != nil {
		return fmt.Errorf("启动安装程序失败: %w", err)
	}
	_ = os.Remove(a.pendingAppUpdatePath())
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
