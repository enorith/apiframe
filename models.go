package apiframe

import (
	"database/sql/driver"

	"github.com/enorith/framework/http/rules"
	"github.com/enorith/http/content"
	"github.com/enorith/supports/dbutil"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"gorm.io/gorm"
)

type QueryField struct {
	Name     string `json:"name"`
	Label    string `json:"label"`
	Width    string `json:"width"`
	DataType string `json:"dataType"`
	Form     string `json:"form"`
	FormHelp string `json:"formHelp"`
	Filter   string `json:"filter"`
	Exclude  bool   `json:"exclude"`
	Required bool   `json:"required"`
	Omit     bool   `json:"omit"`
	Sorter   bool   `json:"sorter"`
}

const (
	QueryTypeList   = "list"
	QueryTypeSave   = "save"
	QueryTypeDelete = "delete"
	QueryTypeDetail = "detail"
)

type QueryDefine struct {
	Table       string       `json:"table"` // 表名
	Fields      []QueryField `json:"fields"`
	Connection  string       `json:"connection"` // 连接名
	PK          string       `json:"pk"`         // 主键
	CountSelect string       `json:"countSelect"`
	WithCreate  bool         `json:"with_create"`
	WithEdit    bool         `json:"with_edit"`
	WithDelete  bool         `json:"with_delete"`
}

// Scan assigns a value from a database driver.
// The src value will be of one of the following types:
//
//	int64
//	float64
//	bool
//	[]byte
//	string
//	time.Time
//	nil - for NULL values
//
// An error should be returned if the value cannot be stored
// without loss of information.
//
// Reference types such as []byte are only valid until the next call to Scan
// and should not be retained. Their underlying memory is owned by the driver.
// If retention is necessary, copy their values before the next call to Scan.
func (qd *QueryDefine) Scan(src any) error {
	if src == nil {
		return nil
	}
	var val []byte
	if s, ok := src.(string); ok {
		val = []byte(s)
	}

	if s, ok := src.([]byte); ok {
		val = s
	}
	return jsoniter.Unmarshal(val, qd)
}

// Value returns a driver Value.
// Value must not panic.
func (qd QueryDefine) Value() (driver.Value, error) {
	return jsoniter.Marshal(qd)
}

type OpenApi struct {
	content.Request `gorm:"-" json:"-"`
	ID              int64 `gorm:"column:id;primaryKey;not null;type:int;autoIncrement" json:"id"`

	GUID    string      `gorm:"column:guid;type:varchar(64);index:idx_guid;uniqueIndex:idx_unique_guid" json:"guid"`
	Query   QueryDefine `gorm:"column:query;type:text" json:"query" input:"query"`
	Enabled bool        `gorm:"column:enabled;type:tinyint;default:1;comment:是否启用" json:"enabled" input:"enabled"`
	Remarks string      `gorm:"column:remarks;type:varchar(255)" json:"remarks" input:"remarks"`
	Preset  bool        `gorm:"column:preset;type:tinyint;default:0;comment:系统预设" json:"preset"`

	WithModel bool `gorm:"column:with_model;type:tinyint;default:1;comment:是否使用模型" json:"with_model"`
	WithAuth  bool `gorm:"column:with_auth;type:tinyint;default:1;comment:是否需要登录" json:"with_auth"`

	dbutil.WithTimestamps
}

func (OpenApi) TableName() string {
	return "open_api"
}

func (oa OpenApi) Rules() map[string][]interface{} {
	return map[string][]interface{}{
		"guid": {rules.UniqueDefault("open_api", "guid").Ignore(oa.ID)},
	}
}

func (oa *OpenApi) BeforeCreate(db *gorm.DB) error {
	if oa.GUID == "" {
		oa.GUID = uuid.New().String()
	}

	return nil
}

type OpenApiHandleRequest[U Model] struct {
	content.Request
	User    U                      `json:"-" `
	GUID    string                 `json:"guid" input:"guid" validate:"required"`
	Type    string                 `json:"type" input:"type" validate:"required"`
	Fields  []string               `json:"fields" input:"fields"`
	Filters map[string]interface{} `json:"filters" input:"filters"`
	Data    content.MapInput       `json:"data" input:"data"`
	PerPage int                    `json:"per_page" input:"per_page"`
	Page    int                    `json:"page" input:"page"`
	Sort    map[string]string      `json:"sort" input:"sort"`
}

type Model interface {
	GetID() int64
}
