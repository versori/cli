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

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type update struct {
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

func NewUpdate(c *config.ConfigFactory) *cobra.Command {
	u := &update{configFactory: c}

	cmd := &cobra.Command{
		Use:   "update --project <project-id> --name <key> [--type <type>] [--description <text>] [--required[=true|false]] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict]",
		Short: "Update fields on a declared dynamic variable (only changes flags you pass)",
		Long: `Update an existing dynamic-variable declaration on the project's DynamicVariablesSchema.
Only the flags you pass are changed — omitting a flag leaves that field untouched.

Behaviour notes:
  --description ""           Clears the description.
  --required=false           Removes the variable from the schema's required[] list.
  --type <new>               Changing the type clears any orphaned JSON-Schema attributes from
                             the previous type (so a string variable with enum doesn't carry
                             the enum forward after flipping to object). Description and the
                             top-level required flag are preserved.
  --field <path>:<type>      For --type object (existing or new), declares / updates a sub-field
                             at the given dotted path; parent objects auto-create. Pass repeatedly
                             to add multiple. Pre-existing sub-fields not mentioned by --field
                             are preserved (use 'remove' + 'add' for a clean reset).
  --items-type <type>        For --type array, replaces the array element type. If --type was
                             not array previously, you must also pass --type array.
  --strict                   Adds additionalProperties:false to every object node in the current
                             schema (cannot be unset by this command; use 'remove' + 'add' or
                             'set --file' to relax it).

Use 'add' for a brand-new variable, 'remove' to delete one, and 'set' to replace the whole schema
from a JSON file (advanced JSON-Schema shapes).`,
		Run: u.Run,
	}

	f := cmd.Flags()
	u.projectId.SetFlag(f)
	f.StringVarP(&u.name, "name", "n", "", "Variable name to update")
	f.StringVarP(&u.varType, "type", "t", "", "New JSON Schema type: string|number|integer|boolean|object|array")
	f.StringVarP(&u.description, "description", "d", "", "New human-readable description")
	f.BoolVarP(&u.required, "required", "r", false, "Whether the variable is required on every activation (pass --required=false to unset)")
	f.StringVar(&u.itemsType, "items-type", "", "Array element type (only valid when the variable is or becomes array)")
	f.StringSliceVar(&u.fields, "field", nil, "Sub-field declaration in the form <path>:<type>[:required] (repeatable; only valid for object or array-of-object variables)")
	f.BoolVar(&u.strict, "strict", false, "Set additionalProperties:false on every object node in the resulting schema")

	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func (u *update) Run(cmd *cobra.Command, _ []string) {
	projectId := u.projectId.GetFlagOrDie(".")

	schema := fetchSchema(u.configFactory, projectId)

	existing, found := schema.Properties[u.name]
	if !found || existing == nil {
		utils.NewExitError().
			WithMessage(fmt.Sprintf("variable %q is not declared on project %s — use 'versori projects variables add' to create it first", u.name, projectId)).
			Done()
	}

	typeChanged := cmd.Flags().Changed("type")
	descChanged := cmd.Flags().Changed("description")
	reqChanged := cmd.Flags().Changed("required")
	itemsChanged := cmd.Flags().Changed("items-type")
	fieldsChanged := cmd.Flags().Changed("field")

	if typeChanged {
		if err := validateType(u.varType); err != nil {
			utils.NewExitError().WithMessage(err.Error()).Done()
		}
		// Preserve the human-meaningful description across the type flip but reset all the
		// JSON-Schema-specific fields the previous type contributed so we don't carry orphans.
		prevDesc := existing.Description
		if u.varType != existing.Type {
			resetForType(existing, u.varType)
			existing.Description = prevDesc
		}
	}

	if descChanged {
		existing.Description = u.description
	}

	if itemsChanged {
		if existing.Type != "array" {
			utils.NewExitError().WithMessage("--items-type is only valid when the variable's --type is array").Done()
		}
		if err := validateType(u.itemsType); err != nil {
			utils.NewExitError().WithMessage(err.Error()).Done()
		}
		existing.Items = &subSchema{Type: u.itemsType}
	}

	if fieldsChanged {
		target, err := containerForFields(existing)
		if err != nil {
			utils.NewExitError().WithMessage(err.Error()).Done()
		}
		for _, raw := range u.fields {
			spec, err := parseFieldSpec(raw)
			if err != nil {
				utils.NewExitError().WithMessage(err.Error()).Done()
			}
			if err := applyFieldSpec(target, spec); err != nil {
				utils.NewExitError().WithMessage(err.Error()).Done()
			}
		}
	}

	if u.strict {
		setStrict(existing)
	}

	schema.Properties[u.name] = existing

	if reqChanged {
		schema.setRequired(u.name, u.required)
	}

	putSchema(u.configFactory, projectId, schema)

	fmt.Printf("Updated variable %q on project %s.\n", u.name, projectId)
}

// containerForFields returns the subSchema whose `properties` --field paths are relative to,
// auto-creating an items subSchema for array-of-object variables. Returns an error if the
// variable's effective type can't hold sub-fields.
func containerForFields(existing *subSchema) (*subSchema, error) {
	switch existing.Type {
	case "object":
		return existing, nil
	case "array":
		if existing.Items == nil {
			existing.Items = &subSchema{Type: "object"}
		}
		if existing.Items.Type != "object" {
			return nil, fmt.Errorf("--field requires --items-type object on array variables (current items type is %q)", existing.Items.Type)
		}
		return existing.Items, nil
	default:
		return nil, fmt.Errorf("--field is only valid for object or array-of-object variables (current type is %q)", existing.Type)
	}
}
