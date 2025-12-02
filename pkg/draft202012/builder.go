// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package draft202012

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/altshiftab/jsonschema/internal/schemacache"
	"github.com/altshiftab/jsonschema/pkg/builder"
	"github.com/altshiftab/jsonschema/pkg/jsonpointer"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// Builder is a JSON schema builder.
// Builder provides a list of methods that may be used to add
// new elements to the schema.
// This should be used by programs that need to create a JSON schema
// from scratch, rather than unmarshaling it from a JSON representation
// or using [schemareflect.Reflect] to construct it from a Go type.
//
// Programs should use [NewBuilder] or [NewSubBuilder] to get a Builder.
type Builder struct {
	b *builder.Builder
}

// NewBuilder returns a [Builder] to use to build a JSON schema.
// Use this to build an entirely new schema.
func NewBuilder() *Builder {
	b := &Builder{builder.New(Vocabulary)}
	return b.AddString(&types.SchemaKeyword, SchemaID)
}

// NewSubBuilder returns a [Builder] like [NewBuilder],
// but is for a schema that will be part of some larger schema.
func NewSubBuilder() *Builder {
	return &Builder{builder.New(Vocabulary)}
}

// Build returns a newly built schema.
func (b *Builder) Build() *types.Schema {
	return b.b.Build()
}

// NewSubBuilder returns a new [Builder] with the same vocabulary.
// This is like the [NewSubBuilder] function in that it is for schemas
// that will be part of some larger schema.
func (b *Builder) NewSubBuilder() *Builder {
	return &Builder{builder.New(Vocabulary)}
}

// BoolSchema returns a newly built schema.
// If acceptAll is true the schema accepts all instance values,
// if false it accepts none.
// This is the JSON schema true and false values.
func (b *Builder) BoolSchema(acceptAll bool) *types.Schema {
	b2 := b.NewSubBuilder()
	b2.b.AddBool(&types.BoolKeyword, acceptAll)
	return b2.Build()
}

// AddSchemaParts adds a list of parts.
func (b *Builder) AddSchemaParts(parts []types.Part) *Builder {
	b.b = b.b.AddSchemaParts(parts)
	return b
}

// Infer adds schema elements to b designed to validate JSON values
// that unmarshal into values of the given type.
// See [builder.Infer] for details.
func Infer[T any](b *Builder, opts *builder.InferOpts) (*Builder, error) {
	return builder.Infer[T](b, opts)
}

// InferType is like [Infer] buts takes a [reflect.Type] rather than
// a type argument.
func InferType(b *Builder, typ reflect.Type, opts *builder.InferOpts) (*Builder, error) {
	return builder.InferType(b, typ, opts)
}

// AddItemsSchema is for builder.Infer. Use the AddItems method instead.
func (b *Builder) AddItemsSchema(s *types.Schema) *Builder {
	return b.AddItems(s)
}

// resolveState holds state during resolveSchema.
type resolveState struct {
	ropts   *types.ResolveOpts
	root    *types.Schema
	schemas map[*types.Schema]schemaData
	uris    map[string]*types.Schema
	anchors map[string]anchorData
	cache   schemacache.Cache
}

// schemaData is information we keep for some schemas.
type schemaData struct {
	uri *url.URL
}

// anchorData is information we keep for an anchor.
type anchorData struct {
	schema  *types.Schema
	dynamic bool // true for $dynamicAnchor
}

// subInfo holds information we pass down to subschemas.
type subInfo struct {
	uri  *url.URL
	name []string
}

// Name returns the name of the current subschema.
func (si subInfo) Name() string {
	return "/" + strings.Join(si.name, "/")
}

// resolveSchema is the Vocabulary.Resolve field.
// It is called to resolve a schema decoded from JSON to
// handle $ref and friends.
func resolveSchema(schema *types.Schema, ropts *types.ResolveOpts) error {
	state := &resolveState{
		ropts: ropts,
		root:  schema,
	}
	var uri *url.URL
	if ropts != nil {
		uri = ropts.URI
	}
	return resolveRefSchema(uri, schema, state)
}

// resolveRefSchema resolves a schema that may have a known URI.
func resolveRefSchema(uri *url.URL, schema *types.Schema, state *resolveState) error {
	subData := subInfo{
		uri: uri,
	}
	if err := resolveIDs(schema, schema, state, subData); err != nil {
		return err
	}
	return resolveRefs(schema, state, subData)
}

// resolveIDs finds the IDs and anchors in a schema.
func resolveIDs(subSchema, base *types.Schema, state *resolveState, subData subInfo) error {
	if subSchema == nil {
		return nil
	}

	var dynamicAnchor string
	for _, part := range subSchema.Parts {
		var err error
		switch part.Keyword.Name {
		case "$id":
			err, subData = resolveID(subSchema, part.Value, state, subData)
			base = subSchema
		case "$anchor":
			_, err = resolveAnchor(subSchema, false, part.Value, state, subData)
		case "$dynamicAnchor":
			if dynamicAnchor != "" {
				return fmt.Errorf("%s: more than one $dynamicAnchor", subData.Name())
			}
			dynamicAnchor, err = resolveAnchor(subSchema, true, part.Value, state, subData)
		case "$ref", "$dynamicRef":
			// We need the URI when resolving references.
			if state.schemas == nil {
				state.schemas = make(map[*types.Schema]schemaData)
			}
			state.schemas[subSchema] = schemaData{uri: subData.uri}
		}
		if err != nil {
			return err
		}
	}

	if dynamicAnchor != "" {
		// Add special keywords to set and clear the dynamic
		// anchor during validation. These keywords need to be
		// added to the root schema, but only if the root doesn't
		// already have a dynamic anchor. This implements the
		// dynamic scoping that resolves to the outermost anchor.

		sawDynamicAnchor := false
		for _, part := range base.Parts {
			if part.Keyword == &recordDynamicAnchorKeyword {
				sawDynamicAnchor = true
				break
			}
		}
		if !sawDynamicAnchor {
			val := &recordDynamicAnchor{
				anchor: dynamicAnchor,
				schema: subSchema,
			}
			recordDynamicAnchor := types.Part{
				Keyword: &recordDynamicAnchorKeyword,
				Value:   types.PartAny{val},
			}
			base.Parts = append([]types.Part{recordDynamicAnchor}, base.Parts...)
			base.Parts = append(base.Parts,
				types.Part{
					Keyword: &clearDynamicAnchorKeyword,
					Value:   types.PartAny{val},
				},
			)
		}
	}

	for name, subsub := range subSchema.Children() {
		subsubData := subInfo{
			uri:  subData.uri,
			name: append(subData.name, name),
		}
		if err := resolveIDs(subsub, base, state, subsubData); err != nil {
			return err
		}
	}

	return nil
}

// resolveID handles the $id keyword when searching for anchors.
func resolveID(subSchema *types.Schema, value types.PartValue, state *resolveState, subData subInfo) (error, subInfo) {
	arg := value.(types.PartString)
	uri, err := url.Parse(string(arg))
	if err != nil {
		return fmt.Errorf(`%s: failed to parse "$id" %q: %v`, subData.Name(), arg, err), subInfo{}
	}
	if uri.Fragment != "" {
		return fmt.Errorf(`%s: "$id" %q contains non-empty fragment`, subData.Name(), err), subInfo{}
	}
	var newURI *url.URL
	if uri.IsAbs() || subData.uri == nil {
		newURI = uri
	} else {
		newURI = subData.uri.ResolveReference(uri)
	}

	if state.uris == nil {
		state.uris = make(map[string]*types.Schema)
	}
	state.uris[newURI.String()] = subSchema

	si := subInfo{
		uri:  newURI,
		name: subData.name,
	}
	return nil, si
}

// resolveAnchor handles the $anchor and $dynamicAnchor keywords
// when searching for anchors.
func resolveAnchor(subSchema *types.Schema, dynamic bool, value types.PartValue, state *resolveState, subData subInfo) (string, error) {
	anchor := string(value.(types.PartString))
	if state.anchors == nil {
		state.anchors = make(map[string]anchorData)
	}

	var anchorURIBase url.URL
	if subData.uri != nil {
		anchorURIBase = *subData.uri
	}
	if anchorURIBase.Fragment != "" {
		panic("can't happen")
	}
	anchorURIBase.Fragment = anchor
	anchorURI := &anchorURIBase
	anchorStr := anchorURI.String()

	if _, ok := state.anchors[anchorStr]; ok {
		fmt.Printf("%v\n", subSchema)
		return "", fmt.Errorf("%s: duplicate anchor %q", subData.Name(), anchorStr)
	}
	state.anchors[anchorStr] = anchorData{
		schema:  subSchema,
		dynamic: dynamic,
	}
	return anchor, nil
}

// resolveRefs resolves all $ref and $dynamicRef keywords in the schema.
func resolveRefs(subSchema *types.Schema, state *resolveState, subData subInfo) error {
	if subSchema == nil {
		return nil
	}

	sawRef, sawDynamicRef := false, false
	for _, part := range subSchema.Parts {
		var err error
		switch part.Keyword.Name {
		case "$ref":
			if sawRef {
				return fmt.Errorf("%s: more than one $ref", subData.Name())
			}
			sawRef = true
			err = resolveRef(subSchema, false, part.Value, state, subData)
		case "$dynamicRef":
			if sawDynamicRef {
				return fmt.Errorf("%s: more than one $dynamicRef", subData.Name())
			}
			sawDynamicRef = true
			err = resolveRef(subSchema, true, part.Value, state, subData)
		}
		if err != nil {
			return err
		}
	}

	for name, subsub := range subSchema.Children() {
		subsubData := subInfo{
			name: append(subData.name, name),
		}
		if err := resolveRefs(subsub, state, subsubData); err != nil {
			return err
		}
	}

	return nil
}

// resolveRef resolves a $ref or $dynamicRef in the schema.
// We record the resolved reference using a magic keyword.
func resolveRef(subSchema *types.Schema, dynamic bool, value types.PartValue, state *resolveState, subData subInfo) error {
	ref := string(value.(types.PartString))
	refURI, err := url.Parse(ref)
	if err != nil {
		return err
	}

	sd, ok := state.schemas[subSchema]
	if !ok {
		// Should have been handled in resolveIDs.
		panic("resolveIDs did not resolve schema URI")
	}
	if sd.uri != nil {
		refURI = sd.uri.ResolveReference(refURI)
	}

	frag := refURI.Fragment

	// A $dynamicRef with a JSON pointer is not really dynamic.
	dynamicFrag := dynamic
	if dynamic && (frag == "" || strings.HasPrefix(frag, "/")) {
		dynamicFrag = false
	}

	addRef := func(refSchema *types.Schema, detached bool) {
		resolvedKey := &resolvedRefKeyword
		if dynamic {
			resolvedKey = &resolvedDynamicRefKeyword
		}
		if detached {
			// This is a backup for a $dynamicRef to a
			// $dynamicAnchor, to be used if we skip over
			// the recordDynamicAnchor.
			resolvedKey = &detachedDynamicRefKeyword
		}

		subSchema.Parts = append(subSchema.Parts,
			types.Part{
				Keyword: resolvedKey,
				Value:   types.PartSchema{refSchema},
			},
		)
	}

	if ad, ok := state.anchors[refURI.String()]; ok {
		addRef(ad.schema, dynamicFrag && ad.dynamic)
		return nil
	}

	refSchema, err := resolveURI(refURI, state, subData)
	if err != nil {
		return err
	}

	// Loading and resolving the schema may have resolved
	// the reference. The schema was loaded without any fragment,
	// but refURI may include a fragment.
	if ad, ok := state.anchors[refURI.String()]; ok {
		addRef(ad.schema, dynamicFrag && ad.dynamic)
		return nil
	}

	// Otherwise, if there is a fragment, we expect it to be a
	// JSON pointer. A reference to an anchor should have been resolved by
	// looking in state.anchors.

	if frag != "" {
		if !strings.HasPrefix(frag, "/") {
			return fmt.Errorf("%s: could not find fragment %q from URI %q", subData.Name(), frag, refURI)
		}

		if refSchema, err = jsonpointer.DerefSchema(SchemaID, refSchema, frag); err != nil {
			return fmt.Errorf("%s: could not resolve JSON pointer %q from URI %q: %v", subData.Name(), frag, refURI, err)
		}
	}

	addRef(refSchema, false)
	return nil
}

// resolveURI returns the schema for a URI.
func resolveURI(refURI *url.URL, state *resolveState, subData subInfo) (*types.Schema, error) {
	// The URI, ignoring the fragment, is either the empty string,
	// meaning the root, or a reference to some $id elsewhere in
	// the schema tree, or a URI to be loaded externally.

	noFragURIBase := *refURI
	noFragURIBase.Fragment = ""
	noFragURI := &noFragURIBase
	noFragStr := noFragURI.String()

	// An empty URI means the schema root.
	if noFragStr == "" {
		return state.root, nil
	}

	// Check for a reference to a known schema $id.
	refSchema, ok := state.uris[noFragStr]
	if ok {
		return refSchema, nil
	}

	// The URI refers to something elsewhere.
	if !noFragURI.IsAbs() {
		return nil, fmt.Errorf("%s: could not resolve ref to %q", subData.Name(), noFragURI)
	}

	// Check for a reference to the metaschema.
	refSchema, err := checkMetaSchema(noFragURI, state.ropts)
	if err != nil {
		return nil, err
	}
	if refSchema != nil {
		return refSchema, nil
	}

	// We need to load the schema from a remote source.
	if state.ropts.Loader == nil {
		return nil, fmt.Errorf("%s: remote loading of URI %q not permitted", subData.Name(), noFragURI)
	}

	// Check the cache.
	refSchema = state.cache.Load(SchemaID, noFragStr)
	if refSchema != nil {
		return refSchema, nil
	}

	// Load the schema remotely.
	refSchema, err = state.ropts.Loader(SchemaID, noFragURI)
	if err != nil {
		return nil, fmt.Errorf("%s: loading of URI %q failed: %v", subData.Name(), noFragURI, err)
	}
	if refSchema == nil {
		return nil, fmt.Errorf("%s: loading of URI %q returned no schema and no error", subData.Name(), noFragURI)
	}

	// Cache the schema. We must do before resolving the schema,
	// as resolving the schema may try to load it again.
	state.cache.Store(SchemaID, noFragStr, refSchema)

	// Resolve the schema in the current resolution state.
	if err := resolveRefSchema(noFragURI, refSchema, state); err != nil {
		return nil, fmt.Errorf("%s: resolving schema at URI %q failed: %v", subData.Name(), noFragURI, err)
	}

	return refSchema, nil
}
