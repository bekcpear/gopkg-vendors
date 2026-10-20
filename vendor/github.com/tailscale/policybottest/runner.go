// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/palantir/policy-bot/policy"
	"github.com/palantir/policy-bot/policy/common"
	"github.com/palantir/policy-bot/pull"
	"github.com/palantir/policy-bot/pull/pulltest"
	"gopkg.in/yaml.v2"
)

// LoadPolicy reads and parses a .policy.yml file.
func LoadPolicy(path string) (common.Evaluator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg policy.Config
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	eval, err := policy.ParsePolicy(&cfg, nil)
	if err != nil {
		return nil, fmt.Errorf("building evaluator from %s: %w", path, err)
	}
	return eval, nil
}

// Run evaluates every test in tc against eval and writes a
// human-readable report to out. It returns the number of failed
// tests; the caller is expected to translate that into an exit code.
func Run(eval common.Evaluator, tc *TestConfig, out io.Writer) (failed int) {
	for _, t := range tc.Tests {
		prctx := buildContext(tc, t)
		result := eval.Evaluate(context.Background(), prctx)

		var errs []string
		if t.Expect.Status != "" {
			want, _ := parseStatus(t.Expect.Status) // validated at load time
			if result.Status != want {
				errs = append(errs, fmt.Sprintf(
					"overall status: want %s, got %s (%s)",
					want, result.Status, result.StatusDescription))
			}
		} else if result.Status == common.StatusSkipped {
			// No explicit assertion, but the top-level result is
			// Skipped, which policy-bot's status-posting layer
			// translates into a failing GitHub status check with
			// the description "All rules were skipped. At least
			// one rule must match." (see policy-bot's
			// server/handler/eval_context.go). Treat this as a
			// failure so users notice the production behavior;
			// callers who legitimately want this can opt in by
			// asserting `expect.status: skipped` explicitly.
			errs = append(errs, "overall status: top-level skipped — policy-bot will post this as a failing GitHub status check (\"All rules were skipped. At least one rule must match.\"); assert expect.status: skipped to opt in")
		}
		// Walk the result tree for any named rules the test checks.
		// Build the index lazily so tests that don't use it skip the work.
		var byName map[string]*common.Result
		if len(t.Expect.Rules) > 0 {
			byName = make(map[string]*common.Result)
			indexResults(&result, byName)
			ruleNames := make([]string, 0, len(t.Expect.Rules))
			for name := range t.Expect.Rules {
				ruleNames = append(ruleNames, name)
			}
			sort.Strings(ruleNames)
			for _, name := range ruleNames {
				wantStr := t.Expect.Rules[name]
				want, _ := parseStatus(wantStr)
				got, ok := byName[name]
				if !ok {
					errs = append(errs, fmt.Sprintf(
						"rule %q not found in evaluation tree (known: %s)",
						name, strings.Join(sortedKeys(byName), ", ")))
					continue
				}
				if got.Status != want {
					errs = append(errs, fmt.Sprintf(
						"rule %q: want %s, got %s (%s)",
						name, want, got.Status, got.StatusDescription))
				}
			}
		}

		if len(errs) == 0 {
			fmt.Fprintf(out, "PASS  %s\n", t.Name)
			continue
		}
		failed++
		fmt.Fprintf(out, "FAIL  %s\n", t.Name)
		for _, e := range errs {
			fmt.Fprintf(out, "      %s\n", e)
		}
	}
	return failed
}

// indexResults walks the result tree adding every Result with a non-empty
// Name to out. The same name appearing twice keeps the deepest occurrence,
// which matters very little in practice but is at least deterministic.
func indexResults(r *common.Result, out map[string]*common.Result) {
	if r == nil {
		return
	}
	if r.Name != "" {
		out[r.Name] = r
	}
	for _, c := range r.Children {
		indexResults(c, out)
	}
}

func sortedKeys(m map[string]*common.Result) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// buildContext converts the test's PullRequest plus the file-level
// teams/orgs maps into a pulltest.Context suitable for passing to
// the evaluator. The pulltest.Context has user-keyed membership
// maps; we invert team->[user] into user->[team] here.
func buildContext(tc *TestConfig, t TestCase) pull.Context {
	files := make([]*pull.File, 0, len(t.PR.ChangedFiles))
	for _, f := range t.PR.ChangedFiles {
		files = append(files, &pull.File{
			Filename:  f.Filename,
			Status:    parseFileStatus(f.Status),
			Additions: f.Additions,
			Deletions: f.Deletions,
		})
	}

	comments := make([]*pull.Comment, 0, len(t.PR.Comments))
	for _, c := range t.PR.Comments {
		comments = append(comments, &pull.Comment{Author: c.User, Body: c.Body})
	}

	reviews := make([]*pull.Review, 0, len(t.PR.Reviews))
	for _, r := range t.PR.Reviews {
		reviews = append(reviews, &pull.Review{
			Author: r.User,
			State:  parseReviewState(r.State),
			Body:   r.Body,
		})
	}

	teamMemberships := invertMembership(tc.Teams)
	orgMemberships := invertMembership(tc.Organizations)

	return &pulltest.Context{
		OwnerValue:        defaultStr(tc.Repository.Owner, "tailscale"),
		RepoValue:         defaultStr(tc.Repository.Name, "test"),
		NumberValue:       1,
		AuthorValue:       t.PR.Author,
		TitleValue:        t.PR.Title,
		BranchBaseName:    defaultStr(t.PR.BaseBranch, "main"),
		BranchHeadName:    defaultStr(t.PR.HeadBranch, "feature"),
		StateValue:        "open",
		LabelsValue:       t.PR.Labels,
		Draft:             t.PR.Draft,
		ChangedFilesValue: files,
		CommentsValue:     comments,
		ReviewsValue:      reviews,
		BodyValue:         &pull.Body{Body: t.PR.Body, Author: t.PR.Author},
		TeamMemberships:   teamMemberships,
		OrgMemberships:    orgMemberships,
	}
}

// invertMembership flips a {group: [users]} map (the natural form for
// humans writing test configs) into a {user: [groups]} map, which is
// what pulltest.Context expects.
func invertMembership(in map[string][]string) map[string][]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string][]string{}
	for group, users := range in {
		for _, u := range users {
			out[u] = append(out[u], group)
		}
	}
	return out
}

func parseFileStatus(s string) pull.FileStatus {
	switch strings.ToLower(s) {
	case "", "modified":
		return pull.FileModified
	case "added":
		return pull.FileAdded
	case "deleted", "removed":
		return pull.FileDeleted
	}
	return pull.FileModified
}

func parseReviewState(s string) pull.ReviewState {
	switch strings.ToLower(s) {
	case "approved", "approve":
		return pull.ReviewApproved
	case "changes_requested", "changes-requested", "rejected":
		return pull.ReviewChangesRequested
	case "commented", "comment":
		return pull.ReviewCommented
	case "dismissed":
		return pull.ReviewDismissed
	case "pending":
		return pull.ReviewPending
	}
	return pull.ReviewCommented
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
