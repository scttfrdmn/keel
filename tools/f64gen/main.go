// Copyright 2026 Scott Friedman
// SPDX-License-Identifier: Apache-2.0

// Command f64gen writes the float64 copy of keel's middle layer: internal/pack64
// from internal/pack and internal/block64 from internal/block (#190).
//
// # Why a generator, and why this one
//
// float64 needs the blocking nest, the packing and the derived Level-3 routines
// over a second element type. Go generics were built and refused (#190,
// docs/generic-middle.md §Refusal register): float32 output stayed bit-identical
// and fast, but gcshape dictionaries changed the float32 object code, and #135
// holds the float32 middle to "nothing moves". A generator keeps the float32
// sources canonical and untouched, so that holds trivially, and a hand copy is
// refused on its face (two copies drift). So the float32 files are the only
// source, and every float64 file is derived from them here, mechanically.
//
// # The whole transform
//
// It is deliberately small, so a reader can audit it in one sitting:
//
//   - the identifier float32 becomes float64, everywhere it is an identifier
//     (never inside a string or comment);
//   - the package clause takes the generated name (pack64, block64);
//   - the import of internal/pack becomes internal/pack64, kept under the name
//     `pack` so no call site changes;
//   - three kern selectors are renamed to their float64 twins (kernRenames), two
//     are kept (MaxMR, MaxNR), and ANY other kern selector is an error: a new
//     dependency of the middle on kern has to be decided, not carried across by
//     a generator that happened to copy it.
//
// Constants that are really bytes are stated in bytes in the float32 source, as
// 64/unsafe.Sizeof(float32(0)), so the identifier rule carries them to float64
// unchanged in meaning (internal/pack's memmoveFloor, internal/block's tStride).
// The float32 object code is identical either way; the constant folds.
//
// Per-type tuning (KC, MC, NC, MB at float64, #191) goes in varOverrides, which
// replaces a package-level var's initializer in the generated copy. It is empty
// until a float64 measurement says otherwise; a value there is a measured one.
//
// Comments are copied verbatim, so the generated files' prose describes the
// float32 original; the header says so.
//
// # Use
//
//	go run ./tools/f64gen          # from the repository root: rewrite the copies
//	go run ./tools/f64gen -check   # exit 1 if any copy differs from what it would write
//
// TestGeneratedIsCurrent runs the -check comparison in the ordinary test suite,
// so a float32 change that is not regenerated fails CI rather than drifting.
//
// It is apparatus, not library, like everything under tools/.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const module = "github.com/scttfrdmn/keel"

// pkgs are the packages generated, in dependency order.
var pkgs = []struct{ src, dst, name string }{
	{"internal/pack", "internal/pack64", "pack64"},
	{"internal/block", "internal/block64", "block64"},
}

// importRenames maps a float32 package to its generated float64 copy. The import
// keeps its original name, so `pack.APanels` in block stays `pack.APanels`.
var importRenames = map[string]string{
	module + "/internal/pack": module + "/internal/pack64",
}

// kernRenames are the kern identifiers that have a float64 twin (internal/kern/
// kern64.go); kernKept are the type-free ones the middle may use as they are.
var (
	kernRenames = map[string]string{"Kernel": "Kernel64", "ScalarKernel": "ScalarKernel64", "ScalarTile": "ScalarTile64"}
	kernKept    = map[string]bool{"MaxMR": true, "MaxNR": true}
)

// varOverrides replaces a package-level var's initializer in a generated package,
// for float64 tuning that differs from float32's. Keyed by generated package name.
var varOverrides = map[string]map[string]string{}

func main() {
	check := flag.Bool("check", false, "report differences instead of writing")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	files, err := generate(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "f64gen:", err)
		os.Exit(2)
	}
	stale := 0
	for _, path := range sortedKeys(files) {
		full := filepath.Join(*root, path)
		if *check {
			if old, err := os.ReadFile(full); err != nil || !bytes.Equal(old, files[path]) {
				fmt.Println("stale:", path)
				stale++
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "f64gen:", err)
			os.Exit(2)
		}
		if err := os.WriteFile(full, files[path], 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "f64gen:", err)
			os.Exit(2)
		}
	}
	if extra, err := orphans(*root, files); err != nil {
		fmt.Fprintln(os.Stderr, "f64gen:", err)
		os.Exit(2)
	} else if len(extra) > 0 {
		// A generated file whose float32 source is gone: report it either way,
		// because writing new files never deletes old ones.
		for _, p := range extra {
			fmt.Println("orphan:", p)
		}
		stale += len(extra)
	}
	if *check && stale > 0 {
		os.Exit(1)
	}
}

// generate returns every generated file, keyed by repository-relative path.
func generate(root string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, p := range pkgs {
		srcs, err := filepath.Glob(filepath.Join(root, p.src, "*.go"))
		if err != nil {
			return nil, err
		}
		for _, src := range srcs {
			if strings.HasSuffix(src, "_test.go") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(p.src, filepath.Base(src)))
			b, err := transform(src, rel, p.name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", rel, err)
			}
			out[filepath.ToSlash(filepath.Join(p.dst, filepath.Base(src)))] = b
		}
	}
	return out, nil
}

func transform(path, rel, pkgName string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if f.Doc != nil && len(f.Doc.List) > 0 {
		// The package doc names the package; keep it truthful about which one.
		first := f.Doc.List[0]
		if old := "// Package " + f.Name.Name + " "; strings.HasPrefix(first.Text, old) {
			first.Text = "// Package " + pkgName + " is the generated float64 copy of package " + f.Name.Name +
				", which " + strings.TrimPrefix(first.Text, old)
		}
	}
	f.Name.Name = pkgName
	for _, imp := range f.Imports {
		ip, _ := strconv.Unquote(imp.Path.Value)
		if to, ok := importRenames[ip]; ok {
			imp.Path.Value = strconv.Quote(to)
			imp.Name = ast.NewIdent(ip[strings.LastIndex(ip, "/")+1:])
		}
	}
	var bad []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && x.Name == "kern" {
				switch {
				case kernRenames[n.Sel.Name] != "":
					n.Sel.Name = kernRenames[n.Sel.Name]
				case kernKept[n.Sel.Name]:
				default:
					bad = append(bad, fmt.Sprintf("%s: kern.%s has no float64 decision (add it to kernRenames or kernKept)", fset.Position(n.Pos()), n.Sel.Name))
				}
				return false
			}
		case *ast.Ident:
			if n.Name == "float32" {
				n.Name = "float64"
			}
		}
		return true
	})
	if len(bad) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(bad, "; "))
	}
	if ov := varOverrides[pkgName]; ov != nil {
		if err := applyOverrides(f, ov); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return nil, err
	}
	return addHeader(buf.Bytes(), rel)
}

// applyOverrides replaces the initializer of each named package-level var.
func applyOverrides(f *ast.File, ov map[string]string) error {
	seen := map[string]bool{}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, s := range gd.Specs {
			vs := s.(*ast.ValueSpec)
			for i, name := range vs.Names {
				expr, ok := ov[name.Name]
				if !ok || i >= len(vs.Values) {
					continue
				}
				e, err := parser.ParseExpr(expr)
				if err != nil {
					return fmt.Errorf("override %s: %w", name.Name, err)
				}
				vs.Values[i] = e
				seen[name.Name] = true
			}
		}
	}
	for name := range ov {
		if !seen[name] {
			return fmt.Errorf("override %s: no such package-level var", name)
		}
	}
	return nil
}

// addHeader inserts the generated-code marker after the copyright/SPDX lines, where
// `go vet` and editors look for it (before the package clause).
func addHeader(src []byte, rel string) ([]byte, error) {
	const spdx = "// SPDX-License-Identifier: Apache-2.0\n"
	i := bytes.Index(src, []byte(spdx))
	if i < 0 {
		return nil, fmt.Errorf("no SPDX line to anchor the generated header")
	}
	i += len(spdx)
	hdr := fmt.Sprintf("\n// Code generated by tools/f64gen from %s. DO NOT EDIT.\n// Its comments are the float32 original's, copied verbatim.\n", rel)
	return append(append(append([]byte{}, src[:i]...), hdr...), src[i:]...), nil
}

// orphans lists files in the generated directories that generate did not produce.
func orphans(root string, files map[string][]byte) ([]string, error) {
	var extra []string
	for _, p := range pkgs {
		have, err := filepath.Glob(filepath.Join(root, p.dst, "*.go"))
		if err != nil {
			return nil, err
		}
		for _, h := range have {
			rel := filepath.ToSlash(filepath.Join(p.dst, filepath.Base(h)))
			if strings.HasSuffix(rel, "_test.go") {
				continue // hand-written tests of the generated package live beside it
			}
			if _, ok := files[rel]; !ok {
				extra = append(extra, rel)
			}
		}
	}
	return extra, nil
}

func sortedKeys(m map[string][]byte) []string {
	k := make([]string, 0, len(m))
	for p := range m {
		k = append(k, p)
	}
	sort.Strings(k)
	return k
}
