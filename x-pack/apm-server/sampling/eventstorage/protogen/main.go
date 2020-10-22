package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/types"
	"log"

	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

type genContext struct {
	types map[*types.TypeName]bool
}

func newGenContext() *genContext {
	return &genContext{
		types: make(map[*types.TypeName]bool),
	}
}

func genPackage(ctx *genContext, pkg *types.Package, w *bufio.Writer) error {
	protoPackage := pkg.Name()
	fmt.Fprintf(w, `
syntax = "proto3";
import "google/protobuf/any.proto";
import "google/protobuf/timestamp.proto";
package %s;
option go_package = "%s";
`, protoPackage, pkg.Path())

	scope := pkg.Scope()
	for _, name := range []string{"Transaction", "Span"} {
		typename, ok := scope.Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		ctx.types[typename] = false
	}

	more := true
	for more {
		more = false
		for typename, done := range ctx.types {
			if done {
				continue
			}
			fmt.Fprintf(w, "message %s ", typename.Name())
			if err := genStruct(ctx, typename.Type().Underlying().(*types.Struct), w); err != nil {
				return err
			}
			fmt.Fprintln(w)
			ctx.types[typename] = true
			more = true
		}
	}
	return nil
}

func genType(ctx *genContext, typ types.Type, w *bufio.Writer) error {
	switch typ := typ.(type) {
	case *types.Basic:
		return genBasic(typ, w)
	case *types.Named:
		typename := typ.Obj()
		if typename.Pkg().Path() == "time" && typename.Name() == "Time" {
			w.WriteString("google.protobuf.Timestamp")
			return nil
		}
		switch underlying := typename.Type().Underlying().(type) {
		case *types.Struct:
			if _, ok := ctx.types[typename]; !ok {
				// Record that the type is referenced, so we know to
				// create a message for it.
				ctx.types[typename] = false
			}
			w.WriteString(typename.Name())
			return nil
		case *types.Map:
			if _, ok := ctx.types[typename]; !ok {
				// Record that the type is referenced, so we know to
				// create a message for it.
				ctx.types[typename] = false
			}
			fmt.Fprintf(w, "repeated %s", typename.Name())
			return nil
		default:
			// Other named types are inlined.
			return genType(ctx, underlying, w)
		}
	case *types.Map:
		w.WriteString("map<")
		if err := genType(ctx, typ.Key(), w); err != nil {
			return err
		}
		w.WriteString(", ")
		if err := genType(ctx, typ.Elem(), w); err != nil {
			return err
		}
		w.WriteRune('>')
		return nil
	case *types.Pointer:
		return genType(ctx, typ.Elem(), w)
	case *types.Interface:
		if typ.Empty() {
			w.WriteString("google.protobuf.Any")
			return nil
		}
	case *types.Slice:
		elem := typ.Elem()
		if elem, ok := elem.(*types.Basic); ok && elem.Kind() == types.Byte {
			w.WriteString("bytes")
			return nil
		}
		w.WriteString("repeated ")
		return genType(ctx, elem, w)
	}
	return fmt.Errorf("unhandled type: %#v", typ)
}

func genBasic(typ *types.Basic, w *bufio.Writer) error {
	switch typ.Kind() {
	case types.Bool:
		w.WriteString("bool")
	case types.Float64:
		w.WriteString("double")
	case types.Int, types.Int64:
		w.WriteString("int64")
	case types.Uint, types.Uint64:
		w.WriteString("uint64")
	case types.String:
		w.WriteString("string")
	default:
		return fmt.Errorf("unhandled basic type: %#v", typ)
	}
	return nil
}

func genStruct(ctx *genContext, typ *types.Struct, w *bufio.Writer) error {
	fmt.Fprintf(w, "{\n")
	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		fmt.Fprintf(w, "\t")
		if err := genType(ctx, field.Type(), w); err != nil {
			return err
		}
		fmt.Fprintf(w, " %s = %d;\n", field.Name(), i+1)
	}
	fmt.Fprintf(w, "}")
	return nil
}

func Main() error {
	pkgpath := "github.com/elastic/apm-server/model"
	log.Printf("loading packages: %s", pkgpath)
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedImports}
	pkgs, err := packages.Load(cfg, pkgpath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	ctx := newGenContext()
	for _, pkg := range pkgs {
		fmt.Println("-", pkg)
		if err := genPackage(ctx, pkg.Types, w); err != nil {
			return errors.Wrapf(err, "failed to generate proto for package %s", pkg)
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Println(buf.String())
	fmt.Println("---")
	return nil
}

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
}
