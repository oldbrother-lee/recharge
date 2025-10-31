package model

import "time"

// PullSource 拉单平台源
type PullSource struct {
	ID               int64     `json:"id" gorm:"primaryKey;type:bigint"`
	Name             string    `json:"name" gorm:"size:64;not null;comment:平台名称"`
	Code             string    `json:"code" gorm:"size:32;uniqueIndex;not null;comment:平台代码"`
	BaseURL          string    `json:"base_url" gorm:"size:255;comment:平台基础URL"`
	AppKey           string    `json:"app_key" gorm:"size:255;comment:平台AppKey/密钥"`
	AccountName      string    `json:"account_name" gorm:"size:255;comment:平台账号名称"`
	AccountPassword  string    `json:"account_password" gorm:"size:255;comment:平台账号密码(按需存储)"`
	BindUserID       *int64    `json:"bind_user_id" gorm:"index;comment:绑定的本地用户ID"`
	PullAction       string    `json:"pull_action" gorm:"size:64;comment:拉单动作名(如orderlist/pull_order)"`
	Enabled          bool      `json:"enabled" gorm:"type:tinyint(1);default:1;comment:是否启用"`
	MaxConcurrency   int       `json:"max_concurrency" gorm:"comment:最大并发拉取数"`
	PollIntervalSec  int       `json:"poll_interval_sec" gorm:"comment:默认轮询间隔秒"`
	Remark           string    `json:"remark" gorm:"size:255;comment:备注"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	
	// 关联字段（用于查询时显示绑定用户名，不存储到数据库）
	BindUserName     string    `json:"bind_user_name" gorm:"-"`
}

func (PullSource) TableName() string { return "pull_sources" }

// PullVariantConfig 拉单平台源变体配置（按运营商+面值）
type PullVariantConfig struct {
	ID              int64      `json:"id" gorm:"primaryKey;type:bigint"`
	SourceID        int64      `json:"source_id" gorm:"index;not null"`
	ISP             int        `json:"isp" gorm:"comment:运营商编码：1移动 2电信 3联通 0未知"`
	FaceValue       float64    `json:"face_value" gorm:"type:decimal(10,2);comment:面值"`
	ProductID       *int64     `json:"product_id" gorm:"index;comment:关联的商品ID"`
	Enabled         bool       `json:"enabled" gorm:"type:tinyint(1);default:1"`
	PollIntervalSec int        `json:"poll_interval_sec" gorm:"comment:轮询间隔秒"`
	Concurrency     int        `json:"concurrency" gorm:"comment:并发度"`
	Cursor          string     `json:"cursor" gorm:"column:cursor_token;size:255;comment:拉取游标"`
	LastPullAt      *time.Time `json:"last_pull_at"`
	FailCount       int        `json:"fail_count" gorm:"default:0"`
	NotifyURL       string     `json:"notify_url" gorm:"size:255"`
	CreatedAt       time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (PullVariantConfig) TableName() string { return "pull_source_variants" }

// PullProductMap 商品映射（外部→内部）
type PullProductMap struct {
	ID             int64     `json:"id" gorm:"primaryKey;type:bigint"`
	SourceID       int64     `json:"source_id" gorm:"index;not null"`
	KeyType        string    `json:"key_type" gorm:"size:32;not null"`
	ExternalCode   string    `json:"external_code" gorm:"size:64"`
	ISP            *int      `json:"isp"`
	FaceValue      *float64  `json:"face_value"`
	ProductID      int64     `json:"product_id" gorm:"not null"`
	AmountOverride *float64  `json:"amount_override"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (PullProductMap) TableName() string { return "pull_source_product_map" }