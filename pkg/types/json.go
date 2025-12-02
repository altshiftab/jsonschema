// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/url"
	"slices"
)

// MarshalJSON marshals a [Schema] into JSON format.
// This implements [encoding/json.Marshaler].
func (s *Schema) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := s.marshalSchema(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalSchema marshals a [Schema] into JSON format,
// storing the results in buf.
func (s *Schema) marshalSchema(buf *bytes.Buffer) error {
	if isBoolSchema, isTrueSchema := s.isBoolSchema(); isBoolSchema {
		if isTrueSchema {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	}

	buf.WriteByte('{')

	first := true
	for _, part := range s.Parts {
		if part.Keyword.Generated {
			continue
		}

		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}

		fmt.Fprintf(buf, "%s:", encodeString(part.Keyword.Name))

		switch v := part.Value.(type) {
		case PartBool:
			fmt.Fprintf(buf, "%t", v)
		case PartString:
			fmt.Fprintf(buf, "%s", encodeString(string(v)))
		case PartStrings:
			buf.WriteByte('[')
			for i, s := range v {
				if i > 0 {
					buf.WriteByte(',')
				}
				fmt.Fprintf(buf, "%s", encodeString(s))
			}
			buf.WriteByte(']')
		case PartStringOrStrings:
			if v.Strings == nil {
				fmt.Fprintf(buf, "%s", encodeString(v.String))
			} else {
				buf.WriteByte('[')
				for i, s := range v.Strings {
					if i > 0 {
						buf.WriteByte(',')
					}
					fmt.Fprintf(buf, "%s", encodeString(s))
				}
				buf.WriteByte(']')
			}
		case PartInt:
			fmt.Fprintf(buf, "%d", v)
		case PartFloat:
			if PartFloat(int64(v)) == v {
				fmt.Fprintf(buf, "%d", int64(v))
			} else if PartFloat(uint64(v)) == v {
				fmt.Fprintf(buf, "%d", uint64(v))
			} else {
				fmt.Fprintf(buf, "%g", v)
			}
		case PartSchema:
			if err := v.S.marshalSchema(buf); err != nil {
				return err
			}
		case PartSchemas:
			buf.WriteByte('[')
			for i, schema := range v {
				if i > 0 {
					buf.WriteByte(',')
				}
				if err := schema.marshalSchema(buf); err != nil {
					return err
				}
			}
			buf.WriteByte(']')
		case PartMapSchema:
			buf.WriteByte('{')
			// Sort the names for predictable results.
			names := slices.Collect(maps.Keys(v))
			slices.Sort(names)
			for i, name := range names {
				if i > 0 {
					buf.WriteByte(',')
				}
				fmt.Fprintf(buf, "%s:", encodeString(name))
				if err := v[name].marshalSchema(buf); err != nil {
					return err
				}
			}
			buf.WriteByte('}')
		case PartSchemaOrSchemas:
			if v.Schema != nil {
				if err := v.Schema.marshalSchema(buf); err != nil {
					return err
				}
			} else {
				buf.WriteByte('[')
				for _, schema := range v.Schemas {
					if err := schema.marshalSchema(buf); err != nil {
						return err
					}
				}
				buf.WriteByte(']')
			}
		case PartMapArrayOrSchema:
			buf.WriteByte('{')
			// Sort the names for predictable results.
			names := slices.Collect(maps.Keys(v))
			slices.Sort(names)
			for i, name := range names {
				if i > 0 {
					buf.WriteByte(',')
				}
				fmt.Fprintf(buf, "%s:", encodeString(name))
				as := v[name]
				if as.Schema != nil {
					if err := as.Schema.marshalSchema(buf); err != nil {
						return err
					}
				} else {
					buf.WriteByte('[')
					for j, s := range as.Array {
						if j > 0 {
							buf.WriteByte(',')
						}
						fmt.Fprintf(buf, "%s", encodeString(s))
					}
					buf.WriteByte(']')
				}
			}
			buf.WriteByte('}')
		case PartAny:
			if err := json.NewEncoder(buf).Encode(v.V); err != nil {
				return err
			}
		default:
			return fmt.Errorf("schema.MarshalJSON: unexpected type %T", part.Value)
		}
	}

	buf.WriteByte('}')

	return nil
}

// isBoolSchema reports whether schema is a boolean schema,
// and reports whether it is the "true" schema.
func (s *Schema) isBoolSchema() (isBoolSchema, isTrueSchema bool) {
	isBoolSchema = false
	for _, part := range s.Parts {
		if part.Keyword == &SchemaKeyword || part.Keyword.Generated {
			continue
		}
		if part.Keyword != &BoolKeyword {
			return false, false
		}
		isBoolSchema = true
		isTrueSchema = bool(part.Value.(PartBool))
	}
	return isBoolSchema, isTrueSchema
}

// encodeString returns the JSON encoding of s.
func encodeString(s string) []byte {
	data, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("json.Marshal failed, which should be impossible: %v", err))
	}
	return data
}

// UnmarshalJSON decodes the JSON representation of a [Schema].
// This is fairly inefficient; we can probably do better with
// encoding/json/v2.
func (s *Schema) UnmarshalJSON(data []byte) error {
	s.Parts = s.Parts[:0:0]

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	vocabulary, err := s.buildTopFromJSON("", nil, v)
	if err != nil {
		return err
	}

	ropts := &ResolveOpts{
		Vocabulary: vocabulary,
		Loader:     loader,
	}
	return s.Resolve(ropts)
}

// buildTopFromJSON builds a [Schema] from JSON parsed into the
// empty interface value v. This assumes that this is the root schema.
func (s *Schema) buildTopFromJSON(schemaID string, uri *url.URL, v any) (*Vocabulary, error) {
	var version string
	if m, ok := v.(map[string]any); ok {
		if schemaVal, ok := m["$schema"]; ok {
			version, ok = schemaVal.(string)
			if !ok {
				return nil, errors.New("jsonschema: $schema does not have a string value")
			}
			s.Parts = append(s.Parts,
				Part{
					&SchemaKeyword,
					PartString(version),
				},
			)
			delete(m, "$schema")
		}
		v = m
	}

	if version == "" && schemaID != "" {
		version = schemaID
	}

	var vocabulary *Vocabulary
	if version == "" {
		vocabulary = DefaultVocabulary()
		if vocabulary == nil {
			return nil, errors.New("jsonschema: JSON schema version not specified and there is no default")
		}
		s.Parts = append(s.Parts,
			Part{
				&SchemaKeyword,
				PartString(vocabulary.Schema),
			},
		)
	} else {
		vocabulary = LookupVocabulary(version)
		if vocabulary == nil {
			return nil, fmt.Errorf("jsonschema: JSON schema version %q not recognized", version)
		}
	}

	err := s.buildFromJSON(v, vocabulary)
	return vocabulary, err
}

// SchemaFromJSON builds a [Schema] from a JSON value that has
// already been parsed. This could be used as something like
//
//	var v any
//	if err := json.Unmarshal(data, &v); err != nil { ... }
//	s, err := schema.SchemaFromJSON(schemaID, uri, v)
//
// This can be useful in cases where it's not clear whether the
// JSON encoding contains a schema or not.
//
// The optional schemaID argument is something like [draft202012.SchemaID].
// The optional uri is where the schema was loaded from.
//
// It is normally necessary to call Resolve on the result.
func SchemaFromJSON(schemaID string, uri *url.URL, v any) (*Schema, error) {
	var s Schema
	if _, err := s.buildTopFromJSON(schemaID, uri, v); err != nil {
		return nil, err
	}
	return &s, nil
}

// buildFromJSON builds a [Schema] from JSON parsed into the
// empty interface value v.
func (s *Schema) buildFromJSON(v any, vocabulary *Vocabulary) error {
	switch v := v.(type) {
	case bool:
		s.Parts = append(s.Parts, Part{
			&BoolKeyword,
			PartBool(v),
		})

	case map[string]any:
		for keyword, val := range v {
			if err := s.addKeywordFromJSON(keyword, val, vocabulary); err != nil {
				return err
			}
		}
		s.Finalize(vocabulary)

	default:
		return fmt.Errorf("jsonschema: unexpected type %T while JSON decoding schema", v)
	}
	return nil
}

// addKeywordFromJSON adds a [Schema] keyword and value parsed from JSON.
func (s *Schema) addKeywordFromJSON(keyword string, val any, vocabulary *Vocabulary) error {
	if len(keyword) == 0 {
		return errors.New("jsonschema: empty JSON keyword")
	}

	sk, ok := vocabulary.Keywords[keyword]
	if !ok {
		// Unrecognized keywords are ignored.
		// They do not affect the validation result.
		s.Parts = append(s.Parts, Part{
			Keyword: &Keyword{
				Name:     keyword,
				ArgType:  ArgTypeAny,
				Validate: validateTrue,
			},
			Value: PartAny{val},
		})
		return nil
	}

	var spv PartValue
	switch sk.ArgType {
	case ArgTypeBool:
		b, ok := val.(bool)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want bool", keyword, val)
		}
		spv = PartBool(b)
	case ArgTypeString:
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want string", keyword, val)
		}
		spv = PartString(s)
	case ArgTypeStrings:
		vals, ok := val.([]any)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want array of string", keyword, val)
		}
		strs := make([]string, 0, len(vals))
		for i, v := range vals {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("jsonschema: %q argument item %d is %T, want string", keyword, i, v)
			}
			strs = append(strs, s)
		}
		spv = PartStrings(strs)
	case ArgTypeStringOrStrings:
		s, ok := val.(string)
		if ok {
			spv = PartStringOrStrings{String: s}
		} else {
			vals, ok := val.([]any)
			if !ok {
				return fmt.Errorf("jsongschema: %q argument is type %T, want string or array of string", keyword, val)
			}
			strs := make([]string, 0, len(vals))
			for i, v := range vals {
				s, ok := v.(string)
				if !ok {
					return fmt.Errorf("jsonschema: %q argument item %d is %T, want string", keyword, i, v)
				}
				strs = append(strs, s)
			}
			spv = PartStringOrStrings{Strings: strs}
		}
	case ArgTypeInt:
		f, ok := val.(float64)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want integer", keyword, val)
		}
		if f != math.Trunc(f) {
			return fmt.Errorf("jsonschema: %q argument is non-integer, want integer", keyword)
		}
		spv = PartInt(f)
	case ArgTypeFloat:
		f, ok := val.(float64)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want number", keyword, val)
		}
		spv = PartFloat(f)
	case ArgTypeSchema:
		var s Schema
		if err := s.buildFromJSON(val, vocabulary); err != nil {
			return err
		}
		spv = PartSchema{&s}
	case ArgTypeSchemas:
		as, ok := val.([]any)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want array", keyword, val)
		}
		schemas := make([]*Schema, 0, len(as))
		for _, a := range as {
			var s Schema
			if err := s.buildFromJSON(a, vocabulary); err != nil {
				return err
			}
			schemas = append(schemas, &s)
		}
		spv = PartSchemas(schemas)
	case ArgTypeMapSchema:
		jm, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want object", keyword, val)
		}
		nm := make(map[string]*Schema, len(jm))
		for k, v := range jm {
			var s Schema
			if err := s.buildFromJSON(v, vocabulary); err != nil {
				return err
			}
			nm[k] = &s
		}
		spv = PartMapSchema(nm)
	case ArgTypeSchemaOrSchemas:
		var (
			schema  *Schema
			schemas []*Schema
		)
		as, ok := val.([]any)
		if ok {
			schemas = make([]*Schema, 0, len(as))
			for _, a := range as {
				var s Schema
				if err := s.buildFromJSON(a, vocabulary); err != nil {
					return err
				}
				schemas = append(schemas, &s)
			}
		} else {
			var s Schema
			if err := s.buildFromJSON(val, vocabulary); err != nil {
				return err
			}
			schema = &s
		}
		spv = PartSchemaOrSchemas{
			Schema:  schema,
			Schemas: schemas,
		}
	case ArgTypeMapArrayOrSchema:
		jm, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("jsonschema: %q argument is type %T, want object", keyword, val)
		}
		nm := make(map[string]ArrayOrSchema, len(jm))
		for k, v := range jm {
			var as ArrayOrSchema
			switch v := v.(type) {
			case bool, map[string]any:
				var s Schema
				if err := s.buildFromJSON(v, vocabulary); err != nil {
					return err
				}
				as.Schema = &s
			case []any:
				strs := make([]string, 0, len(v))
				for i, v := range v {
					s, ok := v.(string)
					if !ok {
						return fmt.Errorf("jsongschema: %q argument item %s:%d is %T, want string", keyword, k, i, v)
					}
					strs = append(strs, s)
				}
				as.Array = strs
			default:
				return fmt.Errorf("jsonschema: %q argument item %s is %T, want schema or array of strings", keyword, k, v)
			}
			nm[k] = as
		}
		spv = PartMapArrayOrSchema(nm)
	case ArgTypeAny:
		spv = PartAny{val}
	default:
		panic("can't happen")
	}

	s.Parts = append(s.Parts, Part{
		Keyword: sk,
		Value:   spv,
	})
	return nil
}
