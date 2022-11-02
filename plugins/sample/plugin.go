package sample

import (
	"github.com/iolivern/ggen"
)

func New() ggen.Plugin {
	return plugin{}
}

var _ ggen.Filterer = plugin{}

type plugin struct{}

func (p plugin) Name() string    { return "sample" }
func (p plugin) Command() string { return "sample" }

func (p plugin) Filter(ft ggen.FilterEngine) error {
	for _, pkg := range ft.ParsingPackages() {
		ft.IncludePackage(pkg.PkgPath)
		ft.Debug("include package", "pkg", pkg.PkgPath)
	}
	return nil
}

func (p plugin) Generate(ng ggen.Engine) error {
	pkgs := ng.GeneratingPackages()
	for _, gpkg := range pkgs {
		ng.Debug("generate package", "pkg", gpkg.Package.PkgPath)
		objects := gpkg.GetObjects()
		for _, obj := range objects {
			ng.Debug("  object", "name", obj.Name(), "type", obj.Type())
		}
	}
	return nil
}
