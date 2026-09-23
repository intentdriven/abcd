BINARY := abcd
BINDIR := bin
TARGETS := darwin/arm64 darwin/amd64 linux/arm64 linux/amd64
# Version stamped into `abcd version`. Defaults to the in-source "dev" value; the
# release build passes the git tag (VERSION=vX.Y.Z). SemVer, v-prefixed.
VERSION ?=
# -s -w strips the symbol table and DWARF debug info; -X stamps the version.
# -trimpath (in the build recipe) rewrites absolute source paths to module paths
# so no local filesystem path is embedded — a smaller, path-clean binary suitable
# for public distribution.
LDFLAGS := -s -w$(if $(VERSION), -X github.com/intentdriven/abcd/internal/core.Version=$(VERSION),)

# The Go toolchain version go.mod declares, read from the declaration rather
# than spelled here: a second spelling is a second thing to bump, and the one
# that falls behind is the one nothing runs. Drives the format gate below.
GO_TOOLCHAIN_VERSION := $(shell sed -n 's/^go \([0-9][0-9.]*\)$$/\1/p' go.mod)

.PHONY: build test vet clean preflight lint-reviews lint-issues lint-decisions record-lint docs-lint site-render smoke \
	evals-cold-reading check-attribution scaffold-sync scaffold-sync-check fmt fmt-check

# Cross-compile every supported target to bin/abcd-<goos>-<arch>.
# Pass VERSION=vX.Y.Z to stamp the version (release builds); omit for a dev build.
build:
	@mkdir -p $(BINDIR)
	@for target in $(TARGETS); do \
		goos=$${target%/*}; goarch=$${target#*/}; \
		out=$(BINDIR)/$(BINARY)-$$goos-$$goarch; \
		echo "building $$out"; \
		GOOS=$$goos GOARCH=$$goarch go build -trimpath -ldflags "$(LDFLAGS)" -o $$out ./cmd/abcd || exit 1; \
	done

test:
	go test ./...

# Every eval in evals/: the self-discovering smoke harness (build the binary, walk
# the Cobra tree, run every command's --help + the read-only verbs) and the
# cold-reading evals below. Behind the `smoke` build tag so they stay out of the
# unit-test lane; run explicitly here, under `make preflight`, in CI's smoke job,
# and in the release verify gate.
smoke:
	go test -tags smoke ./evals/...

# The cold-reading evals alone (evals/coldreading_*_test.go): the read-block eval
# that falsifies the assembler's blindfold by planting sentinel warm content in a
# fixture repository state and asserting its absence from what the assembler
# passes.
#
# Both this target and `smoke` are `preflight` prerequisites, so the push gate
# sees what CI sees for the eval lanes rather than trusting a job that cannot
# block a merge (iss-2608311632382737).
#
# It has its own target, and CI its own always-run job, because the diff
# classifier stands the `smoke` job down on a change confined to docs/,
# .abcd/development/, .abcd/work/ and the root prose files — and those are
# precisely the paths these evals read. A record-only change is the diff MOST
# able to introduce warm content into material the assembler includes, so
# standing the eval down there is anti-correlated with the risk, and a stood-down
# job still reports its check context green. Same reasoning `.github/workflows/ci.yml`
# already documents for the ubuntu unit lane, which never stands down because its
# tests read the live tree under the allowlist.
#
# The selection is a BUILD TAG, not a list of test names: every file carrying
# `//go:build smoke || coldreading` runs here and under `make smoke` both, so a
# second cold-reading eval joins the lane by carrying that constraint, with no
# edit to this target or to the workflow.
evals-cold-reading:
	go test -tags coldreading ./evals/...

# Format gate, resolved through the toolchain go.mod declares
# (iss-2609081953452204). Not `gofmt` off PATH: gofmt's rules move between
# releases — go 1.27 re-indents a multi-value return whose operands are
# composite literals, which this tree contains at internal/core/ahoy/remote.go —
# so on a machine newer than the declaration a bare `gofmt -l .` names a file
# that CI's pinned gofmt calls correctly formatted. The developer reformats what
# the gate names and pushes a file CI then rejects in the other direction, and
# neither direction is visible in the output, which names a file and never says
# which toolchain judged it.
#
# `GOTOOLCHAIN=go<version> go env GOROOT` fetches and caches the declared
# toolchain if the machine lacks it, then reports where it landed; the gofmt
# under that GOROOT is the one CI runs. `fmt` applies the same binary, so the
# remedy and the diagnosis can never disagree.
#
# It REFUSES rather than falling back when the toolchain cannot be resolved
# (offline, or the fetch declined). A fallback would print a filename judged by
# the wrong gofmt, which is precisely the false green this target removes — the
# skew is named in the refusal instead
# (.abcd/development/principles/loud-staging.md).
define pinned_gofmt
	@set -eu; \
	version='$(GO_TOOLCHAIN_VERSION)'; \
	if [ -z "$$version" ]; then \
		echo "gofmt: REFUSING — go.mod declares no \`go <version>\` line, so the format gate has no toolchain to resolve." >&2; \
		exit 2; \
	fi; \
	local_version="$$(go env GOVERSION 2>/dev/null || echo unknown)"; \
	if ! goroot="$$(GOTOOLCHAIN=go$$version go env GOROOT 2>&1)" || [ ! -x "$$goroot/bin/gofmt" ]; then \
		echo "gofmt: REFUSING to judge this tree." >&2; \
		echo "gofmt:   go.mod declares go$$version; the go on PATH is $$local_version." >&2; \
		echo "gofmt:   the go$$version toolchain could not be resolved (the fetch needs network):" >&2; \
		echo "$$goroot" | sed 's/^/gofmt:     /' >&2; \
		echo "gofmt:   NOT falling back to the gofmt on PATH — a different gofmt version judges this" >&2; \
		echo "gofmt:   tree differently, so the fallback would name files CI considers correct." >&2; \
		exit 2; \
	fi; \
	resolved="$$("$$goroot/bin/go" version 2>/dev/null | awk '{print $$3}')"; \
	if [ "$$resolved" != "go$$version" ]; then \
		echo "gofmt: REFUSING — go.mod declares go$$version, but the resolved toolchain reports $$resolved." >&2; \
		echo "gofmt:   GOTOOLCHAIN did not switch, so the gate would run the wrong gofmt." >&2; \
		exit 2; \
	fi; \
	case '$(1)' in \
	check) \
		unformatted="$$("$$goroot/bin/gofmt" -l .)"; \
		if [ -n "$$unformatted" ]; then \
			echo "gofmt: these files are not formatted (run \`make fmt\`):" >&2; \
			echo "$$unformatted" >&2; \
			exit 1; \
		fi; \
		echo "fmt-check: the tree is formatted under go$$version's gofmt"; \
		;; \
	write) \
		"$$goroot/bin/gofmt" -l -w .; \
		;; \
	esac
endef

# The gate: name every file the declared toolchain's gofmt would rewrite, and
# exit non-zero if there is one. This is CI's `Format (gofmt)` step — the
# workflow invokes this target rather than restating the command, so the
# developer's gate and CI's gate are one thing
# (.abcd/development/principles/one-canonical-primitive.md).
fmt-check:
	$(call pinned_gofmt,check)

# The remedy: rewrite them, with the same binary the gate judged them by.
fmt:
	$(call pinned_gofmt,write)

vet:
	go vet ./...

# Deterministic gate for the .abcd/work/reviews/ charter (RD001-RD003) — a
# stopgap until these codes land in internal/core/lint. Needs full git history
# (RD002 is append-only over committed history): on a shallow checkout the
# script refuses (exit 2) rather than pass vacuously with nothing covered. The
# cases run first, as in lint-issues: a gate nobody has watched fail is an
# enforcement claim with no evidence behind it.
lint-reviews:
	@bash scripts/check-reviews-cases.sh
	@bash scripts/check-reviews.sh

# AI-attribution gate (AGENTS.md § Attribution). Checks the commit trailers on
# this branch against the default branch; the pull-request BODY half runs only in
# CI, where the body exists. Not a preflight prerequisite — preflight guards a
# push, and the body it must agree with is not written until the PR is opened.
check-attribution:
	@bash scripts/check-attribution.sh commits origin/main HEAD
	@bash scripts/check-attribution-cases.sh

# Deterministic drift gate for the .abcd/development design record (first slice
# of internal/core/lint). Blocking: any record drift (stale tool names, dropped
# concepts, lifecycle or reference breakage) fails preflight and CI.
record-lint:
	@go run ./cmd/record-lint

# Deterministic issue-resolution gate (iss-2608241347321757). RS001: a
# `Resolves: iss-N` trailer must be accompanied by that record entering a terminal
# folder (resolved/ or wontfix/) in the same change, so resolution lands INSIDE
# the fix and no post-merge step
# exists to forget. RS002/RS003: a resolved_by.commit sha must name a commit
# that is actually reachable — `--commit` is shape-checked only, and the repo
# allows squash and rebase merges, either of which rewrites a cited branch sha
# out of existence. RS005: a `Delivers: itd-N` trailer must be accompanied by
# that intent entering intents/shipped/ in the same change — the intent-store
# twin of RS001 (itd-2609111003026787). The cases run first: a gate nobody has watched fail is an
# enforcement claim with no evidence behind it. Needs full git history, like
# lint-reviews: on a shallow checkout the scripts refuse (exit 2) rather than
# report unfetched commits as ledger violations.
lint-issues:
	@bash scripts/check-issue-resolution-cases.sh
	@bash scripts/check-issue-resolution.sh ledger HEAD
	@bash scripts/check-issue-resolution.sh commits origin/main HEAD

# Deterministic append-only gate for .abcd/work/DECISIONS.md
# (iss-2608271804494867). The ledger declares itself append-only and newest-last,
# and nothing enforced it; the five backwards date steps already in the file are
# historical and are NOT repaired, because reordering a committed append-only log
# is the one thing append-only forbids. Position, not date order, is the first
# rule: a back-dated entry appended at the tail is honest, an entry written above
# existing ones is not. Three rules, each per-commit against that commit's own
# parents:
#
#   DA001 position    — an added line lands after the last line the parent had.
#   DA002 preservation — no committed line below the header is removed, against
#                        EVERY parent. Position alone only fires while content
#                        SURVIVES below the addition, so a rewrite that reaches
#                        end-of-file, and its truncate-then-restore cousin, slip
#                        past it entirely.
#   DA003 merge authors nothing — a merge's ledger holds a line no more times
#                        than its parents hold it between them. Merges cannot be
#                        skipped (a forged resolution is an unchecked write path)
#                        and DA001 cannot be applied to them (the ledger is
#                        merge=union, and the driver's interleaving leaves the
#                        result a tail extension of neither side), so the count
#                        bound is what holds. Set membership alone was blind to a
#                        merge that DUPLICATES a committed decision at the top.
#   DA004 the ledger is text — no commit introduces a NUL byte. One NUL makes git
#                        call the file binary, which empties the diff of hunks and
#                        silently disarms every rule above, permanently. The diffs
#                        are read with --text so that cannot happen; DA004 keeps
#                        the byte out of the file, which --text does not.
#
# Both operations the gate legitimately refuses — redacting a leaked line, and
# the ledger's own planned graduation to per-file decisions/ — are deliberate
# gate-edit-and-review changes, not escape hatches: see the script's header.
#
# The cases run first, as in lint-reviews and lint-issues: a gate nobody has
# watched fail is an enforcement claim with no evidence behind it. Needs full git
# history, like its siblings: on a shallow checkout the script refuses (exit 2)
# rather than read every append as a whole-file add.
lint-decisions:
	@bash scripts/check-decisions-append-cases.sh
	@bash scripts/check-decisions-append.sh commits origin/main HEAD

# Deterministic docs-currency gate (itd-60): the same internal/core/lint engine,
# driven over docs/ and the repo root via the transport-agnostic `abcd docs lint`
# verb. Blocking: change-narration in a doc body, a broken relative link, or a
# stray root markdown file fails preflight and CI.
docs-lint:
	@go run ./cmd/abcd docs lint

# Site-render gate (iss-2608241845109280). The site renders the RECORD as well as
# docs/ — 852 records at the 0.6.4 cut — and the renderer supports a fixed
# markdown subset, refusing anything it would otherwise pass through unrendered.
# So a record is a site input, and a malformed one breaks the render. Nothing
# caught that before this target existed: the other lint gates read records but
# never render them, and site-screenshots.yml is path-filtered to the generator
# and docs/, so a records-only pull request changed the rendered page and
# triggered no audit. It reached a release, where `release.yml`'s site job failed
# AFTER the binaries had published.
#
# Builds into a throwaway directory, then runs `site check` over it before
# discarding it. Build alone answers "does it render"; the publish gates
# (provenance, hero, banned-tokens, snippets, baseline, mobile, figure-labels)
# are what release.yml's post-publish site job runs, so a change that renders but
# trips a check gate would otherwise pass every pre-publish gate and fail only
# AFTER the binaries ship. `site check` needs no mkdocs — it excludes docs/ — so a
# plain build output suffices. Roughly ten seconds, almost all of it writing files.
site-render:
	@rm -rf .abcd/.work.local/scratch/site-render-check
	@go run ./cmd/abcd site build --out .abcd/.work.local/scratch/site-render-check >/dev/null
	@go run ./cmd/abcd site check --out .abcd/.work.local/scratch/site-render-check >/dev/null
	@rm -rf .abcd/.work.local/scratch/site-render-check
	@echo "site-render: the record and docs render and pass the site gates"

# Propagate the pinned action refs in the committed release workflows back into
# the scaffold templates they were rendered from (iss-209). Dependabot only ever
# edits the rendered workflow, which breaks the self-scaffold parity test; this
# is the one-way fix — re-rendering the template over the workflow would revert
# the bump instead.
#
# Nothing in CI calls either target. The drift is GATED by `go test`
# (TestSelfScaffoldParity and TestSyncRepoPinsIsCleanToday, both under
# preflight), which is what fails a bump that only landed on the workflow;
# these are the human's read-only look at it and the one-command fix.
scaffold-sync:
	@go run ./cmd/scaffold-sync

scaffold-sync-check:
	@go run ./cmd/scaffold-sync -check

# Pre-push gate (invoked by .githooks/pre-push): the five lint gates
# (lint-reviews, lint-issues, lint-decisions, record-lint, docs-lint), the
# site-render gate and both tagged eval lanes (smoke, evals-cold-reading) as
# prerequisites, then build, vet, test,
# and race-enabled internal tests natively. CI's check job runs those same four
# Go steps plus the `fmt-check` format gate this target does not, so run
# `make fmt-check` separately before pushing. Host-native `go build` (not the
# cross-compiling build target) because it mirrors CI.
#
# The eval lanes are prerequisites because the untagged `go test ./...` step
# below cannot reach them: every eval file carries a build tag, so a defect in
# the read-block eval — the only component capable of falsifying the assembler's
# firewall — used to pass every local gate and surface only in CI, if at all
# (iss-2608311632382737). That was not hypothetical: a path-elision defect in
# the amnesia eval's own guard was unsatisfiable wherever the process temp
# directory is the Linux one, so it landed green here and was found by an
# adversarial review rather than by a gate.
#
# Both lanes are named even though `smoke` compiles a superset of
# `evals-cold-reading`'s files: they are separate tag sets, so a cold-reading
# file reaching for a smoke-only helper compiles under one and not the other,
# which is the split CI's two jobs cover. About five seconds each on a warm
# cache, against roughly a minute for the gates already here.
preflight: lint-reviews lint-issues lint-decisions record-lint docs-lint site-render smoke evals-cold-reading
	go build ./...
	go vet ./...
	go test ./...
	go test -race ./internal/...

clean:
	rm -rf $(BINDIR)
