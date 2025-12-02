// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package notes defines a type that holds information passed
// between keywords during schema validation.
// This permits validation of one keyword to depend on information
// gathered while validating another keyword.
//
// For example, this supports keywords like "unevaluatedItems",
// that impose requirements only on fields that were not already
// validated by other keywords like "items" or "contains".
//
// This package is visible to support people who want to define their
// JSON schema definitions. People who are only interested in
// validating JSON objects using standard JSON schemas do not need
// to use this package.
package notes

import (
	"fmt"
	"reflect"
)

// Notes is a set of notes. Each note has a name and a value.
// The name should normally be the name of a JSON schema keyword.
// The value may be anything; it is normally a bool, an int,
// or a slice of some elements.
//
// The zero value of Notes is directly usable.
// Notes may not be used concurrently by multiple goroutines.
//
// Notes are not annotations as defined by the JSON schema definition.
// Currently this module does not support JSON schema annotations.
// Notes could evolve into annotations if that seems desirable;
// we would need a way to save them and make them available after validation,
// and we would need to record the instance location and the schema location.
type Notes struct {
	m map[string]any
}

// Set adds a note. If there is an existing note with the same name,
// the old value is replaced with the new one.
func (n *Notes) Set(name string, val any) {
	if n.m == nil {
		n.m = make(map[string]any)
	}
	n.m[name] = val
}

// Get retrieves a note, including reporting whether it exists.
func (n *Notes) Get(name string) (val any, ok bool) {
	val, ok = n.m[name]
	return val, ok
}

// AppendNote appends values to a note.
// This is a function, not a method, so that it can be generic.
// This expects any existing note to have type []E,
// and panics if it does not.
func AppendNote[E any](n *Notes, name string, val ...E) {
	if n.m == nil {
		n.m = make(map[string]any)
	}
	var s []E
	if old := n.m[name]; old != nil {
		var ok bool
		s, ok = old.([]E)
		if !ok {
			panic(fmt.Sprintf("for note %s attempt to append value of type %T to value of type %T", name, val, old))
		}
	}
	n.m[name] = append(s, val...)
}

// AddNotes adds all notes in the elements of ns to n.
// Scalar values in added notes replace those in n.
// Slices in added notes are appended to slices in n;
// if the element types differ,
// or if the value in n is not a slice, AddNotes will panic.
// Names in n that do not appear in added notes are unchanged.
func (n *Notes) AddNotes(ns ...Notes) {
	for _, n2 := range ns {
		for k2, v2 := range n2.m {
			v1, ok1 := n.Get(k2)
			if !ok1 || reflect.TypeOf(v2).Kind() != reflect.Slice {
				n.Set(k2, v2)
			} else {
				n.Set(k2, reflect.AppendSlice(reflect.ValueOf(v1), reflect.ValueOf(v2)).Interface())
			}
		}
	}
}

// Clear clears all current notes.
func (n *Notes) Clear() {
	n.m = nil
}

// IsEmpty reports whether there are no notes.
func (n *Notes) IsEmpty() bool {
	return len(n.m) == 0
}

// String returns a printable Notes.
func (n Notes) String() string {
	return fmt.Sprint(n.m)
}
