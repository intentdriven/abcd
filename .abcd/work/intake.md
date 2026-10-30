# External-contribution intake — the documented pipeline

The protocol for handling a pull request from outside the maintainer pair.
Documented protocol first, verb later (`principles/script-first-mvp.md` — the
three-rung ladder in `principles/README.md`): this page is the MVP, and a
future `abcd contribute triage` absorbs the stages that prove out. Every stage
is typed input → command → **tri-state exit** (`0` clean / `1` findings / `2`
inconclusive). The `2` state is load-bearing: an automated facilitator must
never convert "could not tell" into "fine".

## The stages

| Stage | Actor | What runs | Emits |
|---|---|---|---|
| **S1 Fit** | **HUMAN** | judgement against the accepted issue | `accept` \| `refuse:<reason>` \| `defer` |
| **S2 Fence** | auto | see below | `0` clean / `1` touches publish surface → auto-refuse at S1 / `2` |
| **S3 Volume** | auto | count the author's open PRs | `0` ≤ 3 / `1` over |
| **S4 Review** | auto (delegated) | reviewer agents, **in CI or a container** | `SHIP` \| `NEEDS_WORK` \| `MAJOR_RETHINK` (adr-40 family 1) |
| **S5 Test-reality** | auto | script below | `0` tests failed as they should / `1` tests are decoration / `2` inconclusive |
| **S6 Gate** | **HUMAN** | approve / refuse | pass/fail + remedy |

## S2 — fence

The publish-surface list is `.abcd/work/publish-surface.txt` — one canonical list,
never a second copy:

```sh
gh pr diff <n> --name-only \
  | grep -Ff <(grep -v '^#' .abcd/work/publish-surface.txt | sed 's|^/||') \
  && exit 1 || exit 0
```

A hit is not a refusal of the *work* — it means the change touches the publish
surface, and S1 says so explicitly.

## S4 — review runs contained, never bare

```sh
# Provision modules first: the module cache mount is what lets the build work
# once the network is cut, and the image tag MUST track go.mod's toolchain
# (an older container refuses a newer `go` directive, and --network=none
# blocks the GOTOOLCHAIN auto rescue). Bump the tag with go.mod (a test holds it there).
go mod download
docker run --rm --network=none -v "$PWD:/src" \
  -v "$(go env GOMODCACHE):/go/pkg/mod:ro" -w /src golang:1.26.10
```

`go test` executes contributor Go via `init()` and `TestMain`; a bare run on a
maintainer machine puts `~/.config/gh/hosts.yml` (a token with write on this
repo), `~/.ssh/`, and the private banlist within reach. CI is equally
acceptable — the point is that contributor code never executes with the
maintainer's ambient credentials.

**Calibration:** LLM reviewers measured 21–71% false-negative rates on
execution-validated malicious PRs, worsening sharply when PRs are reviewed
batched (PRWeaver, arXiv:2608.02693). **One PR per fresh context.** S4 is a
filter for the obvious, never the decision.

## S5 — test-reality

Reverting the non-test half and watching the tests is not enough: reverting
usually breaks the *build*, and `[build failed]` is not `--- FAIL:`. To a
non-engineer — or a naive agent — red output reads as success, so the states
are separated:

```sh
BASE=origin/main
git checkout "$BASE" -- $(git diff --name-only "$BASE"..HEAD | grep -v '_test\.go$')
go build ./... || { echo "INCONCLUSIVE: build broke"; exit 2; }
go test ./... 2>&1 | grep -q -- '--- FAIL:' || { echo "TESTS ARE DECORATION"; exit 1; }
echo "tests fail without the change"; exit 0
```

## Typed outputs

Refusal reasons are a **closed enum** — an agent selects, never composes
(`principles/unrecognized-input-never-writes.md`):

`out-of-scope` · `no-accepted-issue` · `touches-publish-surface` ·
`over-volume-cap` · `tests-are-decoration` · `undisclosed-assistance`

A run lands as a receipt in the existing VSA shape at
`.abcd/work/reviews/<sha>/<gate>.json` — the canonical primitive, not a new
format.

## The automation boundary

- **Automates now or later:** S2, S3, S4, S5 — and assembling the inputs to S1.
- **Never automates:** S1 fit and S6 gate
  (`principles/verifier-selects-gates-decide.md` — a probabilistic verdict is a
  selector, never an authority). If the facilitator automates, the human gate
  becomes **more** load-bearing, not less.
- Per `principles/loud-staging.md`, an automated stage that stands down says
  so — it never manufactures a green.
