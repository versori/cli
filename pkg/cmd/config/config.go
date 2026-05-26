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

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/versori/cli/pkg/utils"
	"github.com/versori/cli/pkg/versorifile"
)

const apiBaseURL = "https://platform.versori.com/api/v2/"
const aiBaseURL = "https://platform.versori.com/api/sparkboard/v1alpha1/"

var (
	defaultConfigPath  = "~/.versori/config.yaml"
	configEnvOverwrite = "VERSORI_CONFIG"

	// CurrentContext is the context that is currently active.
	// Users of this should make sure to check if the value is nil before using it
	CurrentContext *Context

	// defaultFactory is set by LoadConfigAndContext to the most recently
	// initialised factory. It exists so flags.ProjectId (and other helpers
	// that don't carry a ConfigFactory pointer) can lazily request the
	// .versori context override after they've resolved a project ID.
	defaultFactory *ConfigFactory
)

func init() {
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		defaultConfigPath = filepath.Join(homeDir, ".versori", "config.yaml")
	}

	if envPath := os.Getenv(configEnvOverwrite); envPath != "" {
		defaultConfigPath = envPath
	}
}

type Config struct {
	ActiveContext string             `yaml:"active_context"`
	Contexts      map[string]Context `yaml:"contexts"`
}

func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

type ConfigFactory struct {
	Context       *Context
	versoriClient *utils.HTTPBuilder
	Config        *Config

	// global flags
	configPath   string
	outputFormat string
	contextName  string

	// versoriFile is the parsed .versori (if any) found in cwd at the
	// time LoadConfigAndContext ran. It pins the project-local context
	// so we don't have to re-read the file on every NewRequest.
	versoriFile *versorifile.VersoriFile
	// versoriFileDir is the cwd from which versoriFile was read. Used
	// only to print human-meaningful warnings.
	versoriFileDir string

	// overrideContextName, when set, replaces the resolved context name.
	// MaybeApplyVersoriContextForProject sets this once a subcommand has
	// confirmed it's operating on the same project the .versori file was
	// synced for; from then on every loadContext picks the override and
	// every NewRequest hits the correct org.
	overrideContextName string
	// versoriNoticeShown ensures the "switched to .versori context" notice
	// is printed at most once per process even though loadContext can be
	// invoked many times via NewRequest.
	versoriNoticeShown bool
}

func (c *ConfigFactory) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.configPath, "config", "c", defaultConfigPath, "Config file (default is $HOME/.versori/config.yaml)")
	flags.StringVarP(&c.contextName, "context", "x", "", "Load a specific context. If not set will use the default active one.")
	flags.StringVarP(&c.outputFormat, "output", "o", "table", "Output format (yaml, json, table)")
}

func (c *ConfigFactory) LoadConfigAndContext() {
	c.loadConfig()
	c.loadVersoriFile()
	c.loadContext()

	// Register as the process-wide default so helpers without a factory
	// pointer (notably flags.ProjectId) can request a .versori context
	// override later in the command's lifecycle.
	defaultFactory = c
}

// loadVersoriFile reads .versori from the current working directory once and
// caches the result on the factory. Errors other than "not found" abort the
// command — a malformed .versori in cwd is the user's bug, not ours.
func (c *ConfigFactory) loadVersoriFile() {
	cwd, err := os.Getwd()
	if err != nil {
		// Without cwd we can't look for .versori, but the rest of the CLI
		// can still function with the persistent active context, so degrade
		// silently rather than abort.
		return
	}

	v, err := versorifile.FromDir(cwd)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read .versori in current directory").WithReason(err).Done()
	}

	c.versoriFile = v
	c.versoriFileDir = cwd
}

// loadConfig makes sure the config file exists
func (c *ConfigFactory) loadConfig() {
	if c.configPath == "" {
		fmt.Fprintln(os.Stderr, "config path is empty")
		os.Exit(1)
	}

	var err error
	c.Config, err = readOrCreateConfig(c.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to load config: %v", err))
		os.Exit(1)
	}
}

// contextOrDie picks the context for this invocation and stores it on the
// factory, or exits with a helpful error message.
//
// Precedence (highest first):
//  1. --context / -x flag — explicit, always wins.
//  2. overrideContextName — set lazily by MaybeApplyVersoriContextForProject
//     when a subcommand has resolved the same project ID the .versori was
//     synced for.
//  3. active_context from ~/.versori/config.yaml — the user's persistent
//     default.
func (c *ConfigFactory) contextOrDie() {
	ctxName, source := c.resolveContextName()

	if ctxName == "" {
		fmt.Fprintln(os.Stderr, "You have no active context\nTry running\n\tversori context set")
		os.Exit(1)
	}

	ctx, ok := c.Config.Contexts[ctxName]
	if !ok {
		switch source {
		case contextSourceVersoriFile:
			fmt.Fprintf(os.Stderr,
				"Context %q referenced by %s/.versori is not configured.\n"+
					"Either add it with\n\tversori context add --name %s ...\n"+
					"or pass --context to use a different one.\n",
				ctxName, c.versoriFileDir, ctxName)
		default:
			fmt.Fprintf(os.Stderr, "Context %q does not exist\nTry running\n\tversori context add\n", ctxName)
		}

		os.Exit(1)
	}

	c.Context = &ctx
	CurrentContext = c.Context
}

// contextSource records how contextOrDie picked the context, so error
// messages can be specific about who supplied the name.
type contextSource int

const (
	contextSourceNone contextSource = iota
	contextSourceFlag
	contextSourceVersoriFile
	contextSourceActive
)

func (c *ConfigFactory) resolveContextName() (string, contextSource) {
	if c.contextName != "" {
		return c.contextName, contextSourceFlag
	}

	if c.overrideContextName != "" {
		return c.overrideContextName, contextSourceVersoriFile
	}

	if c.Config.ActiveContext != "" {
		return c.Config.ActiveContext, contextSourceActive
	}

	return "", contextSourceNone
}

// MaybeApplyVersoriContextForProject opts the active factory into the
// .versori-bound context iff:
//
//  1. No --context flag was passed (explicit user choice always wins).
//  2. A .versori file is present in the directory the subcommand resolved
//     against (typically cwd or the value of --directory).
//  3. The resolved project ID matches .versori's project_id — i.e. the
//     command is genuinely operating on the project the .versori was
//     synced for. A `--project <other-id>` deliberately bypasses this.
//
// When the override fires AND it differs from the persistent active
// context, a one-line notice is written to stderr (once per process).
//
// No-op when there is no active factory (e.g. unit tests that don't go
// through LoadConfigAndContext).
func MaybeApplyVersoriContextForProject(dir, projectId string) {
	if defaultFactory == nil {
		return
	}

	defaultFactory.maybeApplyVersoriContextForProject(dir, projectId)
}

func (c *ConfigFactory) maybeApplyVersoriContextForProject(dir, projectId string) {
	if c.contextName != "" {
		return
	}

	if projectId == "" {
		return
	}

	// Re-read .versori from the dir the caller resolved against, falling
	// back to the eagerly cached one for cwd. This handles --directory
	// cases where the project being acted on lives somewhere other than
	// cwd: the override should follow the *project's* .versori, not the
	// shell's cwd.
	v, sourceDir := c.versoriFileFor(dir)
	if v == nil || v.Context == "" || v.ProjectId == "" {
		return
	}

	if v.ProjectId != projectId {
		return
	}

	c.overrideContextName = v.Context

	// One-shot notice when the override actually changes which context
	// commands will use. Stays quiet when .versori already matches the
	// persistent active context.
	if !c.versoriNoticeShown && c.Config != nil && c.Config.ActiveContext != "" && c.Config.ActiveContext != v.Context {
		fmt.Fprintf(os.Stderr,
			"using context %q from %s/.versori (matches --project %s; pass --context to override)\n",
			v.Context, sourceDir, projectId)
	}

	c.versoriNoticeShown = true
}

// versoriFileFor returns the .versori file relevant to a subcommand operating
// on `dir`, preferring the one inside `dir` when it exists and falling back
// to the cached cwd .versori otherwise. The second return value is the
// directory the file was actually read from, for use in notice messages.
func (c *ConfigFactory) versoriFileFor(dir string) (*versorifile.VersoriFile, string) {
	if dir != "" {
		absDir, err := filepath.Abs(dir)
		if err == nil {
			if absDir == c.versoriFileDir {
				return c.versoriFile, c.versoriFileDir
			}

			if v, readErr := versorifile.FromDir(absDir); readErr == nil && v != nil {
				return v, absDir
			}
		}
	}

	return c.versoriFile, c.versoriFileDir
}

// loadContext loads the context from the config or exits with an error message
func (c *ConfigFactory) loadContext() {
	c.contextOrDie()

	// this is used to help us test the CLI on local dev and stage
	if c.Context.URLOverwrite != "" {
		c.versoriClient = utils.NewHTTPBuilder(c.Context.URLOverwrite)

		return
	}

	c.versoriClient = utils.NewHTTPBuilder(apiBaseURL)
}

func (c *ConfigFactory) AddContext(ctx Context) {
	c.loadConfig()

	_, ok := c.Config.Contexts[ctx.Name]
	if ok {
		utils.NewExitError().WithMessage(fmt.Sprintf("Context '%s' already exists", ctx.Name)).Done()
	}

	c.Config.Contexts[ctx.Name] = ctx
	c.Config.ActiveContext = ctx.Name

	c.saveConfig()
}

func (c *ConfigFactory) RemoveContext(ctxName string) {
	_, ok := c.Config.Contexts[ctxName]
	if !ok {
		utils.NewExitError().WithMessage(fmt.Sprintf("Context '%s' does not exist", ctxName)).Done()
	}

	delete(c.Config.Contexts, ctxName)

	c.saveConfig()
}

func (c *ConfigFactory) SetActiveContext(ctxName string) {
	_, ok := c.Config.Contexts[ctxName]
	if !ok {
		utils.NewExitError().WithMessage(fmt.Sprintf("Context '%s' does not exist", ctxName)).Done()
	}

	c.Config.ActiveContext = ctxName

	c.saveConfig()
}

func (c *ConfigFactory) saveConfig() {
	b, err := yaml.Marshal(c.Config)
	if err != nil {
		utils.NewExitError().WithMessage("failed to marshal CLI config").WithReason(err).Done()
	}

	err = os.WriteFile(c.configPath, b, 0o600)
	if err != nil {
		utils.NewExitError().WithMessage("unable to persist CLI config").WithReason(err).Done()
	}
}

func (c *ConfigFactory) NewRequest() *utils.HTTPRequest {
	c.loadContext()

	return c.versoriClient.
		New().
		WithJWT(c.Context.JWT).
		WithOrganisation(c.Context.OrganisationId)
}

func (c *ConfigFactory) NewAIRequest() *utils.HTTPRequest {
	c.loadContext()

	return c.versoriClient.
		WithURL(aiBaseURL).
		New().
		WithJWT(c.Context.JWT).
		WithOrganisation(c.Context.OrganisationId)
}

func (c *ConfigFactory) Print(resource any) {
	utils.Print(resource, c.outputFormat)
}

func readOrCreateConfig(configPath string) (*Config, error) {
	config := Config{
		Contexts: make(map[string]Context),
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			createErr := os.MkdirAll(filepath.Dir(configPath), 0755)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create config directory: %v", createErr)
			}

			createErr = os.WriteFile(configPath, []byte{}, 0o600)
			if createErr != nil {
				return nil, fmt.Errorf("failed to create config file: %v", createErr)
			}

			return &config, nil
		}

		return nil, err
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
