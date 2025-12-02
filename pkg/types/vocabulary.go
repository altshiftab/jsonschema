// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"strings"
	"sync"
)

// Vocabulary is a vocabulary type: a list of known keywords.
// Each schema version defines an instance of this type.
type Vocabulary struct {
	// The name of this schema version, for messages.
	// Something like draft-2020-12.
	Name string
	// The URI that describes this schema version.
	// The value of the $schema keyword.
	// Something like "https://json-schema.org/draft/2020-12/schema".
	Schema string
	// The keywords of this schema version.
	Keywords map[string]*Keyword
	// A function that resolves references within a schema.
	Resolve func(*Schema, *ResolveOpts) error
	// The sorting function of this schema.
	// Used to sort the keywords of an instance of the schema.
	Cmp func(string, string) int
}

// A registry is a mapping from schema name to Vocabulary.
type registry struct {
	mu      sync.Mutex
	mapping map[string]*Vocabulary
	defval  *Vocabulary // default vocabulary
}

// Adds adds an item to the registry.
func (r *registry) add(s string, v *Vocabulary, def bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mapping == nil {
		r.mapping = make(map[string]*Vocabulary)
	}
	if _, found := r.mapping[s]; found {
		panic(fmt.Sprintf("jsonschema: multiple attempts to add %q to registry", s))
	}
	r.mapping[s] = v
	if def {
		if r.defval != nil {
			panic("jsonschema: multiple default vocabularies")
		}
		r.defval = v
	}
}

// lookup returns an element from the registry,
// or nil if not present.
func (r *registry) lookup(s string) *Vocabulary {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.mapping[s]
}

// def returns the default vocabulary,
// or nil if there isn't one.
func (r *registry) def() *Vocabulary {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.defval != nil {
		return r.defval
	}
	if len(r.mapping) == 1 {
		for _, v := range r.mapping {
			return v
		}
	}
	return nil
}

// setDef sets the default vocabulary.
func (r *registry) setDef(s string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, v := range r.mapping {
		if v.Name == s {
			r.defval = v
			return nil
		}
	}

	return fmt.Errorf("setting default to %q failed: unknown schema ID ", s)
}

// clear clears the registry, removing all entries.
func (r *registry) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mapping = nil
	r.defval = nil
}

// reg is the global registry.
var reg registry

// RegisterVocabulary registers a vocabulary.
// The def argument is true for the default vocabulary.
// It's normally not necessary to call this;
// importing a JSON schema version package will register it.
func RegisterVocabulary(v *Vocabulary, def bool) {
	reg.add(v.Schema, v, def)
}

// LookupVocabulary returns a registered vocabulary, or nil if no vocabulary
// was registered under that name.
// It's normally not necessary to call this;
// instead use something like draft202012.Vocabulary.
func LookupVocabulary(s string) *Vocabulary {
	// For draft7 we can see
	// "http://json-schema.org/draft-07/schema#"
	s = strings.TrimSuffix(s, "#")
	return reg.lookup(s)
}

// DefaultVocabulary returns the default vocabulary, or nil if there isn't one.
func DefaultVocabulary() *Vocabulary {
	return reg.def()
}

// SetDefaultSchema sets the default schema.
// The argument should be something like "draft7" or "draft2020-12".
// This is a global property, as there is no way to pass the desired
// value into the JSON decoder. Callers should use appropriate locking.
// This is mainly for tests.
func SetDefaultSchema(s string) error {
	return reg.setDef(s)
}

// ClearVocabularies discards the vocabulary registry.
// This is for tests.
func ClearVocabularies() {
	reg.clear()
}
