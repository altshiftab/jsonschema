package jsonschema

import (
	"encoding/json"
	"fmt"

	motmedelErrors "github.com/Motmedel/utils_go/pkg/errors"
	motmedelReflect "github.com/Motmedel/utils_go/pkg/reflect"
	_ "github.com/altshiftab/jsonschema/pkg/draft202012"
	schemaPkg "github.com/altshiftab/jsonschema/pkg/types/schema"
	jsonschemaTypeGeneration "github.com/vphpersson/type_generation/pkg/producers/jsonschema"
)

type Schema = schemaPkg.Schema

func New(data []byte) (*Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, motmedelErrors.NewWithTrace(fmt.Errorf("json unmarshal: %w", err))
	}

	return &s, nil
}

func FromType[T any]() (*Schema, error) {
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