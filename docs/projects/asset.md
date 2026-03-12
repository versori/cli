---
title: "projects asset"
description: "Manage assets within a project"
---



## Subcommands

| Subcommand | Description |
|---|---|
| `download` | Download an asset from the Versori platform |
| `list` | List assets for a project |
| `upload` | Upload an asset file to the Versori platform |

---

### `versori projects asset download`


Download retrieves an asset by name from the Versori platform and saves it to the specified directory.


```sh
versori projects asset download --project <id> --asset <name> [flags]
```


**Flags:**
* `-a`, `--asset`: Name of the asset to download (required)
* `-d`, `--directory`: Directory to save the downloaded asset
* `-h`, `--help`: help for download
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects asset list`


List retrieves all assets for a project from the Versori platform.


```sh
versori projects asset list [flags]
```


**Flags:**
* `-h`, `--help`: help for list
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---

### `versori projects asset upload`


Upload uploads a file as an asset to the Versori platform. Assets can be used as context by Versori AI agents.


```sh
versori projects asset upload --file <path> [flags]
```


**Flags:**
* `-f`, `--file`: Path to the file to upload (required)
* `--folder`: Destination folder for the uploaded asset (attachments/raw, research/documents, research/workflows)

* `-h`, `--help`: help for upload
* `--project`: Project ID; defaults from .versori when inside a synced project directory.




---
