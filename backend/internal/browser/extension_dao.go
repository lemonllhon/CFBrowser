package browser

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ExtensionDAO 扩展插件持久化接口。
type ExtensionDAO interface {
	EnsureSchema() error
	List() ([]Extension, error)
	Get(extensionId string) (*Extension, error)
	FindByNameVersion(name string, version string) (*Extension, error)
	Upsert(extension Extension) error
	Delete(extensionId string) error
	CountBindings(extensionId string) (int, error)
	ListBindings(extensionId string) ([]ExtensionBinding, error)
	ListBindingsByProfile(profileId string) ([]ExtensionBinding, error)
	UpsertBinding(binding ExtensionBinding) error
	DeleteBinding(profileId string, extensionId string) error
}

// SQLiteExtensionDAO 基于 SQLite 的 ExtensionDAO 实现。
type SQLiteExtensionDAO struct {
	db *sql.DB
}

// NewSQLiteExtensionDAO 创建 SQLiteExtensionDAO。
func NewSQLiteExtensionDAO(db *sql.DB) *SQLiteExtensionDAO {
	return &SQLiteExtensionDAO{db: db}
}

// EnsureSchema 确保扩展插件表存在，兼容旧库迁移状态异常或手动覆盖程序后的启动场景。
func (d *SQLiteExtensionDAO) EnsureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS browser_extensions (
			extension_id     TEXT PRIMARY KEY,
			name             TEXT NOT NULL DEFAULT '',
			version          TEXT NOT NULL DEFAULT '',
			manifest_version INTEGER NOT NULL DEFAULT 0,
			description      TEXT NOT NULL DEFAULT '',
			source_type      TEXT NOT NULL DEFAULT '',
			source_url       TEXT NOT NULL DEFAULT '',
			install_dir      TEXT NOT NULL DEFAULT '',
			package_path     TEXT NOT NULL DEFAULT '',
			manifest_json    TEXT NOT NULL DEFAULT '',
			created_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_browser_extensions_updated_at ON browser_extensions(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_browser_extensions_source_type ON browser_extensions(source_type)`,
		`CREATE TABLE IF NOT EXISTS browser_profile_extensions (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_id    TEXT NOT NULL,
			extension_id  TEXT NOT NULL,
			mode          TEXT NOT NULL DEFAULT 'shared',
			enabled       INTEGER NOT NULL DEFAULT 1,
			exclusive_dir TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(profile_id, extension_id),
			FOREIGN KEY(extension_id) REFERENCES browser_extensions(extension_id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_browser_profile_extensions_profile_id ON browser_profile_extensions(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_browser_profile_extensions_extension_id ON browser_profile_extensions(extension_id)`,
	}
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			return fmt.Errorf("初始化扩展插件表失败: %w", err)
		}
	}
	return nil
}

// List 查询扩展列表，带绑定实例数量。
func (d *SQLiteExtensionDAO) List() ([]Extension, error) {
	rows, err := d.db.Query(`
		SELECT e.extension_id, e.name, e.version, e.manifest_version, e.description,
		       e.source_type, e.source_url, e.install_dir, e.package_path, e.manifest_json,
		       COALESCE(COUNT(pe.id), 0) AS bound_count,
		       e.created_at, e.updated_at
		FROM browser_extensions e
		LEFT JOIN browser_profile_extensions pe
		  ON pe.extension_id = e.extension_id AND COALESCE(pe.enabled, 1) = 1
		GROUP BY e.extension_id
		ORDER BY e.updated_at DESC, e.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("查询扩展列表失败: %w", err)
	}
	defer rows.Close()

	var list []Extension
	for rows.Next() {
		item, err := scanExtension(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, rows.Err()
}

// Get 查询扩展详情，带绑定实例数量。
func (d *SQLiteExtensionDAO) Get(extensionId string) (*Extension, error) {
	row := d.db.QueryRow(`
		SELECT e.extension_id, e.name, e.version, e.manifest_version, e.description,
		       e.source_type, e.source_url, e.install_dir, e.package_path, e.manifest_json,
		       COALESCE(COUNT(pe.id), 0) AS bound_count,
		       e.created_at, e.updated_at
		FROM browser_extensions e
		LEFT JOIN browser_profile_extensions pe
		  ON pe.extension_id = e.extension_id AND COALESCE(pe.enabled, 1) = 1
		WHERE e.extension_id = ?
		GROUP BY e.extension_id`, extensionId)
	item, err := scanExtension(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("扩展插件不存在: %s", extensionId)
	}
	return item, err
}

// FindByNameVersion 按 manifest 名称和版本查找已登记扩展。
func (d *SQLiteExtensionDAO) FindByNameVersion(name string, version string) (*Extension, error) {
	row := d.db.QueryRow(`
		SELECT e.extension_id, e.name, e.version, e.manifest_version, e.description,
		       e.source_type, e.source_url, e.install_dir, e.package_path, e.manifest_json,
		       COALESCE(COUNT(pe.id), 0) AS bound_count,
		       e.created_at, e.updated_at
		FROM browser_extensions e
		LEFT JOIN browser_profile_extensions pe
		  ON pe.extension_id = e.extension_id AND COALESCE(pe.enabled, 1) = 1
		WHERE lower(e.name) = lower(?) AND e.version = ?
		GROUP BY e.extension_id
		ORDER BY e.updated_at DESC
		LIMIT 1`, name, version)
	item, err := scanExtension(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

// Upsert 新增或更新扩展记录。
func (d *SQLiteExtensionDAO) Upsert(extension Extension) error {
	now := time.Now().Format(time.RFC3339)
	if extension.CreatedAt == "" {
		extension.CreatedAt = now
	}
	if extension.UpdatedAt == "" {
		extension.UpdatedAt = now
	}
	_, err := d.db.Exec(`
		INSERT INTO browser_extensions
		  (extension_id, name, version, manifest_version, description, source_type,
		   source_url, install_dir, package_path, manifest_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(extension_id) DO UPDATE SET
		  name             = excluded.name,
		  version          = excluded.version,
		  manifest_version = excluded.manifest_version,
		  description      = excluded.description,
		  source_type      = excluded.source_type,
		  source_url       = excluded.source_url,
		  install_dir      = excluded.install_dir,
		  package_path     = excluded.package_path,
		  manifest_json    = excluded.manifest_json,
		  updated_at       = excluded.updated_at`,
		extension.ExtensionId, extension.Name, extension.Version, extension.ManifestVersion,
		extension.Description, extension.SourceType, extension.SourceURL, extension.InstallDir,
		extension.PackagePath, extension.ManifestJSON, extension.CreatedAt, extension.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("保存扩展插件失败: %w", err)
	}
	return nil
}

// Delete 删除扩展记录。
func (d *SQLiteExtensionDAO) Delete(extensionId string) error {
	_, err := d.db.Exec(`DELETE FROM browser_extensions WHERE extension_id = ?`, extensionId)
	if err != nil {
		return fmt.Errorf("删除扩展插件失败: %w", err)
	}
	return nil
}

// CountBindings 统计扩展启用中的实例绑定数量。
func (d *SQLiteExtensionDAO) CountBindings(extensionId string) (int, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(1)
		FROM browser_profile_extensions
		WHERE extension_id = ? AND COALESCE(enabled, 1) = 1`, extensionId).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计扩展绑定数量失败: %w", err)
	}
	return count, nil
}

// ListBindings 查询某个扩展绑定的实例列表。
func (d *SQLiteExtensionDAO) ListBindings(extensionId string) ([]ExtensionBinding, error) {
	rows, err := d.db.Query(`
		SELECT pe.id, pe.profile_id, COALESCE(p.profile_name, ''), pe.extension_id,
		       COALESCE(e.name, ''), COALESCE(e.version, ''),
		       pe.mode, pe.enabled, pe.exclusive_dir, pe.created_at, pe.updated_at
		FROM browser_profile_extensions pe
		LEFT JOIN browser_profiles p ON p.profile_id = pe.profile_id
		LEFT JOIN browser_extensions e ON e.extension_id = pe.extension_id
		WHERE pe.extension_id = ?
		ORDER BY pe.updated_at DESC, pe.created_at DESC`, extensionId)
	if err != nil {
		return nil, fmt.Errorf("查询扩展绑定实例失败: %w", err)
	}
	defer rows.Close()

	var list []ExtensionBinding
	for rows.Next() {
		item, err := scanExtensionBinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, rows.Err()
}

// ListBindingsByProfile 查询某个实例绑定的扩展列表。
func (d *SQLiteExtensionDAO) ListBindingsByProfile(profileId string) ([]ExtensionBinding, error) {
	rows, err := d.db.Query(`
		SELECT pe.id, pe.profile_id, COALESCE(p.profile_name, ''), pe.extension_id,
		       COALESCE(e.name, ''), COALESCE(e.version, ''),
		       pe.mode, pe.enabled, pe.exclusive_dir, pe.created_at, pe.updated_at
		FROM browser_profile_extensions pe
		LEFT JOIN browser_profiles p ON p.profile_id = pe.profile_id
		LEFT JOIN browser_extensions e ON e.extension_id = pe.extension_id
		WHERE pe.profile_id = ?
		ORDER BY pe.updated_at DESC, pe.created_at DESC`, profileId)
	if err != nil {
		return nil, fmt.Errorf("查询实例扩展绑定失败: %w", err)
	}
	defer rows.Close()

	var list []ExtensionBinding
	for rows.Next() {
		item, err := scanExtensionBinding(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *item)
	}
	return list, rows.Err()
}

// UpsertBinding 新增或更新实例扩展绑定关系。
func (d *SQLiteExtensionDAO) UpsertBinding(binding ExtensionBinding) error {
	now := time.Now().Format(time.RFC3339)
	if binding.CreatedAt == "" {
		binding.CreatedAt = now
	}
	if binding.UpdatedAt == "" {
		binding.UpdatedAt = now
	}
	enabled := 0
	if binding.Enabled {
		enabled = 1
	}
	_, err := d.db.Exec(`
		INSERT INTO browser_profile_extensions
		  (profile_id, extension_id, mode, enabled, exclusive_dir, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_id, extension_id) DO UPDATE SET
		  mode = excluded.mode,
		  enabled = excluded.enabled,
		  exclusive_dir = excluded.exclusive_dir,
		  updated_at = excluded.updated_at`,
		binding.ProfileId, binding.ExtensionId, binding.Mode, enabled, binding.ExclusiveDir,
		binding.CreatedAt, binding.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("保存扩展绑定关系失败: %w", err)
	}
	return nil
}

// DeleteBinding 删除实例扩展绑定关系。
func (d *SQLiteExtensionDAO) DeleteBinding(profileId string, extensionId string) error {
	_, err := d.db.Exec(`DELETE FROM browser_profile_extensions WHERE profile_id = ? AND extension_id = ?`, profileId, extensionId)
	if err != nil {
		return fmt.Errorf("删除扩展绑定关系失败: %w", err)
	}
	return nil
}

func scanExtension(s scanner) (*Extension, error) {
	var item Extension
	if err := s.Scan(
		&item.ExtensionId, &item.Name, &item.Version, &item.ManifestVersion, &item.Description,
		&item.SourceType, &item.SourceURL, &item.InstallDir, &item.PackagePath, &item.ManifestJSON,
		&item.BoundCount, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanExtensionBinding(s scanner) (*ExtensionBinding, error) {
	var item ExtensionBinding
	var enabled int
	if err := s.Scan(
		&item.Id, &item.ProfileId, &item.ProfileName, &item.ExtensionId,
		&item.ExtensionName, &item.ExtensionVersion, &item.Mode, &enabled,
		&item.ExclusiveDir, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Enabled = enabled != 0
	return &item, nil
}
