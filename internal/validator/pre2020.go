// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package validator

import (
	"reflect"

	"github.com/altshiftab/jsonschema/pkg/notes"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// itemsNote is the type of the note recorded for items.
// We need to track both the length of the array and the schema,
// as items only affects additionalItems in the same schema.
type itemsNote struct {
	all    bool
	idx    int
	schema *types.Schema
}

// ValidatePre2020Items validates an items keyword.
// This differs from the one in validateItems because
// that one is the draft2020-12 one that takes just a single schema,
// but this is the earlier one that takes either a single schema
// or an array of schemas.
func ValidatePre2020Items(arg types.PartSchemaOrSchemas, instance any, state *types.ValidationState) error {
	note := itemsNote{
		all:    false,
		idx:    0,
		schema: state.Schema,
	}

	if arg.Schema != nil {
		if a, ok := instance.([]any); ok {
			// Skip reflection in the common case of a JSON array.
			for _, v := range a {
				if err := arg.Schema.ValidateSubSchema(v, state); err != nil {
					return err
				}
			}
		} else {
			v := reflect.ValueOf(instance)
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil
			}

			for idx := range v.Len() {
				e := v.Index(idx).Interface()
				if err := arg.Schema.ValidateSubSchema(e, state); err != nil {
					return err
				}
			}
		}

		note.all = true
	} else {
		applyDefaults := state.Opts != nil && state.Opts.ApplyDefaults

		if a, ok := instance.([]any); ok {
			// Skip reflection in the common case of a JSON array.
			for i, s := range arg.Schemas {
				if i >= len(a) {
					note.all = true
					break
				}

				val := a[i]
				if applyDefaults && reflect.ValueOf(val).IsZero() {
					pv, hasDefault := s.LookupKeyword("default")
					if hasDefault {
						val = pv.(types.PartAny).V
						a[i] = val
					}
				}

				if err := s.ValidateSubSchema(val, state); err != nil {
					return err
				}
			}
		} else {
			v := reflect.ValueOf(instance)
			if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
				return nil
			}

			ln := v.Len()

			for i, s := range arg.Schemas {
				if i >= ln {
					note.all = true
					break
				}

				indexVal := v.Index(i)
				val := indexVal.Interface()
				if applyDefaults && indexVal.IsZero() {
					pv, hasDefault := s.LookupKeyword("default")
					if hasDefault {
						defVal := pv.(types.PartAny).V
						if err := setDefault(indexVal, defVal); err != nil {
							return err
						}
						val = defVal
					}
				}

				if err := s.ValidateSubSchema(val, state); err != nil {
					return err
				}
			}
		}

		if !note.all {
			note.idx = len(arg.Schemas)
		}
	}

	notes.AppendNote(&state.Notes, "items", note)

	return nil
}

// ValidatePre2020AdditionalItems validates an additionalItems keyword.
func ValidatePre2020AdditionalItems(arg types.PartSchema, instance any, state *types.ValidationState) error {
	found := false
	idx := 0
	if notes, ok := state.Notes.Get("items"); ok {
		for _, note := range notes.([]itemsNote) {
			if note.schema == state.Schema {
				if note.all {
					return nil
				}
				idx = max(idx, note.idx)
				found = true
			}
		}
	}
	if !found {
		return nil
	}

	isArray, err := validateArrayFromIdx(arg.S, instance, state, idx)
	if err != nil {
		return err
	}
	if !isArray {
		return nil
	}

	n := itemsNote{
		schema: state.Schema,
		all:    true,
		idx:    0,
	}
	notes.AppendNote(&state.Notes, "items", n)

	return nil
}

// validateArrayFromIdx validates elements of an array starting at idx.
// It reports whether the instance is an array.
func validateArrayFromIdx(s *types.Schema, instance any, state *types.ValidationState, idx int) (bool, error) {
	if a, ok := instance.([]any); ok {
		// Skip reflection in the common case of a JSON array.
		for ; idx < len(a); idx++ {
			if err := s.ValidateSubSchema(a[idx], state); err != nil {
				return false, err
			}
		}
	} else {
		v := reflect.ValueOf(instance)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return false, nil
		}

		ln := v.Len()
		for ; idx < ln; idx++ {
			e := v.Index(idx).Interface()
			if err := s.ValidateSubSchema(e, state); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// ValidatePre2020UnevaluatedItems validates an unevaluatedItems keyword.
func ValidatePre2020UnevaluatedItems(arg types.PartSchema, instance any, state *types.ValidationState) error {
	idx := 0
	if notes, ok := state.Notes.Get("items"); ok {
		for _, note := range notes.([]itemsNote) {
			if note.all {
				return nil
			}
			idx = max(idx, note.idx)
		}
	}

	if _, err := validateArrayFromIdx(arg.S, instance, state, idx); err != nil {
		return err
	}

	n := itemsNote{
		schema: state.Schema,
		all:    true,
		idx:    0,
	}
	notes.AppendNote(&state.Notes, "items", n)

	return nil
}
