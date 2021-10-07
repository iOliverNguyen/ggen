package ggen

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/olvrng/ggen/errors"
)

type GenerateFileNameInput struct {
	PluginName string
}

type Config struct {
	// default to "zz_generated.{{.Name}}.go"
	GenerateFileName func(GenerateFileNameInput) string

	EnabledPlugins []string

	CleanOnly bool

	Namespace string

	GoimportsArgs []string

	BuildTags []string
}

func Start(cfg Config, patterns ...string) error {
	return theEngine.clone().start(cfg, patterns...)
}

func (ng *engine) start(cfg Config, patterns ...string) (_err error) {
	{
		if len(patterns) == 0 {
			return errors.Errorf(nil, "no patterns")
		}
		if len(ng.plugins) == 0 {
			return errors.Errorf(nil, "no registed plugins")
		}
		if err := ng.validateConfig(&cfg); err != nil {
			return err
		}
		ng.xcfg = cfg
	}
	buildFlags := getBuildFlags(cfg.BuildTags)
	{
		mode := packages.NeedName | packages.NeedImports | packages.NeedDeps |
			packages.NeedFiles | packages.NeedCompiledGoFiles
		ng.pkgcfg = packages.Config{
			Mode:       mode,
			BuildFlags: buildFlags,
		}
		pkgs, err := packages.Load(&ng.pkgcfg, patterns...)
		if err != nil {
			return errors.Errorf(err, "can not load package: %v", err)
		}

		// populate cleanedFileNames
		cleanedFileNames := make(map[string]bool)
		for _, pl := range ng.enabledPlugins {
			input := GenerateFileNameInput{PluginName: pl.name}
			filename := ng.genFilename(input)
			cleanedFileNames[filename] = true
		}

		// list available packages
		availablePkgs := make([]*packages.Package, 0, len(pkgs))
		for _, pkg := range pkgs {
			pkgDir := getPackageDir(pkg)
			if pkgDir == "" {
				ll.V(1).Printf("no Go files found in package: %v", pkg)
				continue
			}
			availablePkgs = append(availablePkgs, pkg)
			if err = cleanDir(cleanedFileNames, pkgDir); err != nil {
				return err
			}
		}
		ng.cleanedFileNames = cleanedFileNames
		if cfg.CleanOnly {
			return nil
		}

		// populate collectedPackages, includes, srcMap
		if err = ng.collectPackages(availablePkgs); err != nil {
			return err
		}

		if ll.Verbosed(4) {
			for _, pkg := range ng.collectedPackages {
				ll.V(4).Printf("collected package: %v", pkg.PkgPath)
			}
		}
	}
	{
		sortedIncludedPackages := make([]includedPackage, 0, len(ng.includedPackages))
		for pkgPath, included := range ng.includedPackages {
			sortedIncludedPackages = append(sortedIncludedPackages, includedPackage{pkgPath, included})
		}
		sort.Slice(sortedIncludedPackages, func(i, j int) bool {
			return sortedIncludedPackages[i].PkgPath < sortedIncludedPackages[j].PkgPath
		})
		ng.sortedIncludedPackages = sortedIncludedPackages
	}
	{
		pkgPatterns := make([]string, 0, len(ng.includedPatterns)+len(ng.sortedIncludedPackages))
		pkgPatterns = append(pkgPatterns, ng.includedPatterns...)
		for _, pkg := range ng.sortedIncludedPackages {
			pkgPatterns = append(pkgPatterns, pkg.PkgPath)
		}
		if ll.Verbosed(3) {
			ll.V(3).Printf("load all syntax from:")
			for _, p := range pkgPatterns {
				ll.V(3).Printf(p)
			}
		}
		if len(pkgPatterns) == 0 {
			fmt.Println("no packages for generating")
			return nil
		}
		pkgPatterns = append(pkgPatterns, builtinPath) // load builtin types

		ng.pkgcfg = packages.Config{
			Mode:       packages.LoadAllSyntax,
			BuildFlags: buildFlags,
			Overlay:    ng.srcMap,
		}
		pkgs, err := packages.Load(&ng.pkgcfg, pkgPatterns...)
		if err != nil {
			return errors.Errorf(err, "can not load package: %v", err)
		}

		// populate xinfo
		ng.xinfo = newExtendedInfo(ng.pkgcfg.Fset)
		packages.Visit(pkgs,
			func(pkg *packages.Package) bool {
				if cfg.Namespace != "" && !strings.HasPrefix(pkg.PkgPath, cfg.Namespace) {
					return true
				}
				if err2 := ng.xinfo.AddPackage(pkg); err2 != nil {
					_err = err2
					return false
				}
				return true
			}, nil)
		if _err != nil {
			return _err
		}

		// populate pkgMap
		packages.Visit(pkgs,
			func(pkg *packages.Package) bool {
				ng.pkgMap[pkg.PkgPath] = pkg
				return true
			}, nil)

		// populate builtin types
		ng.builtinTypes = parseBuiltinTypes(ng.pkgMap[builtinPath])
		delete(ng.pkgMap, builtinPath)
	}
	{
		// populate generatedFiles
		for _, pl := range ng.enabledPlugins {
			wrapNg := &wrapEngine{engine: ng, plugin: pl}
			if err := pl.plugin.Generate(wrapNg); err != nil {
				return errors.Errorf(err, "%v: %v", pl.name, err)
			}
			for _, gpkg := range wrapNg.pkgs {
				prt := gpkg.printer
				if prt != nil && prt.buf.Len() != 0 {
					// close the printer for writing to file, but only if there
					// are any bytes written
					if err := prt.Close(); err != nil {
						return err
					}
				}
			}
		}
	}
	{
		sort.Strings(ng.generatedFiles)
		fmt.Println("Generated files:")
		pwd, err := os.Getwd()
		must(err)
		for _, filename := range ng.generatedFiles {
			filename, err = filepath.Rel(pwd, filename)
			must(err)
			fmt.Printf("\t./%v\n", filename)
		}
		if err = ng.execGoimport(ng.generatedFiles); err != nil {
			return err
		}
	}
	return nil
}

func (ng *engine) collectPackages(pkgs []*packages.Package) error {
	collectedPackages, fileContents, err := collectPackages(pkgs, ng.cleanedFileNames)
	if err != nil {
		return err
	}
	sort.Slice(collectedPackages, func(i, j int) bool {
		return collectedPackages[i].PkgPath < collectedPackages[j].PkgPath
	})
	pkgMap := map[string][]bool{}
	for _, pl := range ng.enabledPlugins {
		filterNg := &filterEngine{
			ng:       ng,
			plugin:   pl,
			pkgs:     collectedPackages,
			pkgMap:   pkgMap,
			patterns: &ng.includedPatterns,
		}
		if err = pl.plugin.Filter(filterNg); err != nil {
			return errors.Errorf(err, "plugin %v: %v", pl.name, err)
		}
	}
	ng.collectedPackages = collectedPackages
	ng.includedPackages = pkgMap
	ng.mapPkgDirectives = make(map[string][]Directive)
	for _, pkg := range collectedPackages {
		ng.mapPkgDirectives[pkg.PkgPath] = pkg.Directives
	}

	srcMap := make(map[string][]byte)
	for _, content := range fileContents {
		srcMap[content.Path] = content.Body
	}
	ng.srcMap = srcMap
	return nil
}

func getBuildFlags(buildTags []string) []string {
	var buildFlags = "-tags generator"
	if len(buildTags) > 0 {
		buildFlags += "," + strings.Join(buildTags, ",")
	}
	return strings.Split(buildFlags, " ")
}

type fileContent struct {
	Path string
	Body []byte
}

func collectPackages(
	pkgs []*packages.Package,
	cleanedFileNames map[string]bool,
) (collectedPackages []filteringPackage, files []fileContent, _err error) {

	var wg0, wg sync.WaitGroup
	wg0.Add(2)

	fileCh := make(chan fileContent, 4)
	go func() {
		defer wg0.Done()
		for file := range fileCh {
			files = append(files, file)
		}
	}()

	// collect errors
	errCh := make(chan error, 4)
	var errs []error
	go func() {
		defer wg0.Done()
		for err := range errCh {
			errs = append(errs, err)
		}
	}()

	limit := make(chan struct{}, 16) // limit concurrency
	collectedPackages = make([]filteringPackage, len(pkgs))
	for i := range pkgs {
		limit <- struct{}{} // limit
		wg.Add(1)
		go func(i int, pkg *packages.Package) {
			defer func() { wg.Done(); <-limit }() // release limit
			directives, inlineDirectives, err := parseDirectivesFromPackage(fileCh, pkg, cleanedFileNames)
			if err != nil {
				_err = errors.Errorf(err, "parsing %v: %v", pkg.PkgPath, err)
			}
			p := filteringPackage{
				PkgPath:          pkg.PkgPath,
				Imports:          pkg.Imports,
				Directives:       directives,
				InlineDirectives: inlineDirectives,
			}
			collectedPackages[i] = p
		}(i, pkgs[i])
	}
	wg.Wait()
	close(fileCh)
	close(errCh)
	wg0.Wait()
	if len(errs) != 0 {
		_err = errors.Errors("can not parse packages", errs)
	}
	return
}

func cleanDir(cleanedFileNames map[string]bool, pkgDir string) error {
	dir, err := os.Open(pkgDir)
	if err != nil {
		return err
	}
	defer func() { must(dir.Close()) }()
	names, err := dir.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, name := range names {
		if cleanedFileNames[name] {
			absFileName := filepath.Join(pkgDir, name)
			if err = os.Remove(absFileName); err != nil {
				return errors.Errorf(err, "can not remove file %v: %v", absFileName, err)
			}
		}
	}
	return nil
}

func parseDirectivesFromPackage(fileCh chan<- fileContent, pkg *packages.Package, cleanedFileNames map[string]bool) (directives, inlineDirectives []Directive, _err error) {
	for _, file := range pkg.CompiledGoFiles {
		if cleanedFileNames[filepath.Base(file)] {
			continue
		}
		body, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, nil, err
		}
		fileCh <- fileContent{Path: file, Body: body}

		errs := parseDirectivesFromBody(body, &directives, &inlineDirectives)
		if len(errs) != 0 {
			// ignore unknown directives
			if ll.Verbosed(2) {
				for _, e := range errs {
					ll.V(1).Printf("ignored %v", e)
				}
			}
		}
	}
	return
}

var startDirective = []byte(startDirectiveStr)

func parseDirectivesFromBody(body []byte, directives, inlineDirectives *[]Directive) (errs []error) {

	// store processing directives
	var tmp []Directive
	lastIdx := -1
	if bytes.HasPrefix(body, startDirective) {
		lastIdx = 0
	}
	for idx := 0; idx < len(body); idx++ {
		if body[idx] != '\n' {
			continue
		}

		// process the last found directive, remove "// " but keep "+"
		if lastIdx >= 0 {
			line := body[lastIdx+len(startDirective)-1 : idx]
			lastIdx = -1

			ds, err := ParseDirective(string(line))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			tmp = append(tmp, ds...)
		}
		// directives are followed by a blank line, accept them
		if idx+1 < len(body) && body[idx+1] == '\n' {
			*directives = append(*directives, tmp...)
			tmp = tmp[:0]
		}
		// find the next directive
		if !bytes.HasPrefix(body[idx+1:], startDirective) && idx+1 != len(body) {
			// put directives not followed by a blank line into inline directives
			if inlineDirectives != nil {
				*inlineDirectives = append(*inlineDirectives, tmp...)
			}
			tmp = tmp[:0]
			continue
		}
		lastIdx = idx
	}
	// source file should end with a newline, so we don't process remaining lastIdx
	*directives = append(*directives, tmp...)
	return errs
}

func (ng *engine) validateConfig(cfg *Config) (_err error) {
	defer func() {
		if _err != nil {
			_err = errors.Errorf(_err, "config error: %v", _err)
		}
	}()

	// populate enabledPlugins
	if cfg.EnabledPlugins != nil {
		for _, enabled := range cfg.EnabledPlugins {
			pl := ng.pluginsMap[enabled]
			if pl == nil {
				return errors.Errorf(nil, "plugin %v not found", enabled)
			}
			pl.enabled = true
			ng.enabledPlugins = append(ng.enabledPlugins, pl)
		}
	} else {
		for _, pl := range ng.plugins {
			pl.enabled = true
		}
		ng.enabledPlugins = ng.plugins
	}

	if cfg.GenerateFileName == nil {
		cfg.GenerateFileName = defaultGeneratedFileName(defaultGeneratedFileNameTpl)
	}

	if ng.bufPool.New == nil {
		ng.bufPool.New = func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, defaultBufSize))
		}
	}
	return nil
}

func (ng *engine) genFilename(input GenerateFileNameInput) string {
	return ng.xcfg.GenerateFileName(input)
}

func (ng *engine) writeFile(filePath string) (io.WriteCloser, error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	ng.generatedFiles = append(ng.generatedFiles, filePath)
	return f, nil
}

func (ng *engine) execGoimport(files []string) error {
	var args []string
	args = append(args, ng.xcfg.GoimportsArgs...)
	args = append(args, "-w")
	args = append(args, files...)
	cmd := exec.Command("goimports", args...)
	ll.V(4).Printf("goimports %v", args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf(err, "goimports: %s\n\n%s\n", err, out)
	}
	return nil
}
