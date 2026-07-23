// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Command policybottest evaluates a palantir/policy-bot .policy.yml
// file against a set of hypothetical pull requests described in a
// companion .policy-test.yml file, and reports which assertions
// pass and fail.
//
// Intended use is in CI alongside `policy-bot validate` (which only
// checks syntax): this binary additionally checks that the policy
// actually evaluates the way the author expects.
//
// Usage:
//
//	policybottest -policy .policy.yml -tests .policy-test.yml
//
// The exit code is 0 if every test case's assertions pass, and 1
// otherwise. See the testdata/ directory in the source tree for an
// example test config.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	policyPath := flag.String("policy", ".policy.yml", "path to the policy file under test")
	testsPath := flag.String("tests", ".policy-test.yml", "path to the test cases file")
	flag.Parse()

	if err := run(*policyPath, *testsPath); err != nil {
		fmt.Fprintln(os.Stderr, "policybottest:", err)
		os.Exit(2)
	}
}

func run(policyPath, testsPath string) error {
	eval, err := LoadPolicy(policyPath)
	if err != nil {
		return err
	}
	tc, err := LoadTestConfig(testsPath)
	if err != nil {
		return err
	}
	failed := Run(eval, tc, os.Stdout)
	fmt.Printf("\n%d/%d passed\n", len(tc.Tests)-failed, len(tc.Tests))
	if failed > 0 {
		os.Exit(1)
	}
	return nil
}
