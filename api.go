package apiframe

import (
	"reflect"
	"sync"

	"github.com/enorith/http/content"
	"github.com/enorith/http/contracts"
	"github.com/enorith/http/validation"
	"gorm.io/gorm"
)

type OpenApiHandler[U Model] struct {
}

type OpenApiQueryHandle[U Model] func(req OpenApiHandleRequest[U], api OpenApi, db *gorm.DB, model any) (any, error)

var (
	openApiQueryHandles = make(map[string]any)
	oaqhMu              = sync.RWMutex{}
)

func RegisterOpenApiQueryHandle[U Model](name string, handle OpenApiQueryHandle[U]) {
	oaqhMu.Lock()
	defer oaqhMu.Unlock()
	openApiQueryHandles[name] = handle
}

func GetOpenApiQueryHandle[U Model](name string) (OpenApiQueryHandle[U], bool) {
	oaqhMu.RLock()
	defer oaqhMu.RUnlock()
	handle, ok := openApiQueryHandles[name]

	if !ok {
		return nil, false
	}

	if h, pass := handle.(OpenApiQueryHandle[U]); pass {
		return h, true
	}

	return nil, false
}

var (
	openApiModels = make(map[string]any)
	oaMu          = sync.RWMutex{}
)

func RegisterOpenApiModel(name string, model any) {
	oaMu.Lock()
	defer oaMu.Unlock()
	openApiModels[name] = model
}

func GetOpenApiModel(name string) (any, bool) {
	oaMu.RLock()
	defer oaMu.RUnlock()
	model, ok := openApiModels[name]

	return model, ok
}

func (OpenApiHandler[U]) Handle(req OpenApiHandleRequest[U], user U, db *gorm.DB) contracts.ResponseContract {
	var api OpenApi
	err := db.Where("guid = ?", req.GUID).Find(&api).Error
	req.User = user

	if err != nil || api.ID == 0 {
		return ErrorMessage("undefind api", 400, map[string]string{"guid": req.GUID})
	}

	if req.User.GetID() == 0 {
		return ErrorMessage("need login", 401, map[string]string{"guid": req.GUID})
	}

	handle, ok := GetOpenApiQueryHandle[U](req.Type)

	if !ok {
		return ErrorMessage("undefind api type", 400, map[string]string{"guid": req.GUID, "type": req.Type})
	}

	var (
		data any
		e    error
	)

	if api.WithModel {
		model, _ := GetOpenApiModel(api.Query.Table)

		if model != nil {
			dataRaw := req.Data.GetRaw()

			if dataRaw != nil {
				validateError := make(validation.ValidateError)
				mt := reflect.TypeOf(model)
				mvp := reflect.New(mt)
				v := mvp.Interface()
				e = req.Data.Unmarshal(&v)
				if e != nil {
					return content.ErrResponseFromError(e, 500, nil)
				}
				// 	defIdx := reflection.SubStructOf(model, models.WithDefaultColumns{})

				// 	if defIdx != -1 {
				// 		var id int64
				// 		req.Data.Get(api.Query.PK, &id)

				// 		reflect.ValueOf(v).Elem().Field(defIdx).Set(reflect.ValueOf(models.WithDefaultColumns{
				// 			ID:        id,
				// 			CreatorID: user.ID,
				// 			EditorID:  user.ID,
				// 			OrgID:     user.OrgID,
				// 			Plat:      req.Platform(),
				// 		}))
				// 	}

				model = v

				if validated, ok := v.(validation.WithValidation); ok {
					rules := validated.Rules()
					for attribute, rules := range rules {
						source := content.JsonInput(dataRaw)
						errs := validation.DefaultValidator.PassesRules(source, attribute, rules)
						if len(errs) > 0 {
							validateError[attribute] = errs
						}
					}
				}

				if len(validateError) > 0 {
					return content.JsonResponse(ErrorResp{
						Code:    422,
						Message: validateError.Error(),
						Errors:  validateError,
					}, 422, nil)
				}
			}
		}
		data, e = handle(req, api, db, model)
	} else {
		data, e = handle(req, api, db, nil)
	}

	if e != nil {
		return content.ErrResponseFromError(e, 500, nil)
	}

	return content.JsonResponse(data, 200, nil)
}
