// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/altshiftab/jsonschema/internal/argtype"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// writeBuilderHeader writes the start of the Builder section.
func writeBuilderHeader(builderBuf *bytes.Buffer) {
	for t := types.ArgTypeBool; t <= types.ArgTypeAny; t++ {
		// We don't add a method for StringOrStrings;
		// instead we use AddString and AddStrings.
		if t == types.ArgTypeStringOrStrings {
			continue
		}

		n := argtype.Name(t)
		fmt.Fprintln(builderBuf)
		fmt.Fprintf(builderBuf, "// Add%s adds a keyword with an argument of type %s.\n", n, n)
		fmt.Fprintf(builderBuf, "func (b *Builder) Add%s(keyword *types.Keyword, v %s) *Builder {\n", n, argtype.GoType(t))
		fmt.Fprintf(builderBuf, "\tb.b = b.b.Add%s(keyword, v)\n", n)
		fmt.Fprintf(builderBuf, "\treturn b\n")
		fmt.Fprintln(builderBuf, "}")
	}
}

// printKeywordsBuilder writes out the builder methods
// for the keywords.
func printKeywordsBuilder(builderBuf *bytes.Buffer, keywords *keywordsData) {
	first := true
	for _, k := range keywords.Keywords {
		if k.SkipBuilder {
			if k.BuilderComment != "" {
				fmt.Fprintf(os.Stderr, "%s: has both skipBuilder and builderComment", k.Name)
				os.Exit(1)
			}
			continue
		}
		if first {
			first = false
		} else {
			fmt.Fprintln(builderBuf)
		}
		name := "Add" + oneup(k.Name[keywords.Prefix:])
		if k.ArgType != "stringOrStrings" {
			typ, ok := goArgType[k.ArgType]
			if !ok {
				fmt.Fprintf(os.Stderr, "%s: unrecognized keyword type %q\n", k.Name, k.ArgType)
				os.Exit(2)
			}
			fmt.Fprintf(builderBuf, "// %s adds the %s keyword to the schema.\n", name, k.Name)
			if k.BuilderComment != "" {
				fmt.Fprintf(builderBuf, "// %s", k.BuilderComment)
			}
			fmt.Fprintf(builderBuf, "func (b *Builder) %s(arg %s) *Builder {\n", name, typ)
			fmt.Fprintf(builderBuf, "\treturn b.Add%s(&%sKeyword, arg)\n", oneup(k.ArgType), k.Name[keywords.Prefix:])
			fmt.Fprintln(builderBuf, "}")
		} else {
			fmt.Fprintf(builderBuf, "// %s adds the %s keyword with one or more strings to the schema.\n", name, k.Name)
			if k.BuilderComment != "" {
				fmt.Fprintf(builderBuf, "// %s", k.BuilderComment)
			}
			fmt.Fprintf(builderBuf, "func (b *Builder) %s(args ...string) *Builder {\n", name)
			fmt.Fprintf(builderBuf, "\tif len(args) == 1 {\n")
			fmt.Fprintf(builderBuf, "\t\treturn b.AddString(&%sKeyword, args[0])\n", k.Name[keywords.Prefix:])
			fmt.Fprintf(builderBuf, "\t} else {\n")
			fmt.Fprintf(builderBuf, "\t\treturn b.AddStrings(&%sKeyword, args)\n", k.Name[keywords.Prefix:])
			fmt.Fprintf(builderBuf, "\t}\n")
			fmt.Fprintln(builderBuf, "}")
		}
	}
}

// goArgType is the Go type to use for a SchemaArgType value
// as it appears in an input JSON file.
var goArgType = map[string]string{
	"bool":    "bool",
	"string":  "string",
	"strings": "[]string",
	// no entry for "stringOrStrings"; uses string and strings instead
	"int":              "int64",
	"float":            "float64",
	"schema":           "*types.Schema",
	"schemas":          "[]*types.Schema",
	"mapSchema":        "map[string]*types.Schema",
	"schemaOrSchemas":  "types.PartSchemaOrSchemas",
	"mapArrayOrSchema": "map[string]types.ArrayOrSchema",
	"any":              "any",
}
