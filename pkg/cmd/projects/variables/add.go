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

package variables

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type add struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	name          string
	varType       string
	description   string
	required      bool
	itemsType     string
	fields        []string
	strict        bool
}

func NewAdd(c *config.ConfigFactory) *cobra.Command {
	a := &add{configFactory: c}

	cmd := &cobra.Command{
		Use:   "add --project <project-id> --name <key> [--type <type>] [--description <text>] [--required] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict]",
		Short: "Declare a dynamic variable on the project (primitive, object with fields, or array with item shape)",
		Long: `Add a dynamic-variable declaration to the project's DynamicVariablesSchema. Activations
on this project will then be allowed to set this key via 'versori projects users set-variable[s]'.

Modes:
  Interactive   Omit --name and the command prompts for each field (Name, Type, Description,
                Required, and recursively for nested fields when Type is object or array).
  Non-interactive  Pass --name and (optionally) the structural flags below. Useful in scripts and CI.

Top-level types: string, number, integer, boolean, object, array.

Nested structure for --type object (and --type array --items-type object):

  --field <path>:<type>[:required]      Repeatable. Path is dot-separated relative to the
                                        container's properties; missing parent objects are
                                        auto-created. Append ':required' to mark the leaf
                                        required on its immediate parent.
  --strict                              Set additionalProperties:false on every object node so
                                        unknown sub-keys are rejected at activation time
                                        (default: unknown sub-keys are accepted).

Array element type for --type array:

  --items-type <type>                   Element type (one of the top-level types). When
                                        --items-type object, --field paths describe the item
                                        object's properties. Omit to accept any element type.

Existing variables: this command updates type/description/required on a variable that already
exists with the same name. Any advanced JSON-Schema fields previously set via 'set --file' or
'patch' that aren't managed by this CLI (enum, default, format, pattern, etc.) are preserved on
the leaf. Use 'remove' + 'add' for a clean reset, or 'update' for piecemeal field-by-field edits.

Examples:

  versori projects variables add --name tenant_org_id --type string --required \
    --description "Tenant's Versori organisation ID"

  versori projects variables add --name addresses --type array --items-type object --strict \
    --field street:string:required \
    --field zip:string:required \
    --field country:string

  versori projects variables add --name feature_flags --type object --strict \
    --field enabled:boolean:required \
    --field metadata.version:string \
    --field metadata.notes:string`,
		Run: a.Run,
	}

	f := cmd.Flags()
	a.projectId.SetFlag(f)
	f.StringVarP(&a.name, "name", "n", "", "Variable name (the key activations set via set-variable)")
	f.StringVarP(&a.varType, "type", "t", "", "JSON Schema type: string|number|integer|boolean|object|array (default: string)")
	f.StringVarP(&a.description, "description", "d", "", "Human-readable description shown in the platform UI")
	f.BoolVarP(&a.required, "required", "r", false, "Mark this variable as required on every activation")
	f.StringVar(&a.itemsType, "items-type", "", "Array element type (only valid with --type array)")
	f.StringSliceVar(&a.fields, "field", nil, "Sub-field declaration in the form <path>:<type>[:required] (repeatable; only valid for object/array-of-object variables)")
	f.BoolVar(&a.strict, "strict", false, "Set additionalProperties:false on every object node (rejects unknown sub-keys)")

	return cmd
}

func (a *add) Run(_ *cobra.Command, _ []string) {
	projectId := a.projectId.GetFlagOrDie(".")

	interactive := a.name == ""

	if interactive {
		if err := elements.NewEditor("Name (the key activations will set):", false,
			elements.WithValidation(nonEmpty("Name"))).Edit(&a.name); err != nil {
			utils.NewExitError().WithMessage("failed to read name").WithReason(err).Done()
		}
	}

	if interactive && a.varType == "" {
		a.varType = "string"
		if err := selectType("Type:", &a.varType); err != nil {
			utils.NewExitError().WithMessage("failed to read type").WithReason(err).Done()
		}
	}

	if a.varType == "" {
		a.varType = "string"
	}
	if err := validateType(a.varType); err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	if interactive && a.description == "" {
		if err := elements.NewEditor("Description (optional, press enter to skip):", false).Edit(&a.description); err != nil {
			utils.NewExitError().WithMessage("failed to read description").WithReason(err).Done()
		}
	}

	// Resolve the existing schema entry up-front so we can preserve Extras / nested structure
	// the user didn't re-specify (matches the 'update fields piecemeal' semantics described in
	// --help). For a fresh variable this returns nil and we start from a zero-valued container.
	schema := fetchSchema(a.configFactory, projectId)
	existing := schema.Properties[a.name]
	if existing == nil {
		existing = &subSchema{}
	}
	existing.Type = a.varType
	existing.Description = a.description

	a.populateContainer(existing, interactive)

	if a.strict {
		setStrict(existing)
	}

	if interactive {
		if err := elements.NewConfirm(fmt.Sprintf("Mark %q as required on every activation?", a.name)).Confirm(&a.required); err != nil {
			utils.NewExitError().WithMessage("failed to read required flag").WithReason(err).Done()
		}
	}

	schema.Properties[a.name] = existing
	schema.setRequired(a.name, a.required)

	putSchema(a.configFactory, projectId, schema)

	fmt.Printf("Added variable %q (type=%s, required=%t) to project %s.\n", a.name, a.varType, a.required, projectId)
}

// populateContainer fills in the object/array nested shape on `existing` from the --field /
// --items-type flags (non-interactive) or by recursing through interactive prompts. For
// primitive types it's a no-op. Clears stale Properties / Items / Required when the user passed
// explicit structural flags so the new shape replaces the old one.
func (a *add) populateContainer(existing *subSchema, interactive bool) {
	switch a.varType {
	case "object":
		if interactive {
			interactiveBuildObject(existing, 1, fmt.Sprintf("(inside %q)", a.name))
			return
		}
		if a.itemsType != "" {
			utils.NewExitError().WithMessage("--items-type is only valid with --type array").Done()
		}
		if len(a.fields) > 0 {
			// Replace the property bag wholesale so the user's --field flags fully specify the
			// new shape; advanced Extras on the container itself are still preserved.
			existing.Properties = map[string]*subSchema{}
			existing.Required = nil
			a.applyFieldFlags(existing)
		}

	case "array":
		if len(a.fields) > 0 && a.itemsType != "object" {
			utils.NewExitError().WithMessage("--field requires --items-type object on array variables").Done()
		}
		if interactive {
			a.itemsType = "string"
			if err := selectType("Element type:", &a.itemsType); err != nil {
				utils.NewExitError().WithMessage("failed to read items type").WithReason(err).Done()
			}
			existing.Items = &subSchema{Type: a.itemsType}
			if a.itemsType == "object" {
				interactiveBuildObject(existing.Items, 1, fmt.Sprintf("(inside item of %q)", a.name))
			}
			return
		}
		if a.itemsType != "" {
			if err := validateType(a.itemsType); err != nil {
				utils.NewExitError().WithMessage(err.Error()).Done()
			}
			existing.Items = &subSchema{Type: a.itemsType}
		}
		if len(a.fields) > 0 {
			if existing.Items == nil {
				existing.Items = &subSchema{Type: "object"}
			}
			existing.Items.Properties = map[string]*subSchema{}
			existing.Items.Required = nil
			a.applyFieldFlagsTo(existing.Items)
		}

	default:
		if len(a.fields) > 0 || a.itemsType != "" {
			utils.NewExitError().WithMessage(fmt.Sprintf("--field / --items-type are not valid for --type %s (use object or array)", a.varType)).Done()
		}
	}
}

func (a *add) applyFieldFlags(root *subSchema) {
	a.applyFieldFlagsTo(root)
}

func (a *add) applyFieldFlagsTo(root *subSchema) {
	for _, raw := range a.fields {
		spec, err := parseFieldSpec(raw)
		if err != nil {
			utils.NewExitError().WithMessage(err.Error()).Done()
		}
		if err := applyFieldSpec(root, spec); err != nil {
			utils.NewExitError().WithMessage(err.Error()).Done()
		}
	}
}

// interactiveBuildObject loops prompting for sub-fields on `obj` until the user submits an empty
// name. Recurses into nested object / array-of-object children up to maxInteractiveNestingDepth.
// At max depth it tells the user to use --field with a dotted path and skips further recursion.
func interactiveBuildObject(obj *subSchema, depth int, context string) {
	if obj.Properties == nil {
		obj.Properties = map[string]*subSchema{}
	}

	indent := strings.Repeat("  ", depth-1)

	for {
		var fieldName string
		title := fmt.Sprintf("%sField name %s (empty to finish):", indent, context)
		if err := elements.NewEditor(title, false).Edit(&fieldName); err != nil {
			utils.NewExitError().WithMessage("failed to read sub-field name").WithReason(err).Done()
		}
		fieldName = strings.TrimSpace(fieldName)
		if fieldName == "" {
			return
		}

		fieldType := "string"
		if err := selectType(fmt.Sprintf("%s  Type of %q:", indent, fieldName), &fieldType); err != nil {
			utils.NewExitError().WithMessage("failed to read sub-field type").WithReason(err).Done()
		}

		fieldRequired := false
		if err := elements.NewConfirm(fmt.Sprintf("%s  Mark %q as required?", indent, fieldName)).Confirm(&fieldRequired); err != nil {
			utils.NewExitError().WithMessage("failed to read sub-field required flag").WithReason(err).Done()
		}

		child := &subSchema{Type: fieldType}
		obj.Properties[fieldName] = child
		if fieldRequired {
			addToRequired(obj, fieldName)
		}

		if containerTypes[fieldType] {
			if depth >= maxInteractiveNestingDepth {
				fmt.Printf("%s  (max interactive nesting depth %d reached; use --field %s.<deeper>:<type> in non-interactive mode for deeper nesting)\n",
					indent, maxInteractiveNestingDepth, fieldName)
				continue
			}
			switch fieldType {
			case "object":
				interactiveBuildObject(child, depth+1, fmt.Sprintf("(inside %q)", fieldName))
			case "array":
				itemsType := "string"
				if err := selectType(fmt.Sprintf("%s    Element type of %q:", indent, fieldName), &itemsType); err != nil {
					utils.NewExitError().WithMessage("failed to read items type").WithReason(err).Done()
				}
				child.Items = &subSchema{Type: itemsType}
				if itemsType == "object" {
					interactiveBuildObject(child.Items, depth+1, fmt.Sprintf("(inside item of %q)", fieldName))
				}
			}
		}
	}
}

func selectType(title string, value *string) error {
	sel := elements.NewListSelect(title)
	for _, t := range validVariableTypes {
		sel.AddOption(t, t)
	}
	return sel.Select(value)
}

func nonEmpty(label string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s must not be empty", label)
		}
		return nil
	}
}
