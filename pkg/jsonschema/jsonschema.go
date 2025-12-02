package jsonschema

import (
	"encoding/json"
	"fmt"

	motmedelErrors "github.com/Motmedel/utils_go/pkg/errors"
	_ "github.com/altshiftab/jsonschema/pkg/draft202012"
	"github.com/altshiftab/jsonschema/pkg/types/schema"
)

type Schema = schema.Schema

func New(data []byte) (*Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, motmedelErrors.NewWithTrace(fmt.Errorf("json unmarshal: %w", err))
	}

	return &s, nil
}
