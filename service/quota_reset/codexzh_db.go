package quota_reset

import (
	"net/url"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// CodexzhDB codexzh 数据库连接实例
var CodexzhDB *gorm.DB

// InitCodexzhDB 初始化 codexzh 数据库连接
// DSN 格式: postgresql://user:password@host:port/database?schema=public
func InitCodexzhDB() error {
	dsn := common.CodexzhSqlDSN
	if dsn == "" {
		common.SysLog("CODEXZH_SQL_DSN 未配置，跳过额度重置功能初始化")
		return nil
	}

	// 处理 DSN 格式，仅移除 schema 参数（GORM 不需要），保留其它参数（例如 sslmode）
	dsn = normalizePostgresDsn(dsn)

	// 使用 PostgreSQL 驱动连接
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // 禁用隐式预编译语句
	}), &gorm.Config{
		PrepareStmt: false, // 关闭预编译语句缓存
	})

	if err != nil {
		common.SysLog("连接 codexzh 数据库失败: " + err.Error())
		return err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		common.SysLog("获取 codexzh 数据库连接池失败: " + err.Error())
		return err
	}

	// 设置较小的连接池，因为这个连接只用于定时任务
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)

	CodexzhDB = db
	common.SysLog("codexzh 数据库连接成功")
	return nil
}

// normalizePostgresDsn 规范化 DSN：移除 schema 参数，保留其它查询参数
func normalizePostgresDsn(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.Scheme == "" || u.Host == "" {
		// 非 URL 形式 DSN（例如 host=... user=...），直接返回
		return dsn
	}
	q := u.Query()
	if q.Has("schema") {
		q.Del("schema")
		u.RawQuery = q.Encode()
		return u.String()
	}
	return dsn
}

// CloseCodexzhDB 关闭 codexzh 数据库连接
func CloseCodexzhDB() error {
	if CodexzhDB != nil {
		sqlDB, err := CodexzhDB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// IsCodexzhDBConnected 检查 codexzh 数据库是否已连接
func IsCodexzhDBConnected() bool {
	return CodexzhDB != nil
}
