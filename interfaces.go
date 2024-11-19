package apiframe

import (
	"gorm.io/gorm"
)

type ApiModelBeforeSave[U Model] interface {
	BeforeModelSave(tx *gorm.DB, req OpenApiHandleRequest[U]) error
}

type ApiModelSaveControl[U Model] interface {
	ModelSaveControl(selects []string, req OpenApiHandleRequest[U]) []string
}
type ApiModelQuerySelect[U Model] interface {
	WithQuerySelect(req OpenApiHandleRequest[U]) []string
}

type ApiModelAfterSave[U Model] interface {
	AfterModelSave(tx *gorm.DB, req OpenApiHandleRequest[U]) error
}

type ApiModelDeleteHook[U Model] interface {
	AfterModelDelete(tx *gorm.DB, req OpenApiHandleRequest[U]) error
	BeforeModelDelete(tx *gorm.DB, req OpenApiHandleRequest[U]) error
}

type ApiModelWithQueryScope[U Model] interface {
	WithQueryScope(req OpenApiHandleRequest[U]) func(db *gorm.DB) *gorm.DB
}
