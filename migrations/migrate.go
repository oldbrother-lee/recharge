package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Migration 表示一个数据库迁移
type Migration struct {
	Version string
	Up      string
	Down    string
}

// Migrator 处理数据库迁移
type Migrator struct {
	db *sql.DB
}

// NewMigrator 创建一个新的迁移器
func NewMigrator(dsn string) (*Migrator, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// 创建迁移表
	if err := createMigrationTable(db); err != nil {
		return nil, err
	}

	return &Migrator{db: db}, nil
}

// createMigrationTable 创建迁移记录表
func createMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS migrations (
            version VARCHAR(14) NOT NULL,
            applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (version)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
    `)
	return err
}

// RunMigrations 执行所有未应用的迁移
func (m *Migrator) RunMigrations() error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if _, ok := applied[migration.Version]; !ok {
			if err := m.applyMigration(migration); err != nil {
				return err
			}
		}
	}

	return nil
}

// loadMigrations 加载所有迁移文件
func (m *Migrator) loadMigrations() ([]Migration, error) {
	files, err := ioutil.ReadDir("migrations/sql")
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(file.Name(), ".sql")
		content, err := ioutil.ReadFile(filepath.Join("migrations/sql", file.Name()))
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, Migration{
			Version: version,
			Up:      string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations 获取已应用的迁移
func (m *Migrator) getAppliedMigrations() (map[string]struct{}, error) {
	rows, err := m.db.Query("SELECT version FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]struct{})
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = struct{}{}
	}

	return applied, nil
}

// applyMigration 应用单个迁移
func (m *Migrator) applyMigration(migration Migration) error {
	// 分割 SQL 语句，使用正则表达式匹配 CREATE TABLE 语句
	statements := strings.Split(migration.Up, "CREATE TABLE")
	if len(statements) == 0 {
		return fmt.Errorf("no SQL statements found")
	}

	// 第一个元素可能是空字符串或注释，跳过
	for i := 1; i < len(statements); i++ {
		stmt := "CREATE TABLE" + statements[i]
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		// 执行 SQL 语句
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute SQL: %v\nSQL: %s", err, stmt)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	// 记录迁移
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for recording migration: %v", err)
	}

	if _, err := tx.Exec("INSERT INTO migrations (version) VALUES (?)", migration.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %s: %v", migration.Version, err)
	}

	return tx.Commit()
}

// Close 关闭数据库连接
func (m *Migrator) Close() error {
	return m.db.Close()
}

// CheckSourceTables 检查源数据库中的表结构和数据
func (m *Migrator) CheckSourceTables() error {
	// 检查产品表
	table := "dyr_product"
	log.Printf("\n检查产品表: %s", table)

	// 检查表结构
	log.Printf("\n表结构:")
	rows, err := m.db.Query(fmt.Sprintf("DESCRIBE recharge.%s", table))
	if err != nil {
		return fmt.Errorf("无法获取表结构: %v", err)
	}
	defer rows.Close()

	var fields []string
	for rows.Next() {
		var field, type_ string
		var null, key sql.NullString
		var default_ sql.NullString
		var extra string
		if err := rows.Scan(&field, &type_, &null, &key, &default_, &extra); err != nil {
			return fmt.Errorf("failed to scan field: %v", err)
		}

		nullStr := "NO"
		if null.Valid {
			nullStr = null.String
		}

		keyStr := ""
		if key.Valid {
			keyStr = key.String
		}

		defaultStr := "NULL"
		if default_.Valid {
			defaultStr = default_.String
		}

		log.Printf("字段: %s, 类型: %s, 允许空: %s, 键: %s, 默认值: %s, 额外: %s",
			field, type_, nullStr, keyStr, defaultStr, extra)
		fields = append(fields, field)
	}

	// 检查数据量
	var count int
	err = m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM recharge.%s", table)).Scan(&count)
	if err != nil {
		return fmt.Errorf("无法获取数据量: %v", err)
	}
	log.Printf("\n总数据量: %d", count)

	// 显示前5条数据的详细信息
	if count > 0 {
		log.Printf("\n前5条数据:")
		query := fmt.Sprintf("SELECT * FROM recharge.%s LIMIT 5", table)
		rows, err := m.db.Query(query)
		if err != nil {
			return fmt.Errorf("无法获取数据: %v", err)
		}
		defer rows.Close()

		// 获取列名
		cols, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get columns: %v", err)
		}

		// 准备数据容器
		values := make([]sql.NullString, len(cols))
		scanArgs := make([]interface{}, len(cols))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// 打印每一行数据
		rowNum := 1
		for rows.Next() {
			if err := rows.Scan(scanArgs...); err != nil {
				return fmt.Errorf("failed to scan row: %v", err)
			}

			log.Printf("\n--- 记录 %d ---", rowNum)
			for i, col := range cols {
				if values[i].Valid {
					log.Printf("%s: %s", col, values[i].String)
				} else {
					log.Printf("%s: NULL", col)
				}
			}
			rowNum++
		}
	}

	return nil
}

// CleanMigration 清理指定版本的迁移记录
func (m *Migrator) CleanMigration(version string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	if _, err := tx.Exec("DELETE FROM migrations WHERE version = ?", version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete migration record: %v", err)
	}

	return tx.Commit()
}

// RunSpecificMigration 执行指定版本的迁移
func (m *Migrator) RunSpecificMigration(version string) error {
	// 清理迁移记录
	if err := m.CleanMigration(version); err != nil {
		return err
	}

	// 检查源数据库中的表结构和数据
	if err := m.CheckSourceTables(); err != nil {
		return err
	}

	files, err := ioutil.ReadDir("migrations/sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	var migrationFile string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), version+"_") {
			migrationFile = file.Name()
			break
		}
	}

	if migrationFile == "" {
		return fmt.Errorf("migration file not found for version %s", version)
	}

	content, err := ioutil.ReadFile(filepath.Join("migrations/sql", migrationFile))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %v", err)
	}

	// 创建迁移对象
	migration := Migration{
		Version: version,
		Up:      string(content),
	}

	// 执行迁移
	if err := m.applyMigration(migration); err != nil {
		return err
	}

	return nil
}

// MigrateData 迁移数据
func (m *Migrator) MigrateData() error {
	// 读取数据迁移SQL文件
	files, err := ioutil.ReadDir("migrations/sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	// 按文件名排序
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	// 执行数据迁移
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "_data.sql") {
			continue
		}

		content, err := ioutil.ReadFile(filepath.Join("migrations/sql", file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
		}

		// 分割SQL语句
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// 执行SQL
			if _, err := m.db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %v\nSQL: %s", file.Name(), err, stmt)
			}

			log.Printf("成功执行SQL语句: %s", stmt[:50]+"...")
		}

		log.Printf("成功执行数据迁移: %s", file.Name())
	}

	return nil
}
