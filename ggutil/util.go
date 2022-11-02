package ggutil

import (
	"go/types"
)

type Qualifier struct{}

func (q Qualifier) Qualify(pkg *types.Package) string {
	alias := pkg.Name()
	return alias
}
