package backend

import (
	appconfig "ant-chrome/backend/internal/config"
	"path/filepath"
	"testing"
)

func TestLoadConfigMigratesLocalLicenseStateToUnlimited(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")

	cfg := appconfig.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	if err := saveLocalLicenseState(configPath, &localLicenseState{
		MaxProfileLimit: appconfig.GithubStarProfileTotal + appconfig.StandardCDKeyProfileBonus,
		UsedCDKeys:      []string{"GITHUB_STAR_REWARD", "ANT-AAAA-BBBB-CCCC-DDDD-EEEEEEEE"},
	}); err != nil {
		t.Fatalf("写入本机额度状态失败: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	if loaded.App.MaxProfileLimit != appconfig.DefaultMaxProfileLimit {
		t.Fatalf("本机额度状态未迁移为无限制: got=%d", loaded.App.MaxProfileLimit)
	}
	if len(loaded.App.UsedCDKeys) != 0 {
		t.Fatalf("兑换记录应清空: %+v", loaded.App.UsedCDKeys)
	}
}

func TestLoadConfigClearsLicenseStateFromConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")

	cfg := appconfig.DefaultConfig()
	cfg.App.MaxProfileLimit = appconfig.GithubStarProfileTotal
	cfg.App.UsedCDKeys = []string{"GITHUB_STAR_REWARD"}
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	if loaded.App.MaxProfileLimit != appconfig.DefaultMaxProfileLimit {
		t.Fatalf("LoadConfig 应迁移为无限制: got=%d", loaded.App.MaxProfileLimit)
	}
	if len(loaded.App.UsedCDKeys) != 0 {
		t.Fatalf("LoadConfig 应清空兑换记录: %+v", loaded.App.UsedCDKeys)
	}

	state, exists, err := loadLocalLicenseState(configPath)
	if err != nil {
		t.Fatalf("读取本机额度状态失败: %v", err)
	}
	if !exists {
		t.Fatalf("应当从现有配置补建本机额度状态")
	}
	if state.MaxProfileLimit != appconfig.DefaultMaxProfileLimit {
		t.Fatalf("本机额度状态应为无限制: got=%d", state.MaxProfileLimit)
	}
	if len(state.UsedCDKeys) != 0 {
		t.Fatalf("本机兑换记录应清空: %+v", state.UsedCDKeys)
	}
}

func TestRedeemGithubStarKeepsUnlimitedLicenseState(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")

	cfg := appconfig.DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	app := NewApp(root)
	app.config = cfg

	if err := app.RedeemGithubStar(); err != nil {
		t.Fatalf("RedeemGithubStar 失败: %v", err)
	}

	state, exists, err := loadLocalLicenseState(configPath)
	if err != nil {
		t.Fatalf("读取本机额度状态失败: %v", err)
	}
	if !exists {
		t.Fatalf("兑换后应写入本机额度状态")
	}
	if state.MaxProfileLimit != appconfig.DefaultMaxProfileLimit {
		t.Fatalf("兑换后本机额度状态应保持无限制: got=%d", state.MaxProfileLimit)
	}
	if len(state.UsedCDKeys) != 0 {
		t.Fatalf("兑换后本机兑换记录应清空: %+v", state.UsedCDKeys)
	}
}
