---
title: "projects variables"
description: "Manage a project's DynamicVariablesSchema (declares valid activation-variable keys)"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `add` | Declare a dynamic variable on the project (primitive, object with fields, or array with item shape) |
| `get` | Get the project's DynamicVariablesSchema (the JSON schema defining valid activation-variable keys) |
| `list` | List the project's declared dynamic variables (name / type / required / description) |
| `remove` | Remove a single dynamic variable from the project's DynamicVariablesSchema |
| `set` | Replace the project's DynamicVariablesSchema in full (low-level escape hatch) |
| `update` | Update fields on a declared dynamic variable (only changes flags you pass) |

---

### `versori projects variables add`


Add a dynamic-variable declaration to the project's DynamicVariablesSchema. Activations
on this project will then be allowed to set this key via 'versori projects users set-variable[s]'.

Modes:
  Interactive   Omit --name and the command prompts for each field (Name, Type, Description,
                Required, and recursively for nested fields when Type is object or array).
  Non-interactive  Pass --name and (optionally) the structural flags below. Useful in scripts and CI.

Top-level types: string, number, integer, boolean, object, array.

Nested structure for --type object (and --type array --items-type object):

  --field \<path\>:\<type\>[:required]      Repeatable. Path is dot-separated relative to the
                                        container's properties; missing parent objects are
                                        auto-created. Append ':required' to mark the leaf
                                        required on its immediate parent.
  --strict                              Set additionalProperties:false on every object node so
                                        unknown sub-keys are rejected at activation time
                                        (default: unknown sub-keys are accepted).

Array element type for --type array:

  --items-type \<type\>                   Element type (one of the top-level types). When
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
    --field metadata.notes:string

```sh
versori projects variables add --project <project-id> --name <key> [--type <type>] [--description <text>] [--required] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict] [flags]
```


**Flags:**
* `-d`, `--description`: Human-readable description shown in the platform UI
* `--field`: Sub-field declaration in the form \<path\>:\<type\>[:required] (repeatable; only valid for object/array-of-object variables)

* `-h`, `--help`: help for add
* `--items-type`: Array element type (only valid with --type array)
* `-n`, `--name`: Variable name (the key activations set via set-variable)
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-r`, `--required`: Mark this variable as required on every activation
* `--strict`: Set additionalProperties:false on every object node (rejects unknown sub-keys)

* `-t`, `--type`: JSON Schema type: string|number|integer|boolean|object|array (default: string)




---

### `versori projects variables get`


Fetch the JSON schema that defines which dynamic-variable keys end-user activations
on this project may set. Activations whose variables don't match this schema are rejected at
creation time. Use this to discover which keys exist before running 'versori projects users
activate' or 'versori projects users set-variable[s]'.

```sh
versori projects variables get --project <project-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for get
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects variables list`


List the dynamic-variable keys declared on this project's DynamicVariablesSchema in
a friendly table. Activation variables set with 'versori projects users set-variable[s]' must
appear in this list — unknown keys are rejected by the platform at activation time. Use 'versori
projects variables add' to declare a new variable, or 'versori projects variables get' to dump
the raw JSON schema (escape hatch for advanced shapes like enum/default/nested object).

```sh
versori projects variables list --project <project-id> [flags]
```


**Flags:**
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects variables remove`


Remove a dynamic-variable declaration (and its entry in the required[] list) from the
project's DynamicVariablesSchema. Activations that previously set this key keep the value on
their record but workflow code that reads it via ctx.activation.getVariable() will continue to
return the stored value. Confirms before deleting unless --yes is passed.

```sh
versori projects variables remove --project <project-id> --name <key> [--yes] [flags]
```


**Flags:**
* `-h`, `--help`: help for remove
* `-n`, `--name`: Variable name to remove
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-y`, `--yes`: Skip the confirmation prompt



---

### `versori projects variables set`


Replace the entire JSON schema defining the valid activation-variable keys for this project.
This is a full PUT — any keys not present in the new schema will be removed.

Most users should prefer the high-level commands ('add', 'update', 'remove', 'list'), which work
in terms of variable Name / Type / Description / Required without requiring you to write raw JSON
Schema. Use 'set' only when you need advanced JSON Schema features (enum, default, nested object
shapes, patternProperties, etc.) that the high-level commands don't expose, or to bulk-import a
schema from a file in CI.

Example schema declaring two string variables:

```json
{
  "type": "object",
  "properties": {
    "tenant_org_id": { "type": "string" },
    "channel_id":    { "type": "string" }
  }
}
```

After updating the schema, end-user activations can set those keys via 'versori projects users
activate --variable key=value' or 'versori projects users set-variable[s]'.

```sh
versori projects variables set --project <project-id> --file <schema.json> [flags]
```


**Flags:**
* `-f`, `--file`: Path to a JSON file containing the full DynamicVariablesSchema

* `-h`, `--help`: help for set
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects variables update`


Update an existing dynamic-variable declaration on the project's DynamicVariablesSchema.
Only the flags you pass are changed — omitting a flag leaves that field untouched.

Behaviour notes:
  --description ""           Clears the description.
  --required=false           Removes the variable from the schema's required[] list.
  --type \<new\>               Changing the type clears any orphaned JSON-Schema attributes from
                             the previous type (so a string variable with enum doesn't carry
                             the enum forward after flipping to object). Description and the
                             top-level required flag are preserved.
  --field \<path\>:\<type\>      For --type object (existing or new), declares / updates a sub-field
                             at the given dotted path; parent objects auto-create. Pass repeatedly
                             to add multiple. Pre-existing sub-fields not mentioned by --field
                             are preserved (use 'remove' + 'add' for a clean reset).
  --items-type \<type\>        For --type array, replaces the array element type. If --type was
                             not array previously, you must also pass --type array.
  --strict                   Adds additionalProperties:false to every object node in the current
                             schema (cannot be unset by this command; use 'remove' + 'add' or
                             'set --file' to relax it).

Use 'add' for a brand-new variable, 'remove' to delete one, and 'set' to replace the whole schema
from a JSON file (advanced JSON-Schema shapes).

```sh
versori projects variables update --project <project-id> --name <key> [--type <type>] [--description <text>] [--required[=true|false]] [--items-type <type>] [--field <path>:<type>[:required]]... [--strict] [flags]
```


**Flags:**
* `-d`, `--description`: New human-readable description
* `--field`: Sub-field declaration in the form \<path\>:\<type\>[:required] (repeatable; only valid for object or array-of-object variables)

* `-h`, `--help`: help for update
* `--items-type`: Array element type (only valid when the variable is or becomes array)

* `-n`, `--name`: Variable name to update
* `--project`: Project ID; defaults from .versori when inside a synced project directory.

* `-r`, `--required`: Whether the variable is required on every activation (pass --required=false to unset)

* `--strict`: Set additionalProperties:false on every object node in the resulting schema

* `-t`, `--type`: New JSON Schema type: string|number|integer|boolean|object|array




---
