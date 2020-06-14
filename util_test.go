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
