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
