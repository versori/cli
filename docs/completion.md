---
title: "completion"
description: "Generate the autocompletion script for the specified shell"
---

Generate the autocompletion script for versori for the specified shell.
See each sub-command's help for details on how to use the generated script.


## Subcommands

| Subcommand | Description |
|---|---|
| `bash` | Generate the autocompletion script for bash |
| `fish` | Generate the autocompletion script for fish |
| `powershell` | Generate the autocompletion script for powershell |
| `zsh` | Generate the autocompletion script for zsh |

---

### `versori completion bash`


Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(versori completion bash)

To load completions for every new session, execute once:

#### Linux:

	versori completion bash > /etc/bash_completion.d/versori

#### macOS:

	versori completion bash > $(brew --prefix)/etc/bash_completion.d/versori

You will need to start a new shell for this setup to take effect.


```sh
versori completion bash
```


**Flags:**
* `-h`, `--help`: help for bash
* `--no-descriptions`: disable completion descriptions



---

### `versori completion fish`


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	versori completion fish | source

To load completions for every new session, execute once:

	versori completion fish > ~/.config/fish/completions/versori.fish

You will need to start a new shell for this setup to take effect.


```sh
versori completion fish [flags]
```


**Flags:**
* `-h`, `--help`: help for fish
* `--no-descriptions`: disable completion descriptions



---

### `versori completion powershell`


Generate the autocompletion script for powershell.

To load completions in your current shell session:

	versori completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```sh
versori completion powershell [flags]
```


**Flags:**
* `-h`, `--help`: help for powershell
* `--no-descriptions`: disable completion descriptions



---

### `versori completion zsh`


Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(versori completion zsh)

To load completions for every new session, execute once:

#### Linux:

	versori completion zsh > "${fpath[1]}/_versori"

#### macOS:

	versori completion zsh > $(brew --prefix)/share/zsh/site-functions/_versori

You will need to start a new shell for this setup to take effect.


```sh
versori completion zsh [flags]
```


**Flags:**
* `-h`, `--help`: help for zsh
* `--no-descriptions`: disable completion descriptions



---
