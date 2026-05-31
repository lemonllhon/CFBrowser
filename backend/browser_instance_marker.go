package backend

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
)

type browserInstanceIdentity struct {
	ProfileId string
	Marker    string
	Index     int
	DebugPort int
	Pid       int
}

func (a *App) browserInstanceIdentityLocked(profile *BrowserProfile) browserInstanceIdentity {
	if profile == nil {
		return browserInstanceIdentity{}
	}
	index := a.browserInstanceDisplayIndexLocked(profile.ProfileId)
	name := browserInstanceMarkerName(profile.ProfileName, profile.ProfileId)
	return browserInstanceIdentity{
		ProfileId: profile.ProfileId,
		Marker:    fmt.Sprintf("[Trace #%02d] %s", index, name),
		Index:     index,
		DebugPort: profile.DebugPort,
		Pid:       profile.Pid,
	}
}

func (a *App) browserInstanceIdentityForProfileID(profileId string) (browserInstanceIdentity, bool) {
	if a == nil || a.browserMgr == nil {
		return browserInstanceIdentity{}, false
	}
	profileId = strings.TrimSpace(profileId)
	if profileId == "" {
		return browserInstanceIdentity{}, false
	}
	a.browserMgr.Mutex.Lock()
	defer a.browserMgr.Mutex.Unlock()
	profile, ok := a.browserMgr.Profiles[profileId]
	if !ok || profile == nil {
		return browserInstanceIdentity{}, false
	}
	return a.browserInstanceIdentityLocked(profile), true
}

func (a *App) browserInstanceDisplayIndexLocked(profileId string) int {
	if a == nil || a.browserMgr == nil || len(a.browserMgr.Profiles) == 0 {
		return 1
	}
	type profileOrderItem struct {
		id        string
		name      string
		createdAt string
	}
	items := make([]profileOrderItem, 0, len(a.browserMgr.Profiles))
	for _, profile := range a.browserMgr.Profiles {
		if profile == nil || strings.TrimSpace(profile.ProfileId) == "" {
			continue
		}
		items = append(items, profileOrderItem{
			id:        profile.ProfileId,
			name:      strings.ToLower(strings.TrimSpace(profile.ProfileName)),
			createdAt: strings.TrimSpace(profile.CreatedAt),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].name != items[j].name {
			return items[i].name < items[j].name
		}
		if items[i].createdAt != items[j].createdAt {
			return items[i].createdAt < items[j].createdAt
		}
		return items[i].id < items[j].id
	})
	for i, item := range items {
		if item.id == profileId {
			return i + 1
		}
	}
	return 1
}

func browserInstanceMarkerName(profileName string, profileId string) string {
	name := strings.Join(strings.Fields(strings.TrimSpace(profileName)), " ")
	if name == "" {
		name = shortBrowserProfileID(profileId)
	}
	name = strings.ReplaceAll(name, "|", "/")
	runes := []rune(name)
	if len(runes) > 28 {
		name = string(runes[:28])
	}
	if strings.TrimSpace(name) == "" {
		return "Instance"
	}
	return name
}

func shortBrowserProfileID(profileId string) string {
	value := strings.TrimSpace(profileId)
	if value == "" {
		return "Instance"
	}
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func (a *App) applyBrowserInstanceIdentityMarker(identity browserInstanceIdentity) {
	if strings.TrimSpace(identity.ProfileId) == "" || strings.TrimSpace(identity.Marker) == "" {
		return
	}

	log := logger.New("Browser")
	script := browserInstanceMarkerScript(identity.Marker)
	iconPath := ""
	if identity.Index > 0 && stdruntime.GOOS == "windows" {
		if path, err := a.browserInstanceMarkerIconPath(identity.Index); err == nil {
			iconPath = path
		} else {
			log.Warn("实例窗口数字图标生成失败",
				logger.F("profile_id", identity.ProfileId),
				logger.F("error", err.Error()),
			)
		}
	}

	delays := []time.Duration{0, 700 * time.Millisecond, 2 * time.Second}
	titleApplied := identity.DebugPort <= 0
	iconApplied := iconPath == ""
	var lastTitleErr error
	var lastIconErr error
	for _, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		if !titleApplied && identity.DebugPort > 0 {
			if err := applyBrowserInstanceTitleMarkerToPort(identity.DebugPort, script); err != nil {
				lastTitleErr = err
			} else {
				titleApplied = true
			}
		}
		if !iconApplied && iconPath != "" {
			if err := setBrowserWindowsIcon(identity.Pid, identity.Marker, iconPath); err != nil {
				lastIconErr = err
			} else {
				iconApplied = true
			}
		}
		if titleApplied && iconApplied {
			return
		}
	}

	if lastTitleErr != nil {
		log.Warn("实例窗口标题标识应用失败",
			logger.F("profile_id", identity.ProfileId),
			logger.F("debug_port", identity.DebugPort),
			logger.F("error", lastTitleErr.Error()),
		)
	}
	if lastIconErr != nil {
		log.Warn("实例窗口数字图标应用失败",
			logger.F("profile_id", identity.ProfileId),
			logger.F("pid", identity.Pid),
			logger.F("error", lastIconErr.Error()),
		)
	}
}

func applyBrowserInstanceTitleMarkerToPort(debugPort int, script string) error {
	if debugPort <= 0 {
		return fmt.Errorf("debug port is required")
	}
	targets, err := pageWebSocketTargets(debugPort)
	if err != nil {
		return err
	}
	applied := 0
	var lastErr error
	for _, target := range targets {
		wsURL := strings.TrimSpace(target.WebSocketURL)
		if wsURL == "" {
			continue
		}
		_, _ = cdpCallWebSocket(wsURL, "Page.enable", nil)
		_, _ = cdpCallWebSocket(wsURL, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": script})
		if _, err := cdpCallWebSocket(wsURL, "Runtime.evaluate", map[string]any{
			"expression":   script,
			"awaitPromise": false,
		}); err != nil {
			lastErr = err
			continue
		}
		applied++
	}
	if applied == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("未找到可标记的浏览器页面")
	}
	return nil
}

func browserInstanceMarkerScript(marker string) string {
	markerJSON, _ := json.Marshal(marker)
	return fmt.Sprintf(`(() => {
  const marker = %s;
  if (!marker) return;
  window.__traceBrowserInstanceMarker = marker;
  if (window.__traceBrowserInstanceMarkerInstalled) {
    if (typeof window.__traceBrowserApplyInstanceMarker === "function") {
      window.__traceBrowserApplyInstanceMarker();
    }
    return;
  }
  window.__traceBrowserInstanceMarkerInstalled = true;
  let applying = false;
  const markerPattern = /^\[Trace #\d{1,3}\]\s+[^|]{0,80}\s+\|\s*/;
  const cleanTitle = (value) => String(value || "").replace(markerPattern, "").trim();
  window.__traceBrowserApplyInstanceMarker = () => {
    if (applying || !document) return;
    const currentMarker = window.__traceBrowserInstanceMarker || marker;
    const prefix = currentMarker + " | ";
    const clean = cleanTitle(document.title);
    const next = prefix + (clean || "New Tab");
    if (document.title === next) return;
    applying = true;
    try {
      document.title = next;
    } finally {
      setTimeout(() => { applying = false; }, 0);
    }
  };
  const installObserver = () => {
    try {
      let title = document.querySelector("title");
      if (!title && document.head) {
        title = document.createElement("title");
        document.head.appendChild(title);
      }
      if (title) {
        const observer = new MutationObserver(() => {
          if (!applying) window.__traceBrowserApplyInstanceMarker();
        });
        observer.observe(title, { childList: true, subtree: true, characterData: true });
      }
    } catch (_) {}
  };
  installObserver();
  window.__traceBrowserApplyInstanceMarker();
  window.addEventListener("pageshow", window.__traceBrowserApplyInstanceMarker, true);
  setInterval(window.__traceBrowserApplyInstanceMarker, 1500);
})();`, string(markerJSON))
}

func (a *App) browserInstanceMarkerIconPath(index int) (string, error) {
	if index <= 0 {
		index = 1
	}
	dir := a.resolveAppPath(filepath.Join("data", "generated", "instance-icons"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("trace-instance-%02d.ico", index))
	data, err := renderBrowserInstanceIcon(index)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func renderBrowserInstanceIcon(index int) ([]byte, error) {
	if index <= 0 {
		index = 1
	}
	label := fmt.Sprintf("%d", index)
	if index > 99 {
		label = "99"
	}
	bg := browserInstanceIconColor(index)
	frames := []struct {
		size int
		data []byte
	}{
		{size: 16},
		{size: 32},
		{size: 48},
	}
	for i := range frames {
		pngData, err := renderBrowserInstanceIconPNG(frames[i].size, label, bg)
		if err != nil {
			return nil, err
		}
		frames[i].data = pngData
	}

	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(frames))); err != nil {
		return nil, err
	}
	offset := uint32(6 + len(frames)*16)
	for _, frame := range frames {
		buf.WriteByte(byte(frame.size))
		buf.WriteByte(byte(frame.size))
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(buf, binary.LittleEndian, uint32(len(frame.data)))
		_ = binary.Write(buf, binary.LittleEndian, offset)
		offset += uint32(len(frame.data))
	}
	for _, frame := range frames {
		if _, err := buf.Write(frame.data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func renderBrowserInstanceIconPNG(size int, label string, bg color.RGBA) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size-1) / 2
	cy := float64(size-1) / 2
	radius := float64(size) * 0.47
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := dx*dx + dy*dy
			if dist <= radius*radius {
				img.SetRGBA(x, y, bg)
			}
		}
	}
	drawBrowserInstanceIconLabel(img, label, color.RGBA{R: 15, G: 23, B: 42, A: 120}, 1, 1)
	drawBrowserInstanceIconLabel(img, label, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0, 0)

	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func browserInstanceIconColor(index int) color.RGBA {
	palette := []color.RGBA{
		{R: 37, G: 99, B: 235, A: 255},
		{R: 5, G: 150, B: 105, A: 255},
		{R: 220, G: 38, B: 38, A: 255},
		{R: 147, G: 51, B: 234, A: 255},
		{R: 202, G: 138, B: 4, A: 255},
		{R: 8, G: 145, B: 178, A: 255},
		{R: 219, G: 39, B: 119, A: 255},
		{R: 22, G: 163, B: 74, A: 255},
	}
	if index <= 0 {
		index = 1
	}
	return palette[(index-1)%len(palette)]
}

func drawBrowserInstanceIconLabel(img *image.RGBA, label string, clr color.RGBA, offsetX int, offsetY int) {
	bounds := img.Bounds()
	size := bounds.Dx()
	if label == "" {
		return
	}
	scale := maxInt(1, size/14)
	if len(label) == 1 {
		scale = maxInt(1, size/8)
	}
	charWidth := 5 * scale
	charHeight := 7 * scale
	gap := scale
	totalWidth := len(label)*charWidth + maxInt(0, len(label)-1)*gap
	startX := bounds.Min.X + (size-totalWidth)/2 + offsetX
	startY := bounds.Min.Y + (size-charHeight)/2 + offsetY
	for i, ch := range label {
		pattern, ok := browserInstanceDigitFont[ch]
		if !ok {
			continue
		}
		x0 := startX + i*(charWidth+gap)
		for row, line := range pattern {
			for col, pixel := range line {
				if pixel != '1' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						x := x0 + col*scale + dx
						y := startY + row*scale + dy
						if image.Pt(x, y).In(bounds) {
							img.SetRGBA(x, y, clr)
						}
					}
				}
			}
		}
	}
}

var browserInstanceDigitFont = map[rune][]string{
	'0': {"11111", "10001", "10011", "10101", "11001", "10001", "11111"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"11111", "00001", "00001", "11111", "10000", "10000", "11111"},
	'3': {"11111", "00001", "00001", "11111", "00001", "00001", "11111"},
	'4': {"10001", "10001", "10001", "11111", "00001", "00001", "00001"},
	'5': {"11111", "10000", "10000", "11111", "00001", "00001", "11111"},
	'6': {"11111", "10000", "10000", "11111", "10001", "10001", "11111"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"11111", "10001", "10001", "11111", "10001", "10001", "11111"},
	'9': {"11111", "10001", "10001", "11111", "00001", "00001", "11111"},
}
