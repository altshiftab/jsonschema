package format_test

import (
	"encoding/json"
	"testing"

	"github.com/altshiftab/jsonschema/pkg/draft202012"
	_ "github.com/altshiftab/jsonschema/pkg/format"
	"github.com/altshiftab/jsonschema/pkg/types/schema"
)

// buildEmailSchema returns a schema: {"type":"string","format":"email"}
func buildEmailSchema() *schema.Schema {
	b := draft202012.NewBuilder()
	b = b.AddType("string")
	b = b.AddFormat("email")
	return b.Build()
}

// buildArrayOfEmailSchema returns a schema:
//
//	{"type":"array","items":{"type":"string","format":"email"}}
func buildArrayOfEmailSchema() *schema.Schema {
	itemsSchema := draft202012.NewSubBuilder()
	itemsSchema = itemsSchema.AddType("string")
	itemsSchema = itemsSchema.AddFormat("email")

	b := draft202012.NewBuilder()
	b = b.AddType("array")
	b = b.AddItems(itemsSchema.Build())
	return b.Build()
}

// buildFlatArrayFormatSchema returns a schema with format on the array itself:
//
//	{"type":"array","format":"email"}
func buildFlatArrayFormatSchema() *schema.Schema {
	b := draft202012.NewBuilder()
	b = b.AddType("array")
	b = b.AddFormat("email")
	return b.Build()
}

func TestEmailFormatSingleString(t *testing.T) {
	s := buildEmailSchema()

	// Valid email should pass.
	if err := s.Validate("user@example.com"); err != nil {
		t.Errorf("expected valid email to pass, got error: %v", err)
	}

	// Invalid email should fail.
	if err := s.Validate("not-an-email"); err == nil {
		t.Error("expected invalid email to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_ItemsSchema(t *testing.T) {
	// Using items-level format: {"type":"array","items":{"type":"string","format":"email"}}
	s := buildArrayOfEmailSchema()

	// A slice with all valid emails should pass.
	valid := []any{"alice@example.com", "bob@example.com"}
	if err := s.Validate(valid); err != nil {
		t.Errorf("expected all-valid email slice to pass, got error: %v", err)
	}

	// A slice containing an invalid email should fail.
	invalid := []any{"alice@example.com", "not-an-email"}
	if err := s.Validate(invalid); err == nil {
		t.Error("expected slice with invalid email to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_FlatFormat(t *testing.T) {
	// Schema: {"type":"array","format":"email"}
	// Format is on the array itself. ValidateFormat now iterates over
	// slice elements, so this catches invalid emails too.
	s := buildFlatArrayFormatSchema()

	// All valid emails should pass.
	valid := []any{"alice@example.com", "bob@example.com"}
	if err := s.Validate(valid); err != nil {
		t.Errorf("expected all-valid email slice to pass, got error: %v", err)
	}

	// A slice with invalid emails should now fail.
	invalid := []any{"not-an-email", "also-not-email"}
	if err := s.Validate(invalid); err == nil {
		t.Error("expected slice with invalid email to fail validation, but got nil")
	}

	// Mixed: one valid, one invalid should fail.
	mixed := []any{"alice@example.com", "not-an-email"}
	if err := s.Validate(mixed); err == nil {
		t.Error("expected slice with mixed valid/invalid emails to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_GoStringSlice(t *testing.T) {
	// Using the items schema with a Go []string (not []any).
	s := buildArrayOfEmailSchema()

	valid := []string{"alice@example.com", "bob@example.com"}
	if err := s.Validate(valid); err != nil {
		t.Errorf("expected Go []string with valid emails to pass, got error: %v", err)
	}

	invalid := []string{"alice@example.com", "not-an-email"}
	if err := s.Validate(invalid); err == nil {
		t.Error("expected Go []string with invalid email to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_GoStringSlice_FlatFormat(t *testing.T) {
	// Format directly on the array, with a Go []string.
	s := buildFlatArrayFormatSchema()

	valid := []string{"alice@example.com", "bob@example.com"}
	if err := s.Validate(valid); err != nil {
		t.Errorf("expected Go []string with valid emails to pass, got error: %v", err)
	}

	invalid := []string{"alice@example.com", "not-an-email"}
	if err := s.Validate(invalid); err == nil {
		t.Error("expected Go []string with invalid email to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_JSONUnmarshal(t *testing.T) {
	// Validate JSON-unmarshaled arrays with items-level format.
	s := buildArrayOfEmailSchema()

	validJSON := `["alice@example.com", "bob@example.com"]`
	var validInstance any
	if err := json.Unmarshal([]byte(validJSON), &validInstance); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(validInstance); err != nil {
		t.Errorf("expected valid JSON email array to pass, got error: %v", err)
	}

	invalidJSON := `["alice@example.com", "not-an-email"]`
	var invalidInstance any
	if err := json.Unmarshal([]byte(invalidJSON), &invalidInstance); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(invalidInstance); err == nil {
		t.Error("expected JSON array with invalid email to fail validation, but got nil")
	}
}

func TestEmailFormatSlice_EmptySlice(t *testing.T) {
	s := buildFlatArrayFormatSchema()

	// An empty slice should pass.
	if err := s.Validate([]any{}); err != nil {
		t.Errorf("expected empty slice to pass, got error: %v", err)
	}
	if err := s.Validate([]string{}); err != nil {
		t.Errorf("expected empty Go string slice to pass, got error: %v", err)
	}
}
