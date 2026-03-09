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
)

const apiBaseURL = "https://platform.versori.com/api/v2/"
const aiBaseURL = "https://platform.versori.com/api/sparkboard/v1alpha1/"

var (
	defaultConfigPath  = "~/.versori/config.yaml"
	configEnvOverwrite = "VERSORI_CONFIG"

	// CurrentContext is the context that is currently active.
	// Users of this should make sure to check if the value is nil before using it
	CurrentContext *Context
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
}

func (c *ConfigFactory) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.configPath, "config", "c", defaultConfigPath, "Config file (default is $HOME/.versori/config.yaml)")
	flags.StringVarP(&c.contextName, "context", "x", "", "Load a specific context. If not set will use the default active one.")
	flags.StringVarP(&c.outputFormat, "output", "o", "table", "Output format (yaml, json, table)")
}

func (c *ConfigFactory) LoadConfigAndContext() {
	c.loadConfig()
	c.loadContext()
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

// contextOrDie loads the context from the config or exits with an error message
func (c *ConfigFactory) contextOrDie() {
	ctxName := ""

	switch {
	case c.contextName != "":
		ctxName = c.contextName
	case c.Config.ActiveContext != "":
		ctxName = c.Config.ActiveContext
	}

	if ctxName == "" {
		fmt.Fprintln(os.Stderr, "You have no active context\nTry running\n\tversori context set")
		os.Exit(1)
	}

	ctx, ok := c.Config.Contexts[ctxName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Context '%s' does not exist\nTry running\n\tversori context add\n", c.contextName)
		os.Exit(1)
	}

	c.Context = &ctx
	CurrentContext = c.Context
}

// loadContext loads the context from the config or exits with an error message
func (c *ConfigFactory) loadContext() {
	c.contextOrDie()

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
