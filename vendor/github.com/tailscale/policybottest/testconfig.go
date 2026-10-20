// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"fmt"
	"os"

	"github.com/palantir/policy-bot/policy/common"
	"gopkg.in/yaml.v2"
)

// TestConfig is the schema of a .policy-test.yml file.
//
// It describes hypothetical GitHub teams, org memberships, and a list
// of test cases to evaluate against a separately-loaded .policy.yml.
type TestConfig struct {
	// Repository optionally controls the values returned by the
	// fake pull.Context's RepositoryOwner / RepositoryName methods.
	// Useful for policies that gate on `repository:` predicates.
	Repository struct {
		Owner string `yaml:"owner,omitempty"`
		Name  string `yaml:"name,omitempty"`
	} `yaml:"repository,omitempty"`

	// Teams maps a fully qualified team slug ("org/team") to its
	// members' GitHub usernames. Anything the policy references via
	// `requires.teams` must be listed here for membership checks to
	// succeed.
	Teams map[string][]string `yaml:"teams,omitempty"`

	// Organizations maps an org name to its member usernames, used
	// by `requires.organizations` and `has_author_in.organizations`.
	Organizations map[string][]string `yaml:"organizations,omitempty"`

	// Tests is the list of cases to evaluate.
	Tests []TestCase `yaml:"tests"`
}

// TestCase is a single hypothetical pull request plus a set of
// assertions about how the policy should evaluate it.
type TestCase struct {
	// Name identifies the test case in output. Required.
	Name string `yaml:"name"`

	// PR describes the hypothetical pull request.
	PR PullRequest `yaml:"pull_request"`

	// Expect lists the assertions to check after evaluating the policy.
	Expect Expectations `yaml:"expect"`
}

// PullRequest is the subset of pull request state that the test
// harness fakes out for policy-bot's evaluator.
type PullRequest struct {
	Author       string    `yaml:"author"`
	Title        string    `yaml:"title,omitempty"`
	Body         string    `yaml:"body,omitempty"`
	BaseBranch   string    `yaml:"base_branch,omitempty"`
	HeadBranch   string    `yaml:"head_branch,omitempty"`
	Labels       []string  `yaml:"labels,omitempty"`
	Draft        bool      `yaml:"draft,omitempty"`
	ChangedFiles []File    `yaml:"changed_files,omitempty"`
	Comments     []Comment `yaml:"comments,omitempty"`
	Reviews      []Review  `yaml:"reviews,omitempty"`
}

// File is a hypothetical changed file.
//
// In the YAML it can appear either as a bare string (the filename)
// or as an explicit object with status/additions/deletions, e.g.:
//
//	changed_files:
//	  - tailcfg/foo.go
//	  - filename: tailcfg/bar.go
//	    status: added
//	    additions: 42
type File struct {
	Filename  string `yaml:"filename"`
	Status    string `yaml:"status,omitempty"` // "modified" (default), "added", "deleted"
	Additions int    `yaml:"additions,omitempty"`
	Deletions int    `yaml:"deletions,omitempty"`
}

// UnmarshalYAML lets `- some/path` desugar to {Filename: "some/path"}.
func (f *File) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		f.Filename = s
		return nil
	}
	type raw File
	var r raw
	if err := unmarshal(&r); err != nil {
		return err
	}
	*f = File(r)
	return nil
}

// Comment is a hypothetical PR comment.
type Comment struct {
	User string `yaml:"user"`
	Body string `yaml:"body"`
}

// Review is a hypothetical PR review.
//
// State is one of "approved", "changes_requested", "commented",
// "dismissed", or "pending" (case-insensitive).
type Review struct {
	User  string `yaml:"user"`
	State string `yaml:"state"`
	Body  string `yaml:"body,omitempty"`
}

// Expectations are the assertions checked after running the
// evaluator against the test case.
type Expectations struct {
	// Status is the expected overall status of the policy.approval
	// evaluation: one of "skipped", "pending", "approved",
	// "disapproved". Empty means "do not assert on overall status".
	Status string `yaml:"status,omitempty"`

	// Rules maps an approval_rule name to its expected status.
	// The runner walks the evaluation tree and looks the rule up
	// by Name; unknown names cause the test to fail.
	Rules map[string]string `yaml:"rules,omitempty"`
}

// LoadTestConfig reads and parses a .policy-test.yml file.
func LoadTestConfig(path string) (*TestConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tc TestConfig
	if err := yaml.UnmarshalStrict(data, &tc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if err := tc.validate(); err != nil {
		return nil, fmt.Errorf("invalid test config %s: %w", path, err)
	}
	return &tc, nil
}

func (tc *TestConfig) validate() error {
	if len(tc.Tests) == 0 {
		return fmt.Errorf("no tests defined")
	}
	seen := map[string]bool{}
	for i, t := range tc.Tests {
		if t.Name == "" {
			return fmt.Errorf("tests[%d]: missing name", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("tests[%d]: duplicate name %q", i, t.Name)
		}
		seen[t.Name] = true
		if t.Expect.Status != "" {
			if _, ok := parseStatus(t.Expect.Status); !ok {
				return fmt.Errorf("tests[%d] %q: bad expect.status %q", i, t.Name, t.Expect.Status)
			}
		}
		for rule, status := range t.Expect.Rules {
			if _, ok := parseStatus(status); !ok {
				return fmt.Errorf("tests[%d] %q: bad expect.rules[%q] status %q",
					i, t.Name, rule, status)
			}
		}
	}
	return nil
}

// parseStatus maps the lowercase status name used in YAML to the
// policy-bot EvaluationStatus constant.
func parseStatus(s string) (common.EvaluationStatus, bool) {
	switch s {
	case "skipped":
		return common.StatusSkipped, true
	case "pending":
		return common.StatusPending, true
	case "approved":
		return common.StatusApproved, true
	case "disapproved":
		return common.StatusDisapproved, true
	}
	return 0, false
}
