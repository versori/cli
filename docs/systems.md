---
title: "systems"
description: "Manage systems"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `add-auth-scheme` | Add or update an auth scheme config for a system |
| `create` | Create a new system in the current organisation |
| `list` | Lists systems for the current organisation context |

---

### `versori systems add-auth-scheme`


Adds or updates an auth scheme config for the given system.
If the auth scheme name already exists on the system it will be updated, otherwise it will be created.

Supported types: none, api-key, basic-auth, oauth2, hmac, certificate

```sh
versori systems add-auth-scheme --system-id <system-id> --type <type> --name <name> [flags]
```


**Flags:**
* `--api-key.in`: Where to send the API key: header, query or cookie (required when type=api-key)

* `--api-key.name`: Header/query/cookie name for API key (required when type=api-key)

* `--description`: Human-readable description for this auth scheme config
* `-h`, `--help`: help for add-auth-scheme
* `--hmac.algorithm`: HMAC algorithm: sha1, sha256, sha512 (required when type=hmac)

* `--hmac.digest-input`: HMAC digest input: body, url (repeatable)
* `--hmac.encoding`: HMAC encoding: hex, base64, base64url
* `--hmac.in`: Where to send the HMAC signature: header, query or cookie (required when type=hmac)

* `--hmac.name`: Header/query/cookie name for HMAC signature (required when type=hmac)

* `--name`: Name of the auth scheme config
* `--oauth2.authorize-url`: OAuth2 authorization URL
* `--oauth2.client-id`: OAuth2 client ID
* `--oauth2.client-secret`: OAuth2 client secret
* `--oauth2.default-scope`: OAuth2 default scope in name=description format (repeatable)
* `--oauth2.grant-type`: OAuth2 grant type: authorizationCode, clientCredentials, password

* `--oauth2.scope`: OAuth2 scope in name=description format, e.g. read=Read access (repeatable)

* `--oauth2.token-url`: OAuth2 token URL (required when type=oauth2)
* `--system-id`: ID of the system to add the auth scheme to (required)
* `--type`: Auth scheme type (none, api-key, basic-auth, oauth2, hmac, certificate)




---

### `versori systems create`




```sh
versori systems create --name <name> --domain <domain> --template-base-url <url> [flags]
```


**Flags:**
* `--domain`: Domain of the system
* `-h`, `--help`: help for create
* `--name`: Name of the system
* `--template-base-url`: Template base URL for the system



---

### `versori systems list`




```sh
versori systems list [flags]
```


**Flags:**
* `-h`, `--help`: help for list



---
