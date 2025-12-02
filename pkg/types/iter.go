// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"iter"
	"slices"
	"strings"
)

// Children returns an iterator over the immediate subschemas.
// The first iterator value is the name of the schema as used in a JSON pointer,
// the second is the schema itself.
func (s *Schema) Children() iter.Seq2[string, *Schema] {
	return func(yield func(string, *Schema) bool) {
		for _, part := range s.Parts {
			if part.Keyword.Generated {
				continue
			}

			switch part.Keyword.ArgType {
			case ArgTypeSchema:
				if !yield(part.Keyword.Name, part.Value.(PartSchema).S) {
					return
				}

			case ArgTypeSchemas:
				for i, sub := range part.Value.(PartSchemas) {
					name := fmt.Sprintf("%s/%d", part.Keyword.Name, i)
					if !yield(name, sub) {
						return
					}
				}

			case ArgTypeMapSchema:
				// Sort for determinism.
				type keyVal struct {
					key string
					val *Schema
				}
				m := part.Value.(PartMapSchema)
				keyVals := make([]keyVal, 0, len(m))
				for k, v := range m {
					keyVals = append(keyVals, keyVal{k, v})
				}
				slices.SortFunc(keyVals, func(a, b keyVal) int {
					return strings.Compare(a.key, b.key)
				})
				for _, kv := range keyVals {
					name := part.Keyword.Name + "/" + kv.key
					if !yield(name, kv.val) {
						return
					}
				}

			case ArgTypeSchemaOrSchemas:
				pv := part.Value.(PartSchemaOrSchemas)
				if pv.Schema != nil {
					if !yield(part.Keyword.Name, pv.Schema) {
						return
					}
				} else {
					for i, sub := range pv.Schemas {
						name := fmt.Sprintf("%s/%d", part.Keyword.Name, i)
						if !yield(name, sub) {
							return
						}
					}
				}

			case ArgTypeMapArrayOrSchema:
				// Sort for determinism.
				type keyVal struct {
					key string
					val *Schema
				}
				m := part.Value.(PartMapArrayOrSchema)
				keyVals := make([]keyVal, 0, len(m))
				for k, v := range m {
					if v.Schema != nil {
						keyVals = append(keyVals, keyVal{k, v.Schema})
					}
				}
				slices.SortFunc(keyVals, func(a, b keyVal) int {
					return strings.Compare(a.key, b.key)
				})
				for _, kv := range keyVals {
					name := part.Keyword.Name + "/" + kv.key
					if !yield(name, kv.val) {
						return
					}
				}
			}
		}
	}
}
