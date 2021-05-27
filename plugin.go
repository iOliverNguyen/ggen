package ggen

import (
	"go/types"

	"github.com/olvrng/ggen/errors"
)

type Filterer interface {
	Filter(FilterEngine) error
}

type Qualifier interface {
	Qualify(*types.Package) string
}

type Plugin interface {
	Name() string
	Filter(FilterEngine) error
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
