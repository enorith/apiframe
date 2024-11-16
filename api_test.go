package apiframe_test

import (
	"testing"

	"github.com/enorith/apiframe"
)

func TestLoadRelation(t *testing.T) {
	apiframe.WithLoadRelations([]apiframe.QueryField{
		{
			Name: "user.name",
		},
		{
			Name: "user.id",
		},
		{
			Name: "profile.id",
		},
		{
			Name: "profile.age",
		},
	})
}
