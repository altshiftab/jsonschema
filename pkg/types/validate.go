// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"net/url"

	"github.com/altshiftab/jsonschema/internal/validerr"
	"github.com/altshiftab/jsonschema/pkg/notes"
)

// Validate reports whether instance satisfies schema.
// If it does, this will return nil.
// If it does not, this will return an error with type either
// [*ValidationError] or [*ValidationErrors].
// A non-nil error with a different type indicates some error
// during validation processing.
//
// An instance may be an object read from JSON,
// with a Go type like map[string]any or []any.
// An instance may also be a Go struct or a pointer to a Go struct;
// in this case json tags on fields are used when matching field names.
func (s *Schema) Validate(instance any) error {
	return s.ValidateWithOpts(instance, nil)
}

// ValidateOpts describes validation options.
// These are uncommon so we use a separate method for them.
type ValidateOpts struct {
	// Whether to modify the instance being validated by setting defaults.
	// If this is true, then defaults are applied when:
	//   - a "properties" keyword is applied to a map or a struct
	//   - a "prefixItems" keyword is applied to a slice or array
	//   - a "items" keyword with an array argument (pre draft2020-12)
	//     is applied to a slice or array.
	// In these cases, if the subschema has a "default" keyword,
	// and the value in question is the zero value of its type
	// (or, in the case of a map, is missing), then the instance
	// is modified to be set to the default.
	// Defaults are ignored for required properties,
	// as the user must supply them.
	//
	// This operation may panic if the instance can't be modified.
	//
	// The modification is made before validation;
	// if the default value is not permitted by the rest of
	// the schema, validation may fail.
	ApplyDefaults bool

	// If this is true, then when a "properties" keyword is applied to a
	// map or struct, or an "items" keyword with a list of schemas is
	// applied to a slice or array, and if the schema uses a
	// "default" keyword, and if the corresponding

	// Whether to validate the format keyword.
	// In order for this to be effective, the package
	// jsonschema/format must be blank imported;
	// by default the format keyword always matches.
	ValidateFormat bool
}

// ValidateWithOpts is like Validate but supports options.
func (s *Schema) ValidateWithOpts(instance any, opts *ValidateOpts) error {
	var versionData any
	state := &ValidationState{
		Root:        s,
		VersionData: &versionData,
		Opts:        opts,
	}
	state.RootState = state
	return s.ValidateSubSchema(instance, state)
}

// ValidateInPlaceSchema reports whether instance satisfies schema,
// where schema is a subschema that is evaluated in the same context
// as the parent schema.
func (s *Schema) ValidateInPlaceSchema(instance any, state *ValidationState) error {
	subState, err := state.Child()
	if err != nil {
		return err
	}
	subState.Schema = s

	var topErr error
	for i, p := range s.Parts {
		if p.Keyword.Validate == nil {
			continue
		}
		subState.Index = i
        if err := p.Keyword.Validate(p.Value, instance, subState); err != nil {
            // Prefix with the current keyword name only if the error lacks any location.
            if hasAnyLocation(err) {
                validerr.AddError(&topErr, err, "")
            } else {
                validerr.AddError(&topErr, err, p.Keyword.Name)
            }
        }
    }

	state.Notes.AddNotes(subState.Notes)

	return topErr
}

// ValidateSubSchema reports whether instance satisfies schema,
// where schema is a sub-schema of some larger validation request.
// This is like Validate but also accepts the current validation state.
func (s *Schema) ValidateSubSchema(instance any, state *ValidationState) error {
	subState, err := state.Child()
	if err != nil {
		return err
	}
	subState.Schema = s

	var topErr error
	for i, p := range s.Parts {
		if p.Keyword.Validate == nil {
			continue
		}
		subState.Index = i
        if err := p.Keyword.Validate(p.Value, instance, subState); err != nil {
            // Prefix with the current keyword name only if the error lacks any location.
            if hasAnyLocation(err) {
                validerr.AddError(&topErr, err, "")
            } else {
                validerr.AddError(&topErr, err, p.Keyword.Name)
            }
        }
    }
    return topErr
}

// hasAnyLocation reports whether err already has a populated keyword or instance location.
func hasAnyLocation(err error) bool {
    switch e := err.(type) {
    case *validerr.ValidationError:
        return e.KeywordLocation != "" || e.InstanceLocation != ""
    case *validerr.ValidationErrors:
        for _, ve := range e.Errs {
            if ve.KeywordLocation != "" || ve.InstanceLocation != "" {
                return true
            }
        }
        return false
    default:
        return false
    }
}

// ValidationState is state we maintain while validating a schema.
// This does not apply to subschemas or parent schemas.
// This is exported for use by additional schema implementations.
// It is not expected to be used by code that just wants to validate a schema.
type ValidationState struct {
	// The root of the Schema being validatedd.
	Root *Schema
	// The ValidationState attached to the root Schema,
	// for global information.
	RootState *ValidationState
	// The Schema being validated.
	Schema *Schema
	// The index in schema.Parts of the keyword currently being validated.
	Index int
	// Current URI, from $id keyword.
	URI *url.URL
	// Notes created during validation.
	Notes notes.Notes
	// Depth of tree when validating. Used to avoid infinite recursion.
	Depth int
	// Validation options. Nil for the defaults.
	Opts *ValidateOpts
    // For use by version-specific code.
    VersionData *any

    // InstancePath holds the JSON Pointer tokens to the current location
    // within the instance being validated.
    InstancePath []string
}

// Child returns a new ValidationState that is a child of vs.
// This can be used to validate a subschema without changing
// the notes stored in vs.
func (vs *ValidationState) Child() (*ValidationState, error) {
	if vs.Depth > 1000 {
		return nil, errors.New("recursion while validating schema too deep")
	}

    ret := &ValidationState{
        Root:        vs.Root,
        RootState:   vs.RootState,
        Schema:      vs.Schema,
        Index:       vs.Index,
        URI:         vs.URI,
        Depth:       vs.Depth + 1,
        Opts:        vs.Opts,
        VersionData: vs.VersionData,
        InstancePath: append([]string(nil), vs.InstancePath...),
    }
    return ret, nil
}

// PushInstanceToken appends a token to the instance path.
func (vs *ValidationState) PushInstanceToken(tok string) {
    vs.InstancePath = append(vs.InstancePath, tok)
}

// PopInstanceToken removes the last token from the instance path.
func (vs *ValidationState) PopInstanceToken() {
    if n := len(vs.InstancePath); n > 0 {
        vs.InstancePath = vs.InstancePath[:n-1]
    }
}

// InstancePointer returns the current instance location as a JSON Pointer
// string starting with '#'.
func (vs *ValidationState) InstancePointer() string {
    if len(vs.InstancePath) == 0 {
        return "#"
    }
    // Escape per RFC 6901
    b := make([]byte, 0, 2*len(vs.InstancePath))
    b = append(b, '#', '/')
    for i, t := range vs.InstancePath {
        if i > 0 {
            b = append(b, '/')
        }
        // Replace ~ with ~0, / with ~1
        for j := 0; j < len(t); j++ {
            switch t[j] {
            case '~':
                b = append(b, '~', '0')
            case '/':
                b = append(b, '~', '1')
            default:
                b = append(b, t[j])
            }
        }
    }
    return string(b)
}

// EnsureInstanceLocation sets InstanceLocation on validation errors if empty.
func EnsureInstanceLocation(err error, ptr string) error {
    switch e := err.(type) {
    case *validerr.ValidationError:
        if e.InstanceLocation == "" || e.InstanceLocation == "#" {
            e.InstanceLocation = ptr
        }
        return e
    case *validerr.ValidationErrors:
        for _, ve := range e.Errs {
            if ve.InstanceLocation == "" || ve.InstanceLocation == "#" {
                ve.InstanceLocation = ptr
            }
        }
        return e
    default:
        return err
    }
}

// ValidationError is returned by a validation function
// when an instance fails validation.
type ValidationError = validerr.ValidationError

// ValidationErrors is a collection of ValidationError values.
type ValidationErrors = validerr.ValidationErrors

// IsValidationError reports whether err is a validation error.
func IsValidationError(err error) bool {
	return validerr.IsValidationError(err)
}
