#!/usr/bin/env bash
# Deterministic gate for the .abcd/work/reviews/ charter (RD001-RD004).
#
# Stopgap: enforces the machine-checkable half of the reviews-folder charter
# until these codes are implemented in `internal/core/lint` (Go). The semantic
# half (provenance discriminator, "not a shadow backlog") is not mechanisable
# and stays a convention. This script is where the RD codes are defined; the
# charter they enforce is .abcd/work/reviews/README.md.
set -euo pipefail

# Every git probe below is fail-closed: a swallowed error reads exactly like a
# clean history, and a gate that cannot tell them apart is a false green.
rc=0
toplevel="$(git rev-parse --show-toplevel 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
  echo "check-reviews: not a git repository (git rev-parse --show-toplevel exit $rc) — refusing rather than reporting a vacuous pass:" >&2
  echo "$toplevel" >&2
  exit 2
fi
cd "$toplevel"

# An unborn HEAD is legitimate — a freshly scaffolded repo has no committed
# history at all — so it is the one git condition that is not a fault. Say loudly
# that RD002 covered nothing, so the pass cannot be mistaken for a verdict.
if ! git rev-parse --verify -q HEAD >/dev/null 2>&1; then
  echo "check-reviews: unborn HEAD — no committed history — RD002 covers nothing (nothing has been committed yet)"
  exit 0
fi

# A shallow checkout blinds RD002: past the graft boundary every file reports
# as newly added, so the append-only check covers nothing and passes vacuously
# — a false green, the worse polarity. Report the environment fault as itself.
rc=0
shallow="$(git rev-parse --is-shallow-repository 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
  echo "check-reviews: git rev-parse --is-shallow-repository failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
  echo "$shallow" >&2
  exit 2
fi
case "$shallow" in
true)
  echo "check-reviews: shallow checkout — RD002 (append-only over committed history) cannot see past the graft; run 'git fetch --unshallow' first (CI checks out with fetch-depth: 0)." >&2
  exit 2
  ;;
false) ;;
*)
  echo "check-reviews: unexpected git rev-parse --is-shallow-repository output \"$shallow\" — refusing rather than guessing." >&2
  exit 2
  ;;
esac

ROOT=".abcd/work/reviews"
fail=0
note() { echo "  $1" >&2; }

[ -d "$ROOT" ] || { echo "check-reviews: no $ROOT — nothing to check"; exit 0; }

# RD001 — review directory shape: each directory under reviews/ is named
# <YYYY-MM-DD>-<kebab-scope> and carries a 00-summary.md. (reviews/README.md,
# the charter itself, sits at the root and is exempt.)
for d in "$ROOT"/*/; do
  [ -d "$d" ] || continue
  base="$(basename "$d")"
  # Semantic-gate receipt directories are sha-keyed (.abcd/work/reviews/<40-hex>/
  # <gate>.json, iss-35) — a distinct artifact class from the dated human-review
  # dirs this charter governs, with their own integrity check (the receipt_gate
  # record-lint rule + release.yml attestation). Exempt them from RD001's
  # <date>-<scope>/00-summary.md shape.
  printf '%s' "$base" | grep -Eq '^[0-9a-f]{40}$' && continue
  printf '%s' "$base" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$' \
    || { note "RD001 $d — directory name must be <YYYY-MM-DD>-<kebab-scope>"; fail=1; }
  [ -f "${d}00-summary.md" ] \
    || { note "RD001 $d — missing required 00-summary.md"; fail=1; continue; }
  # RD004 — the pin (itd-28): the summary's leading frontmatter block names the
  # commit the review read, `review_of_commit: <full sha>`, so the status board
  # can say how far the default branch has moved since. The reading is the
  # board's own (internal/core/reviews.Pin): a block opened on line 1 and closed
  # by the next `---`, the first `review_of_commit` key in it, and a bare full
  # object name in git's lowercase hex, optionally followed by a comment. A
  # folder from before the rule is named as legacy and not refused; the legacy
  # set is closed, and a folder added after the rule cannot join it.
  case "$base" in
  2026-07-06-plan-consistency | 2026-07-07-roadmap-consistency | 2026-08-19-pr-294-null-predicate)
    echo "  RD004 legacy $d — predates the review_of_commit pin; named, not refused"
    continue
    ;;
  esac
  pin_line=""
  lineno=0
  opened=0
  while IFS= read -r line || [ -n "$line" ]; do
    lineno=$((lineno + 1))
    line="${line%$'\r'}"
    [ "$lineno" -eq 1 ] && line="${line#$'\xef\xbb\xbf'}"
    trimmed="${line%"${line##*[! $'\t']}"}"
    if [ "$lineno" -eq 1 ]; then
      [ "$trimmed" = "---" ] || break
      opened=1
      continue
    fi
    if [ "$trimmed" = "---" ]; then
      opened=2
      break
    fi
    if [ -z "$pin_line" ] && printf '%s' "$line" | grep -Eq '^review_of_commit[ 	]*:'; then
      pin_line="$line"
    fi
  done <"${d}00-summary.md"
  if [ "$opened" -ne 2 ] || [ -z "$pin_line" ]; then
    note "RD004 $d — 00-summary.md names no review_of_commit in its frontmatter (the full sha of the commit the review read)"
    fail=1
  elif ! printf '%s' "$pin_line" | grep -Eq '^review_of_commit[ 	]*:[ 	]*([0-9a-f]{40}|[0-9a-f]{64})([ 	]+#.*)?[ 	]*$'; then
    note "RD004 $d — review_of_commit is not a bare full sha in lowercase hex"
    fail=1
  fi
done

# RD002 — append-only: no post-creation modify/rename/delete in committed
# history. ONE history pass over the whole reviews root, not a probe per file: a
# pathspec-scoped log can never report an R (rename detection needs both sides in
# view, and a pathspec hides one of them), and a file that was deleted or renamed
# away has no working-tree path left to probe from. Scanning the history instead
# of the tree sees all three shapes.
rc=0
history="$(git log --format='' --name-status --diff-filter=DMRT -- "$ROOT" 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
  echo "check-reviews: git log over $ROOT failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
  echo "$history" >&2
  exit 2
fi
# Each emitted line is a status then one path, or — for an R — a status then the
# old and new path, tab-separated. Both sides of a rename are review files in
# their own right, so both columns are checked. The population is the dated
# review dirs only; reviews/README.md and the sha-keyed receipt dirs are outside
# it and stay mutable.
while IFS="$(printf '\t')" read -r status old new; do
  [ -n "$status" ] || continue
  for p in "$old" "$new"; do
    [ -n "$p" ] || continue
    printf '%s' "$p" | grep -Eq '^\.abcd/work/reviews/[0-9]{4}-[0-9]{2}-[0-9]{2}-[^/]+/.*\.md$' || continue
    note "RD002 $p — review files are append-only; changed after creation (history status $status)"
    fail=1
  done
done <<EOF
$history
EOF

# Review files live inside dated dirs (depth >=2); the root README is mutable.
# (while-read, not mapfile — portable to the bash 3.2 that macOS ships.)
files=()
while IFS= read -r line; do files+=("$line"); done < <(find "$ROOT" -mindepth 2 -type f -name '*.md' | sort)

for f in "${files[@]:-}"; do
  [ -n "$f" ] || continue
  # RD003 — path hygiene: repo-relative only, no absolute personal paths.
  if grep -nE '/Users/|/home/[a-z]|C:\\Users' "$f" >/dev/null 2>&1; then
    note "RD003 $f — contains an absolute personal path (use repo-relative)"; fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  echo "check-reviews: FAILED — reviews-charter discipline (RD001-RD004)" >&2
  exit 1
fi
echo "check-reviews: OK — $ROOT (${#files[@]} review files), RD001-RD004 clean"
