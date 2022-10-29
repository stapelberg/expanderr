package main

import (
	"bytes"
	"encoding/json"
	"flag"
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
		name        string
		fn          string
		posn        string
		errcallback string
	}{
		{"SingleErrorAfter", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#90", ""},
		{"SingleErrorBefore", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#69", ""},
		{"SingleErrorMiddle", "testdata/singleerror.got/src/singleerror/singleerror.go", ":#81", ""},
		{"NoReturn", "testdata/nocalleereturn.got/src/nocalleereturn/nocalleereturn.go", ":#75", ""},
		{"VariableAndError", "testdata/varanderror.got/src/varanderror/varanderror.go", ":#148", ""},
		{"Comment", "testdata/comment.got/src/comment/comment.go", ":#90", ""},
		{"CommentInline", "testdata/commentinline.got/src/commentinline/commentinline.go", ":#109", ""},
		{"NoReturnCaller", "testdata/noreturncaller.got/src/noreturncaller/noreturncaller.go", ":#77", ""},
		{"NoErrReturn", "testdata/noerrreturn.got/src/noerrreturn/noerrreturn.go", ":#81", ""},
		{"ReturnErrCall", "testdata/returnerrcall.got/src/returnerrcall/returnerrcall.go", ":#101", "log.Fatal(err.Error())"},
		{"FunctionLiteral", "testdata/functionliteral.got/src/functionliteral/functionliteral.go", ":#87", ""},
		// The following test spreads out one package over two files, exercising
		// the code path for loading multiple files.
		{"2Files1Pkg", "testdata/pkg.got/src/pkg/pkg2.go", ":#49", ""},
		// MultiPkg calls a function in another not-compiled, non-stdlib package.
		{"MultiPkg", "testdata/multipkg.got/src/multipkg/multipkg.go", ":#79", ""},
		{"MultiPkgVendor", "testdata/multipkgvendor.got/src/multipkg/multipkg.go", ":#79", ""},
		{"Underscore", "testdata/underscore.got/src/underscore/underscore.go", ":#162", ""},
		{"IntroduceErr", "testdata/introduceerr.got/src/introduceerr/introduceerr.go", ":#176", ""},
		{"NoIntroduce", "testdata/nointroduce.got/src/nointroduce/nointroduce.go", ":#165", ""},
		{"PresentSingle", "testdata/presentsingle.got/src/presentsingle/presentsingle.go", ":#90", ""},
		{"PresentDouble", "testdata/presentdouble.got/src/presentdouble/presentdouble.go", ":#105", ""},
		{"CustomTypes", "testdata/customtypes.got/src/customtypes/customtypes.go", ":#191", ""},
	} {
		entry := entry // copy
		t.Run(entry.name, func(t *testing.T) {
			// Cannot be safely run in parallel as long as build.Default is overridden
			// t.Parallel()

			flag.Set("format", "source")

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
			if err := logic(&buf, &buildctx, entry.fn+entry.posn, entry.errcallback); err != nil {
				t.Fatal(err)
			}

			if got, want := buf.String(), string(wantContents); got != want {
				t.Fatalf("unexpected result: have:\n%s\nwant:\n%s", got, want)
			}

			// Test the JSON output format as well.

			buf.Reset()
			flag.Set("format", "json")

			gotContents, err := ioutil.ReadFile(entry.fn)
			if err != nil {
				t.Fatal(err)
			}

			if err := logic(&buf, &buildctx, entry.fn+entry.posn, entry.errcallback); err != nil {
				t.Fatal(err)
			}

			var change struct {
				Start    int      `json:"start"`
				End      int      `json:"end"`
				Lines    []string `json:"lines"`
				Warnings []string `json:"warnings"`
			}
			if err := json.Unmarshal(buf.Bytes(), &change); err != nil {
				t.Fatal(err)
			}

			lines := strings.Split(string(gotContents), "\n")

			replaced := make([]string, len(lines[:change.Start-1]))
			copy(replaced, lines)
			replaced = append(replaced, change.Lines...)
			replaced = append(replaced, lines[change.End:]...)

			if got, want := strings.Join(replaced, "\n"), string(wantContents); got != want {
				t.Fatalf("unexpected result: have:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
