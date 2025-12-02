package types_test

import (
	"errors"
	"testing"

	"github.com/altshiftab/jsonschema/pkg/draft202012"
	"github.com/altshiftab/jsonschema/pkg/types"
)

// Ensure the draft 2020-12 vocabulary is registered via init.

func TestBasicOutput_TypeUnderProperties(t *testing.T) {
	schemaJSON := map[string]any{
		"$schema": draft202012.SchemaID,
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
	}

	s, err := types.SchemaFromJSON(draft202012.SchemaID, nil, schemaJSON)
	if err != nil {
		t.Fatalf("SchemaFromJSON: %v", err)
	}

	inst := map[string]any{"name": 123}
	err = s.Validate(inst)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	// Expect a single validation error describing type mismatch at:
	// keywordLocation: #/properties/name/type
	// instanceLocation: #/name
	var ve *types.ValidationError
	if !errors.As(err, &ve) {
		var ves *types.ValidationErrors
		if !errors.As(err, &ves) || len(ves.Errs) != 1 {
			t.Fatalf("expected single ValidationError, got %T: %v", err, err)
		}
		ve = ves.Errs[0]
	}

	if ve.KeywordLocation != "#/properties/name/type" {
		t.Fatalf("keywordLocation: got %q, want %q", ve.KeywordLocation, "#/properties/name/type")
	}
	if ve.InstanceLocation != "#/name" {
		t.Fatalf("instanceLocation: got %q, want %q", ve.InstanceLocation, "#/name")
	}
	if ve.Message == "" {
		t.Fatalf("error message should not be empty")
	}
	// And Error() should render as "<keywordLocation>: <error>"
	if got := ve.Error(); got[:len("#/")] != "#/" {
		t.Fatalf("Error() prefix: got %q", got)
	}
}

func TestBasicOutput_RequiredMissing(t *testing.T) {
	schemaJSON := map[string]any{
		"$schema": draft202012.SchemaID,
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
		"required": []any{"name"},
	}

	s, err := types.SchemaFromJSON(draft202012.SchemaID, nil, schemaJSON)
	if err != nil {
		t.Fatalf("SchemaFromJSON: %v", err)
	}

	inst := map[string]any{}
	err = s.Validate(inst)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	var ve *types.ValidationError
	if !errors.As(err, &ve) {
		var ves *types.ValidationErrors
		if !errors.As(err, &ves) || len(ves.Errs) != 1 {
			t.Fatalf("expected single ValidationError, got %T: %v", err, err)
		}
		ve = ves.Errs[0]
	}

	if ve.KeywordLocation != "#/required/name" {
		t.Fatalf("keywordLocation: got %q, want %q", ve.KeywordLocation, "#/required/name")
	}
	if ve.InstanceLocation != "#" {
		t.Fatalf("instanceLocation: got %q, want %q", ve.InstanceLocation, "#")
	}
	if ve.Message == "" {
		t.Fatalf("error message should not be empty")
	}
}
