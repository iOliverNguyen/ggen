package ggen

import (
	"golang.org/x/tools/go/packages"
)

type FilteringPackage struct {
	PkgPath          string
	Imports          map[string]*packages.Package
	Directives       Directives
	InlineDirectives Directives

	ng *filterEngine
}

func (p *FilteringPackage) Include() {
	p.ng.IncludePackage(p.PkgPath)
}

type filteringPackage struct {
	PkgPath          string
	Imports          map[string]*packages.Package
	Directives       Directives
	InlineDirectives Directives
}

type includedPackage struct {
	PkgPath  string
	Included []bool
}

type FilterEngine interface {
	IncludePackage(pkgPath string)
	ParsePackage(pkgPath string)
	ParsePackages(patterns ...string)
	ParsingPackages() []*FilteringPackage
}

var _ FilterEngine = &filterEngine{}

type filterEngine struct {
	ng       *engine
	plugin   *pluginStruct
	pkgs     []filteringPackage
	pkgMap   map[string][]bool
	patterns *[]string
}

// IncludePackage indicates that the given package will be included for
// generating. It will be returned later in Engine.GeneratingPackages(). If it
// does not exist, an error with be returned later.
func (ng *filterEngine) IncludePackage(pkgPath string) {
	if pkgPath == "" {
		panic("invalid package path")
	}
	flags := ng.pkgMap[pkgPath]
	if flags == nil {
		flags = make([]bool, len(ng.ng.plugins))
		ng.pkgMap[pkgPath] = flags
	}
	flags[ng.plugin.index] = true
}

// ParsePackage indicates that the given package should be parsed. If it does
// not exist, an error with be returned later.
func (ng *filterEngine) ParsePackage(pkgPath string) {
	if pkgPath == "" {
		panic("invalid package path")
	}
	_, ok := ng.pkgMap[pkgPath]
	if !ok {
		ng.pkgMap[pkgPath] = nil
	}
}

func (ng *filterEngine) ParsePackages(patterns ...string) {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		*ng.patterns = append(*ng.patterns, p)
	}
}

func (ng *filterEngine) ParsingPackages() []*FilteringPackage {
	buf := make([]FilteringPackage, len(ng.pkgs))
	res := make([]*FilteringPackage, len(ng.pkgs))
	for i, pkg := range ng.pkgs {
		p := FilteringPackage{
			PkgPath:          pkg.PkgPath,
			Imports:          pkg.Imports,
			Directives:       cloneDirectives(pkg.Directives),
			InlineDirectives: cloneDirectives(pkg.InlineDirectives),

			ng: ng,
		}
		buf[i] = p
		res[i] = &buf[i]
	}
	return res
}
