package apiframe

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/enorith/gormdb"
	"github.com/enorith/supports/dbutil"
	jsoniter "github.com/json-iterator/go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func WithListHandler[U Model]() {
	const DefaultPageSize = 15

	type PageMeta struct {
		Total    int64           `json:"total"`
		PerPage  int             `json:"per_page"`
		Page     int             `json:"page"`
		LastPage int             `json:"last_page"`
		From     int             `json:"from"`
		To       int             `json:"to"`
		Fields   []QueryField    `json:"fields"`
		Permits  map[string]bool `json:"permits"`
	}

	RegisterOpenApiQueryHandle(QueryTypeList, func(req OpenApiHandleRequest[U], apiModel OpenApi, db *gorm.DB, model any) (any, error) {
		dbl, e := getDBFromApi(apiModel, db)
		if e != nil {
			return nil, e
		}

		var (
			meta PageMeta
		)

		if model != nil {
			mt := reflect.TypeOf(model)

			v := reflect.New(mt).Interface()
			dbl = dbl.Model(v)
			if qs, ok := v.(ApiModelWithQueryScope[U]); ok {
				dbl = dbl.Scopes(qs.WithQueryScope(req))
			}
			dbl = dbl.Scopes(WithLoadRelations(apiModel.Query.Fields))
		}

		newTx := dbl.Session(&gorm.Session{
			NewDB: true,
		})

		pk := apiModel.Query.PK
		selects := make([]string, 0)
		var qPk string
		if pk != "" {
			qPk = fmt.Sprintf("%s.%s", apiModel.Query.Table, apiModel.Query.PK)
			selects = append(selects, qPk)
		}

		for _, field := range apiModel.Query.Fields {
			if field.Omit || field.Name == pk {
				continue
			}
			if !strings.Contains(field.Name, ".") {
				selects = append(selects, field.Name)
			}
		}

		dbl = dbutil.ApplyFilters(dbl.Table(apiModel.Query.Table).Select(selects), req.Filters)

		page := req.Page
		perPage := req.PerPage
		if page < 1 {
			page = 1
		}
		if perPage < 1 {
			perPage = DefaultPageSize
		}

		meta.Page = page
		meta.PerPage = perPage
		meta.From = perPage*(page-1) + 1
		meta.Fields = apiModel.Query.Fields

		meta.Permits = map[string]bool{
			"list":   true,
			"create": apiModel.Query.WithCreate,
			"edit":   apiModel.Query.WithEdit,
			"delete": apiModel.Query.WithDelete,
		}

		aggTable := dbl.Session(&gorm.Session{})
		delete(aggTable.Statement.Clauses, "ORDER BY")

		e = newTx.Table("(?) aggragate", aggTable.Scopes(func(d *gorm.DB) *gorm.DB {
			countSelect := qPk
			if apiModel.Query.CountSelect != "" {
				countSelect = apiModel.Query.CountSelect
			}

			if countSelect == "" {
				return d
			}

			return d.Select(countSelect)
		})).Count(&meta.Total).Error
		if e != nil {
			return nil, e
		}

		callFind := func(v interface{}) error {
			var defSorts []string
			if qPk != "" {
				defSorts = append(defSorts, qPk+" DESC")
			}

			return dbutil.ApplySorts(dbl.Session(&gorm.Session{}), req.Sort, defSorts...).Limit(perPage).Offset((page - 1) * perPage).Find(v).Error
		}

		data := make([]any, 0)
		if model != nil {
			mt := reflect.TypeOf(model)
			sv := reflect.MakeSlice(reflect.SliceOf(mt), 0, 0)
			dv := sv.Interface()
			e = callFind(&dv)
			if e != nil {
				return nil, e
			}
			vdv := reflect.ValueOf(dv)
			for i := 0; i < vdv.Len(); i++ {
				data = append(data, vdv.Index(i).Interface())
			}
		} else {
			rows := make([]map[string]any, 0)
			e = callFind(&rows)
			if e != nil {
				return nil, e
			}
			for _, row := range rows {
				data = append(data, row)
			}
		}

		meta.LastPage = int(math.Ceil(float64(meta.Total) / float64(perPage)))
		meta.To = meta.From + int(db.RowsAffected-1)

		return map[string]any{
			"meta": meta,
			"data": data,
		}, nil
	})
}

func printJson(v any) {
	data, _ := jsoniter.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func ucfirst(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

// ConvertSnakeToCamel converts a snake_case string to camelCase.
func snakeToCamel(s string) string {
	// Split the string by underscores
	words := strings.Split(s, "_")

	// Lowercase the first word
	// words[0] = strings.ToLower(words[0])

	// Capitalize the first letter of each subsequent word
	for i := 0; i < len(words); i++ {
		words[i] = ucfirst(words[i])
	}

	// Join the words together into a single string
	return strings.Join(words, "")
}

func WithLoadRelations(fields []QueryField) func(*gorm.DB) *gorm.DB {
	relations := make(map[string][]string, 0)
	for _, field := range fields {
		if field.Omit {
			continue
		}

		if strings.Contains(field.Name, ".") {
			parts := strings.SplitN(field.Name, ".", 2)
			key := snakeToCamel(parts[0])
			relations[key] = append(relations[key], parts[1])
		}
	}
	// printJson(relations)

	return func(db *gorm.DB) *gorm.DB {

		for k, v := range relations {
			db = db.Preload(k, func(tx *gorm.DB) *gorm.DB {
				return tx.Select(v)
			})
		}

		return db
	}
}

func WithSaveHandler[U Model]() {

	RegisterOpenApiQueryHandle(QueryTypeSave, func(req OpenApiHandleRequest[U], apiModel OpenApi, db *gorm.DB, model any) (any, error) {
		define := apiModel.Query

		dbc, e := getDBFromApi(apiModel, db)
		if e != nil {
			return nil, e
		}

		var data any

		callSave := func(fn func(tx *gorm.DB, selects []string, update bool) (any, error)) (any, error) {
			var id int64
			req.Data.Get(define.PK, &id)
			selects := make([]string, 0)

			if len(req.Fields) > 0 {
				selects = req.Fields
			} else {
				for _, field := range define.Fields {
					if field.Form == "" {
						continue
					}
					selects = append(selects, field.Name)
				}
			}
			var val any
			e = dbc.Transaction(func(tx *gorm.DB) error {

				var err error
				if model == nil && id > 0 {
					tx = tx.Where(define.PK+" = ?", id)
				}

				if model == nil {
					tx = tx.Table(define.Table)
				}

				tx.Omit(clause.Associations)
				val, err = fn(tx.Select(selects), selects, id > 0)
				return err
			})
			return val, e
		}

		if model != nil {
			data, e = callSave(func(tx *gorm.DB, selects []string, update bool) (any, error) {
				var err error

				if am, ok := model.(ApiModelSaveControl[U]); ok {
					selects = am.ModelSaveControl(selects, req)
				}

				tx.Select(selects)

				if am, ok := model.(ApiModelBeforeSave[U]); ok {
					err = am.BeforeModelSave(tx.Session(&gorm.Session{NewDB: true}), req)
					if err != nil {
						return nil, err
					}
				}

				if update {
					err = tx.Model(model).Updates(model).Error
				} else {
					err = tx.Model(model).Create(model).Error
				}

				if err != nil {
					return nil, err
				}

				if am, ok := model.(ApiModelAfterSave[U]); ok {
					err = am.AfterModelSave(tx.Session(&gorm.Session{
						NewDB: true,
					}), req)
				}

				return model, err
			})
		} else {
			var mapData = make(map[string]any, 0)
			e = req.Data.Unmarshal(&mapData)

			if e != nil {
				return nil, e
			}

			data, e = callSave(func(tx *gorm.DB, selects []string, update bool) (any, error) {
				if update {
					return req.Data, tx.Updates(mapData).Error
				}

				return req.Data, tx.Create(mapData).Error
			})
		}

		return data, e
	})
}

func WithDeleteHandler[U Model]() {

	RegisterOpenApiQueryHandle(QueryTypeDelete, func(req OpenApiHandleRequest[U], api OpenApi, db *gorm.DB, model any) (any, error) {
		dbc, e := getDBFromApi(api, db)

		if e != nil {
			return nil, e
		}

		define := api.Query
		var id int64
		req.Data.Get(define.PK, &id)
		if model != nil {
			err := dbc.Transaction(func(tx *gorm.DB) error {
				tx.Omit(clause.Associations)
				am, ok := model.(ApiModelDeleteHook[U])
				if ok {
					if e = am.BeforeModelDelete(tx, req); e != nil {
						return e
					}
				}

				if e = tx.Model(model).Delete(model, define.PK+" = ?", id).Error; e != nil {
					return e
				}

				if ok {
					e = am.AfterModelDelete(tx, req)
				}

				return e
			})

			return req.Data, err
		}

		return req.Data, dbc.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", define.Table, define.PK), id).Error
	})
}

func WithDetailHandler[U Model]() {
	RegisterOpenApiQueryHandle(QueryTypeDetail, func(req OpenApiHandleRequest[U], apiModel OpenApi, db *gorm.DB, model any) (any, error) {
		dbc, e := getDBFromApi(apiModel, db)

		if e != nil {
			return nil, e
		}
		define := apiModel.Query
		qPk := fmt.Sprintf("%s.%s", define.Table, define.PK)

		var id int64
		req.Data.Get(define.PK, &id)
		if model != nil {
			dbc = dbc.Model(model).Scopes(WithLoadRelations(define.Fields))
			if qs, ok := model.(ApiModelWithQueryScope[U]); ok {
				dbc = dbc.Scopes(qs.WithQueryScope(req))
			}
			e = dbc.First(model).Error

			return model, e
		}

		var data = map[string]any{}

		e = dbc.Table(define.Table).Where(qPk+" = ?", id).Find(&data).Error

		return data, e
	})
}

func getDBFromApi(api OpenApi, def *gorm.DB) (*gorm.DB, error) {
	define := api.Query
	conn := define.Connection

	if conn != "" {
		return gormdb.DefaultManager.GetConnection(conn)
	}

	return def, nil
}
