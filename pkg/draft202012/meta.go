// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package draft202012

import (
	"embed"
	"net/url"

	"github.com/altshiftab/jsonschema/internal/metaschema"
	"github.com/altshiftab/jsonschema/pkg/types"
)

//go:embed metaschema/*.json metaschema/*/*.json
var metaFS embed.FS

// checkMetaSchema checks whether uri refers to the meta-schema,
// and loads the schema if it does. If uri is not the meta-schema,
// this returns nil, nil.
func checkMetaSchema(uri *url.URL, ropts *types.ResolveOpts) (*types.Schema, error) {
	return metaschema.Load(SchemaID, "/draft/2020-12/", &metaFS, uri, ropts)
}
