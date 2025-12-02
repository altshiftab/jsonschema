// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package validator

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// instanceField returns the value of a field name in an instance,
// the JSON field name, and whether the field is found at all.
func instanceField(name string, instance any) (any, string, bool) {
	if instance == nil {
		return nil, "", false
	}

	v := reflect.Indirect(reflect.ValueOf(instance))

	if m, ok := v.Interface().(map[string]any); ok {
		// This is a JSON object.
		v, ok := m[name]
		return v, name, ok
	}

	if v.Kind() != reflect.Struct {
		return nil, "", false
	}
	fields := cachedTypeFields(v.Type())
	field := fields.byExactName[name]
	if field == nil {
		field = fields.byFoldedName[foldName(name)]
	}
	if field == nil {
		return nil, "", false
	}
	vf, err := v.FieldByIndexErr(field.index)
	if err != nil {
		return nil, "", false
	}
	return vf.Interface(), field.name, true
}

// instanceFieldNames returns the field names found in an instance,
// and reports whether the instance is an object.
// For efficiency this returns a structFields value,
// but for a JSON object the maps in the structFields will
// have all nil values. The only thing that matters is the
// keys of the byExactName field.
func instanceFieldNames(instance any) (structFields, bool) {
	if instance == nil {
		return structFields{}, false
	}

	v := reflect.Indirect(reflect.ValueOf(instance))

	if m, ok := v.Interface().(map[string]any); ok {
		mf := make(map[string]*field)
		for k := range m {
			mf[k] = nil
		}
		return structFields{byExactName: mf}, true
	}

	typ := v.Type()
	if typ.Kind() != reflect.Struct {
		return structFields{}, false
	}
	return cachedTypeFields(typ), true
}

// setField sets the value of a field in instance.
func setField(instance any, jsonName string, val any) error {
	v := reflect.Indirect(reflect.ValueOf(instance))

	if m, ok := v.Interface().(map[string]any); ok {
		m[jsonName] = val
		return nil
	}

	fields := cachedTypeFields(v.Type())
	field := fields.byExactName[jsonName]
	if field == nil {
		// This should be impossible, since instanceField succeeded.
		panic("could not find field in setField")
	}
	vf := v.FieldByIndex(field.index)
	return setDefault(vf, val)
}

// Much of the rest of this file is copied from the standard library's
// encoding/json package, with some modifications.

// structFields is used to map from field names that appear in the
// JSON schema to fields in a Go struct.
type structFields struct {
	byExactName  map[string]*field
	byFoldedName map[string]*field
}

// A field represents a single field found in a struct.
type field struct {
	name      string
	tag       bool
	index     []int
	typ       reflect.Type
	omitEmpty bool
}

// typeFields returns a list of fields that JSON should recognize for a type.
func typeFields(t reflect.Type) structFields {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	var count, nextCount map[reflect.Type]int

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				if sf.Anonymous {
					t := sf.Type
					if t.Kind() == reflect.Pointer {
						t = t.Elem()
					}
					if !sf.IsExported() && t.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if !sf.IsExported() {
					// Ignore unexported non-embedded fields.
					continue
				}
				tag := sf.Tag.Get("json")
				if tag == "-" {
					continue
				}
				name, opts := parseTag(tag)
				if !isValidTag(name) {
					name = ""
				}
				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Record found field and index sequence.
				if name != "" || !sf.Anonymous || ft.Kind() != reflect.Struct {
					tagged := name != ""
					if name == "" {
						name = sf.Name
					}
					field := field{
						name:      name,
						tag:       tagged,
						index:     index,
						typ:       ft,
						omitEmpty: opts.Contains("omitempty"),
					}

					fields = append(fields, field)
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 and 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	slices.SortFunc(fields, func(a, b field) int {
		// sort field by name, breaking ties with depth, then
		// breaking ties with "name came from json tag", then
		// breaking ties with index sequence.
		if c := strings.Compare(a.name, b.name); c != 0 {
			return c
		}
		if c := cmp.Compare(len(a.index), len(b.index)); c != 0 {
			return c
		}
		if a.tag != b.tag {
			if a.tag {
				return -1
			}
			return +1
		}
		return slices.Compare(a.index, b.index)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with JSON tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		// One iteration per name.
		// Find the sequence of fields with the name of this first field.
		fi := fields[i]
		name := fi.name
		for advance = 1; i+advance < len(fields); advance++ {
			fj := fields[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, fi)
			continue
		}
		dominant, ok := dominantField(fields[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
	}

	fields = out
	slices.SortFunc(fields, func(i, j field) int {
		return slices.Compare(i.index, j.index)
	})

	exactNameIndex := make(map[string]*field, len(fields))
	foldedNameIndex := make(map[string]*field, len(fields))
	for i, field := range fields {
		exactNameIndex[field.name] = &fields[i]
		// For historical reasons, first folded match takes precedence.
		if _, ok := foldedNameIndex[foldName(field.name)]; !ok {
			foldedNameIndex[foldName(field.name)] = &fields[i]
		}
	}
	return structFields{exactNameIndex, foldedNameIndex}
}

// dominantField looks through the fields, all of which are known to
// have the same name, to find the single field that dominates the
// others using Go's embedding rules, modified by the presence of
// JSON tags. If there are multiple top-level fields, the boolean
// will be false: This condition is an error in Go and we skip all
// the fields.
func dominantField(fields []field) (field, bool) {
	// The fields are sorted in increasing index-length order, then by presence of tag.
	// That means that the first field is the dominant one. We need only check
	// for error cases: two fields at top level, either both tagged or neither tagged.
	if len(fields) > 1 && len(fields[0].index) == len(fields[1].index) && fields[0].tag == fields[1].tag {
		return field{}, false
	}
	return fields[0], true
}

var fieldCache sync.Map // map[reflect.Type]structFields

// cachedTypeFields is like typeFields but uses a cache to avoid repeated work.
func cachedTypeFields(t reflect.Type) structFields {
	if f, ok := fieldCache.Load(t); ok {
		return f.(structFields)
	}
	f, _ := fieldCache.LoadOrStore(t, typeFields(t))
	return f.(structFields)
}

// foldName returns a folded string such that foldName(x) == foldName(y)
// is identical to bytes.EqualFold(x, y).
func foldName(in string) string {
	var sb strings.Builder
	for i := 0; i < len(in); {
		// Handle single-byte ASCII.
		if c := in[i]; c < utf8.RuneSelf {
			if 'a' <= c && c <= 'z' {
				c -= 'a' - 'A'
			}
			sb.WriteByte(c)
			i++
			continue
		}
		// Handle multi-byte Unicode.
		r, n := utf8.DecodeRuneInString(in[i:])
		sb.WriteRune(foldRune(r))
		i += n
	}
	return sb.String()
}

// foldRune is returns the smallest rune for all runes in the same fold set.
func foldRune(r rune) rune {
	for {
		r2 := unicode.SimpleFold(r)
		if r2 <= r {
			return r2
		}
		r = r2
	}
}

// tagOptions is the string following a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	tag, opt, _ := strings.Cut(tag, ",")
	return tag, tagOptions(opt)
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	for s != "" {
		var name string
		name, s, _ = strings.Cut(s, ",")
		if name == optionName {
			return true
		}
	}
	return false
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}
