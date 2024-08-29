package apiframe

import (
	"github.com/enorith/http/content"
	"github.com/enorith/http/contracts"
	"github.com/enorith/http/validation"
	"github.com/enorith/language"
)

type ErrorResp struct {
	Message string                   `json:"message"`
	Code    int                      `json:"code"`
	Errors  validation.ValidateError `json:"errors"`
}

func ErrorMessage(msg string, code int, params ...map[string]string) contracts.ResponseContract {
	lang, _ := language.T("errors", msg, params...)
	if lang == "" {
		lang = msg
	}

	return content.JsonResponse(ErrorResp{Message: lang, Code: code}, 422, nil)
}
