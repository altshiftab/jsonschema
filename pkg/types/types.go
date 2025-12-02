// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types defines the JSON schema types.
// Most programs do not need to use this package.
//
// This package is used with a specific set of JSON schema drafts,
// that must be imported separately.
// For example, to use the current draft, a program should also
//
//	import _ "github.com/ianlancetaylor/jsonschema/pkg/draft20212"
package types

import (
	"fmt"
	"slices"
	"strings"
)

// Schema is a JSON schema.
// A JSON schema determines whether an instance is valid or not.
// Do not create values of this type directly.
// Instead, unmarshal from JSON or use a draft-specific Builder.
//
// If you have an existing Schema, you can edit the Parts list,
// but you must call [Schema.Finalize] afterward.
// When adding a new Part it will help to use [Vocabulary.Keywords];
// each supported JSON schema draft has a Vocabulary package variable.
// You can't add keywords that refer to other parts of the schema by name,
// such as $ref.
type Schema struct {
	// The different elements of this Schema.
	Parts []Part
}

// Clone returns a copy of a Schema.
func (s *Schema) Clone() *Schema {
	return &Schema{Parts: slices.Clone(s.Parts)}
}

// String returns a somewhat readable representation of a Schema.
// The format differs from JSON output, and also includes internal
// information not stored in JSON.
func (s *Schema) String() string {
	var sb strings.Builder
	sb.WriteString("Schema{")
	for i, part := range s.Parts {
		if i > 0 {
			sb.WriteString(", ")
		}
		val := part.Value
		if part.Keyword.Generated {
			// Don't try to print schemas of generated keywords.
			// They can cause infinite recursion.
			switch part.Keyword.ArgType {
			case ArgTypeBool, ArgTypeString, ArgTypeStrings, ArgTypeInt, ArgTypeFloat:
			default:
				val = PartString("<not printed>")
			}
		}
		fmt.Fprintf(&sb, "{%s %v}", part.Keyword.Name, val)
	}
	sb.WriteByte('}')
	return sb.String()
}

// Part is one part of a JSON schema.
// This is a keyword, such as "$id" or "properties",
// along with the value associated with that keyword in the schema.
type Part struct {
	Keyword *Keyword
	Value   PartValue
}

// MakePart builds a Part.
func MakePart(keyword *Keyword, value PartValue) Part {
	return Part{
		Keyword: keyword,
		Value:   value,
	}
}

// Keyword is a schema keyword.
type Keyword struct {
	// Name is the keyword, such as allOf, anyOf, and so forth.
	Name string

	// ArgType is the type of argument expected.
	ArgType ArgType

	// Validate is a function that checks whether the schema matches
	// the keyword. arg is the value from the schema, which is [Part.Value].
	// instance is the object to validate.
	//
	// The function returns an error if any.
	// A failure to validate will be type [*ValidationError]
	// or type [*ValidationErrors].
	// Any other error type indicates a problem with the schema itself,
	// not the instance.
	Validate func(arg PartValue, instance any, state *ValidationState) error

	// Generated is true if this keyword is not represented in JSON,
	// but is added to record additional information.
	// If this is true the keyword should be ignored by anything
	// that wants to treat the Schema as a JSON object.
	Generated bool
}

// Equal reports whether two keywords are equal.
// This is for the benefit of the github.com/google/go-cmp package,
// which won't compare the Validate function values.
func (k1 Keyword) Equal(k2 Keyword) bool {
	return k1.Name == k2.Name && k1.ArgType == k2.ArgType && k1.Generated == k2.Generated
}

// PartValue is the value of a JSON schema element.
// This is accessed via a type switch.
// The possible types are
//   - [PartBool]
//   - [PartString]
//   - [PartStrings]
//   - [PartStringOrStrings]
//   - [PartInt]
//   - [PartFloat]
//   - [PartSchema]
//   - [PartSchemas]
//   - [PartMapSchema]
//   - [PartSchemaOrSchemas]
//   - [PartMapArrayOrSchema]
//   - [PartAny]
type PartValue interface {
	partValue() // restrict to types defined in this package
}

// PartBool is a schema part value that is a bool.
// This is a compact representation of a JSON schema.
// A value of true is the schema that matches every value.
// A value of false is the schema that matches no values.
type PartBool bool

// PartString is a schema part value that is a string.
// For example, the schema keyword "pattern" has a string
// value that must be a regexp that must match the instance value.
type PartString string

// PartStrings is a schema part value that is a list of strings.
// For example, the schema keyword "required" takes a list of strings
// where each string is a property that the instance is required to have.
type PartStrings []string

// PartStringOrStrings is a schema part that is either a single string
// or a list of strings. This is basically just for the "type" keyword,
// which takes either a single type string or an array of type strings.
// If the Strings is not nil, the String field must be the empty string.
type PartStringOrStrings struct {
	String  string
	Strings []string
}

// PartInt is a schema part value that is an integer.
// For example, the schema keyword "minLength" specifies
// the minimum length of a string.
type PartInt int64

// PartFloat is a schema part value that is a floating-point number.
// For example, the schema keyword "maximum" specifies the maximum
// value of a number.
type PartFloat float64

// PartSchema is a schema part value that is a reference to a schema.
// For example, the schema keyword "not" refers to a schema;
// the instance matches if it does not match that schema.
type PartSchema struct {
	S *Schema
}

// PartSchemas is a schema part value that is a list of schemas.
// For example, the schema keyword "allOf" matches an instance
// if the instance matches each schema in the list.
type PartSchemas []*Schema

// PartMapSchema is a schema part value that is a map from strings to schemas.
// For example, the schema keyword "properties" has a mapping
// from field names to schemas, and matches an instance if the
// corresponding instance fields match the schemas.
type PartMapSchema map[string]*Schema

// PartSchemaOrSchemas is either a single schema (like [PartSchema])
// or a list of schemas (like [PartSchemas]). For example,
// the draft201909 keyword "items" takes either a single schema
// or a list of schemas. Exactly one of the fields will be nil.
type PartSchemaOrSchemas struct {
	Schema  *Schema
	Schemas []*Schema
}

// PartMapArrayOrSchema is a map from strings to elements,
// where each element is either an array of strings or a schema.
// This is used for the draft7 "dependencies" keyword.
type PartMapArrayOrSchema map[string]ArrayOrSchema

// ArrayOrSchema is the element type of the PartMapArrayOrSchema map.
// Exactly one of the fields will be nil.
type ArrayOrSchema struct {
	Array  []string // a zero-length slice is []string{}, not nil
	Schema *Schema
}

// PartAny is a schema part value that is an arbitrary type.
// For example, the schema keyword "$vocabulary" expects an
// object where each property is a URI.
// For example, the schema keyword "enum" expects an array,
// and matches an instance if the instance is equal to one of the
// elements in the array.
type PartAny struct {
	V any
}

// Define a schemaPartValue method for each permitted Part type.
// This implements the [PartValue] interface.

func (PartBool) partValue()             {}
func (PartString) partValue()           {}
func (PartStrings) partValue()          {}
func (PartStringOrStrings) partValue()  {}
func (PartInt) partValue()              {}
func (PartFloat) partValue()            {}
func (PartSchema) partValue()           {}
func (PartSchemas) partValue()          {}
func (PartMapSchema) partValue()        {}
func (PartSchemaOrSchemas) partValue()  {}
func (PartMapArrayOrSchema) partValue() {}
func (PartAny) partValue()              {}

// ArgType is an enumeration of the possible schema part types.
type ArgType int

const (
	ArgTypeBool ArgType = iota + 1
	ArgTypeString
	ArgTypeStrings
	ArgTypeStringOrStrings
	ArgTypeInt
	ArgTypeFloat
	ArgTypeSchema
	ArgTypeSchemas
	ArgTypeMapSchema
	ArgTypeSchemaOrSchemas
	ArgTypeMapArrayOrSchema
	ArgTypeAny
)

// LookupKeyword returns the value associated with a keyword in the schema.
// The bool result reports whether the keyword is present at all.
func (s *Schema) LookupKeyword(keyword string) (PartValue, bool) {
	for _, part := range s.Parts {
		if !part.Keyword.Generated && part.Keyword.Name == keyword {
			return part.Value, true
		}
	}
	return nil, false
}
