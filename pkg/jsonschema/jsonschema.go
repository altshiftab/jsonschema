package jsonschema

import (
	jsonv2 "encoding/json/v2"
	"fmt"

	motmedelErrors "github.com/Motmedel/utils_go/pkg/errors"
	motmedelReflect "github.com/Motmedel/utils_go/pkg/reflect"
	_ "github.com/altshiftab/jsonschema/pkg/draft202012"
	_ "github.com/altshiftab/jsonschema/pkg/format"
	schemaPkg "github.com/altshiftab/jsonschema/pkg/schema"
	jsonschemaTypeGeneration "github.com/vphpersson/type_generation/pkg/producers/jsonschema"
)

type Schema = schemaPkg.Schema

// ValidateError is the error returned by [Schema.Validate] when an
// instance fails validation. It matches
// [motmedelErrors.ErrValidationError] with errors.Is.
type ValidateError = schemaPkg.ValidateError

// ValidationError is a single validation failure within a [ValidateError].
type ValidationError = schemaPkg.ValidationError

func New(data []byte) (*Schema, error) {
	var s Schema
	if err := jsonv2.Unmarshal(data, &s); err != nil {
		return nil, motmedelErrors.New(fmt.Errorf("json unmarshal: %w", err))
	}

	return &s, nil
}

func NewFromType[T any]() (*Schema, error) {
	schemaData, err := jsonschemaTypeGeneration.Convert(motmedelReflect.TypeOf[T]())
	if err != nil {
		return nil, motmedelErrors.New(fmt.Errorf("type generation jsonschema convert: %w", err))
	}

	schema, err := New([]byte(schemaData))
	if err != nil {
		return nil, motmedelErrors.New(fmt.Errorf("new: %w", err))
	}

	return schema, nil
}
