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

// Package kv implements the `versori kv` command group: a read tier (stores list, list, count,
// get) for inspecting Versori KV store entries and a guarded mutation tier (set, delete, wipe).
//
// Store + prefix targeting works in two modes, controlled by kvScope:
//   - raw mode (--store / --prefix / --key): values map straight onto the KV HTTP API.
//   - friendly mode (--scope + --project/--environment/...): the CLI derives the store name and
//     key prefix exactly the way the runtime SDK does (see SDKKeyValueProvider in @versori/run),
//     so a human can address the same keys a workflow wrote without copying ULIDs by hand.
//
// Value encoding mirrors the SDK: the SDK JSON.stringify()s on write and JSON.parse()s on read,
// so workflow-written values are stored as JSON-encoded strings. `set` matches that encoding and
// the read commands unwrap one level by default (toggle with --raw-values).
package kv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

// tableValueWidth caps the value column in table output so a single large blob doesn't wrap the
// whole table. Inspecting full values is what -o json / -o yaml is for.
const tableValueWidth = 60

func init() {
	utils.RegisterResource(printableEntry{}, []string{"Key", "Value", "Versionstamp", "Created", "Expires"})
	utils.RegisterResource(printableStore{}, []string{"Name", "ID"})
	utils.RegisterResource(printableCount{}, []string{"Count"})
}

// kvScope resolves which store and key prefix a command targets. It is embedded by every kv
// subcommand and registers the shared flags via addFlags.
type kvScope struct {
	// raw mode
	store string

	// friendly mode
	project      flags.ProjectId
	environment  string
	scope        string
	externalID   string
	executionID  string
	activationID string
	fullKeys     bool
}

func (s *kvScope) addFlags(f *pflag.FlagSet) {
	f.StringVar(&s.store, "store", "", "KV store ID (raw mode). Mutually exclusive with --scope.")
	s.project.SetFlag(f)
	f.StringVar(&s.environment, "environment", "", "Environment name, e.g. production (friendly mode).")
	f.StringVar(&s.scope, "scope", "", "Friendly scope: organization | workspace | project | user | execution.")
	f.StringVar(&s.externalID, "external-id", "", "End-user external ID (required for --scope user).")
	f.StringVar(&s.executionID, "execution-id", "", "Execution ID (required for --scope execution).")
	f.StringVar(&s.activationID, "activation-id", "", "Activation ID (optional; narrows project/execution scope).")
	f.BoolVar(&s.fullKeys, "full-keys", false, "Show full resolved keys instead of stripping the scope prefix.")
}

// resolveStore returns the target store ID and, in friendly mode, the resolved scope prefix that
// every key under this scope shares. friendly reports whether scope-based resolution was used so
// callers know whether to strip the prefix for display.
func (s *kvScope) resolveStore(c *config.ConfigFactory) (storeID string, scopePrefix []string, friendly bool) {
	if s.store != "" && s.scope != "" {
		utils.NewExitError().WithMessage("pass either --store (raw) or --scope (friendly), not both").Done()
	}

	if s.store != "" {
		return s.store, nil, false
	}

	if s.scope == "" {
		utils.NewExitError().
			WithMessage("provide --store <id> (raw mode) or --scope <organization|workspace|project|user|execution> (friendly mode)").
			Done()
	}

	// organization scope needs only the org; everything else resolves a project (which may switch
	// the active context to the one pinned by .versori) and requires --environment.
	var org, project, env string
	if normalizeScope(s.scope) == "organization" {
		org = mustOrg(c)
	} else {
		org, project, env = s.mustProjectEnv(c)
	}

	res, err := resolveScopePrefix(s.scope, org, project, env, s.externalID, s.executionID, s.activationID)
	if err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	return resolveStoreByName(c, res.storeName), res.prefix, true
}

// scopeResolution is the pure result of mapping a friendly scope onto the runtime's store name and
// key prefix.
type scopeResolution struct {
	storeName string
	prefix    []string
}

// resolveScopePrefix maps a friendly scope + identifiers onto the store name and key prefix exactly
// as @versori/run's SDKKeyValueProvider does (store names ORG_/PROJECT_/EXECUTION_, subjectPrefix
// segment order, sha256 fingerprint for :user:). Pure and unit-tested; callers supply org/project/
// env after resolving them from flags/context.
func resolveScopePrefix(scope, org, project, env, externalID, executionID, activationID string) (scopeResolution, error) {
	switch normalizeScope(scope) {
	case "organization":
		return scopeResolution{storeName: "ORG_" + org, prefix: []string{org}}, nil
	case "workspace":
		return scopeResolution{storeName: "PROJECT_" + project, prefix: []string{org, project, env}}, nil
	case "project":
		prefix := []string{org, project, env}
		if activationID != "" {
			prefix = append(prefix, activationID)
		}

		return scopeResolution{storeName: "PROJECT_" + project, prefix: prefix}, nil
	case "user":
		if externalID == "" {
			return scopeResolution{}, errors.New("--external-id is required for --scope user")
		}

		return scopeResolution{
			storeName: "PROJECT_" + project,
			prefix:    []string{org, project, env, fingerprintExternalUserID(externalID)},
		}, nil
	case "execution":
		if executionID == "" {
			return scopeResolution{}, errors.New("--execution-id is required for --scope execution")
		}
		prefix := []string{org, project, env, executionID}
		if activationID != "" {
			prefix = append(prefix, activationID)
		}

		return scopeResolution{storeName: "EXECUTION_" + project, prefix: prefix}, nil
	default:
		return scopeResolution{}, fmt.Errorf("unknown --scope %q (use organization|workspace|project|user|execution)", scope)
	}
}

// stripCount returns how many leading key segments to hide in display output: the resolved scope
// prefix in friendly mode (unless --full-keys), zero otherwise.
func (s *kvScope) stripCount(scopePrefix []string, friendly bool) int {
	if friendly && !s.fullKeys {
		return len(scopePrefix)
	}

	return 0
}

// mustProjectEnv resolves the project ID (which may switch the active context to the one pinned by
// .versori) and then reads the org from that resolved context, enforcing that --environment is set.
func (s *kvScope) mustProjectEnv(c *config.ConfigFactory) (org, project, env string) {
	project = s.project.GetFlagOrDie(".")
	org = mustOrg(c)

	if s.environment == "" {
		utils.NewExitError().
			WithMessage(fmt.Sprintf("--environment is required for --scope %s", normalizeScope(s.scope))).
			Done()
	}

	return org, project, s.environment
}

func mustOrg(c *config.ConfigFactory) string {
	org := c.OrganisationID()
	if org == "" {
		utils.NewExitError().WithMessage("no organisation in the active context; run `versori context add` or `versori context select`").Done()
	}

	return org
}

// normalizeScope maps the friendly aliases users actually type onto the canonical scope names.
func normalizeScope(scope string) string {
	switch strings.ToLower(strings.Trim(scope, ": ")) {
	case "organization", "organisation", "org":
		return "organization"
	case "workspace", "ws":
		return "workspace"
	case "project", "activation":
		return "project"
	case "user":
		return "user"
	case "execution", "exec":
		return "execution"
	default:
		return scope
	}
}

// fingerprintExternalUserID mirrors @versori/run's fingerprintExternalUserId: a stable sha256 hex
// of the external user ID, used as the KV path segment for :user: scope.
func fingerprintExternalUserID(externalUserID string) string {
	sum := sha256.Sum256([]byte(externalUserID))

	return hex.EncodeToString(sum[:])
}

// resolveStoreByName looks up a store ID by its name. The runtime creates stores lazily on first
// write (ORG_<org>, PROJECT_<project>, EXECUTION_<project>), so a missing store usually means the
// workflow hasn't run yet rather than a typo.
func resolveStoreByName(c *config.ConfigFactory, name string) string {
	resp := storesResponse{}
	err := c.NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/store").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list KV stores").WithReason(err).Done()
	}

	for _, st := range resp.Stores {
		if st.Name == name {
			return st.ID
		}
	}

	utils.NewExitError().
		WithMessage(fmt.Sprintf("no KV store named %q found for this scope; stores are created lazily on first write — has the workflow run yet?", name)).
		Done()

	return "" // unreachable
}

// countEntries returns how many entries match a prefix. Shared by `kv count` and the `kv wipe`
// pre-flight.
func countEntries(c *config.ConfigFactory, storeID string, prefix []string) int {
	resp := struct {
		Count float64 `json:"count"`
	}{}
	err := c.NewRequest().
		WithMethod(http.MethodPost).
		JSONBody(countBody{Selector: countSelector{Prefix: prefix}}).
		Into(&resp).
		WithPath(kvPath(storeID, "count")).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to count KV entries").WithReason(err).Done()
	}

	return int(resp.Count)
}

// kvPath builds a store-scoped sub-path, e.g. kvPath(id, "list") -> "store/<id>/kv/list".
func kvPath(storeID, op string) string {
	return "store/" + storeID + "/kv/" + op
}

// ensureWipePrefixSafe rejects prefixes that would cascade-delete an entire store. The backend
// matches a cascade delete with a prefix LIKE, so an empty prefix — or one containing an empty
// segment (e.g. from `--prefix ''`) — serialises to a match-everything pattern. A length check
// alone is not enough because []string{""} has length 1 but joins to "".
func ensureWipePrefixSafe(prefix []string) error {
	if len(prefix) == 0 {
		return errors.New("refusing to wipe with an empty prefix — that would delete the entire store; pass --prefix or use --scope")
	}

	for _, seg := range prefix {
		if seg == "" {
			return errors.New("refusing to wipe: the prefix contains an empty segment, which would match the entire store; remove the empty --prefix value")
		}
	}

	return nil
}

// splitKey turns a slash-delimited key string into segments, trimming surrounding slashes.
func splitKey(key string) []string {
	key = strings.Trim(key, "/")
	if key == "" {
		return nil
	}

	return strings.Split(key, "/")
}

// stripKey hides the first stripN segments of a key and joins the rest with "/".
func stripKey(key []string, stripN int) string {
	if stripN > len(key) {
		stripN = len(key)
	}

	return strings.Join(key[stripN:], "/")
}

// encodeValueForSet matches the SDK's write encoding: parse the user input as JSON when possible
// (so numbers/objects/arrays keep their type), fall back to a raw string, then JSON-encode the
// result. The encoded string is what gets stored, so a workflow's JSON.parse(get()) round-trips.
func encodeValueForSet(raw string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		parsed = raw
	}

	payload, err := json.Marshal(parsed)
	if err != nil {
		utils.NewExitError().WithMessage("failed to encode value").WithReason(err).Done()
	}

	return string(payload)
}

// decodeValue is the inverse of encodeValueForSet for display. Unless rawValues is set, a value
// that is itself a JSON-encoded string is unwrapped one level so workflow-written data renders as
// the structure it represents rather than an escaped string.
func decodeValue(raw json.RawMessage, rawValues bool) interface{} {
	if len(raw) == 0 {
		return nil
	}

	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}

	if rawValues {
		return v
	}

	if s, ok := v.(string); ok {
		var inner interface{}
		if err := json.Unmarshal([]byte(s), &inner); err == nil {
			return inner
		}
	}

	return v
}

func compactValue(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}

	return string(b)
}

// filterFlags holds the optional server-side list filters. It is embedded by `kv list` only:
// the API's CountSelector supports a prefix but no filter, so `kv count` deliberately omits these.
type filterFlags struct {
	createdAfter  string
	createdBefore string
	metadata      []string
}

func (ff *filterFlags) addFlags(f *pflag.FlagSet) {
	f.StringVar(&ff.createdAfter, "created-after", "", "Only entries created at/after this time (RFC3339 or YYYY-MM-DD).")
	f.StringVar(&ff.createdBefore, "created-before", "", "Only entries created at/before this time (RFC3339 or YYYY-MM-DD).")
	f.StringArrayVar(&ff.metadata, "metadata", nil, "Filter by metadata key=value (repeatable; value parsed as JSON when valid, else a string).")
}

// build returns the request filter, or nil when no filter flags were supplied.
func (ff *filterFlags) build() *listFilter {
	filter := &listFilter{}
	set := false

	if ff.createdAfter != "" {
		filter.CreatedAfter = parseFilterTime(ff.createdAfter, "--created-after")
		set = true
	}

	if ff.createdBefore != "" {
		filter.CreatedBefore = parseFilterTime(ff.createdBefore, "--created-before")
		set = true
	}

	if len(ff.metadata) > 0 {
		filter.Metadata = parseMetadataPairs(ff.metadata)
		set = true
	}

	if !set {
		return nil
	}

	return filter
}

// parseFilterTime accepts a few common layouts and normalises to RFC3339 (what the API expects),
// dying with a clear message on anything it can't parse.
func parseFilterTime(value, flag string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}

	utils.NewExitError().
		WithMessage(fmt.Sprintf("invalid %s %q: use RFC3339 (e.g. 2026-06-01T00:00:00Z) or YYYY-MM-DD", flag, value)).
		Done()

	return "" // unreachable
}

// parseMetadataPairs turns repeated key=value flags into a metadata map, parsing each value as JSON
// when possible (so numbers/bools/objects keep their type) and falling back to a string otherwise.
func parseMetadataPairs(pairs []string) map[string]interface{} {
	out := make(map[string]interface{}, len(pairs))
	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		if !found || key == "" {
			utils.NewExitError().
				WithMessage(fmt.Sprintf("invalid --metadata %q: expected key=value", pair)).
				Done()
		}

		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			parsed = value
		}

		out[key] = parsed
	}

	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}

	return s[:max-1] + "…"
}

// isInteractive reports whether stdin is a TTY. Mutation commands refuse to run without explicit
// confirmation in non-interactive shells (CI, pipes, agent sandboxes).
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}

// confirmOrAbort gates a mutation: skipped when yes is set; in a TTY it prompts; otherwise it
// exits asking for --yes so an agent or script never mutates KV by accident.
func confirmOrAbort(yes bool, prompt string) {
	if yes {
		return
	}

	if !isInteractive() {
		utils.NewExitError().
			WithMessage("refusing to mutate KV without confirmation in a non-interactive shell; pass --yes to proceed").
			Done()
	}

	confirmed := false
	if err := elements.NewConfirm(prompt).Confirm(&confirmed); err != nil {
		utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
	}

	if !confirmed {
		fmt.Println("Aborted; no changes were made.")
		os.Exit(0)
	}
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		utils.NewExitError().WithMessage("failed to marshal JSON output").WithReason(err).Done()
	}

	fmt.Println(string(b))
}

func printYAML(v any) {
	b, err := yaml.Marshal(v)
	if err != nil {
		utils.NewExitError().WithMessage("failed to marshal YAML output").WithReason(err).Done()
	}

	fmt.Print(string(b))
}

// ---- request bodies ----

type listBody struct {
	Selector listSelector `json:"selector"`
	Options  listOptions  `json:"options"`
}

type listSelector struct {
	Prefix []string    `json:"prefix,omitempty"`
	After  string      `json:"after,omitempty"`
	Before string      `json:"before,omitempty"`
	Filter *listFilter `json:"filter,omitempty"`
}

type listFilter struct {
	CreatedAfter  string                 `json:"createdAfter,omitempty"`
	CreatedBefore string                 `json:"createdBefore,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type listOptions struct {
	Limit   int  `json:"limit,omitempty"`
	Reverse bool `json:"reverse,omitempty"`
}

type countBody struct {
	Selector countSelector `json:"selector"`
}

type countSelector struct {
	Prefix []string `json:"prefix,omitempty"`
}

type setBody struct {
	Value   string      `json:"value"`
	Options *setOptions `json:"options,omitempty"`
}

type setOptions struct {
	ExpireIn    int64 `json:"expireIn,omitempty"`
	IfNotExists bool  `json:"ifNotExists,omitempty"`
}

type deleteBody struct {
	Options deleteOptions `json:"options"`
}

type deleteOptions struct {
	Cascade bool `json:"cascade,omitempty"`
}

// ---- response shapes ----

type storesResponse struct {
	Stores []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"stores"`
}

type rawKVEntry struct {
	Key          []string        `json:"key"`
	Value        json.RawMessage `json:"value"`
	Versionstamp string          `json:"versionstamp"`
	CreatedAt    string          `json:"createdAt"`
	ExpiresAt    *string         `json:"expiresAt"`
}

type rawListResponse struct {
	Data       []rawKVEntry   `json:"data"`
	Pagination paginationInfo `json:"pagination"`
}

type rawGetResponse struct {
	Value        json.RawMessage `json:"value"`
	Versionstamp string          `json:"versionstamp"`
	CreatedAt    string          `json:"createdAt"`
	ExpiresAt    *string         `json:"expiresAt"`
}

type paginationInfo struct {
	NextCursor string `json:"nextCursor,omitempty"`
	PrevCursor string `json:"prevCursor,omitempty"`
}

// ---- display shapes ----

// displayEntry is the structured (json/yaml) view of one entry, carrying the real decoded value.
type displayEntry struct {
	Key          string      `json:"key"`
	Value        interface{} `json:"value"`
	Versionstamp string      `json:"versionstamp,omitempty"`
	CreatedAt    string      `json:"createdAt,omitempty"`
	ExpiresAt    *string     `json:"expiresAt,omitempty"`
}

func (e displayEntry) printable() printableEntry {
	expires := ""
	if e.ExpiresAt != nil {
		expires = *e.ExpiresAt
	}

	return printableEntry{
		Key:          e.Key,
		Value:        truncate(compactValue(e.Value), tableValueWidth),
		Versionstamp: e.Versionstamp,
		Created:      e.CreatedAt,
		Expires:      expires,
	}
}

// printableEntry is the flat table view (value compacted + truncated).
type printableEntry struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	Versionstamp string `json:"versionstamp"`
	Created      string `json:"created"`
	Expires      string `json:"expires"`
}

type printableStore struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type printableCount struct {
	Count int `json:"count"`
}

type listOutput struct {
	Data       []displayEntry `json:"data"`
	Pagination paginationInfo `json:"pagination"`
}

// renderEntries prints list results: structured (json/yaml) keeps real nested values; table uses
// the compact printable view and surfaces the next-page cursor on stderr.
func renderEntries(c *config.ConfigFactory, entries []displayEntry, pagination paginationInfo) {
	switch c.OutputFormat() {
	case "json":
		printJSON(listOutput{Data: entries, Pagination: pagination})
	case "yaml":
		printYAML(listOutput{Data: entries, Pagination: pagination})
	default:
		if len(entries) == 0 {
			fmt.Println("(no entries)")

			return
		}

		items := make([]printableEntry, 0, len(entries))
		for _, e := range entries {
			items = append(items, e.printable())
		}
		c.Print(items)

		if pagination.NextCursor != "" {
			fmt.Fprintf(os.Stderr, "\nmore results available; pass --after %q to fetch the next page\n", pagination.NextCursor)
		}
	}
}

// renderEntry prints a single entry (used by `kv get`).
func renderEntry(c *config.ConfigFactory, e displayEntry) {
	switch c.OutputFormat() {
	case "json":
		printJSON(e)
	case "yaml":
		printYAML(e)
	default:
		c.Print([]printableEntry{e.printable()})
	}
}
