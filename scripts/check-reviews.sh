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
  # A review folder, and the summary in it, is committed content, never a
  # pointer elsewhere: the board does not follow a symlinked entry
  # (internal/core/reviews.readEntries), so a symlinked folder is not on it at all and
  # a symlinked summary reads as unpinned. `[ -d ]` and `[ -f ]` follow a
  # symlink, so the gate asks `[ -L ]` first rather than vouch for either.
  if [ -L "${d%/}" ]; then
    note "RD001 $d — a review folder is a directory, not a symlink"
    fail=1
    continue
  fi
  # Semantic-gate receipt directories are sha-keyed (.abcd/work/reviews/<40-hex>/
  # <gate>.json, iss-35) — a distinct artifact class from the dated human-review
  # dirs this charter governs, with their own integrity check (the receipt_gate
  # record-lint rule + release.yml attestation). Exempt them from RD001's
  # <date>-<scope>/00-summary.md shape.
  printf '%s' "$base" | grep -Eq '^[0-9a-f]{40}$' && continue
  printf '%s' "$base" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$' \
    || { note "RD001 $d — directory name must be <YYYY-MM-DD>-<kebab-scope>"; fail=1; }
  if [ -L "${d}00-summary.md" ]; then
    note "RD001 $d — 00-summary.md is a symlink; the summary is a regular file"
    fail=1
    continue
  fi
  [ -f "${d}00-summary.md" ] \
    || { note "RD001 $d — missing required 00-summary.md"; fail=1; continue; }
  # RD004 — the pin (itd-28): the summary's leading frontmatter block names the
  # commit the review read, `review_of_commit: <full sha>`, so the status board
  # can say how far the default branch has moved since. The reading is the
  # board's own (internal/core/reviews.summaryPin): a block opened on line 1 and closed
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
  # The board reads at most maxSummaryBytes of a summary (1 MiB,
  # internal/core/reviews) and shows a larger one unpinned, so the gate
  # refuses it rather than reading a pin the board never will.
  size=$(($(wc -c <"${d}00-summary.md")))
  if [ "$size" -gt 1048576 ]; then
    note "RD004 $d — 00-summary.md is $size bytes, past the 1 MiB the board reads a summary within"
    fail=1
    continue
  fi
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
  # bash `read` drops a NUL byte (3.2 truncates the line at it), so the loop
  # above can see a clean block where reviews.summaryPin sees none. The block is lines
  # 1..lineno, the closing fence included; any NUL in those bytes is refused.
  nul=0
  if [ "$opened" -eq 2 ]; then
    block_bytes=$(($(head -n "$lineno" "${d}00-summary.md" | wc -c)))
    text_bytes=$(($(head -n "$lineno" "${d}00-summary.md" | LC_ALL=C tr -d '\000' | wc -c)))
    [ "$block_bytes" -eq "$text_bytes" ] || nul=1
  fi
  if [ "$nul" -ne 0 ]; then
    note "RD004 $d — 00-summary.md's frontmatter carries a NUL byte, which the board reads as no pin"
    fail=1
  elif [ "$opened" -ne 2 ] || [ -z "$pin_line" ]; then
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
#
# ONE exception, and a narrow one (iss-2610290855523061): a commit that removes
# a person's login or name from a review. It is exempt only when ALL hold — its
# message carries the trailer `Review-Redaction: names`; every review file it
# touches is modified in place (M; a D, R or T is never exempt); and each of
# those changes is a pure substitution, every removed span name-shaped and every
# inserted span exactly one member of REDACTION_PLACEHOLDERS. Anything else in
# the same file — a reworded word, an added or dropped sentence, a re-spaced
# line — leaves the commit under RD002 like any other edit.
#
# What that guarantees, plainly: without the private name list (in CI, in a
# fresh clone) the gate cannot tell a name from any other word, so ANY single
# word, or up to three capitalised words, may be replaced by a redaction marker.
# The marker is what tells a reader the text was redacted, which is why every
# member of the set reads as one and none reads as ordinary prose. Only where
# the private list exists is a removed span also proved to be a name.
REDACTION_TRAILER="Review-Redaction: names"
# The fixed public set of redaction markers. Each visibly reads as a redaction,
# never as content a reader could take for the original; extend the set only in
# its own reviewed change.
REDACTION_PLACEHOLDERS=(
  "(the product thinker's login)"
  "<personal account>"
  "<login>"
  "<name>"
  "<redacted>"
)
# Name-shaped: a forge login (alphanumerics and hyphens, at most 39, not
# starting with a hyphen), optionally @-prefixed and optionally in code
# backticks; or one to three capitalised words, a person's name.
NAME_SHAPE='^(`@?[A-Za-z0-9][A-Za-z0-9-]{0,38}`|@?[A-Za-z0-9][A-Za-z0-9-]{0,38}|[A-Z][A-Za-z'"'"'-]*( [A-Z][A-Za-z'"'"'-]*){0,2})$'
# The word-diff tokens: a run of word characters, or any one other non-blank
# character, so a login inside a path or a URL is its own token.
WORD_RE='[A-Za-z0-9_-]+|[^[:space:]]'
# The per-machine private name list (internal/core/banlist.PrivateRelPath).
# Where it exists, a removed span must also match one of its entries — the
# public shape check above admits any single word, the list admits only names.
# Absent (CI, a fresh clone), the public check is the whole test.
PRIVATE_NAMES=".abcd/.work.local/private-names.txt"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

is_placeholder() {
  local p
  for p in "${REDACTION_PLACEHOLDERS[@]}"; do
    [ "$1" = "$p" ] && return 0
  done
  return 1
}

# load_private reads the private list once, from THIS checkout's file only —
# unlike .githooks/pre-commit, it never falls back to the primary checkout's
# store, so a linked worktree without its own copy gets the public check alone.
# The entry format is the hook's: line 1 exactly `# abcd-banlist: keyed` makes
# every entry KEY<space-or-tab>PATTERN, otherwise each line is one whole-line
# pattern; blank and `#` lines are skipped; patterns are grep -iE. A store that
# cannot be read for what it is refuses the run — never an entry's content, only
# its line number, is ever printed.
private_state=unread
private_pats=()
private_lines=()
load_private() {
  [ "$private_state" = unread ] || return 0
  private_state=absent
  [ -e "$PRIVATE_NAMES" ] || [ -L "$PRIVATE_NAMES" ] || return 0
  if [ -L "$PRIVATE_NAMES" ] || [ ! -f "$PRIVATE_NAMES" ]; then
    echo "check-reviews: $PRIVATE_NAMES is not a regular file — refusing rather than skipping the private name check." >&2
    exit 2
  fi
  local raw line keyed=0 n=0 key pat
  while IFS= read -r raw || [ -n "$raw" ]; do
    n=$((n + 1))
    line="${raw%$'\r'}"
    [ "$n" -eq 1 ] && line="${line#$'\xef\xbb\xbf'}"
    line="${line%"${line##*[! $'\t']}"}"
    line="${line#"${line%%[! $'\t']*}"}"
    if [ "$n" -eq 1 ] && [ "$line" = "# abcd-banlist: keyed" ]; then
      keyed=1
      continue
    fi
    case "$line" in '' | \#*) continue ;; esac
    pat="$line"
    if [ "$keyed" -eq 1 ]; then
      key="${line%%[ $'\t']*}"
      pat="${line#"$key"}"
      pat="${pat#"${pat%%[! $'\t']*}"}"
      if [ -z "$pat" ] || ! printf '%s' "$key" | grep -Eq '^[A-Za-z0-9][A-Za-z0-9._/-]*$'; then
        echo "check-reviews: $PRIVATE_NAMES line $n is not a KEY<space-or-tab>PATTERN entry — refusing; its content is withheld." >&2
        exit 2
      fi
    fi
    private_pats+=("$pat")
    private_lines+=("$n")
  done <"$PRIVATE_NAMES"
  private_state=present
}

# on_private_list <span> reports whether the span, its backticks and @ shed,
# matches an entry of the private list. A pattern grep refuses is a fault.
on_private_list() {
  local core="$1" i rc
  core="${core#\`}"
  core="${core%\`}"
  core="${core#@}"
  i=0
  while [ "$i" -lt "${#private_pats[@]}" ]; do
    rc=0
    grep -Eiq -e "${private_pats[$i]}" <<<"$core" 2>/dev/null || rc=$?
    case "$rc" in
    0) return 0 ;;
    1) ;;
    *)
      echo "check-reviews: $PRIVATE_NAMES line ${private_lines[$i]} is a pattern grep -E refuses — refusing; its content is withheld." >&2
      exit 2
      ;;
    esac
    i=$((i + 1))
  done
  return 1
}

# pure_redaction <sha> <path> reports whether the commit's change to the file is
# a pure name-to-placeholder substitution; on refusal it sets why_not. Removed
# spans are never printed: they are the names being taken out.
#
# The word diff is read whole (one hunk spanning the file) and both sides are
# rebuilt from it and compared with the real blobs byte for byte, because a
# word diff ignores whitespace between tokens: without the rebuild a re-spaced
# or re-wrapped line would pass as unchanged. A side the stream does not rebuild
# exactly — a file without a final newline among them — is refused.
why_not=""
pure_redaction() {
  local sha="$1" p="$2" rc n rem add pairs=0
  rc=0
  git cat-file blob "$sha^1:$p" >"$work/old" 2>"$work/err" || rc=$?
  [ "$rc" -eq 0 ] && { git cat-file blob "$sha:$p" >"$work/new" 2>"$work/err" || rc=$?; }
  if [ "$rc" -ne 0 ]; then
    echo "check-reviews: git cat-file for $p at $sha failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
    cat "$work/err" >&2
    exit 2
  fi
  n=$(($(wc -l <"$work/old") + $(wc -l <"$work/new") + 1))
  rc=0
  git diff --no-color --no-ext-diff --no-textconv -U"$n" --word-diff=porcelain --word-diff-regex="$WORD_RE" "$sha^1..$sha" -- "$p" >"$work/wd" 2>"$work/err" || rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "check-reviews: git diff for $p at $sha failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
    cat "$work/err" >&2
    exit 2
  fi
  # Porcelain: ' ' common text, '-' removed, '+' added, '~' end of line. A '-'
  # must be followed at once by its '+', and a '+' preceded by its '-': a bare
  # removal or insertion is not a substitution. Pairs go out two lines each.
  rc=0
  LC_ALL=C awk -v old="$work/old.re" -v new="$work/new.re" -v pairs="$work/pairs" '
    BEGIN { state = "head"; prev = ""; hunks = 0; bad = 0
            printf "" > old; printf "" > new; printf "" > pairs }
    state == "head" {
      if ($0 ~ /^@@ /) { state = "hunk"; hunks++; next }
      if ($0 ~ /^(diff --git |index |--- |\+\+\+ )/) next
      bad = 1; exit
    }
    {
      c = substr($0, 1, 1); t = substr($0, 2)
      if (c == " ") { if (prev == "-") { bad = 1; exit }; printf "%s", t > old; printf "%s", t > new }
      else if (c == "-") { if (prev == "-") { bad = 1; exit }; printf "%s", t > old; rem = t }
      else if (c == "+") { if (prev != "-") { bad = 1; exit }; printf "%s", t > new; print rem > pairs; print t > pairs }
      else if (c == "~") { if (prev == "-") { bad = 1; exit }; printf "\n" > old; printf "\n" > new }
      else { bad = 1; exit }
      prev = c
    }
    END { if (bad || hunks != 1 || prev == "-") exit 1 }
  ' "$work/wd" || rc=$?
  if [ "$rc" -ne 0 ] || ! cmp -s "$work/old" "$work/old.re" || ! cmp -s "$work/new" "$work/new.re"; then
    why_not="the change is not a pure substitution of spans"
    return 1
  fi
  load_private
  while IFS= read -r rem && IFS= read -r add; do
    pairs=$((pairs + 1))
    if ! grep -Eq -- "$NAME_SHAPE" <<<"$rem"; then
      why_not="a removed span is not name-shaped"
      return 1
    fi
    if ! is_placeholder "$add"; then
      why_not="an inserted span is not one of the fixed placeholders"
      return 1
    fi
    if [ "$private_state" = present ] && ! on_private_list "$rem"; then
      why_not="a removed span is not on the private name list"
      return 1
    fi
  done <"$work/pairs"
  if [ "$pairs" -eq 0 ]; then
    why_not="the change substitutes nothing"
    return 1
  fi
  return 0
}

rc=0
history="$(git -c core.quotePath=false log --format='commit %H' --name-status --diff-filter=DMRT -- "$ROOT" 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
  echo "check-reviews: git log over $ROOT failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
  echo "$history" >&2
  exit 2
fi
# Each commit opens with `commit <sha>`; each line after it is a status then one
# path, or — for an R — a status then the old and new path, tab-separated. Both
# sides of a rename are review files in their own right, so both columns are
# checked. The population is the dated review dirs only; reviews/README.md and
# the sha-keyed receipt dirs are outside it and stay mutable. A commit's review
# entries are gathered first, because the exemption is judged per commit.
review_re='^\.abcd/work/reviews/[0-9]{4}-[0-9]{2}-[0-9]{2}-[^/]+/.*\.md$'
c_sha=""
c_status=()
c_path=()

judge_commit() {
  [ "${#c_path[@]}" -gt 0 ] || return 0
  local i reason="" trailers
  rc=0
  trailers="$(git log -1 --format='%(trailers:only,unfold)' "$c_sha" 2>&1)" || rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "check-reviews: reading the trailers of $c_sha failed (exit $rc) — refusing rather than reporting a vacuous pass:" >&2
    echo "$trailers" >&2
    exit 2
  fi
  if grep -Fxq -- "$REDACTION_TRAILER" <<<"$trailers"; then
    i=0
    while [ "$i" -lt "${#c_path[@]}" ]; do
      if [ "${c_status[$i]}" != M ]; then
        reason="the $REDACTION_TRAILER trailer covers an in-place edit only, not history status ${c_status[$i]}"
        break
      fi
      case "${c_path[$i]}" in
      \"*)
        reason="the $REDACTION_TRAILER trailer cannot cover a path git quotes"
        break
        ;;
      esac
      if ! pure_redaction "$c_sha" "${c_path[$i]}"; then
        reason="the $REDACTION_TRAILER trailer is present, but $why_not"
        break
      fi
      i=$((i + 1))
    done
    if [ -z "$reason" ]; then
      for i in "${c_path[@]}"; do
        echo "  RD002 redaction $i — a pure name redaction in ${c_sha:0:12}; allowed"
      done
      return 0
    fi
  fi
  i=0
  while [ "$i" -lt "${#c_path[@]}" ]; do
    note "RD002 ${c_path[$i]} — review files are append-only; changed after creation (history status ${c_status[$i]})${reason:+; $reason}"
    i=$((i + 1))
  done
  fail=1
}

while IFS="$(printf '\t')" read -r status old new; do
  [ -n "$status" ] || continue
  case "$status" in
  "commit "*)
    judge_commit
    c_sha="${status#commit }"
    c_status=()
    c_path=()
    continue
    ;;
  esac
  for p in "$old" "$new"; do
    [ -n "$p" ] || continue
    # core.quotePath=false keeps a non-ASCII name raw, but git still quotes a
    # path holding a `"`, a backslash or a control byte. The log is scoped to
    # the reviews root, so a quoted path is taken as in scope and refused —
    # skipping it as unmatched is the vacuous pass this gate exists to avoid.
    case "$p" in
    \"*) ;;
    *) grep -Eq -- "$review_re" <<<"$p" || continue ;;
    esac
    c_status+=("$status")
    c_path+=("$p")
  done
done <<EOF
$history
EOF
judge_commit

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
