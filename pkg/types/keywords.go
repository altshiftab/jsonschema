// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// SchemaKeyword is a keyword to hold the schema version.
var SchemaKeyword = Keyword{
	Name:     "$schema",
	ArgType:  ArgTypeString,
	Validate: validateTrue,
}

// BoolKeyword is not a real keyword, but is used to represent the
// special schema values "true" and "false".
var BoolKeyword = Keyword{
	Name:     "$bool",
	ArgType:  ArgTypeBool,
	Validate: validateBool,
}

// validateTrue is a validator function that always succeeds.
func validateTrue(PartValue, any, *ValidationState) error {
	return nil
}

// validateBool handles the special $bool keyword,
// which does not actually appear in schema definitions.
func validateBool(arg PartValue, instance any, state *ValidationState) error {
	b := arg.(PartBool)
	if !b {
		return &ValidationError{
			Message: "false schema never matches",
		}
	}
	return nil
}
