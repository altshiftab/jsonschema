// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jsonpointer implements JSON pointers for the jsonschema package.
// This is not a fully general package.
package jsonpointer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/altshiftab/jsonschema/internal/argtype"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// DerefSchema takes a JSON pointer and a root schema and returns
// the schema to which the pointer refers.
// The schemaID parameter is the default schema ID.
func DerefSchema(schemaID string, root *types.Schema, pointer string) (*types.Schema, error) {
	s := root
	pointer = strings.TrimPrefix(pointer, "/")
	toks := strings.Split(pointer, "/")
	for i := 0; i < len(toks); i++ {
		tok := decodeToken(toks[i])
		for _, part := range s.Parts {
			if part.Keyword.Generated {
				continue
			}
			if part.Keyword.Name != tok {
				continue
			}

			switch part.Keyword.ArgType {
			case types.ArgTypeSchema:
				s = part.Value.(types.PartSchema).S

			case types.ArgTypeSchemas:
				i++
				if i >= len(toks) {
					return nil, fmt.Errorf("when dereferencing pointer %q expected array index after %q", pointer, tok)
				}
				tok = decodeToken(toks[i])
				idx, err := strconv.Atoi(tok)
				if err != nil {
					return nil, fmt.Errorf("when dereferencing pointer %q got token %q, expected array index", pointer, tok)
				}
				schemas := part.Value.(types.PartSchemas)
				if idx < 0 || idx >= len(schemas) {
					return nil, fmt.Errorf("when dereferencing pointer %q array index %d out of range (length %d)", pointer, idx, len(schemas))
				}
				s = schemas[idx]

			case types.ArgTypeMapSchema:
				i++
				if i >= len(toks) {
					return nil, fmt.Errorf("when dereferencing pointer %q expected map key after %q", pointer, tok)
				}
				tok = decodeToken(toks[i])
				m := part.Value.(types.PartMapSchema)
				ms, ok := m[tok]
				if !ok {
					return nil, fmt.Errorf("when dereferencing pointer %q map key %q not present", pointer, tok)
				}
				s = ms

			case types.ArgTypeSchemaOrSchemas:
				pv := part.Value.(types.PartSchemaOrSchemas)
				if pv.Schema != nil {
					s = pv.Schema
				} else {
					i++
					if i >= len(toks) {
						return nil, fmt.Errorf("when dereferencing pointer %q expected array index after %q", pointer, tok)
					}
					tok = decodeToken(toks[i])
					idx, err := strconv.Atoi(tok)
					if err != nil {
						return nil, fmt.Errorf("when dereferencing pointer %q got token %q, expected array index", pointer, tok)
					}
					if idx < 0 || idx >= len(pv.Schemas) {
						return nil, fmt.Errorf("when dereferencing pointer %q array index %d out of range (length %d)", pointer, idx, len(pv.Schemas))
					}
					s = pv.Schemas[idx]
				}

			case types.ArgTypeMapArrayOrSchema:
				i++
				if i >= len(toks) {
					return nil, fmt.Errorf("when dereferencing pointer %q expected map key after %q", pointer, tok)
				}
				tok = decodeToken(toks[i])
				m := part.Value.(types.PartMapArrayOrSchema)
				mv, ok := m[tok]
				if !ok {
					return nil, fmt.Errorf("when dereferencing pointer %q map key %q not present", pointer, tok)
				}
				if mv.Schema == nil {
					return nil, fmt.Errorf("when dereferencing pointer %q map key %q is not a schema", pointer, tok)
				}
				s = mv.Schema

			case types.ArgTypeAny:
				pv := part.Value.(types.PartAny).V
			resolveLoop:
				for {
					switch v := pv.(type) {
					case bool, map[string]any:
						var err error
						s, err = types.SchemaFromJSON(schemaID, nil, v)
						if err != nil {
							return nil, fmt.Errorf("when dereferencing pointer %q failed to resolve unrecognized schema: %v", pointer, err)
						}
						break resolveLoop

					case []any:
						i++
						if i >= len(toks) {
							return nil, fmt.Errorf("when dereferencing pointer %q expected array index after %q", pointer, tok)
						}
						tok = decodeToken(toks[i])
						idx, err := strconv.Atoi(tok)
						if err != nil {
							return nil, fmt.Errorf("when dereferencing pointer %q for token %q, expected array index", pointer, tok)
						}
						if idx < 0 || idx >= len(v) {
							return nil, fmt.Errorf("when dereferencing pointer %q array index %d out of range (length %d)", pointer, idx, len(v))
						}
						pv = v[idx]

					default:
						return nil, fmt.Errorf("when dereferencing pointer %q unexpected type %T", pointer, v)
					}
				}

			default:
				return nil, fmt.Errorf("when dereferencing pointer %q unexpected part type %s", pointer, argtype.Name(part.Keyword.ArgType))
			}

			break
		}
	}

	return s, nil
}

// decodeToken unmangles a token in a JSON pointer.
func decodeToken(tok string) string {
	tok = strings.ReplaceAll(tok, "~1", "/")
	return strings.ReplaceAll(tok, "~0", "~")
}
