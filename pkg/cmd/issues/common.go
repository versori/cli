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

package issues

import (
	"encoding/json"
	"fmt"
	"time"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/utils"
)

// Enum value sets accepted by the issue filter/update flags, mirrored from the API's
// IssueStatusEnum / IssueSeverityEnum / IssueResolutionStatusEnum so an invalid value fails fast
// with a helpful message instead of a raw server rejection.
var (
	validStatuses           = []string{"open", "closed", "acked", "resolved"}
	validSeverities         = []string{"critical", "high", "low", "medium"}
	validResolutionStatuses = []string{"resolved", "negated", "ignored"}
)

func validateStatus(v string) error   { return validateEnum("status", v, validStatuses) }
func validateSeverity(v string) error { return validateEnum("severity", v, validSeverities) }
func validateResolutionStatus(v string) error {
	return validateEnum("resolution-status", v, validResolutionStatuses)
}

func validateEnum(flag, value string, allowed []string) error {
	if value == "" {
		return nil
	}
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}

	return fmt.Errorf("invalid --%s %q (must be one of %v)", flag, value, allowed)
}

// printableIssue is the reduced view rendered as a table row by `issues list`. Enum fields on the
// generated v1.Issue are pointers to string-typed enums; rendering those directly through the table
// printer would show hex addresses, so we flatten to plain strings here (same pattern as
// connections list).
type printableIssue struct {
	Title      string `json:"title"`
	Severity   string `json:"severity,omitempty"`
	Status     string `json:"status"`
	SeenCount  int    `json:"seenCount"`
	LastSeenAt string `json:"lastSeenAt"`
	Id         string `json:"id"`
}

// printableIssueDetail is the fuller single-issue view rendered by `issues get`. It adds the
// message, project/environment IDs, and the flattened labels/annotations so an agent can identify
// an issue (e.g. title "OOM Killed") and act without a second call. Labels/Annotations are
// compact-JSON strings so they survive the table renderer; use -o json / -o yaml for structured
// output.
type printableIssueDetail struct {
	Id            string `json:"id"`
	Title         string `json:"title"`
	Message       string `json:"message"`
	Severity      string `json:"severity,omitempty"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	ProjectId     string `json:"projectId"`
	EnvironmentId string `json:"environmentId"`
	SeenCount     int    `json:"seenCount"`
	CreatedAt     string `json:"createdAt"`
	LastSeenAt    string `json:"lastSeenAt"`
	ResolvedAt    string `json:"resolvedAt,omitempty"`
	Labels        string `json:"labels,omitempty"`
	Annotations   string `json:"annotations,omitempty"`
}

func init() {
	utils.RegisterResource(printableIssue{}, []string{"Title", "Severity", "Status", "SeenCount", "LastSeenAt", "Id"})
	utils.RegisterResource(printableIssueDetail{}, []string{"Id", "Title", "Severity", "Status", "Reason", "ProjectId", "EnvironmentId", "SeenCount", "LastSeenAt", "Message"})
}

func toPrintableIssue(i v1.Issue) printableIssue {
	return printableIssue{
		Title:      i.Title,
		Severity:   derefSeverity(i.Severity),
		Status:     string(i.Status),
		SeenCount:  i.SeenCount,
		LastSeenAt: i.LastSeenAt.Format(time.RFC3339),
		Id:         i.Id.String(),
	}
}

func toPrintableIssueDetail(i v1.Issue) printableIssueDetail {
	d := printableIssueDetail{
		Id:            i.Id.String(),
		Title:         i.Title,
		Message:       i.Message,
		Severity:      derefSeverity(i.Severity),
		Status:        string(i.Status),
		Reason:        string(i.Reason),
		ProjectId:     i.ProjectId.String(),
		EnvironmentId: i.EnvironmentId.String(),
		SeenCount:     i.SeenCount,
		CreatedAt:     i.CreatedAt.Format(time.RFC3339),
		LastSeenAt:    i.LastSeenAt.Format(time.RFC3339),
		Labels:        marshalMap(i.Labels),
		Annotations:   marshalMap(i.Annotations),
	}
	if i.ResolvedAt != nil {
		d.ResolvedAt = i.ResolvedAt.Format(time.RFC3339)
	}

	return d
}

func derefSeverity(s *v1.IssueSeverityEnum) string {
	if s == nil {
		return ""
	}

	return string(*s)
}

// marshalMap renders a labels/annotations map as a compact JSON string for the table view. Empty
// maps collapse to "" so the column stays blank rather than showing "{}".
func marshalMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}

	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}

	return string(b)
}
