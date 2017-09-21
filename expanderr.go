// Copyright 2013 The Go Authors. All rights reserved.
// Copyright 2014 The Go Authors. All rights reserved.
// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"unicode"

	"github.com/stapelberg/expanderr/internal/srcimporter"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	unsafeFastImporter = flag.Bool("unsafe_fast_importer",
		false,
		"import installed packages when possible (unsafe until golang.org/issues/19337 is fixed)")
)

// parseQueryPos parses the source query position pos and returns the path
// enclosing the specified interval.
// (based on parseQueryPos from github.com/golang/tools/cmd/guru/guru.go)
func parseQueryPos(fset *token.FileSet, root *ast.File, pos string, needExact bool) ([]ast.Node, error) {
	filename, startOffset, endOffset, err := parsePos(pos)
	if err != nil {
		return nil, err
	}

	// Find the named file among those in the loaded program.
	var file *token.File
	fset.Iterate(func(f *token.File) bool {
		if sameFile(filename, f.Name()) {
			file = f
			return false // done
		}
		return true // continue
	})
	if file == nil {
		return nil, fmt.Errorf("file %s not found in loaded program", filename)
	}

	start, end, err := fileOffsetToPos(file, startOffset, endOffset)
	if err != nil {
		return nil, err
	}

	// decrement startOffset as long as it points to <whitespace>|")", so that PathEnclosingInterval returns an ast.CallExpr
	//log.Printf("filename = %q, startOffset = %d, endOffset = %d\n", filename, startOffset, endOffset)
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	//log.Printf("before: %q (rune: %v)", string(b[startOffset-5:endOffset+5]), rune(b[startOffset-1]))
	for unicode.IsSpace(rune(b[startOffset-1])) || b[startOffset-1] == ')' {
		startOffset--
		endOffset--
		//log.Printf("decremented to startOffset = %d, endOffset = %d\n", startOffset, endOffset)
	}
	//log.Printf("after: %q", string(b[startOffset-5:endOffset+5]))

	start, end, err = fileOffsetToPos(file, startOffset, endOffset)
	if err != nil {
		return nil, err
	}

	path, exact := astutil.PathEnclosingInterval(root, start, end)
	if path == nil {
		return nil, fmt.Errorf("no syntax here")
	}
	if needExact && !exact {
		return nil, fmt.Errorf("ambiguous selection within %s", astutil.NodeDescription(path[0]))
	}
	return path, nil
}

func unparen(e ast.Expr) ast.Expr { return astutil.Unparen(e) }

var errUnknownSignature = errors.New("unknown signature")

func signatureOf(info *types.Info, e *ast.CallExpr) (*types.Signature, error) {
	// Deal with obviously static calls before constructing SSA form.
	// Some static calls may yet require SSA construction,
	// e.g.  f := func(){}; f().
	switch funexpr := unparen(e.Fun).(type) {
	case *ast.Ident:
		switch obj := info.Uses[funexpr].(type) {
		case *types.Builtin:
			// Reject calls to built-ins.
			return nil, fmt.Errorf("this is a call to the built-in '%s' operator", obj.Name())
		case *types.Func:
			// This is a static function call
			return obj.Type().(*types.Signature), nil
		default:
			// TODO: better error message: the function signature for <TODO> could not be found
			//return nil, fmt.Errorf("unhandled: info.Uses[%v] = %T", funexpr, obj)
			return nil, errUnknownSignature
		}
	case *ast.SelectorExpr:
		sel := info.Selections[funexpr]
		if sel == nil {
			// qualified identifier.
			// May refer to top level function variable
			// or to top level function.
			switch callee := info.Uses[funexpr.Sel].(type) {
			case *types.Func:
				return callee.Type().(*types.Signature), nil
			default:
				// TODO: better error message (see above)
				return nil, errUnknownSignature
			}
		} else if sel.Kind() == types.MethodVal {
			// Inspect the receiver type of the selected method.
			// If it is concrete, the call is statically dispatched.
			// (Due to implicit field selections, it is not enough to look
			// at sel.Recv(), the type of the actual receiver expression.)
			method := sel.Obj().(*types.Func)
			return method.Type().(*types.Signature), nil
		}
	}
	return nil, fmt.Errorf("unhandled: signature of %T", unparen(e.Fun))
}

func currentSignature(path []ast.Node) (*ast.FuncType, error) {
	for _, n := range path {
		switch n := n.(type) {
		case *ast.FuncDecl:
			return n.Type, nil
		case *ast.FuncLit:
			return n.Type, nil
		}
	}
	return nil, fmt.Errorf("no function definition found in path")
}

// newZeroValueNode returns an AST expr representing the zero value of
// typ. If determining the zero value requires additional information
// (e.g., type-checking output), it returns nil.
// (from github.com/sqs/goreturns/returns/fix.go)
func newZeroValueNode(typ ast.Expr) ast.Expr {
	switch v := typ.(type) {
	case *ast.Ident:
		switch v.Name {
		case "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64", "byte", "rune", "uint", "int", "uintptr":
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		case "float32", "float64":
			return &ast.BasicLit{Kind: token.FLOAT, Value: "0"}
		case "complex64", "complex128":
			return &ast.BasicLit{Kind: token.IMAG, Value: "0"}
		case "bool":
			return &ast.Ident{Name: "false"}
		case "string":
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}
		case "error":
			return &ast.Ident{Name: "nil"}
		}
	case *ast.ArrayType:
		if v.Len == nil {
			// slice
			return &ast.Ident{Name: "nil"}
		}
		return &ast.CompositeLit{Type: v}
	case *ast.StarExpr:
		return &ast.Ident{Name: "nil"}
	}
	return nil
}

// Like newZeroValueNode, but with type information.
//
// TODO: can we safely get rid of newZeroValueNode or does it handle cases which
// would otherwise go unhandled?
func newZeroValueNodeTypeName(id *ast.Ident, name *types.TypeName) ast.Expr {
	switch t := name.Type().Underlying().(type) {
	case *types.Struct:
		return &ast.Ident{Name: id.Name + "{}"}

	case *types.Interface:
		return &ast.Ident{Name: "nil"}

	case *types.Basic:
		switch t.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
			return &ast.BasicLit{Kind: token.INT, Value: "0"}

		case types.Float32, types.Float64:
			return &ast.BasicLit{Kind: token.FLOAT, Value: "0"}

		case types.Complex64, types.Complex128:
			return &ast.BasicLit{Kind: token.IMAG, Value: "0"}

		case types.Bool:
			return &ast.Ident{Name: "false"}

		case types.String:
			return &ast.BasicLit{Kind: token.STRING, Value: `""`}
		}
	}
	return nil
}

// fallbackImporter tries to import using importer first, falling back to
// srcImporter on any error. This allows us to load binary files (significantly
// faster) where possible, but import from source where necessary.
type fallbackImporter struct {
	importer    types.ImporterFrom
	srcImporter types.ImporterFrom
}

func (fi *fallbackImporter) Import(path string) (*types.Package, error) {
	p, err := fi.importer.Import(path)
	if err != nil {
		return fi.srcImporter.Import(path)
	}
	return p, err
}

func (fi *fallbackImporter) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	p, err := fi.importer.ImportFrom(path, srcDir, mode)
	if err != nil {
		return fi.srcImporter.ImportFrom(path, srcDir, mode)
	}
	return p, err
}

func callExprAtPath(path []ast.Node) *ast.CallExpr {
	var ce *ast.CallExpr
	// Return the outer-most *ast.CallExpr in path, if any.
	for _, p := range path {
		if e, ok := p.(*ast.CallExpr); ok {
			ce = e
		}
	}
	if ce != nil {
		return ce
	}

	// Look for an *ast.CallExpr within the *ast.BlockStmt, if path starts with
	// an *ast.BlockStmt.
	if first, ok := path[0].(*ast.BlockStmt); ok {
		ast.Inspect(first, func(n ast.Node) bool {
			if n, ok := n.(*ast.CallExpr); ok {
				ce = n
				return false // found, stop
			}

			return true // recurse
		})
		if ce != nil {
			return ce
		}
	}

	return nil // no *ast.CallExpr found
}

func defaultImporter() types.Importer {
	// TODO(golang.org/issues/19337): default to fallbackImporter once packages
	// are augmented.
	if *unsafeFastImporter {
		return &fallbackImporter{
			importer:    importer.Default().(types.ImporterFrom),
			srcImporter: importer.For("source", nil).(types.ImporterFrom),
		}
	}
	i := importer.For("source", nil)
	if i == nil {
		// Fallback for Go <1.9
		i = srcimporter.New(&build.Default, token.NewFileSet(), make(map[string]*types.Package))
	}
	return i
}

// expansion holds state during the error expansion.
type expansion struct {
	fset    *token.FileSet
	file    *ast.File        // the file under cursor
	ce      *ast.CallExpr    // the call expression under the cursor
	callee  *types.Signature // the callee’s signature
	caller  *ast.FuncType    // the caller’s type (including signature)
	results []ast.Expr       // return values for the new error check
	info    *types.Info      // type information of the type-checked package
	pkg     *types.Package
	path    []ast.Node // node under cursor and all its ancestors
}

func (e *expansion) getScope() *types.Scope {
	for _, p := range e.path {
		if funcDecl, ok := p.(*ast.FuncDecl); ok {
			if s, ok := e.info.Scopes[funcDecl.Type]; ok {
				return s
			}
		}
		if s, ok := e.info.Scopes[p]; ok {
			return s
		}
	}
	return nil
}

func errPresent(lhs []ast.Expr) bool {
	for _, expr := range lhs {
		if id, ok := expr.(*ast.Ident); ok && id.Name == "err" {
			return true
		}
	}
	return false
}

// parent returns the parent node of n within e.path.
func (e *expansion) parent(n ast.Node) ast.Node {
	found := false
	for _, p := range e.path {
		if found {
			return p
		}
		found = (p == n)
	}
	return nil
}

func (e *expansion) typeCheck(pkgname string, files []*ast.File) error {
	e.info = &types.Info{
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	conf := types.Config{
		Importer: defaultImporter(),
		Error: func(err error) {
			log.Printf("ignoring type-checking error: %v", err)
		}, // keep going on errors
	}
	pkg, _ := conf.Check(pkgname, e.fset, files, e.info)
	// Type checking errors are ignored so that we can write expressions like
	// “n := io.Write(p)” (two values assigned to one variable).

	e.pkg = pkg

	var err error
	e.caller, err = currentSignature(e.path)
	if err != nil {
		return err
	}

	if e.caller.Results == nil {
		return fmt.Errorf("current function returns no values, cannot return error")
	}

	e.results = make([]ast.Expr, len(e.caller.Results.List))
	for idx, res := range e.caller.Results.List {
		if id, ok := res.Type.(*ast.Ident); ok && id.Name == "error" {
			e.results[idx] = &ast.Ident{Name: "err"}
		} else {
			e.results[idx] = newZeroValueNode(res.Type)
			if e.results[idx] == nil {
				// We could not figure out from the AST what the type is, so
				// it’s not a builtin type, array type or pointer type.
				if id, ok := res.Type.(*ast.Ident); ok {
					if tn, ok := e.info.Uses[id].(*types.TypeName); ok {
						e.results[idx] = newZeroValueNodeTypeName(id, tn)
					}
				}
			}
		}
	}

	e.ce = callExprAtPath(e.path)
	if e.ce == nil {
		return fmt.Errorf("no ast.CallExpr found")
	}
	e.callee, err = signatureOf(e.info, e.ce)
	return err
}

func logic(w io.Writer, buildctx *build.Context, posn string) error {
	e := expansion{
		fset: token.NewFileSet(),
	}

	// Short-cut: parse+type-check a single file before loading the entire
	// package.
	filename, _, _, err := parsePos(posn)
	if err != nil {
		return err
	}

	e.file, err = parser.ParseFile(e.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing: %v", err)
	}

	e.path, err = parseQueryPos(e.fset, e.file, posn, false)
	if err != nil {
		return err
	}

	// TODO(golang.org/issues/21418): hack: importer.For always uses
	// build.Default, so we need to change build.Default
	build.Default = *buildctx

	if err := e.typeCheck("main", []*ast.File{e.file}); err != nil {
		if err != errUnknownSignature {
			return err
		}

		// Parse all files, type-check again.
		d, err := os.Open(filepath.Dir(filename))
		if err != nil {
			return err
		}
		defer d.Close()
		names, err := d.Readdirnames(-1)
		if err != nil {
			return err
		}
		files := []*ast.File{e.file}
		// TODO: parallelize
		for _, n := range names {
			if n == filepath.Base(filename) {
				continue // already parsed
			}
			if strings.HasPrefix(n, "expanderr") {
				continue // skip expanderr temp file when working in /tmp
			}
			if !strings.HasSuffix(n, ".go") {
				continue
			}
			f, err := parser.ParseFile(e.fset, filepath.Join(filepath.Dir(filename), n), nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parsing: %v", err)
			}
			files = append(files, f)
		}
		if err := e.typeCheck(e.pkg.Name(), files); err != nil {
			return err
		}
	}

	var subject ast.Node // what will be replaced
	subject = e.ce
	var repl []ast.Node
	switch e.callee.Results().Len() {
	case 0:
		// nothing to replace, i.e. keep the original *ast.CallExpr
		repl = []ast.Node{e.ce}
	case 1: // TODO: verify the return value is of type error

		// TODO: check if this CallExpr is within an AssignStmt. if so, replace the AssignStmt instead
		if parent := e.parent(subject); parent != nil {
			if stmt, ok := parent.(*ast.AssignStmt); ok {
				subject = stmt
			}
		}
		// e.g. os.Remove(…) → if err := os.Remove(…); err != nil { return 0, err }
		repl = []ast.Node{&ast.IfStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "err"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{e.ce},
			},
			Cond: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "err"},
				Op: token.NEQ,
				Y:  &ast.Ident{Name: "nil"},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{Results: e.results},
				},
			},
		}}
	default:
		// e.g. f := os.Create(…) → f, err := os.Create(…); if err != nil { return 0, err }

		// walk up to the *ast.AssignStmt and append an *ast.Ident to its .Lhs
		var as *ast.AssignStmt
		for _, p := range e.path {
			if p, ok := p.(*ast.AssignStmt); ok {
				as = p
				break
			}
		}
		if as == nil {
			return fmt.Errorf("no *ast.AssignStmt found in path") // TODO: better error msg
		}

		scope := e.getScope()
		if scope == nil {
			return fmt.Errorf("could not find scope") // TODO: better error msg. can this happen at all?
		}
		errInScope := scope.Lookup("err") != nil

		subject = as

		onlyUnderscore := true
		for _, lhs := range as.Lhs {
			switch lhs := lhs.(type) {
			case *ast.Ident:
				if lhs.Name != "_" {
					onlyUnderscore = false
				}
			default:
				onlyUnderscore = false
			}
		}

		// TODO: verify all other parameters are assigned

		if !errPresent(as.Lhs) {
			as.Lhs = append(as.Lhs, &ast.Ident{Name: "err"})
		}

		if !onlyUnderscore && as.Tok == token.DEFINE {
			// Insert a new *ast.IfStmt after the *ast.CallExpr.
			repl = []ast.Node{
				subject,
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  &ast.Ident{Name: "err"},
						Op: token.NEQ,
						Y:  &ast.Ident{Name: "nil"},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{Results: e.results},
						},
					},
				},
			}
		} else {
			tok := as.Tok
			if onlyUnderscore {
				tok = token.DEFINE
			}
			if !onlyUnderscore && !errInScope {
				// The “err” identifier is not yet in scope, so insert a “var
				// err error” declaration before the *ast.IfStmt.
				repl = append(repl, &ast.DeclStmt{
					Decl: &ast.GenDecl{
						Tok: token.VAR,
						Specs: []ast.Spec{
							&ast.ValueSpec{
								Names: []*ast.Ident{&ast.Ident{Name: "err"}},
								Type:  &ast.Ident{Name: "error"},
							},
						},
					},
				})
			}
			// Embed the *ast.CallExpr in an *ast.IfStmt.
			repl = append(repl, &ast.IfStmt{
				Init: &ast.AssignStmt{
					Lhs: as.Lhs,
					Tok: tok,
					Rhs: []ast.Expr{e.ce},
				},
				Cond: &ast.BinaryExpr{
					X:  &ast.Ident{Name: "err"},
					Op: token.NEQ,
					Y:  &ast.Ident{Name: "nil"},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{Results: e.results},
					},
				},
			})
		}
	}

	// TODO(golang.org/issues/20744): switch from textual replacement to
	// formatting the AST once comments are represented in a more convenient
	// way.

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var src bytes.Buffer
	// Copy everything before the subject as-is.
	if _, err := src.Write(b[:subject.Pos()-1]); err != nil {
		return err
	}
	// Print the replacement
	for _, node := range repl {
		// Hack: inline comments (e.g. os.Remove("/foo" /* path */)) get lost
		// when formatting node, so we string-replace the formatted CallExpr
		// with the original CallExpr.
		var stmtFmt, ceFmt bytes.Buffer
		if err := format.Node(&stmtFmt, e.fset, node); err != nil {
			return fmt.Errorf("formatting replacement: %v", err)
		}
		if err := format.Node(&ceFmt, e.fset, e.ce); err != nil {
			return fmt.Errorf("formatting replacement: %v", err)
		}

		ceOrig := string(b[e.ce.Pos()-1 : e.ce.End()-1])
		if _, err := src.Write([]byte(strings.Replace(stmtFmt.String(), ceFmt.String(), ceOrig, 1))); err != nil {
			return err
		}

		if _, err := src.Write([]byte(";")); err != nil {
			return err
		}
	}
	// Copy everything after the subject as-is.
	if _, err := src.Write(b[subject.End():]); err != nil {
		return err
	}

	// Format the entire source code — formatting the replacement is not
	// sufficient, as the replacement is not formatted in context.
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		return fmt.Errorf("formatting source: %v.\nsource:\n%s", err, src.String())
	}

	if _, err := w.Write(formatted); err != nil {
		return err
	}

	return nil
}

var wFlag = flag.String("w", "", "write")

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile `file`")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}
	posn := args[0]

	o := io.Writer(os.Stdout)
	if *wFlag != "" {
		f, err := os.Create(*wFlag)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		o = f
	}

	if err := logic(o, &build.Default, posn); err != nil {
		log.Fatal(err)
	}
}
