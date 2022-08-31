package ggen

import (
	"go/types"

	"github.com/iolivern/ggen/errors"
)

type Filterer interface {
	Filter(FilterEngine) error
}

type Qualifier interface {
	Qualify(*types.Package) string
}

type Plugin interface {

	// Name returns name of the plugin. Each plugin must have a different name.
	Name() string

	// Filter is called to determine which packages will be parsed and which will be skipped. It will be called before
	// Generate. It received a FilterEngine and need to call the following methods:
	//
	//     ParsingPackages: Get a list of all packages available for IncludePackage.
	//                      The plugin can only include packages in this list.
	//     IncludePackage:  Make the package available for Generate.
	Filter(FilterEngine) error

	// Generate is called to actually generate code for the given packages. Only packages passed to
	// FilterEngine.IncludePackage are available for Generate.
	Generate(Engine) error
}

type pluginStruct struct {
	name      string
	index     int
	plugin    Plugin
	enabled   bool
	qualifier types.Qualifier
}

func RegisterPlugin(plugins ...Plugin) error {
	for _, plugin := range plugins {
		if err := theEngine.registerPlugin(plugin); err != nil {
			return errors.Errorf(err, "register plugin %v: %v", plugin.Name(), err)
		}
	}
	return nil
}

func (ng *engine) registerPlugin(plugin Plugin) error {
	name := plugin.Name()
	if name == "" {
		return errors.Errorf(nil, "empty name")
	}
	if plugin == nil {
		return errors.Errorf(nil, "nil plugin")
	}
	if ng.pluginsMap[name] != nil {
		return errors.Errorf(nil, "duplicated pluginStruct name: %v", name)
	}

	pl := &pluginStruct{name: name, plugin: plugin, index: len(ng.plugins)}
	if q, ok := plugin.(Qualifier); ok {
		pl.qualifier = q.Qualify
	}

	ng.plugins = append(ng.plugins, pl)
	ng.pluginsMap[name] = pl
	return nil
}
