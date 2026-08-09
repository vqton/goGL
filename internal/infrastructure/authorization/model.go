package authorization

import (
	_ "embed"

	"github.com/casbin/casbin/v3/model"
)

// rbacModelSource is the embedded access model definition.
//
//go:embed rbac_model.conf
var rbacModelSource string

// RBACModel parses the embedded RBAC access model into a casbin model.
func RBACModel() (model.Model, error) {
	return model.NewModelFromString(rbacModelSource)
}
