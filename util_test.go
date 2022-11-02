package ggen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestBuiltinTypes(t *testing.T) {
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, builtinPath)
	require.NoError(t, err)

	pkg := pkgs[0]
	require.NotNil(t, pkg)

	m := parseBuiltinTypes(pkg)
	require.Equal(t, types.Int, m["int"].(*types.Basic).Kind())
	require.Equal(t, "Error", m["error"].Underlying().(*types.Interface).Method(0).Name())
}

func TestParseGoBuild(t *testing.T) {
	t.Run("go:build", func(t *testing.T) {
		directives, err := ParseDirective("go:build")
		require.NoError(t, err)
		require.Equal(t, []Directive{
			{
				Raw: "go:build",
				Cmd: "build",
				Arg: "",
			},
		}, directives)
	})
	t.Run("go:build generator", func(t *testing.T) {
		directives, err := ParseDirective("go:build generator")
		require.NoError(t, err)
		require.Equal(t, []Directive{
			{
				Raw: "go:build generator",
				Cmd: "build",
				Arg: "generator",
			},
		}, directives)
	})
}

func TestParseDirective(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("")
		require.NoError(t, err)
		require.Empty(t, directives)
	})
	t.Run("invalid", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("invalid")
		require.Error(t, err)
		require.Empty(t, []Directive{}, directives)
	})
	t.Run("single", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("+sample")
		require.NoError(t, err)
		require.Equal(t, directives, parser.result)
		require.Equal(t, []Directive{
			{
				Raw: "+sample",
				Cmd: "sample",
				Arg: "",
			},
		}, directives)
	})
	t.Run("single with args", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("+sample=arg1,arg2")
		require.NoError(t, err)
		require.Equal(t, directives, parser.result)
		require.Equal(t, []Directive{
			{
				Raw: "+sample=arg1,arg2",
				Cmd: "sample",
				Arg: "arg1,arg2",
			},
		}, directives)
	})
	t.Run("single with args (2)", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("+sample=arg1,key2=arg2")
		require.NoError(t, err)
		require.Equal(t, directives, parser.result)
		require.Equal(t, []Directive{
			{
				Raw: "+sample=arg1,key2=arg2",
				Cmd: "sample",
				Arg: "arg1,key2=arg2",
			},
		}, directives)
	})
	t.Run("multiple", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("+three +sample +directives")
		require.NoError(t, err)
		require.Equal(t, directives, parser.result)
		require.Equal(t, []Directive{
			{
				Raw: "+three",
				Cmd: "three",
				Arg: "",
			},
			{
				Raw: "+sample",
				Cmd: "sample",
				Arg: "",
			},
			{
				Raw: "+directives",
				Cmd: "directives",
				Arg: "",
			},
		}, directives)
	})
	t.Run("multiple with args", func(t *testing.T) {
		parser := &directiveParser{}
		directives, err := parser.parseDirective("+three=arg1 +sample=arg2 +directives=arg3")
		require.NoError(t, err)
		require.Equal(t, directives, parser.result)
		require.Equal(t, []Directive{
			{
				Raw: "+three=arg1",
				Cmd: "three",
				Arg: "arg1",
			},
			{
				Raw: "+sample=arg2",
				Cmd: "sample",
				Arg: "arg2",
			},
			{
				Raw: "+directives=arg3",
				Cmd: "directives",
				Arg: "arg3",
			},
		}, directives)
	})
}
