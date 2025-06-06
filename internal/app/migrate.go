package app

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrateApp 数据库迁移应用
type MigrateApp struct {
	container *Container
	migrator  *migrate.Migrate
}

// NewMigrateApp 创建迁移应用实例
func NewMigrateApp() (Application, error) {
	container, err := NewContainerWithConfigAndService("configs/config.yaml", "migrate")
	if err != nil {
		return nil, err
	}

	return &MigrateApp{
		container: container,
	}, nil
}

// Start 启动迁移应用
func (m *MigrateApp) Start(ctx context.Context) error {
	return m.Initialize()
}

// Stop 停止迁移应用
func (m *MigrateApp) Stop(ctx context.Context) error {
	return nil
}

// Initialize 初始化迁移应用
func (m *MigrateApp) Initialize() error {
	// 解析命令行参数
	checkFlag := flag.Bool("check", false, "检查源数据库表结构")
	dataFlag := flag.Bool("data", false, "迁移数据")
	flag.Parse()

	// 创建迁移器
	sqlDB, err := m.container.GetSQLDB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %v", err)
	}

	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("创建数据库驱动失败: %v", err)
	}

	m.migrator, err = migrate.NewWithDatabaseInstance(
		"file://migrations",
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("创建迁移器失败: %v", err)
	}

	// 根据参数执行不同操作
	if *checkFlag {
		return m.checkMigrations()
	}

	if *dataFlag {
		return m.migrateData()
	}

	return m.runMigrations()
}

// checkMigrations 检查迁移
func (m *MigrateApp) checkMigrations() error {
	log.Println("检查源数据库表结构...")
	// 这里实现检查逻辑
	return nil
}

// runMigrations 执行迁移
func (m *MigrateApp) runMigrations() error {
	log.Println("执行数据库迁移...")
	if err := m.migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("执行迁移失败: %v", err)
	}
	log.Println("数据库迁移完成")
	return nil
}

// migrateData 迁移数据
func (m *MigrateApp) migrateData() error {
	log.Println("迁移数据...")
	// 这里实现数据迁移逻辑
	return nil
}
