package apiframe

import (
	"github.com/enorith/framework"
	"github.com/enorith/http/router"
)

type Config struct {
	ApiURL      string   `yaml:"api_url" default:"api/open"`
	Middlewares []string `yaml:"middlewares"`
}

type APIService[U Model] struct {
	config Config
}

// Register service when app starting, before http server start
// you can configure service, prepare global vars etc.
// running at main goroutine
func (as *APIService[U]) Register(app *framework.App) error {
	app.Configure("api_frame", &as.config)

	WithListHandler[U]()
	WithSaveHandler[U]()
	WithDeleteHandler[U]()
	WithDetailHandler[U]()
	WithModels()

	return nil
}

func WithModels() {
	//RegisterOpenApiModel("open_api", OpenApi{})
}

func (as *APIService[U]) RegisterRoutes(rw *router.Wrapper) {
	rw.Group(func(r *router.Wrapper) {
		var handler OpenApiHandler[U]
		r.Post("/", handler.Handle)
	}, as.config.ApiURL).Middleware(as.config.Middlewares...)
}

func NewApiService[U Model]() *APIService[U] {
	return &APIService[U]{}
}

// func MigratePresetApis(db *gorm.DB) {
// 	log.Printf("[api] migrating apis")

// 	apis, e := resources.GetJson[[]models.OpenApi]("api")
// 	if e != nil {
// 		log.Printf("[api] migrate apis get json failed: %v", e)
// 		return
// 	}
// 	e = db.Scopes(database.WithUpsert(
// 		database.UpsertOptColumns("guid"),
// 		database.UpsertOptUpdateColumns("query", "remarks", "preset", "enabled"),
// 	)).Create(&apis).Error
// 	if e != nil {
// 		log.Printf("[api] migrate apis failed: %v", e)
// 	}
// }
