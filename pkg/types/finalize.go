// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
)

// Finalize sorts the schema keywords into the order required for validation.
// Normally there is no need to call this explicitly.
// It will be called automatically by a Builder or by the JSON unmarshaler.
func (s *Schema) Finalize(v *Vocabulary) {
	slices.SortFunc(s.Parts, func(a, b Part) int {
		return v.Cmp(a.Keyword.Name, b.Keyword.Name)
	})
}

// Resolve resolves references across a schema and its subschemas.
// Normally there is no need to call this explicitly.
// It will be called automatically by the JSON unmarshaler.
func (s *Schema) Resolve(opts *ResolveOpts) error {
	var v *Vocabulary
	if opts != nil {
		v = opts.Vocabulary
	}

	if v == nil {
		for _, part := range s.Parts {
			if part.Keyword == &SchemaKeyword {
				v = LookupVocabulary(string(part.Value.(PartString)))
				if v == nil {
					return fmt.Errorf("no registered vocabulary for schema %q when resolving", part.Value.(PartString))
				}
				break
			}
		}
		if v == nil {
			return errors.New("unknown schema vocabulary when resolving")
		}
	}

	if opts == nil {
		opts = &ResolveOpts{
			Vocabulary: v,
			Loader:     loader,
		}
	}

	return v.Resolve(s, opts)
}

// ResolveOpts is options to use when resolving the schema.
// These are all optional.
type ResolveOpts struct {
	// The vocabulary to use.
	// This overrides anything recorded with the schema.
	Vocabulary *Vocabulary
	// URI of root of schema.
	// This is overridden by a $id keyword, if present.
	URI *url.URL
	// Load a remote reference, specifying the default schema.
	// This will be resolved by the resolver of the schema that
	// references it; no need for Loader to call (*Schema).Resolve.
	Loader func(schemaID string, uri *url.URL) (*Schema, error)
}

// SetLoader sets a function to call when resolving a $ref
// to an external schema. This is a global property,
// as there is no way to pass the desired value into the JSON decoder.
// Callers should use appropriate locking.
//
// Note that when unmarshaling user-written schemas,
// the loader function can be called with arbitrary URIs.
// It's probably unwise to simply call [net/http.Get] in all cases.
//
// To fully support JSON schema cross references, the loader should call
// [SchemaFromJSON]. The caller will handle calling [Schema.Resolve].
//
// This returns the old loader function.
// The default loader function is nil, which will produce an
// error for a $ref to an external schema.
func SetLoader(fn func(schemaID string, uri *url.URL) (*Schema, error)) func(string, *url.URL) (*Schema, error) {
	ret := loader
	loader = fn
	return ret
}

// loader is the default loader function.
var loader func(schemaID string, uri *url.URL) (*Schema, error)
