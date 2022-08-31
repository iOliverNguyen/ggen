package sample

import (
	"fmt"

	"github.com/iolivern/ggen"
)

func New() ggen.Plugin {
	return plugin{}
}

var _ ggen.Filterer = plugin{}

type plugin struct{}

func (p plugin) Name() string    { return "sample" }
func (p plugin) Command() string { return "sample" }

func (p plugin) Filter(_ ggen.FilterEngine) error {
	return nil
}

func (p plugin) Generate(ng ggen.Engine) error {
	pkgs := ng.GeneratingPackages()
	for _, gpkg := range pkgs {
		fmt.Printf("package %v:\n", gpkg.Package.PkgPath)
		objects := gpkg.GetObjects()
		for _, obj := range objects {
			fmt.Printf("  %v\t%v\n", obj.Name(), obj.Type())
		}
		fmt.Println()
	}
	return nil
}
