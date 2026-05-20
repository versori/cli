/*
 * Copyright (c) 2026 Versori Group Inc
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in the LICENSE file at the root of this repository.
 *
 * Change Date: 2030-03-01
 * Change License: Apache License, Version 2.0
 *
 * As of the Change Date, in accordance with the Business Source License,
 * use of this software will be governed by the Apache License, Version 2.0.
 */

// Package variables implements the `versori projects variables` subcommands. The schema below is
// the high-level Go view of the project's DynamicVariablesSchema (a JSON Schema), used by the
// add/list/remove/update commands. The schema.go file shields callers from the JSON-Schema shape
// so the CLI surface stays in terms of variable name / type / description / required and a small
// dotted-path DSL for nested fields.
package variables

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// validVariableTypes lists the JSON Schema "type" values the CLI exposes for variables (at any
// nesting level). Object and array are first-class — for them the recursive --field / --items-type
// DSL fills in properties / items so the resulting schema actually validates structure rather
// than accepting any object / array shape.
var validVariableTypes = []string{"string", "number", "integer", "boolean", "object", "array"}

// containerTypes is the subset of validVariableTypes that have nested structure managed by the
// CLI (object → properties + required, array → items). Used to decide when --field / --items-type
// flags apply and when interactive mode should recurse.
var containerTypes = map[string]bool{"object": true, "array": true}

// maxInteractiveNestingDepth caps how far interactive mode will recurse before refusing further
// nesting. Beyond this users should use the --field flag with a dotted path. The cap exists so a
// confused interactive user can't get stuck in a 20-level prompt loop with no escape.
const maxInteractiveNestingDepth = 5

// variablesSchema is the in-memory view of a project's DynamicVariablesSchema. It mirrors the
// JSON Schema "object with properties" shape that the platform expects but is friendlier to mutate.
type variablesSchema struct {
	Properties map[string]*subSchema `json:"properties"`
	Required   []string              `json:"required,omitempty"`
}

// subSchema is the recursive view of a JSON Schema fragment used for both top-level variables and
// any nested object property / array items. Extras preserves JSON-Schema attributes the high-level
// CLI doesn't surface directly (enum, default, format, pattern, etc.) so set --file / patch
// shapes survive round-trips through add / update.
type subSchema struct {
	Type                 string                `json:"type"`
	Description          string                `json:"description,omitempty"`
	Properties           map[string]*subSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Items                *subSchema            `json:"items,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
	Extras               map[string]any        `json:"-"`
}

// variableRow is the table-printable view of one top-level variable.
type variableRow struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Shape       string `json:"shape,omitempty"`
}

func init() {
	utils.RegisterResource(variableRow{}, []string{"Name", "Type", "Required", "Shape", "Description"})
}

// MarshalJSON for subSchema merges the named fields with Extras so round-trips preserve any
// JSON-Schema attributes the CLI doesn't surface. Canonical fields always win over Extras with
// the same key (so we never silently re-emit a stale description, for example).
func (s *subSchema) MarshalJSON() ([]byte, error) {
	out := map[string]any{}
	if s.Type != "" {
		out["type"] = s.Type
	}
	if s.Description != "" {
		out["description"] = s.Description
	}
	if len(s.Properties) > 0 {
		out["properties"] = s.Properties
	}
	if len(s.Required) > 0 {
		out["required"] = s.Required
	}
	if s.Items != nil {
		out["items"] = s.Items
	}
	if s.AdditionalProperties != nil {
		out["additionalProperties"] = *s.AdditionalProperties
	}

	for k, v := range s.Extras {
		if _, taken := out[k]; taken {
			continue
		}
		out[k] = v
	}

	return json.Marshal(out)
}

func (s *subSchema) UnmarshalJSON(data []byte) error {
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if t, ok := raw["type"].(string); ok {
		s.Type = t
	}
	if d, ok := raw["description"].(string); ok {
		s.Description = d
	}

	// Nested properties / items / required need a second pass with strict types so we get
	// back fully-typed *subSchema trees rather than map[string]any blobs.
	if propsRaw, ok := raw["properties"]; ok {
		propBytes, err := json.Marshal(propsRaw)
		if err == nil {
			props := map[string]*subSchema{}
			if err := json.Unmarshal(propBytes, &props); err == nil {
				s.Properties = props
			}
		}
	}
	if itemsRaw, ok := raw["items"]; ok {
		itemBytes, err := json.Marshal(itemsRaw)
		if err == nil {
			items := &subSchema{}
			if err := json.Unmarshal(itemBytes, items); err == nil {
				s.Items = items
			}
		}
	}
	if reqRaw, ok := raw["required"]; ok {
		if reqArr, ok := reqRaw.([]any); ok {
			req := make([]string, 0, len(reqArr))
			for _, v := range reqArr {
				if str, ok := v.(string); ok {
					req = append(req, str)
				}
			}
			s.Required = req
		}
	}
	if apRaw, ok := raw["additionalProperties"]; ok {
		if b, ok := apRaw.(bool); ok {
			s.AdditionalProperties = &b
		}
	}

	for _, k := range []string{"type", "description", "properties", "items", "required", "additionalProperties"} {
		delete(raw, k)
	}
	if len(raw) > 0 {
		s.Extras = raw
	}

	return nil
}

// shape returns a one-line summary of the property's nested structure for the list view, e.g.
// "object{street, zip}" or "array<string>". Returns "" for primitives so the column stays empty.
func (s *subSchema) shape() string {
	switch s.Type {
	case "object":
		if len(s.Properties) == 0 {
			return "object (any)"
		}
		keys := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 4 {
			return fmt.Sprintf("object{%s, +%d}", strings.Join(keys[:3], ", "), len(keys)-3)
		}
		return fmt.Sprintf("object{%s}", strings.Join(keys, ", "))
	case "array":
		if s.Items == nil || s.Items.Type == "" {
			return "array (any)"
		}
		return fmt.Sprintf("array<%s>", s.Items.shape1())
	}
	return ""
}

// shape1 is a compact rendering of a child schema for use inside array<...> in the list view.
func (s *subSchema) shape1() string {
	if shape := s.shape(); shape != "" {
		return shape
	}
	return s.Type
}

// schemaPath returns the API path for the DynamicVariablesSchema endpoint of a project.
func schemaPath(projectId string) string {
	return "o/:organisation/projects/" + projectId + "/variables"
}

// fetchSchema GETs the project's DynamicVariablesSchema and returns it in the friendlier
// in-memory shape. Treats an empty / null / non-object schema as an empty schema so callers can
// always add to it without special-casing first-time setup.
func fetchSchema(cf *config.ConfigFactory, projectId string) *variablesSchema {
	raw := json.RawMessage{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&raw).
		WithPath(schemaPath(projectId)).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get DynamicVariablesSchema").WithReason(err).Done()
	}

	out := &variablesSchema{Properties: map[string]*subSchema{}}

	if len(raw) == 0 || string(raw) == "null" {
		return out
	}

	probe := struct {
		Properties map[string]*subSchema `json:"properties"`
		Required   []string              `json:"required"`
	}{}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return out
	}

	if probe.Properties != nil {
		out.Properties = probe.Properties
	}
	out.Required = probe.Required

	return out
}

// putSchema PUTs the friendlier in-memory schema back to the platform as a JSON-Schema object.
func putSchema(cf *config.ConfigFactory, projectId string, schema *variablesSchema) {
	body := map[string]any{
		"type":       "object",
		"properties": schema.Properties,
	}
	if len(schema.Required) > 0 {
		body["required"] = schema.Required
	}

	resp := json.RawMessage{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodPut).
		Into(&resp).
		WithPath(schemaPath(projectId)).
		JSONBody(body).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update DynamicVariablesSchema").WithReason(err).Done()
	}
}

// toRows flattens the top-level schema into sorted table rows.
func (s *variablesSchema) toRows() []variableRow {
	requiredSet := map[string]struct{}{}
	for _, name := range s.Required {
		requiredSet[name] = struct{}{}
	}

	rows := make([]variableRow, 0, len(s.Properties))
	for name, prop := range s.Properties {
		_, isRequired := requiredSet[name]
		rows = append(rows, variableRow{
			Name:        name,
			Type:        prop.Type,
			Required:    isRequired,
			Description: prop.Description,
			Shape:       prop.shape(),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return rows
}

// setRequired toggles whether `name` appears in the schema's top-level required[] array.
func (s *variablesSchema) setRequired(name string, required bool) {
	for i, n := range s.Required {
		if n == name {
			if !required {
				s.Required = append(s.Required[:i], s.Required[i+1:]...)
			}
			return
		}
	}
	if required {
		s.Required = append(s.Required, name)
		sort.Strings(s.Required)
	}
}

// validateType returns an error if `t` is not one of the supported JSON Schema types.
func validateType(t string) error {
	for _, v := range validVariableTypes {
		if t == v {
			return nil
		}
	}
	return fmt.Errorf("invalid type %q: must be one of %v", t, validVariableTypes)
}

// fieldSpec is a parsed --field flag value of the form `path:type[:required]`. Path uses `.` to
// separate levels and is interpreted relative to the variable's properties (for --type object) or
// items.properties (for --type array with --items-type object).
type fieldSpec struct {
	Path     []string
	Type     string
	Required bool
}

// parseFieldSpec parses one --field flag value. Spec format: `path:type[:required]`. Returns a
// descriptive error if any component is malformed so the user sees the bad flag value rather
// than a generic "invalid schema" later.
func parseFieldSpec(raw string) (fieldSpec, error) {
	parts := strings.Split(raw, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return fieldSpec{}, fmt.Errorf("--field %q: expected format <path>:<type>[:required]", raw)
	}

	pathStr := strings.TrimSpace(parts[0])
	if pathStr == "" {
		return fieldSpec{}, fmt.Errorf("--field %q: path must not be empty", raw)
	}
	pathSegs := strings.Split(pathStr, ".")
	for _, seg := range pathSegs {
		if strings.TrimSpace(seg) == "" {
			return fieldSpec{}, fmt.Errorf("--field %q: path segments must not be empty (got %q)", raw, pathStr)
		}
	}

	t := strings.TrimSpace(parts[1])
	if err := validateType(t); err != nil {
		return fieldSpec{}, fmt.Errorf("--field %q: %w", raw, err)
	}

	required := false
	if len(parts) == 3 {
		switch strings.ToLower(strings.TrimSpace(parts[2])) {
		case "required", "r", "yes", "y", "true":
			required = true
		case "":
			required = false
		default:
			return fieldSpec{}, fmt.Errorf("--field %q: third segment must be 'required' or empty, got %q", raw, parts[2])
		}
	}

	return fieldSpec{Path: pathSegs, Type: t, Required: required}, nil
}

// applyFieldSpec inserts the parsed field into the given root schema, auto-creating any missing
// parent object schemas along the path. The root is the object whose `properties` the path is
// relative to (variable itself for --type object, variable.Items for --type array of objects).
func applyFieldSpec(root *subSchema, spec fieldSpec) error {
	if root.Type != "object" {
		return fmt.Errorf("--field requires an object container (root type is %q)", root.Type)
	}
	if root.Properties == nil {
		root.Properties = map[string]*subSchema{}
	}

	current := root
	for i, seg := range spec.Path {
		isLeaf := i == len(spec.Path)-1
		if current.Properties == nil {
			current.Properties = map[string]*subSchema{}
		}

		child, exists := current.Properties[seg]

		if isLeaf {
			if exists {
				// Preserve any pre-existing nested structure / Extras on the child but update
				// its leaf type to whatever the user specified.
				child.Type = spec.Type
			} else {
				child = &subSchema{Type: spec.Type}
				current.Properties[seg] = child
			}

			if spec.Required {
				addToRequired(current, seg)
			} else {
				removeFromRequired(current, seg)
			}
			return nil
		}

		// Non-leaf: walk into the parent, auto-creating it as an object if absent. We do NOT
		// auto-mark interior segments as required; only the leaf carries the user-specified flag.
		if !exists {
			child = &subSchema{Type: "object", Properties: map[string]*subSchema{}}
			current.Properties[seg] = child
		}
		if child.Type != "object" {
			return fmt.Errorf("--field %q: path traverses non-object %q (currently %q) — declare the parent as object first", strings.Join(spec.Path, "."), seg, child.Type)
		}
		current = child
	}

	return nil
}

func addToRequired(s *subSchema, name string) {
	for _, existing := range s.Required {
		if existing == name {
			return
		}
	}
	s.Required = append(s.Required, name)
	sort.Strings(s.Required)
}

func removeFromRequired(s *subSchema, name string) {
	for i, existing := range s.Required {
		if existing == name {
			s.Required = append(s.Required[:i], s.Required[i+1:]...)
			return
		}
	}
}

// setStrict applies additionalProperties:false to every object node reachable via Properties /
// Items so the resulting schema rejects unknown sub-keys at activation time. Only called when the
// user passes --strict; default is to leave additionalProperties unset (≡ true).
func setStrict(s *subSchema) {
	if s == nil {
		return
	}
	if s.Type == "object" {
		f := false
		s.AdditionalProperties = &f
		for _, child := range s.Properties {
			setStrict(child)
		}
	}
	if s.Type == "array" && s.Items != nil {
		setStrict(s.Items)
	}
}

// resetForType clears any nested structure / Extras that don't make sense for the new top-level
// type. Called by `update` when the user explicitly changes --type so a string variable with an
// enum doesn't carry the enum forward after flipping to object. Description / Required are
// preserved because they're meaningful at any type.
func resetForType(s *subSchema, newType string) {
	s.Type = newType
	s.Properties = nil
	s.Required = nil
	s.Items = nil
	s.AdditionalProperties = nil
	s.Extras = nil
}
