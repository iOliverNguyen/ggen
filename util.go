package ggen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"

	"github.com/ng-vu/ggen/log"
)

const defaultGeneratedFileNameTpl = "zz_generated.%v.go"
const defaultBufSize = 1024 * 4
const startDirectiveStr = "// +"

var ll = log.New()
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

// processDoc splits directive and text comment
func processDoc(doc, cmt *ast.CommentGroup) (Comment, error) {
	if doc == nil {
		return Comment{Comment: cmt}, nil
	}

	directives := make([]Directive, 0, 4)
	for _, line := range doc.List {
		if !strings.HasPrefix(line.Text, startDirectiveStr) {
			continue
		}

		// remove "// " but keep "+"
		text := line.Text[len(startDirectiveStr)-1:]
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
		if !strings.HasPrefix(line.Text, startDirectiveStr) {
			processedDoc = append(processedDoc, line)
			continue
		}
	}
	return (&ast.CommentGroup{List: processedDoc}).Text()
}

func ParseDirective(text string) (result []Directive, _ error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "+build") {
		return nil, nil
	}
	result, err := parseDirective(text, result)
	if err != nil {
		return nil, Errorf(err, "%v (%v)", err, text)
	}
	return result, nil
}

func parseDirective(text string, result []Directive) ([]Directive, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if text[0] != '+' {
		return nil, Errorf(nil, "invalid directive")
	}
	cmdIdx := reCommand.FindStringIndex(text)
	if cmdIdx == nil {
		return nil, Errorf(nil, "invalid directive")
	}
	if cmdIdx[0] != 1 {
		return nil, Errorf(nil, "invalid directive")
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
			return nil, Errorf(nil, "invalid directive")
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
				return nil, Errorf(nil, "invalid directive")
			}
			return append(result, directive), nil
		}
		directive.Raw = text[:idx]
		directive.Arg = strings.TrimSpace(text[len(dtext)+1 : idx])
		if directive.Arg == "" {
			return nil, Errorf(nil, "invalid directive")
		}
		result = append(result, directive)
		return parseDirective(text[idx:], result)
	}
	if strings.HasPrefix(remain, "_") {
		return nil, Errorf(nil, "invalid directive (directive commands should contain -, not _)")
	}
	return nil, Errorf(nil, "invalid directive")
}

type listErrors struct {
	Msg    string
	Errors []error
}

func newErrors(msg string, errs []error) error {
	return listErrors{Msg: msg, Errors: errs}
}

func (es listErrors) Error() string {
	return fmt.Sprint(es)
}

func (es listErrors) Format(st fmt.State, c rune) {
	if es.Msg == "" && len(es.Errors) == 0 {
		_, _ = st.Write([]byte("<nil>"))
		return
	}

	width, ok := st.Width()
	if !ok {
		width = 8
	}

	verbose := st.Flag('#') || st.Flag('+')
	var b bytes.Buffer
	if es.Msg != "" {
		b.WriteString(es.Msg)
		if len(es.Errors) == 0 {
			return
		}
		if verbose {
			b.WriteString(":\n")
		} else {
			b.WriteString(": ")
		}
	}
	for i, e := range es.Errors {
		if verbose {
			for j := 0; j < width; j++ {
				b.WriteByte(' ')
			}
		}
		b.WriteString(e.Error())
		if i > 0 {
			if verbose {
				b.WriteString("\n")
			} else {
				b.WriteString("; ")
			}
		}
	}
	_, _ = st.Write(b.Bytes())
}

type stacker interface {
	StackTrace() errors.StackTrace
}

type withMessage struct {
	cause error
	msg   string
}

func (w *withMessage) Error() string { return w.msg }
func (w *withMessage) Cause() error  { return w.cause }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", w.Cause())
			io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, w.Error())
	}
}

func Errorf(err error, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	if err != nil {
		if _, ok := err.(stacker); !ok {
			err = errors.WithStack(err)
		}
		return &withMessage{
			cause: err,
			msg:   msg,
		}
	}
	return errors.New(msg)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
