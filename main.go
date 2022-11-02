package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/iolivernguyen/ggen/ggen"
	"github.com/iolivernguyen/ggen/plugins/sample"
)

var flClean = flag.Bool("clean", false, "clean generated files without generating new files")
var flPlugins = flag.String("plugins", "", "comma separated list of plugins for generating (default to all plugins)")
var flIgnoredPlugins = flag.String("ignored-plugins", "", "comma separated list of plugins to ignore")
var flNamespace = flag.String("namespace", "", "only parse and generate packages under this namespace (example: github.com/foo)")

func usage() {
	const text = `
Usage: ggen [OPTION] PATTERN ...

Options:
`
	fmt.Print(text[1:])
	flag.PrintDefaults()
}

func main() {
	Start(
		sample.New(), // sample plugin
	)
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
		Namespace:      *flNamespace,
		EnabledPlugins: enabledPlugins,
		GoimportsArgs:  []string{}, // example: -local github.com/foo
	}

	must(ggen.RegisterPlugin(plugins...))
	must(ggen.Start(cfg, patterns...))
}

func must(err error) {
	if err != nil {
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
		if !contains(ignoredPlugins, p) {
			result = append(result, p)
		}
	}
	return result
}

func contains(ss []string, s string) bool {
	for _, _s := range ss {
		if _s == s {
			return true
		}
	}
	return false
}
