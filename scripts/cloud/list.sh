#!/usr/bin/env bash
# List the cloud sessions (spc-2610120941441282 step 10, itd-2610120738075547
# C8, decision 4), read from the shared remote alone: there is no interface
# that lists hosted sessions. Run on the person's own computer.
#
# It reads three things, and nothing else:
#
#   1. the work-in-progress branches: `refs/heads/<wip_prefix>*` for every
#      prefix a host connector declares (connector.sh, so nothing here names a
#      host, C12), fetched with pruning into refs/remotes/origin/, and the
#      report file on each branch's tip, read with `git show`;
#   2. the reservations: `go run ./cmd/abcd peers --json`, which fetches every
#      `refs/heads/abcd-reserved/*` from origin with pruning (the refspec
#      internal/core/claim uses) and judges each one live, expired, released
#      or unreadable by the one lease rule. Liveness is never re-derived here
#      from a commit message;
#   3. the tracking issue's comments (cloud.conf's tracking_issue), through
#      `gh api` over REST, for sessions whose push failed: a comment is a
#      session's report when its body opens with the report's frontmatter
#      (channel: issue-comment) and its author is the repository's owner, a
#      member or a collaborator, since a cloud session posts under the
#      person's own account and anyone can comment on a public issue. With
#      tracking_issue empty, gh missing or origin not a github.com repository,
#      the list is read from the branches alone and says so once.
#
# One row per item, tab-separated under a header: the item, its branch ("-"
# when none is pushed), its state and the first line of its report. The
# report's first line is the first line of its text after the frontmatter that
# is neither blank nor a heading: the opening of what the session did. A
# report from a comment is marked "(issue #<n>)". When a branch and a comment
# both carry a report, the later finished_at wins. The states:
#
#   handed-back, stopped   from the report;
#   working                a live reservation, and no report;
#   unknown                otherwise (a branch with no report, or one whose
#                          report does not read, and no live reservation).
#
# A reservation names no host, so every live reservation without a report
# lists as working, a person's own included: the remote cannot tell them
# apart.
#
# It never opens a transcript, and it writes nothing in the checkout but the
# fetched remote-tracking refs, with git's record of each fetch (FETCH_HEAD)
# and abcd peers' stamp of when it fetched the reservations, in the local tier
# (.abcd/.work.local/reservation/), which that verb keeps.
#
# Usage: list.sh
# Exit 0 listed (a source that could not be read is named on standard error);
# 1 refused; 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

ITEM_RE='^((iss|itd|spc)-[0-9]{16}|(iss|itd)-[0-9]{1,4})$'
FINISHED_RE='^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
SAFE_RE='^[A-Za-z0-9._/-]+$'
MAX_REPORT_BYTES=65536

if [ $# -ne 0 ]; then
	echo "usage: list.sh" >&2
	exit 2
fi

refuse() {
	echo "list: REFUSING $1" >&2
	exit 1
}
note() {
	echo "list: $1" >&2
}

cd "$root"
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || refuse "this is not a git checkout."
prefixes="$(bash "$here/connector.sh" prefixes)" || exit 1

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

tmp="$(mktemp -d "${TMPDIR:-/tmp}/abcd-cloud-list.XXXXXX")"
trap '/bin/rm -rf -- "$tmp"' EXIT

# One line per thing read: item, kind (b a branch, c a comment, r a live
# reservation), branch, state, finished_at, report line. "" is empty.
rows="$tmp/rows"
: >"$rows"

# one_line makes a value safe for one tab-separated cell on a terminal.
one_line() {
	printf '%s' "$1" | LC_ALL=C tr -d '\000-\010\012-\037\177' | tr '\t' ' ' | cut -c1-200
}

# parse_report <file> prints item, state, channel, finished_at and the first
# line, tab-separated ("-" for a field it lacks), from a report; it fails on anything not of the
# report's shape (a frontmatter opening the first line, closed within ten).
parse_report() {
	local size
	size="$(wc -c <"$1")"
	[ "$((size))" -le "$MAX_REPORT_BYTES" ] || return 1
	tr -d '\r' <"$1" | awk '
		NR == 1 { if ($0 != "---") { bad = 1; exit } fm = 1; next }
		fm && $0 == "---" { fm = 0; body = 1; next }
		fm {
			if (NR > 10) { bad = 1; exit }
			k = $0; sub(/:.*/, "", k)
			v = $0; sub(/^[^:]*:[ ]?/, "", v)
			if (k == "item") item = v
			else if (k == "state") state = v
			else if (k == "channel") channel = v
			else if (k == "finished_at") fin = v
			next
		}
		body && line == "" && $0 !~ /^[[:space:]]*$/ && $0 !~ /^#/ { line = $0 }
		END {
			if (bad || !body) exit 1
			# No field is empty: the reader splits on tabs, which collapse.
			if (item == "") item = "-"
			if (state == "") state = "-"
			if (channel == "") channel = "-"
			if (fin == "") fin = "-"
			if (line == "") line = "-"
			printf "%s\t%s\t%s\t%s\t%s\n", item, state, channel, fin, line
		}'
}

# read_report <file> <channel> sets r_item, r_state, r_finished and r_line, or
# fails when the file is not a report of that channel.
read_report() {
	local parsed
	parsed="$(parse_report "$1")" || return 1
	IFS='	' read -r r_item r_state r_channel r_finished r_line <<<"$parsed" || true
	[[ "$r_item" =~ $ITEM_RE ]] || return 1
	case "$r_state" in
	handed-back | stopped) ;;
	*) return 1 ;;
	esac
	[ "$r_channel" = "$2" ] || return 1
	[[ "$r_finished" =~ $FINISHED_RE ]] || r_finished=""
	[ "$r_line" != "-" ] || r_line=""
	r_line="$(one_line "$r_line")"
}

# --- 1. The work-in-progress branches -------------------------------------------

skipped=0
while IFS= read -r prefix; do
	[ -n "$prefix" ] || continue
	GIT_TERMINAL_PROMPT=0 git fetch -q --prune --no-tags origin \
		"+refs/heads/${prefix}*:refs/remotes/origin/${prefix}*" </dev/null 2>"$tmp/fetch.err" ||
		note "could not fetch ${prefix}* from origin; those branches are listed as last fetched."
	while IFS= read -r ref; do
		[ -n "$ref" ] || continue
		branch="${ref#refs/remotes/origin/}"
		item="${branch#"$prefix"}"
		if ! [[ "$item" =~ $ITEM_RE ]]; then
			skipped=$((skipped + 1))
			continue
		fi
		state="" finished="" line=""
		if git cat-file -e "$ref:$report_dir/$item.md" 2>/dev/null; then
			if git show "$ref:$report_dir/$item.md" >"$tmp/report" 2>/dev/null &&
				read_report "$tmp/report" branch && [ "$r_item" = "$item" ]; then
				state="$r_state" finished="$r_finished" line="$r_line"
			else
				line="(the report on the branch does not read)"
			fi
		fi
		printf '%s\tb\t%s\t%s\t%s\t%s\n' "$item" "$branch" "$state" "$finished" "$line" >>"$rows"
	done <<<"$(git for-each-ref --format='%(refname)' "refs/remotes/origin/$prefix")"
done <<<"$prefixes"
[ "$skipped" -eq 0 ] ||
	note "skipped $skipped branch(es) under a work-in-progress prefix that do not name one record id."

# --- 2. The reservations -------------------------------------------------------

rc=0
peers="$(go run ./cmd/abcd peers --json 2>"$tmp/peers.err" </dev/null)" || rc=$?
if [ "$rc" -le 1 ] && [ -n "$peers" ]; then
	[ "$rc" -eq 0 ] ||
		note "the reservations could not be fetched from origin; they are as last fetched ($(tail -n 1 "$tmp/peers.err" | cut -c1-200))."
	# Indented JSON, one key per line: a "record" names the reservation and
	# the "state" after it judges it. A value a writer chose is a JSON string,
	# whose quotes are escaped, so it can never spell either key.
	printf '%s\n' "$peers" | grep -oE '"(record|state)": "[^"\\]*"' |
		awk -F'"' '$2 == "record" { r = $4; next } $2 == "state" { if (r != "" && $4 == "live") print r; r = "" }' |
		while IFS= read -r item; do
			[[ "$item" =~ $ITEM_RE ]] || continue
			printf '%s\tr\t\tlive\t\t\n' "$item"
		done >>"$rows" || true
else
	note "the reservations could not be read (go run ./cmd/abcd peers exited $rc), so no session shows as working."
fi

# --- 3. The fallback channel: the tracking issue's comments ---------------------

not_read() {
	note "the fallback channel (the tracking issue's comments) was not read: $1; listing from the branches only."
}
repo=""
if [ -z "$tracking" ]; then
	not_read "no tracking issue configured (scripts/cloud/cloud.conf, tracking_issue)"
elif ! command -v gh >/dev/null 2>&1; then
	not_read "the forge's command-line tool, gh, is not installed"
elif ! repo="$(bash "$here/forge-repo.sh" 2>/dev/null)"; then
	not_read "origin names no github.com repository"
else
	rc=0
	gh api --paginate "repos/$repo/issues/$tracking/comments" \
		--jq '.[] | [.author_association, (.body | @base64)] | @tsv' \
		</dev/null >"$tmp/comments" 2>"$tmp/gh.err" || rc=$?
	if [ "$rc" -ne 0 ]; then
		not_read "gh exited $rc"
	else
		while IFS='	' read -r assoc encoded; do
			case "$assoc" in
			OWNER | MEMBER | COLLABORATOR) ;;
			*) continue ;;
			esac
			printf '%s' "$encoded" | base64 --decode >"$tmp/comment" 2>/dev/null || continue
			read_report "$tmp/comment" issue-comment || continue
			printf '%s\tc\t\t%s\t%s\t%s\n' "$r_item" "$r_state" "$r_finished" "(issue #$tracking) $r_line" >>"$rows"
		done <"$tmp/comments"
	fi
fi

# --- One row per item ------------------------------------------------------------

printf 'ITEM\tBRANCH\tSTATE\tREPORT\n'
awk -F'\t' '
	function offer(i, s, f, l) {
		if (!(i in rs) || f > rf[i]) { rs[i] = s; rf[i] = f; rl[i] = l }
	}
	{ seen[$1] = 1 }
	$2 == "b" {
		if (!($1 in br)) br[$1] = $3
		if ($4 != "") offer($1, $4, $5, $6)
		else if ($6 != "") why[$1] = $6
	}
	$2 == "c" { offer($1, $4, $5, $6) }
	$2 == "r" { live[$1] = 1 }
	END {
		for (i in seen) {
			b = (i in br) ? br[i] : "-"
			if (i in rs) { s = rs[i]; l = rl[i] }
			else { s = (i in live) ? "working" : "unknown"; l = (i in why) ? why[i] : "" }
			if (l == "") l = "-"
			printf "%s\t%s\t%s\t%s\n", i, b, s, l
		}
	}' "$rows" | LC_ALL=C sort
