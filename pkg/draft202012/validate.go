// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package draft202012

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/altshiftab/jsonschema/internal/validator"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// resolvedRefKeyword is a special Keyword used to record what a
// $ref keyword refers to in a schema.
var resolvedRefKeyword = types.Keyword{
	Name:      "$$resolvedRef",
	ArgType:   types.ArgTypeSchema,
	Validate:  validator.ValidateTrue,
	Generated: true,
}

// resolvedDynamicRefKeyword is a special Keyword used to record
// what a $dynamicRef refers to in a schema.
var resolvedDynamicRefKeyword = types.Keyword{
	Name:      "$$resolvedDynamicRef",
	ArgType:   types.ArgTypeSchema,
	Validate:  validator.ValidateTrue,
	Generated: true,
}

// detachedDynamicRefKeyword is a special Keyword used to record
// what a $dynamicRef refers to in a schema if we did not see
// any $dynamicAnchor while evaluating. We need this fallback for
// a reference to a subschema that skips over the base schema
// that records the dynamic anchor.
var detachedDynamicRefKeyword = types.Keyword{
	Name:      "$$detachedDynamicRef",
	ArgType:   types.ArgTypeSchema,
	Validate:  validator.ValidateTrue,
	Generated: true,
}

// recordDynamicAnchor is the type of the value stored with
// recordDynamicAnchorKeyword and clearDynamicAnchorKeyword.
type recordDynamicAnchor struct {
	anchor string
	schema *types.Schema
}

// recordDynamicAnchorKeyword is a special Keyword that records a
// $dynamicAnchor. The string is the name of the $dynamicAnchor.
var recordDynamicAnchorKeyword = types.Keyword{
	Name:      "$$recordDynamicAnchorKeyword",
	ArgType:   types.ArgTypeString,
	Validate:  validator.ArgTypeAny(validateRecordDynamicAnchor),
	Generated: true,
}

// clearDynamicAnchorKeyword is a special Keyword that removes a
// $dynamicAnchor stored during validation.
var clearDynamicAnchorKeyword = types.Keyword{
	Name:      "$$clearDynamicAnchorKeyword",
	ArgType:   types.ArgTypeString,
	Validate:  validator.ArgTypeAny(validateClearDynamicAnchor),
	Generated: true,
}

// validateRef validates a $ref keyword.
func validateRef(arg types.PartString, instance any, state *types.ValidationState) error {
	for _, part := range state.Schema.Parts {
		if part.Keyword == &resolvedRefKeyword {
			return part.Value.(types.PartSchema).S.ValidateInPlaceSchema(instance, state)
		}
	}
	// This should never happen.
	return fmt.Errorf(`reference %q unresolved`, arg)
}

// validateDynamicRef validates a $dynamicRef keyword.
func validateDynamicRef(arg types.PartString, instance any, state *types.ValidationState) error {
	// See if this was resolved non-dynamically.
	var s *types.Schema
	for _, part := range state.Schema.Parts {
		if part.Keyword == &resolvedDynamicRefKeyword {
			s = part.Value.(types.PartSchema).S
			break
		}
	}

	if s == nil {
		// Resolve dynamically.
		var err error
		s, err = resolveDynamicRef(arg, state)
		if err != nil {
			return err
		}

		if s == nil {
			// Last try: a detached $dynamicAnchor.
			for _, part := range state.Schema.Parts {
				if part.Keyword == &detachedDynamicRefKeyword {
					s = part.Value.(types.PartSchema).S
					break
				}
			}

			if s == nil {
				return fmt.Errorf("dynamic reference %q unresolved", arg)
			}
		}
	}

	return s.ValidateInPlaceSchema(instance, state)
}

// validationData is data specific to the draft used for validation.
// We record the current dynamic anchors.
type validationData struct {
	dynamicAnchors map[string]*types.Schema
}

// validateRecordDynamicAnchor records a dynamic anchor during validation.
// This is added by the builder when we see a $dynamicAnchor.
// We record the dynamic anchor while validating this schema,
// so that a $dynamicRef can see it.
func validateRecordDynamicAnchor(arg types.PartAny, instance any, state *types.ValidationState) error {
	da := arg.V.(*recordDynamicAnchor)
	if *state.VersionData == nil {
		*state.VersionData = &validationData{
			dynamicAnchors: make(map[string]*types.Schema),
		}
	}
	vd := (*state.VersionData).(*validationData)
	if _, ok := vd.dynamicAnchors[da.anchor]; ok {
		// We already have a this dynamic anchors.
		// Dynamic anchors use a top-down scope.
		return nil
	}
	vd.dynamicAnchors[da.anchor] = da.schema
	return nil
}

// validateClearDynamicAnchor clear a dynamic anchor during validation.
// This is added by the builder when we see a $dynamicAnchor,
// at the end of the schema. This removes the dynamic anchor added by
// validateRecordDynamicAnchor, so that the dynamic anchor is only
// visible while processing the scheme that defines int.
func validateClearDynamicAnchor(arg types.PartAny, instance any, state *types.ValidationState) error {
	da := arg.V.(*recordDynamicAnchor)
	vd := (*state.VersionData).(*validationData)
	if vd.dynamicAnchors[da.anchor] == da.schema {
		delete(vd.dynamicAnchors, da.anchor)
	}
	return nil
}

// resolveDynamicRef dynamically resolves a $dynamicRef.
// This returns nil if the reference can't be resolved.
func resolveDynamicRef(arg types.PartString, state *types.ValidationState) (*types.Schema, error) {
	if *state.VersionData == nil {
		return nil, nil
	}

	uri, err := url.Parse(string(arg))
	if err != nil {
		return nil, err
	}
	if uri.Fragment == "" || strings.HasPrefix(uri.Fragment, "/") {
		return nil, nil
	}

	vd := (*state.VersionData).(*validationData)
	s, ok := vd.dynamicAnchors[uri.Fragment]
	if !ok {
		return nil, nil
	}
	return s, nil
}
