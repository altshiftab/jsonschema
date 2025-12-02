// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run ../internal/cmd/keywordgen -p draft202012 -o keywords.go corekeywords.json applicatorkeywords.json

// Package draft202012 defines the keywords used by
// JSON schema version 2020-12.
package draft202012

import (
	"github.com/altshiftab/jsonschema/pkg/types"
)

const SchemaID = "https://json-schema.org/draft/2020-12/schema"

var Vocabulary = &types.Vocabulary{
	Name:     "draft2020-12",
	Schema:   SchemaID,
	Keywords: keywordMap,
	Cmp:      keywordCmp,
	Resolve:  resolveSchema,
}

func init() {
	types.RegisterVocabulary(Vocabulary, true)
}
