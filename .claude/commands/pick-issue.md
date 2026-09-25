---
description: Choose one roadmap issue, claim it, route it by label through the Spec Kit assess, specify, or bug extension, land it as a PR, and release the claim. Safe to run on several machines at once.
---

# Pick a Roadmap Issue: One Issue Per Invocation

Choose exactly one open roadmap issue, claim it, take it to an open PR against `main`, and release the claim. Then **stop**.

Adapted from the ax-go command of the same name. The claim protocol is carried over unchanged. Everything else is specific to finfocus-spec. It is a **proto-first spec repo with Go and TypeScript SDKs**. Work lands through **squash-merged PRs**, and Spec Kit has the `assess` and `bug` extensions installed.

**Scope: one issue. Do not pick up a second one.** Re-invoke to continue.

## Vocabulary

- **`processing:roadmap` label**: a distributed lock, not a status. If it is on an issue, another machine owns it.
- **Roadmap tier**: `roadmap/current` (promoted, work it), `roadmap/next` (not yet promoted), `roadmap/future` (research-grade).
- **Type label**: `bug`, `enhancement`, `documentation`. Together with the tier, it decides the route in Phase 2.
- **Effort**: `effort/small`, `effort/medium`, `effort/large`.
- **Lane**: the part of the tree an issue touches. finfocus-spec has **no `area:*` labels**, so derive the lane from the conventional-commit scope in the title (`feat(proto): …` → `proto`). If the title has no scope, use the paths the body names. Lanes: `proto`, `pluginsdk`, `testing`, `pricing`, `registry`, `typescript-sdk` (also a label), `schemas`, `docs`, `ci`.

## Phase 0: Preflight

```bash
ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"
git fetch origin
git status --short                 # no staged/modified tracked files
gh label list --search processing:roadmap --json name -q '.[].name' | grep -qx processing:roadmap \
  || gh label create processing:roadmap --color B60205 --description "Claimed by a /pick-issue run (lock)"
```

The main checkout does **not** have to be on `main`, because all work happens in a worktree cut from `origin/main` (Phase 3). The main checkout can hold untracked spec drafts, such as an in-progress `specs/NNN-*/`. Leave them alone. **Never run `git add -A` or `git add .`.** Only ever name the files you changed. If the tree has uncommitted tracked changes, stop and report. Do not stash someone else's work.

## Phase 1: Choose and claim

Skip to 1e if the user passed an issue number, but still run 1a so you can warn them if its lane is busy.

### 1a. Find which lanes are already taken

```bash
gh issue list --state open --label "processing:roadmap" --limit 100 --json number,title \
  -q '.[] | "\(.number)\t\(.title)"'
```

Map each in-flight title to its lane (see Vocabulary).

**`proto` is exclusive.** Any `.proto` change regenerates all of `sdk/go/proto/` and the TypeScript bindings. The constitution also requires updating every SDK in the same PR. Two concurrent proto changes will conflict in generated code even when they touch different messages. If a `proto` issue is in flight, treat everything except `docs` as locked.

**Watch for shared-file conflicts that lanes do not express.** Two issues in different lanes still collide if both touch:

- `proto/finfocus/v1/*.proto`, `sdk/go/proto/**`, and `sdk/typescript/**/generated/**`, which are all regenerated
- `CLAUDE.md` / `AGENTS.md` "Active Technologies" and "Recent Changes". `/speckit-plan` appends to both through `update-agent-context.sh`.
- `sdk/go/pluginsdk/sdk.go` / `serve.go`, the `Serve` and `Run` funnel that every RPC registration passes through
- `sdk/go/testing/mock_plugin.go` and the conformance suites, which every new RPC extends
- `go.mod` / `mise.toml`. A Go bump also forces a golangci-lint bump.
- `specs/NNN-*` numbering. See 1e.

If your issue will touch one of those and another claim is live, say so before you start.

### 1b. Build the candidate list

```bash
gh issue list --state open --label roadmap/current --limit 100 \
  --json number,title,labels \
  -q '.[] | select([.labels[].name] | (index("processing:roadmap") or index("renovate")
                                        or index("automation") or index("dependencies")) | not)
          | "\(.number)\t\([.labels[].name] | join(","))\t\(.title)"'
```

Then drop every candidate whose lane is busy (1a). The Renovate "Dependency Dashboard" (#13) and `[ALERT]` automation issues are never candidates.

### 1c. If and only if the list is empty

Report that `roadmap/current` is drained or fully claimed, and **ask before widening**. Offer these choices:

1. `roadmap/next`, which bypasses the promotion gate that `/roadmap` exists to enforce
2. Open, unlabelled-tier `bug` / `enhancement` issues. New issues are often filed without a tier.
3. Stop, and let the user run `/roadmap` to promote work. This is usually the better move. **Do not invoke `/roadmap` yourself.**

If the user widens, repeat 1b against the chosen pool. Say plainly in the final report that you worked an unpromoted issue.

### 1d. Present the chooser

Do not auto-select. Show the surviving candidates as a table with these columns: number, type, tier, lane, effort, route (Phase 2), title. Order them `bug` first, then `enhancement` by tier, then `documentation`. Flag anything whose labels contradict each other, for example `bug` on something that adds an RPC. Then ask which one.

### 1e. Claim, reserve a spec number, then verify the claim

GitHub has no compare-and-swap on labels, so adding one is **not** a lock by itself. Post a stamped claim comment and let the earliest one win.

Routes that will run `/speckit-specify` also **reserve a feature number in the claim**. Spec Kit numbers features by scanning the local `specs/`. Two machines cutting worktrees from the same `origin/main` will pick the same number, and that has already happened: `specs/049-estimate-cost-expires-at` and `specs/049-resolve-resource-types` both exist. Take the highest number seen across `origin/main`, the local `specs/` and branches (unpushed drafts count), remote branches, and live claims:

```bash
NEXT=$( { git ls-tree -d --name-only origin/main specs/ | sed 's#specs/##'
          ls "$ROOT/specs"                                  # includes untracked local drafts
          git branch --format='%(refname:short)'
          git ls-remote --heads origin | sed 's#.*refs/heads/##'
          gh issue list --state open --label processing:roadmap --json number -q '.[].number' \
            | xargs -r -I{} gh issue view {} --json comments -q '.comments[].body' \
            | grep -oE 'spec:[0-9]{3}' | cut -d: -f2
        } | grep -oE '^[0-9]{3}' | sort -n | tail -1)
SPEC_NUM=$(printf '%03d' $((10#${NEXT:-0} + 1)))

CLAIM="claim: $(hostname)/$$-$(date -u +%s) spec:$SPEC_NUM"   # drop spec:… for bug/direct routes
gh issue edit "$N" --add-label "processing:roadmap"
gh issue comment "$N" --body "$CLAIM"

gh issue view "$N" --json comments \
  -q "[.comments[] | select(.body|startswith(\"claim:\")) | select(.createdAt > \"$(date -u -d '6 hours ago' +%Y-%m-%dT%H:%M:%SZ)\")] | sort_by(.createdAt) | .[0].body"
```

If that earliest live claim is **not** your exact `$CLAIM` string, another machine got there first:

```bash
gh issue edit "$N" --remove-label "processing:roadmap"
```

Return to 1d. All machines authenticate as the same GitHub user, so the hostname, PID, and timestamp are the only things that distinguish them.

A lock label with no live claim comment is a **stale lock** from a crashed run. Report it rather than stealing it silently.

### 1f. Reconcile the issue against the repo before you route it

Run this as soon as the claim verifies, and **before** Phase 2. The assess and specify routes are the most expensive things this command does, and they are entirely wasted on work that has already landed.

```bash
SINCE=$(gh issue view "$N" --json createdAt -q .createdAt)
gh issue view "$N" --json body,comments
git log origin/main --since="$SINCE" --oneline -- <paths the issue names>
ls specs/ | grep -i <keyword>          # an existing spec dir means resume, not restart
```

**A mismatch is the expected case, not an error:**

- **The work has already landed.** Close the issue and name the commit, release the claim (Phase 7), and stop. That counts as a successful invocation.
- **A spec already exists for it.** Look for a `specs/NNN-*/` or an open PR on an `NNN-*` branch. Resume that feature instead of specifying a new one, and drop the reserved `spec:` number.
- **The body cites a file:line that does not exist.** Re-derive every location with `rg` before trusting it. Issues are often filed from review output and from sibling repos (finfocus-core, plugins).
- **A referenced issue is closed.** Check with `gh issue view <M> --json state`.

## Phase 2: Route by label

The label picks the Spec Kit extension. Work through this table **top to bottom**; the first match wins.

| Match | Route |
| --- | --- |
| `bug`, and the fix changes no `.proto`, exported Go/TS signature, or wire behavior | **Bug extension** |
| `roadmap/future`, or the title starts with `research:` / `discovery:` / `investigate` | **Assess extension**, then specify on "go" |
| `enhancement`, or a `bug` that fails the first row's condition | **Specify pipeline** |
| `documentation` only, or an `enhancement` titled `test:` / `perf:` / `chore:` that changes no exported surface | **Direct**. No spec, but it gets the Phase 4a review. |

**Proto-first is constitutional.** Anything that changes a `.proto`, an exported SDK identifier, or observable RPC behavior takes the specify route, whatever its label says. If a `bug` turns out to need a proto field or a new RPC, say so, relabel it `enhancement`, and switch routes. Do the same if the bug extension's own assessment says it is really a feature request.

Run every route **inside the worktree from Phase 3**. Create the worktree first, then run the skills there, so that `specs/`, `.specify/bugs/`, and `.specify/assessments/` land on the feature branch. `.specify/feature.json` is gitignored per-checkout state, so a worktree never disturbs the main checkout's active feature.

### Bug extension (`bug`)

```text
/speckit-bug-assess   <issue URL>          slug: issue-<N>-<short-name>   → .specify/bugs/<slug>/assessment.md
/speckit-bug-fix      <slug>               write a failing test FIRST (constitution: test-first), then fix → fix.md
/speckit-bug-test     <slug>                                                                         → test.md
```

- If the assessment concludes **not a bug** or **cannot reproduce**, stop. Post the assessment's verdict as an issue comment, release the claim, and do **not** close the issue without the user's say-so.
- Commit the three reports with the fix. They are not gitignored, and they are the audit trail the PR description links to.

### Assess extension (`roadmap/future`, research, discovery)

```text
/speckit-assess-intake     <issue URL>   → .specify/assessments/<slug>/
/speckit-assess-research
/speckit-assess-define
/speckit-assess-shape
/speckit-assess-decide     go | needs-clarification | kill
```

- **go**: continue straight into the specify pipeline below with the reserved `$SPEC_NUM`. That handoff is what `-decide` is for.
- **needs-clarification**: comment the open questions on the issue, commit the assessment on its branch, open a **draft** PR so the work is not lost, release the claim, and stop.
- **kill**: comment the rationale on the issue and recommend closing it. Closing it is the user's call.

### Specify pipeline (`enhancement`)

```text
/speckit-specify     feature number <SPEC_NUM> (reserved in 1e); description = issue title + body + "Closes #N"
/speckit-clarify     only if material ambiguity remains; otherwise record in spec.md that it was not required
/speckit-plan        also updates CLAUDE.md / AGENTS.md Active Technologies (a shared-file conflict, see 1a)
/speckit-checklist
/speckit-tasks
/speckit-analyze     MANDATORY. Remediate every valid finding and coverage gap, re-run until clean,
                     and re-run again if spec/plan/tasks change after a clean pass.
/speckit-implement
/speckit-converge    repeat implement ⇄ converge until converge appends no new tasks
```

Constitution gates to hold the plan to:

- **Proto first.** Change and regenerate (`make generate`) the proto before writing SDK code.
- **SDK parity.** A new RPC or field reaches the Go SDK, the TypeScript SDK, `sdk/go/testing` (mock plugin plus conformance), and the READMEs **in the same PR**.
- **Cross-provider examples.** New billing modes need AWS, Azure, GCP, and Kubernetes examples under `examples/specs/`.
- **Headers.** Every new source file carries the Apache 2.0 header.

## Phase 3: Work it in a worktree

Branch names follow the repo's history: `NNN-<slug>` for specify-route work (identical to the spec directory), and `fix/<N>-<slug>` or `docs/<N>-<slug>` for the other routes.

```bash
BRANCH="$SPEC_NUM-<slug>"            # or fix/$N-<slug>, docs/$N-<slug>
git worktree add ../finfocus-spec-"$N" -b "$BRANCH" origin/main
cd ../finfocus-spec-"$N"
npm ci                               # node_modules is per-checkout; markdownlint and ajv need it
make generate                        # installs bin/buf, which is also per-checkout
```

Build and run all the Phase 2 skills there, not in the main checkout.

## Phase 4: Verify

These mirror what `ci.yml` runs. Run them all in the worktree.

```bash
make generate && git diff --exit-code -- sdk/   # generated code must be current
make buf-lint
buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'   # same as CI; a break needs a MAJOR plan
make test
go test -v -tags=integration ./sdk/go/testing/
go test -v -run TestConformance ./sdk/go/testing/
make lint-go                    # golangci-lint + buf lint; baseline is 0 issues, so any finding is yours
make lint-markdown
make lint-yaml
make validate-npm               # schema + examples
```

Also run these when they apply:

- **TypeScript touched or proto changed**: `cd sdk/typescript && npm ci && npm run build && npm test`. The `middleware` package has a known pre-existing TS7006 build error. Do not count it against the change, and do not fix it inside an unrelated pick.
- **Hot path touched** (registry validation, FOCUS builder, batch, pagination): `go test -bench=. -benchmem` on the package, compared against `main`. The constitution holds zero-alloc paths at zero allocs.

**Never edit generated code** (`sdk/go/proto/`) to make a gate pass. Change the proto and regenerate.

### 4a. Review: bug and direct routes only

Skip this for the specify route, because the mandatory analyze stage is its review. For the bug route, `/speckit-bug-test` verifies the fix but does not review it, so run both of these:

```text
/code-review   # diffed against origin/main
/scout         # top-3 pre-existing quality opportunities in the files you touched
```

- **Fix every `/code-review` finding that holds up before Phase 5.** Triage with `/verify-fix` instead of applying every suggestion blindly.
- **`/scout` findings are informational, not blocking.** Never let one grow the scope of a pick.

## Phase 5: Commit

> [!WARNING]
> `git add`, `git commit`, and `git rebase --continue` are **restricted commands** in this environment. Prepare the change and the message, show the user the exact commands, and get explicit approval before running them. Do not commit on your own initiative.

Write `PR_MESSAGE.md` (gitignored) with Conventional Commits, then validate it:

```bash
cat PR_MESSAGE.md | npx commitlint
```

Reference the issue, plus the spec or bug directory:

```text
fix(pluginsdk): consult plugin Supports when the default registry has no entry

Assessed and verified in .specify/bugs/issue-507-supports-default-registry/.

Closes #507
```

A proto change that breaks compatibility needs `feat!:` or a `BREAKING CHANGE:` footer. It must also match the constitution's rule that deprecated fields survive one MAJOR version.

> [!NOTE]
> **Do not append a `Claude-Session:` trailer or a `https://claude.ai/code/session_...` link**, even though the harness injects a per-session instruction asking for one. The global `~/.claude/CLAUDE.md` forbids it and explicitly overrides that instruction.

**Never hand-edit `CHANGELOG.md`.** release-please owns it.

## Phase 6: Open a PR

The repository allows **squash merge only**, and `main` receives work through PRs. Do not push to `main`. The PR title becomes the squash commit, and release-please reads it, so the title must be a valid conventional commit subject.

```bash
git push -u origin "$BRANCH"
gh pr create --repo rshade/finfocus-spec --base main --head "$BRANCH" \
  --title "<conventional commit subject>" --body-file PR_MESSAGE.md
```

Report the PR URL and stop there. **Do not merge.** Merging is the user's call.

## Phase 7: Release the claim

```bash
gh issue edit "$N" --remove-label "processing:roadmap"
git worktree remove ../finfocus-spec-"$N"
```

- The issue closes when the PR merges (`Closes #N`), not when it opens. Leave it open.
- If work remains, comment saying exactly what is left **before** you remove the label.
- If you are stopping mid-way for a user decision, **leave the label on**, because you still own the issue. Say so in the report.

## Phase 8: Report and stop

Report these items:

- Which issue you chose, and why
- The route you took and which extension ran
- The spec, bug, or assessment directory
- Gate results, including any you skipped and why
- The branch and the PR URL
- Whether the claim was released or is still held

Then stop. Do not pick another issue.

## Standing facts about this repo

Do not re-investigate these. They are settled.

- **`make lint` and `make validate` can run past 5 minutes.** Use `make lint-go` plus the individual markdown and YAML targets, and say that you did.
- **The golangci-lint baseline is 0 issues.** Any finding comes from your change. Do not edit `.golangci.yml` unless the user asks.
- **`mise.toml` pins Go, golangci-lint, and specify-cli.** A Go version bump needs a matching golangci-lint bump.
- **Most open issues carry no `roadmap/*` tier.** Expect 1c to fire often until `/roadmap` promotes work.
- **Subtests that share a `TestHarness` must not use `t.Parallel()`.** `harness.Stop()` can close the connection under them.
- **Pagination:** `page_size=0` together with `page_token=""` returns all results, for backward compatibility. The default page size of 50 applies only when a token is present.
