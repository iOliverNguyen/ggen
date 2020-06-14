package ggencmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ng-vu/ggen"
)

var flClean = flag.Bool("clean", false, "clean generated files without generating new files")
var flPlugins = flag.String("plugins", "", "comma separated list of plugins for generating (default to all plugins)")
var flIgnoredPlugins = flag.String("ignored-plugins", "", "comma separated list of plugins to ignore")

func usage() {
	const text = `
Usage: generator [OPTION] PATTERN ...

Options:
`
	fmt.Print(text[1:])
	flag.PrintDefaults()
}

func Start(plugins ...ggen.Plugin) {
	flag.Parse()
	patterns := flag.Args()
	if len(patterns) == 0 {
		usage()
		os.Exit(2)
	}

	enabledPlugins := allPluginNames(plugins)
	if *flPlugins != "" {
		enabledPlugins = strings.Split(*flPlugins, ",")
	}
	if *flIgnoredPlugins != "" {
		ignoredPlugins := strings.Split(*flIgnoredPlugins, ",")
		enabledPlugins = calcEnabledPlugins(enabledPlugins, ignoredPlugins)
	}

	cfg := ggen.Config{
		CleanOnly:      *flClean,
		Namespace:      "o.o",
		EnabledPlugins: enabledPlugins,
		GoimportsArgs:  []string{"-local", "o.o"},
	}

	if err := ggen.RegisterPlugin(plugins...); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	if err := ggen.Start(cfg, patterns...); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func allPluginNames(plugins []ggen.Plugin) []string {
	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name()
	}
	return names
}

func calcEnabledPlugins(plugins []string, ignoredPlugins []string) []string {
	result := make([]string, 0, len(plugins))
	for _, p := range plugins {
		include := true
		for _, ip := range ignoredPlugins {
			if p == ip {
				include = false
				break
			}
		}
		if include {
			result = append(result, p)
		}
	}
	return result
}
