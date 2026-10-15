#!/usr/bin/env bash
# The last step of every cloud session, finished or stopped
# (spc-2610120941441282 step 8, itd-2610120738075547 C6, decision 4). In order:
#
#   1. refuse, writing nothing, unless the item is one record id, the state is
#      handed-back or stopped, and the three texts are given;
#   2. compose the report: a short frontmatter (item, state, channel,
#      finished_at) and three sections, what the session did, what failed and
#      what it needs from the person. It passes
#      `go run ./cmd/abcd lint outbound --label report` or nothing is written:
#      the finding is printed, so the session can fix its text and run this
#      again;
#   3. write it as <report_dir>/<item-id>.md (cloud.conf) and commit it ALONE,
#      as the branch's last commit: the session's other changes, staged or
#      not, stay out of it. The commit carries `Assisted-by: <value>`, the
#      value --assisted-by gives or, without it, the one the session's own
#      commits since origin's default branch carry (the latest). It is
#      authored under the checkout's configured identity; this script never
#      sets one (the setup does, decision 18);
#   4. push HEAD to <wip_prefix><item-id> on origin through the committed
#      hooks: never --no-verify, and only while core.hooksPath names the
#      checkout's .githooks, since a push the hook never sees is a skipped
#      hook. The connector's prefix is read through connector.sh (C12);
#   5. only when that fails (the commit, the hooks or the push), post the same
#      report, with `channel: issue-comment` and why the push failed, as a
#      comment on cloud.conf's tracking_issue through the forge's REST
#      interface: `gh api --method POST repos/<owner>/<name>/issues/<n>/comments
#      --input -`, after the same outbound check. Owner and name come from
#      origin's URL, read strictly as a plain github.com one (forge-repo.sh).
#      With tracking_issue empty the fallback refuses by name ("no tracking
#      issue configured"); it never posts anywhere else;
#   6. when neither channel takes the report, print it on standard output, so
#      the session's transcript holds it as a last resort, and exit 1.
#
# The channel field says where the report went: `branch` in the committed
# file, `issue-comment` in the comment.
#
# Usage:
#   hand-back.sh --item <record-id> --state <handed-back|stopped>
#                --did <text> --failed <text> --needs <text>
#                [--assisted-by <Vendor>:<model-version>] [--connector <name>]
#   (default connector: the only one declared)
# Exit 0 the report reached the branch or, failing that, the tracking issue;
# 1 refused, or neither channel took it; 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

# One record id: the hook's own shape for what follows a work-in-progress
# prefix (.githooks/pre-push, wip_id_re).
ITEM_RE='^((iss|itd|spc)-[0-9]{16}|(iss|itd)-[0-9]{1,4})$'
# The model half of the attribution gate's TRAILER_RE (scripts/check-attribution.sh).
ASSISTED_RE='^[A-Za-z][A-Za-z0-9._-]*:[A-Za-z0-9._-]+(\[[A-Za-z0-9._-]+\])?$'
SAFE_RE='^[A-Za-z0-9._/-]+$'

usage() {
	echo "usage: hand-back.sh --item <record-id> --state <handed-back|stopped> --did <text> --failed <text> --needs <text> [--assisted-by <Vendor>:<model-version>] [--connector <name>]" >&2
	exit 2
}

refuse() {
	echo "hand-back: REFUSING $1 Nothing was committed or sent." >&2
	exit 1
}

# clean drops every control character but the tab and the newline.
clean() {
	printf '%s' "$1" | LC_ALL=C tr -d '\000-\010\013-\037\177'
}

item="" state="" did="" failed="" needs="" assisted="" connector=""
seen=" "
while [ $# -gt 0 ]; do
	case "$1" in
	--item | --state | --did | --failed | --needs | --assisted-by | --connector)
		[ $# -ge 2 ] || usage
		case "$seen" in
		*" $1 "*) usage ;;
		esac
		seen="$seen$1 "
		case "$1" in
		--item) item="$2" ;;
		--state) state="$2" ;;
		--did) did="$2" ;;
		--failed) failed="$2" ;;
		--needs) needs="$2" ;;
		--assisted-by) assisted="$2" ;;
		--connector) connector="$2" ;;
		esac
		shift 2
		;;
	*) usage ;;
	esac
done
for flag in --item --state --did --failed --needs; do
	case "$seen" in
	*" $flag "*) ;;
	*) usage ;;
	esac
done

# --- 1. What is handed back --------------------------------------------------

[[ "$item" =~ $ITEM_RE ]] ||
	refuse "--item: the item is not a record id (iss-, itd- or spc- and sixteen digits, or an ordinal iss- or itd- id)."
case "$state" in
handed-back | stopped) ;;
*) refuse "--state: the state is handed-back (the work is done) or stopped (a check or a question stopped it)." ;;
esac
did="$(clean "$did")"
failed="$(clean "$failed")"
needs="$(clean "$needs")"
for pair in "did:$did" "failed:$failed" "needs:$needs"; do
	[ -n "$(printf '%s' "${pair#*:}" | tr -d '[:space:]')" ] ||
		refuse "--${pair%%:*}: it is empty; say it in words (\"Nothing.\" is an answer)."
done

cd "$root"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || refuse "this is not a git checkout."

if [ -z "$connector" ]; then
	connector="$(bash "$here/connector.sh" default)" || exit 1
fi
prefix="$(bash "$here/connector.sh" get "$connector" wip_prefix)" || exit 1
branch="$prefix$item"

# cloud.conf, read as data and never sourced.
conf_value() {
	sed -n "s/^$1=//p" "$here/cloud.conf" | tail -n 1
}
report_dir="$(conf_value report_dir)"
report_dir="${report_dir%/}"
case "/$report_dir/" in
*/../* | */./* | //*) report_dir="" ;;
esac
[[ "$report_dir" =~ $SAFE_RE ]] ||
	refuse "scripts/cloud/cloud.conf: report_dir must be a relative path inside the checkout."
tracking="$(conf_value tracking_issue)"
[ -z "$tracking" ] || [[ "$tracking" =~ ^[1-9][0-9]{0,9}$ ]] ||
	refuse "scripts/cloud/cloud.conf: tracking_issue must be an issue number, or empty."
report="$report_dir/$item.md"

# The disclosure: given, or the session's own latest since the default branch.
if [ -z "$assisted" ]; then
	if base="$(bash "$here/default-branch.sh" 2>/dev/null)" &&
		fork="$(git merge-base "refs/remotes/origin/$base" HEAD 2>/dev/null)"; then
		assisted="$(git log --format='%(trailers:key=Assisted-by,valueonly)' "$fork..HEAD" 2>/dev/null |
			awk 'NF { print; exit }' | tr -d '\r')" || assisted=""
	fi
	[ -n "$assisted" ] ||
		refuse "--assisted-by: no commit of this session carries an Assisted-by: trailer to take it from; pass --assisted-by <Vendor>:<model-version>."
fi
case "$assisted" in
None | abcd:*)
	refuse "--assisted-by: '$assisted' is not a model; a cloud session's report is assisted work, so it names <Vendor>:<model-version>."
	;;
esac
[[ "$assisted" =~ $ASSISTED_RE ]] ||
	refuse "--assisted-by: the value is not <Vendor>:<model-version> (the attribution gate's shape)."

# --- 2. The report, and the outbound check -----------------------------------

tmp="$(mktemp -d "${TMPDIR:-/tmp}/abcd-hand-back.XXXXXX")"
trap '/bin/rm -rf -- "$tmp"' EXIT
finished="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# compose <channel> [<why the push failed>] writes the report on standard output.
compose() {
	printf -- '---\nitem: %s\nstate: %s\nchannel: %s\nfinished_at: %s\n---\n\n' "$item" "$state" "$1" "$finished"
	printf '# Cloud session report: %s\n\n' "$item"
	printf '## What it did\n\n%s\n\n' "$did"
	printf '## What failed\n\n%s\n\n' "$failed"
	printf '## What it needs from the person\n\n%s\n' "$needs"
	if [ $# -ge 2 ]; then
		printf '\n## Why the push failed\n\n'
		printf '%s\n' "$2" | sed 's/^/    /'
	fi
}

# outbound <file> runs the outbound check; its output is in $lint.
lint=""
outbound() {
	local rc=0
	lint="$(go run ./cmd/abcd lint outbound --label report "$1" 2>&1)" || rc=$?
	return "$rc"
}

compose branch >"$tmp/report.md"
rc=0
outbound "$tmp/report.md" || rc=$?
case "$rc" in
0) ;;
1)
	printf '%s\n' "$lint" >&2
	refuse "the report fails the outbound check (go run ./cmd/abcd lint outbound); the finding is above. Fix its text and hand back again."
	;;
*)
	printf '%s\n' "$lint" >&2
	echo "hand-back: the outbound check (go run ./cmd/abcd lint outbound) could not run (exit $rc), so the report was not sent. It follows, so this transcript holds it." >&2
	cat "$tmp/report.md"
	exit 1
	;;
esac

# --- 3. Commit it alone, 4. push it through the hooks -------------------------

why=""
if [ -L "$report_dir" ] || [ -L "$report" ]; then
	why="the report's path, $report, is a symlink; it is never written through"
fi
if [ -z "$why" ]; then
	mkdir -p "$report_dir"
	cat "$tmp/report.md" >"$report"
	printf 'chore(cloud): hand back the report on %s\n\nThe session %s; the report alone, after its work.\n\nAssisted-by: %s\n' \
		"$item" "$([ "$state" = stopped ] && echo stopped || echo handed its work back)" "$assisted" >"$tmp/message"
	if ! out="$(git add -- "$report" 2>&1 && git commit -q --only -F "$tmp/message" -- "$report" 2>&1)"; then
		why="the report could not be committed: $out"
	fi
fi

if [ -z "$why" ]; then
	hooks="$(git config --get core.hooksPath 2>/dev/null)" || hooks=""
	if [ -z "$hooks" ] || [ ! -d "$hooks" ] || [ ! -x .githooks/pre-push ] ||
		[ "$(cd "$hooks" && pwd -P)" != "$(cd .githooks && pwd -P)" ]; then
		why="the committed hooks are not active (core.hooksPath does not name this checkout's .githooks), and a push they never see skips them"
	fi
fi

if [ -z "$why" ]; then
	rc=0
	out="$(GIT_TERMINAL_PROMPT=0 git push origin "HEAD:refs/heads/$branch" 2>&1)" || rc=$?
	if [ "$rc" -eq 0 ]; then
		[ -z "$out" ] || printf '%s\n' "$out" >&2
		echo "hand-back: the report is committed as $(git rev-parse --short HEAD) and pushed with the work to $branch."
		exit 0
	fi
	printf '%s\n' "$out" >&2
	why="the push to $branch failed (exit $rc): $out"
fi

# --- 5. The fallback: the tracking issue -------------------------------------

# Why, as the comment carries it: the last lines, with no path on this
# computer and no control character.
why="$(printf '%s\n' "$why" | tail -n 20)"
why="${why//"$root"/.}"
if [ -n "${HOME:-}" ] && [ "$HOME" != / ]; then
	why="${why//"$HOME"/\~}"
fi
why="$(clean "$why")"
compose issue-comment "$why" >"$tmp/comment.md"

# neither <reason> prints the report and stops: no channel took it.
neither() {
	echo "hand-back: the report reached neither channel: $1. It follows, so this transcript holds it." >&2
	cat "$tmp/comment.md"
	exit 1
}

echo "hand-back: the report could not go with the work ($why)." >&2
[ -n "$tracking" ] ||
	neither "the fallback refuses: no tracking issue configured (scripts/cloud/cloud.conf, tracking_issue)"
repo="$(bash "$here/forge-repo.sh")" || neither "origin names no forge repository to post on"
command -v gh >/dev/null 2>&1 || neither "the forge's command-line tool, gh, is not installed"

rc=0
outbound "$tmp/comment.md" || rc=$?
if [ "$rc" -ne 0 ]; then
	printf '%s\n' "$lint" >&2
	neither "the comment fails the outbound check (go run ./cmd/abcd lint outbound, exit $rc), so it was not posted"
fi

# The request body, {"body": "<the comment>"}, as JSON: backslash, quote and
# tab escaped, lines joined by \n. clean() left no other control character.
{
	printf '{"body": "'
	sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e 's/	/\\t/g' "$tmp/comment.md" |
		awk 'NR > 1 { printf "\\n" } { printf "%s", $0 }'
	printf '"}\n'
} >"$tmp/comment.json"

rc=0
out="$(gh api --method POST "repos/$repo/issues/$tracking/comments" --input - <"$tmp/comment.json" 2>&1)" || rc=$?
if [ "$rc" -ne 0 ]; then
	printf '%s\n' "$out" | tail -n 5 >&2
	neither "the comment on the tracking issue, #$tracking, was not posted (gh exited $rc)"
fi
echo "hand-back: the push failed, so the report was posted as a comment on the tracking issue, #$tracking, instead."
exit 0
