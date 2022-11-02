package ggen

import (
	"fmt"
	"go/ast"
	"go/types"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"

	_ "github.com/iolivern/ggen/builtin"
	"github.com/iolivern/ggen/errors"
	"github.com/iolivern/ggen/lg"
)

const defaultGeneratedFileNameTpl = "zz_generated.%v.go"
const defaultBufSize = 1024 * 4
const startDirectiveStr = "// +"   // +directive
const startDirectiveStr2 = "//go:" // go:build generator

var ll = lg.New()
var reCommand = regexp.MustCompile(`[a-z]([a-z0-9.:-]*[a-z0-9])?`)

func FilterByCommand(command string) CommandFilter {
	return CommandFilter(command)
}

type CommandFilter string

func (cmd CommandFilter) Filter(ng FilterEngine) error {
	for _, p := range ng.ParsingPackages() {
		if cmd.Include(p.Directives) {
			p.Include()
		}
	}
	return nil
}

func (cmd CommandFilter) FilterAll(ng FilterEngine) error {
	for _, p := range ng.ParsingPackages() {
		if cmd.Include(p.Directives) {
			p.Include()
		} else if cmd.Include(p.InlineDirectives) {
			p.Include()
		}
	}
	return nil
}

func (cmd CommandFilter) Include(ds Directives) bool {
	for _, d := range ds {
		if d.Cmd == string(cmd) ||
			strings.HasPrefix(d.Cmd, string(cmd)) && d.Cmd[len(cmd)] == ':' {
			return true
		}
	}
	return false
}

func defaultGeneratedFileName(tpl string) func(GenerateFileNameInput) string {
	return func(input GenerateFileNameInput) string {
		return fmt.Sprintf(tpl, input.PluginName)
	}
}

var builtinPath = reflect.TypeOf((*Engine)(nil)).Elem().PkgPath() + "/builtin"

func parseBuiltinTypes(pkg *packages.Package) map[string]types.Type {
	if pkg.PkgPath != builtinPath {
		panic(fmt.Sprintf("unexpected path %v", pkg.PkgPath))
	}
	m := map[string]types.Type{}
	s := pkg.Types.Scope()
	for _, name := range s.Names() {
		if !strings.HasPrefix(name, "_") {
			continue
		}
		typ := s.Lookup(name).Type()
		m[typ.String()] = typ
	}
	return m
}

func getPackageDir(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return ""
}

func hasStartDirective(line string) bool {
	return strings.HasPrefix(line, startDirectiveStr) || strings.HasPrefix(line, startDirectiveStr2)
}

// processDoc splits directive and text comment
func processDoc(doc, cmt *ast.CommentGroup) (Comment, error) {
	if doc == nil {
		return Comment{Comment: cmt}, nil
	}

	directives := make([]Directive, 0, 4)
	for _, line := range doc.List {
		if hasStartDirective(line.Text) {
			continue
		}

		// remove "// " but keep "+"
		text := strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
		_directives, err := ParseDirective(text)
		if err != nil {
			return Comment{}, err
		}
		directives = append(directives, _directives...)
	}

	comment := Comment{
		Doc:        doc,
		Comment:    cmt,
		Directives: directives,
	}
	return comment, nil
}

func processDocText(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	processedDoc := make([]*ast.Comment, 0, len(doc.List))
	for _, line := range doc.List {
		if hasStartDirective(line.Text) {
			processedDoc = append(processedDoc, line)
			continue
		}
	}
	return (&ast.CommentGroup{List: processedDoc}).Text()
}

// ParseDirectiveFromFile reads from file and returns the parsed directives.
func ParseDirectiveFromFile(filename string) (directives, inlineDirective []Directive, err error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return ParseDirectiveFromBody(body)
}

// ParseDirectiveFromBody reads directives from body and returns the parsed directives.
func ParseDirectiveFromBody(body []byte) (directives, inlineDirective []Directive, err error) {
	errs := parseDirectivesFromBody(body, &directives, &inlineDirective)
	err = Errors("can not parse directive", errs)
	return
}

// ParseDirective parses directives from a single line.
func ParseDirective(line string) (result []Directive, _ error) {
	line = strings.TrimSpace(strings.TrimPrefix(line, "//"))
	if line == "+build" || strings.HasPrefix(line, "+build ") ||
		line == "go:build" || strings.HasPrefix(line, "go:build ") {
		return parseBuildDirective(line)
	}
	result, err := parseDirective(line, result)
	if err != nil {
		return nil, errors.Errorf(err, "%v (%v)", err, line)
	}
	return result, nil
}

// TODO(iOliverN): support new directive format //go:build
// https://go.googlesource.com/proposal/+/master/design/draft-gobuild.md
func parseBuildDirective(text string) ([]Directive, error) {
	arg := strings.TrimPrefix(text, "+build")
	if arg == text {
		arg = strings.TrimPrefix(text, "go:build")
	}
	arg = strings.TrimSpace(arg)
	directive := Directive{
		Raw: text,
		Cmd: "build",
		Arg: arg,
	}
	return []Directive{directive}, nil
}

func parseDirective(text string, result []Directive) ([]Directive, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if text[0] != '+' {
		return nil, errors.Errorf(nil, "invalid directive")
	}
	cmdIdx := reCommand.FindStringIndex(text)
	if cmdIdx == nil {
		return nil, errors.Errorf(nil, "invalid directive")
	}
	if cmdIdx[0] != 1 {
		return nil, errors.Errorf(nil, "invalid directive")
	}
	dtext := text[:cmdIdx[1]]
	directive := Directive{
		Cmd: dtext[1:], // remove "+"
	}
	remain := text[len(dtext):]
	if remain == "" {
		directive.Raw = dtext
		return append(result, directive), nil
	}
	if remain[0] == ' ' || remain[0] == '\t' {
		directive.Raw = dtext
		result = append(result, directive)
		return parseDirective(remain, result)
	}
	if remain[0] == ':' {
		remain = remain[1:] // remove ":"
		directive.Raw = text
		directive.Arg = strings.TrimSpace(remain)
		if directive.Arg == "" {
			return nil, errors.Errorf(nil, "invalid directive")
		}
		return append(result, directive), nil
	}
	if remain[0] == '=' {
		remain = remain[1:] // remove "="
		idx := strings.IndexAny(text, " \t")
		if idx < 0 {
			directive.Raw = text
			directive.Arg = strings.TrimSpace(remain)
			if directive.Arg == "" {
				return nil, errors.Errorf(nil, "invalid directive")
			}
			return append(result, directive), nil
		}
		directive.Raw = text[:idx]
		directive.Arg = strings.TrimSpace(text[len(dtext)+1 : idx])
		if directive.Arg == "" {
			return nil, errors.Errorf(nil, "invalid directive")
		}
		result = append(result, directive)
		return parseDirective(text[idx:], result)
	}
	if strings.HasPrefix(remain, "_") {
		return nil, errors.Errorf(nil, "invalid directive (directive commands should contain -, not _)")
	}
	return nil, errors.Errorf(nil, "invalid directive")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
