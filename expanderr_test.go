package main

import (
	"bytes"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	// Cannot be safely run in parallel as long as build.Default is overridden
	// t.Parallel()

	for _, entry := range []struct {
		name string
		fn   string
		posn string
	}{
		{"SingleErrorAfter", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#90"},
		{"SingleErrorBefore", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#69"},
		{"SingleErrorMiddle", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#81"},
		{"NoReturn", "testdata/noreturn.got/src/noreturn/noreturn.go", ":#75"},
		{"VariableAndError", "testdata/varanderror.got/src/varanderror/varanderror.go", ":#148"},
		{"Comment", "testdata/comment.got/src/comment/comment.go", ":#90"},
		{"CommentInline", "testdata/commentinline.got/src/commentinline/commentinline.go", ":#109"},
		// The following test spreads out one package over two files, exercising
		// the code path for loading multiple files.
		{"2Files1Pkg", "testdata/pkg.got/src/pkg/pkg2.go", ":#49"},
		// MultiPkg calls a function in another not-compiled, non-stdlib package.
		{"MultiPkg", "testdata/multipkg.got/src/multipkg/multipkg.go", ":#79"},
		{"MultiPkgVendor", "testdata/multipkgvendor.got/src/multipkg/multipkg.go", ":#79"},
		{"Underscore", "testdata/underscore.got/src/underscore/underscore.go", ":#162"},
		{"IntroduceErr", "testdata/introduceerr.got/src/introduceerr/introduceerr.go", ":#176"},
		{"NoIntroduce", "testdata/nointroduce.got/src/nointroduce/nointroduce.go", ":#165"},
		{"PresentSingle", "testdata/presentsingle.got/src/presentsingle/presentsingle.go", ":#90"},
		{"PresentDouble", "testdata/presentdouble.got/src/presentdouble/presentdouble.go", ":#105"},
		{"CustomTypes", "testdata/customtypes.got/src/customtypes/customtypes.go", ":#191"},
	} {
		entry := entry // copy
		t.Run(entry.name, func(t *testing.T) {
			// Cannot be safely run in parallel as long as build.Default is overridden
			// t.Parallel()

			wantContents, err := ioutil.ReadFile(strings.Replace(entry.fn, ".got", ".want", 1))
			if err != nil {
				t.Fatal(err)
			}

			gopath, err := filepath.Abs(filepath.Join(strings.Split(entry.fn, "/")[:2]...))
			if err != nil {
				t.Fatal(err)
			}
			buildctx := build.Context{
				GOARCH:   build.Default.GOARCH,
				GOOS:     build.Default.GOOS,
				GOROOT:   build.Default.GOROOT,
				GOPATH:   gopath,
				Compiler: build.Default.Compiler,
			}

			var buf bytes.Buffer
			if err := logic(&buf, &buildctx, entry.fn+entry.posn); err != nil {
				t.Fatal(err)
			}

			if got, want := buf.String(), string(wantContents); got != want {
				t.Fatalf("unexpected result: have:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
