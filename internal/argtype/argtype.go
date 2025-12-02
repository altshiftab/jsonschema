// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package argtype defines a few helpers for schema.ArgType.
// This is only used by the generator commands, not by user code.
package argtype

import (
	"fmt"

	"github.com/altshiftab/jsonschema/pkg/types/arg_type"
)

// nameToString maps [types.ArgType] to a name used in generated
// function and type names.
var nameToString = map[arg_type.ArgType]string{
	arg_type.ArgTypeBool:             "Bool",
	arg_type.ArgTypeString:           "String",
	arg_type.ArgTypeStrings:          "Strings",
	arg_type.ArgTypeStringOrStrings:  "StringOrStrings",
	arg_type.ArgTypeInt:              "Int",
	arg_type.ArgTypeFloat:            "Float",
	arg_type.ArgTypeSchema:           "Schema",
	arg_type.ArgTypeSchemas:          "Schemas",
	arg_type.ArgTypeMapSchema:        "MapSchema",
	arg_type.ArgTypeSchemaOrSchemas:  "SchemaOrSchemas",
	arg_type.ArgTypeMapArrayOrSchema: "MapArrayOrSchema",
	arg_type.ArgTypeAny:              "Any",
}

// Name returns a name to use for a [types.ArgType] in
// generated function and type names.
func Name(sat arg_type.ArgType) string {
	if n, ok := nameToString[sat]; ok {
		return n
	}
	panic(fmt.Sprintf("unexpected ArgType value %d", sat))
}

// nameToGoType maps [types.ArgType] to the underlying Go type.
var nameToGoType = map[arg_type.ArgType]string{
	arg_type.ArgTypeBool:             "bool",
	arg_type.ArgTypeString:           "string",
	arg_type.ArgTypeStrings:          "[]string",
	arg_type.ArgTypeStringOrStrings:  "types.PartStringOrStrings",
	arg_type.ArgTypeInt:              "int64",
	arg_type.ArgTypeFloat:            "float64",
	arg_type.ArgTypeSchema:           "*types.Schema",
	arg_type.ArgTypeSchemas:          "[]*types.Schema",
	arg_type.ArgTypeMapSchema:        "map[string]*types.Schema",
	arg_type.ArgTypeSchemaOrSchemas:  "types.PartSchemaOrSchemas",
	arg_type.ArgTypeMapArrayOrSchema: "map[string]types.ArrayOrSchema",
	arg_type.ArgTypeAny:              "any",
}

// GoType returns the Go type of a [types.ArgType], as a string.
func GoType(sat arg_type.ArgType) string {
	if t, ok := nameToGoType[sat]; ok {
		return t
	}
	panic(fmt.Sprintf("unexpected Argtype %d", sat))
}
