# policybottest

`policybottest` is a small CLI for unit-testing
[palantir/policy-bot](https://github.com/palantir/policy-bot)
`.policy.yml` files. Upstream's `policy-bot validate` checks
syntax; this tool additionally checks that the policy actually
*evaluates* the way the author expected for a set of hypothetical
pull requests.

It works by loading the policy with the same Go evaluator
policy-bot itself uses and running it against a fake
`pull.Context` populated from a companion `.policy-test.yml`
file.

## Usage

```
policybottest -policy .policy.yml -tests .policy-test.yml
```

Exit code is `0` if every test's assertions pass and `1`
otherwise. `2` is reserved for setup errors (bad YAML, missing
files, an unparseable policy).

## Test file schema

The test file is YAML, parsed in strict mode (unknown fields
fail the load). The canonical Go definition is in
[`testconfig.go`](./testconfig.go); this section is the
human-readable reference.

### Top-level keys

| Key             | Type                              | Required | Default                                                          |
| --------------- | --------------------------------- | -------- | ---------------------------------------------------------------- |
| `repository`    | object (`{owner, name}`)          | no       | `{owner: tailscale, name: test}`                                 |
| `teams`         | map of `org/team` → list of users | no       | `{}`                                                             |
| `organizations` | map of `org` → list of users      | no       | `{}`                                                             |
| `tests`         | list of test cases                | **yes**  | —                                                                |

`teams` and `organizations` populate the membership lookups the
policy evaluator uses for `requires.teams` /
`requires.organizations` / `has_author_in.teams` /
`has_author_in.organizations`. Any team or org the policy
references must appear here, otherwise membership checks always
return false. Team keys must include the org prefix
(e.g. `myorg/admins`, not just `admins`).

### `tests[]` — a single test case

| Key            | Type                | Required | Notes                                       |
| -------------- | ------------------- | -------- | ------------------------------------------- |
| `name`         | string              | **yes**  | Must be unique across the file.             |
| `pull_request` | object (see below)  | yes      | The hypothetical PR.                        |
| `expect`       | object (see below)  | yes      | Assertions to check after evaluation.       |

### `tests[].pull_request`

| Key             | Type            | Default          | Notes                                                                 |
| --------------- | --------------- | ---------------- | --------------------------------------------------------------------- |
| `author`        | string          | `""`             | GitHub login of the PR author.                                        |
| `title`         | string          | `""`             | Used by `title:` predicates.                                          |
| `body`          | string          | `""`             | Used by predicates that read the PR body.                             |
| `base_branch`   | string          | `"main"`         | Used by `targets_branch:`.                                            |
| `head_branch`   | string          | `"feature"`      | Used by `from_branch:`.                                               |
| `labels`        | list of string  | `[]`             | Used by `has_labels:`.                                                |
| `draft`         | bool            | `false`          |                                                                       |
| `changed_files` | list of `File`  | `[]`             | See below.                                                            |
| `comments`      | list of object  | `[]`             | Each: `{user, body}`.                                                 |
| `reviews`       | list of object  | `[]`             | Each: `{user, state, body?}`.                                         |

### `File` shorthand

A changed file may be written either as a bare string (the path)
or as an explicit object. Both forms are equivalent for path-only
rules:

```yaml
changed_files:
  - tailcfg/tailcfg.go            # status defaults to "modified"
  - filename: cmd/new/main.go
    status: added                 # "modified" | "added" | "deleted"
    additions: 42
    deletions: 0
```

### `tests[].expect`

Either assertion may be omitted; an empty `expect:` block is
legal and asserts nothing about per-rule behavior, but see the
note on top-level `skipped` below.

| Key      | Type                          | Notes                                                                 |
| -------- | ----------------------------- | --------------------------------------------------------------------- |
| `status` | string (status enum)          | The overall status of `policy.approval` evaluation.                   |
| `rules`  | map of `rule name` → status   | Per-rule assertions, by `approval_rule.name` from the policy file.    |

When `rules` is set, the runner walks the policy-bot evaluation
result tree and looks up each rule by its `Name`. A name that is
not found in the tree fails the test and prints the list of
names that *were* found, so typos are caught immediately.

#### Top-level `skipped` is treated as failure

The policy-bot evaluator's `Result.Status` has four values:
`skipped`, `pending`, `approved`, `disapproved`. The server-side
status-posting layer (`eval_context.go` in upstream) translates
a top-level `skipped` result into a **failing** GitHub status
check with the description "All rules were skipped. At least one
rule must match." — this is something policy-bot fails the PR
on, not a no-op.

To make tests catch this rather than rubber-stamping it, the
runner reports a top-level `skipped` result as a failure whenever
`expect.status` is not set. If you genuinely want to assert that
the top-level skipped state is intentional, opt in explicitly:

```yaml
expect:
  status: skipped
```

Per-rule (child) `skipped` is never auto-failed, since "this rule
didn't apply" is a normal evaluation outcome inside an `or` block
or a path-gated rule.

### Enums

Status values (used in `expect.status` and `expect.rules` values),
mirroring policy-bot's `common.EvaluationStatus`:

| Value          | Meaning                                                                  |
| -------------- | ------------------------------------------------------------------------ |
| `skipped`      | The rule's `if:` predicates didn't match; it's not part of the decision. |
| `pending`      | The rule applies but its requirements aren't yet satisfied.              |
| `approved`     | The rule applies and is satisfied.                                       |
| `disapproved`  | The rule applies and the PR is actively blocked.                         |

File status values (used in `changed_files[].status`):

| Value      | Aliases    |
| ---------- | ---------- |
| `modified` | (default)  |
| `added`    |            |
| `deleted`  | `removed`  |

Review state values (used in `reviews[].state`), all
case-insensitive:

| Value               | Aliases                            |
| ------------------- | ---------------------------------- |
| `approved`          | `approve`                          |
| `changes_requested` | `changes-requested`, `rejected`    |
| `commented`         | `comment` (also the implicit default) |
| `dismissed`         |                                    |
| `pending`           |                                    |

## Worked example

```yaml
teams:
  myorg/admins: [alice, bob]
  myorg/devs:   [alice, bob, carol]

tests:
  - name: an admin's review approves an admin-only path
    pull_request:
      author: carol
      changed_files:
        - admin/secrets.go
      reviews:
        - user: alice
          state: approved
    expect:
      status: approved
      rules:
        "admin paths require an admin review": approved

  - name: the author cannot self-approve
    pull_request:
      author: alice
      changed_files:
        - admin/secrets.go
      reviews:
        - user: alice
          state: approved
    expect:
      status: pending
      rules:
        "admin paths require an admin review": pending

  - name: changes outside admin/ skip the admin rule
    pull_request:
      author: carol
      changed_files:
        - README.md
    expect:
      status: skipped
      rules:
        "admin paths require an admin review": skipped
```

See [`testdata/`](./testdata/) for a real worked example: the
tailscale.com policy with ten cases covering path-scoping,
team-gated review, and a comment-based override escape hatch.

## Limitations

* No support yet for `has_successful_status` /
  `has_workflow_result` predicates (no fake CI statuses are
  wired up); add fields to `PullRequest` in `testconfig.go` if
  you need them.
* The fake context does not model `RequestedReviewers`,
  `RepositoryCollaborators`, or `Permissions`; rules that gate
  on those will see empty sets.
* Test cases are evaluated independently; there is no notion of
  time-ordering between events on a single PR.

## Development

```
go test ./...
go run . -policy testdata/tailscale.com.policy.yml \
         -tests  testdata/tailscale.com.policy-test.yml
```
