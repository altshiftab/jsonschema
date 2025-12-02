// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package argtype defines a few helpers for schema.ArgType.
// This is only used by the generator commands, not by user code.
package argtype

import (
	"fmt"

	"github.com/altshiftab/jsonschema/pkg/types"
)

// nameToString maps [types.ArgType] to a name used in generated
// function and type names.
var nameToString = map[types.ArgType]string{
	types.ArgTypeBool:             "Bool",
	types.ArgTypeString:           "String",
	types.ArgTypeStrings:          "Strings",
	types.ArgTypeStringOrStrings:  "StringOrStrings",
	types.ArgTypeInt:              "Int",
	types.ArgTypeFloat:            "Float",
	types.ArgTypeSchema:           "Schema",
	types.ArgTypeSchemas:          "Schemas",
	types.ArgTypeMapSchema:        "MapSchema",
	types.ArgTypeSchemaOrSchemas:  "SchemaOrSchemas",
	types.ArgTypeMapArrayOrSchema: "MapArrayOrSchema",
	types.ArgTypeAny:              "Any",
}

// Name returns a name to use for a [types.ArgType] in
// generated function and type names.
func Name(sat types.ArgType) string {
	if n, ok := nameToString[sat]; ok {
		return n
	}
	panic(fmt.Sprintf("unexpected ArgType value %d", sat))
}

// nameToGoType maps [types.ArgType] to the underlying Go type.
var nameToGoType = map[types.ArgType]string{
	types.ArgTypeBool:             "bool",
	types.ArgTypeString:           "string",
	types.ArgTypeStrings:          "[]string",
	types.ArgTypeStringOrStrings:  "types.PartStringOrStrings",
	types.ArgTypeInt:              "int64",
	types.ArgTypeFloat:            "float64",
	types.ArgTypeSchema:           "*types.Schema",
	types.ArgTypeSchemas:          "[]*types.Schema",
	types.ArgTypeMapSchema:        "map[string]*types.Schema",
	types.ArgTypeSchemaOrSchemas:  "types.PartSchemaOrSchemas",
	types.ArgTypeMapArrayOrSchema: "map[string]types.ArrayOrSchema",
	types.ArgTypeAny:              "any",
}

// GoType returns the Go type of a [types.ArgType], as a string.
func GoType(sat types.ArgType) string {
	if t, ok := nameToGoType[sat]; ok {
		return t
	}
	panic(fmt.Sprintf("unexpected Argtype %d", sat))
}
