package ggutil

import (
	"go/types"

	"github.com/olvrng/ggen"
)

var _ ggen.Qualifier = &Qualifier{}

type Qualifier struct{}

func (q Qualifier) Qualify(pkg *types.Package) string {
	alias := pkg.Name()
	return alias
}
