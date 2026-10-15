#!/usr/bin/env bash
# Send one item to a hosted cloud session (spc-2610120941441282 step 9,
# itd-2610120738075547 C10 and C5, decisions 2 and 11). Run on the person's own
# computer. In order:
#
#   1. read the environment id from .abcd/.work.local/cloud/environment, the
#      local tier: it is the person's, never committed;
#   2. compose the item's brief through brief.sh, which refuses an item that is
#      not ready, not public or under embargo (C4, C13);
#   3. keep the brief as a preview at .abcd/.work.local/cloud/briefs/<item>.md,
#      so the person can read what was sent;
#   4. start one session through the connector's launch script, from origin's
#      default branch, and print the session's id and link. The link is
#      printed to the person and written to no file.
#
# No fallback, ever (C5): when the cloud cannot be reached (no host tool, no
# login, an expired login, an unknown environment, a start the host refuses)
# it says so and why, and starts nothing on this computer in its place: no
# local session, no sub-agent, no worktree, and no file beyond the brief
# preview. An expired login stops the run; the person renews it by hand, since
# abcd stores no login (decision 11).
#
# The connector's launch script answers through a fixed contract (documented
# in it): exit 0 with `id=` and `link=` lines; 3 no host tool; 4 no login;
# 5 an expired login; 6 an unknown environment; 7 another refusal; 8 an answer
# naming no session. This script renders each without naming the host (C12).
#
# Usage: send.sh [--connector <name>] <item-id>
#   (default connector: the only one declared)
# Exit 0 started; 1 refused or not started; 2 misuse.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

ITEM_RE='^(iss|itd)-([0-9]{16}|[0-9]{1,4})$'
ENV_FILE=".abcd/.work.local/cloud/environment"
PREVIEW_DIR=".abcd/.work.local/cloud/briefs"

usage() {
	echo "usage: send.sh [--connector <name>] <item-id>   (an iss- or itd- record id)" >&2
	exit 2
}

refuse() {
	echo "send: REFUSING $1 No session was started." >&2
	exit 1
}

connector=""
item=""
while [ $# -gt 0 ]; do
	case "$1" in
	--connector)
		[ $# -ge 2 ] || usage
		connector="$2"
		shift 2
		;;
	-*) usage ;;
	*)
		[ -z "$item" ] || usage
		item="$1"
		shift
		;;
	esac
done
[[ "$item" =~ $ITEM_RE ]] || usage

cd "$root"

if [ -z "$connector" ]; then
	connector="$(bash "$here/connector.sh" default)" || exit 1
fi
launch="$(bash "$here/connector.sh" get "$connector" launch)" || exit 1
prefix="$(bash "$here/connector.sh" get "$connector" wip_prefix)" || exit 1

# --- 1. The environment id, from the local tier ------------------------------

if [ ! -f "$ENV_FILE" ] || [ -L "$ENV_FILE" ]; then
	refuse "there is no environment id: write the id of the cloud environment to start sessions in, alone on one line, to $ENV_FILE (it is never committed)."
fi
env_id=""
lines=0
while IFS= read -r line || [ -n "$line" ]; do
	line="${line%$'\r'}"
	line="$(printf '%s' "$line" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')"
	case "$line" in
	'' | '#'*) continue ;;
	esac
	lines=$((lines + 1))
	env_id="$line"
done <"$ENV_FILE"
if [ "$lines" -ne 1 ] || ! [[ "$env_id" =~ ^[A-Za-z0-9][A-Za-z0-9_.:-]*$ ]]; then
	refuse "$ENV_FILE does not hold one environment id (letters, digits, and . _ : -, alone on one line)."
fi

# --- 2. The brief --------------------------------------------------------------

brief="$(bash "$here/brief.sh" --connector "$connector" "$item")" ||
	refuse "no brief could be composed for $item (the reason is above)."
base="$(bash "$here/default-branch.sh")" ||
	refuse "origin's default branch cannot be determined, so there is no branch to start the session from."

# --- 3. The preview ------------------------------------------------------------

preview="$PREVIEW_DIR/$item.md"
mkdir -p "$PREVIEW_DIR"
printf '%s\n' "$brief" >"$preview"

# --- 4. The start --------------------------------------------------------------

nothing_local="Nothing was started on this computer in its place: no local session, no sub-agent, no worktree. The brief that would have been sent is at $preview."
unreachable() {
	echo "send: the cloud cannot be reached: $1" >&2
	echo "send: $nothing_local" >&2
	exit 1
}

rc=0
answer="$("$launch" "$env_id" "$base" "$preview")" || rc=$?
case "$rc" in
0) ;;
3) unreachable "the host's command-line tool is not installed on this computer." ;;
4) unreachable "this computer is not logged in to the host. Log in by hand, then send again." ;;
5) unreachable "the login cloud sessions are started with has expired: the person must renew it by hand (abcd stores no login), then send again. An unattended run stops here." ;;
6) unreachable "the host does not know the environment named in $ENV_FILE. Check the id there." ;;
7) unreachable "the host refused the start (its answer is above)." ;;
8)
	echo "send: the host answered without naming a session, so a session may have started. Check the host's own list of sessions before sending $item again." >&2
	exit 1
	;;
*) unreachable "the connector's launch script failed (exit $rc)." ;;
esac

id="$(printf '%s\n' "$answer" | sed -n 's/^id=//p' | head -n 1)"
link="$(printf '%s\n' "$answer" | sed -n 's/^link=//p' | head -n 1)"
if ! [[ "$id" =~ ^[A-Za-z0-9_-]+$ ]] || ! [[ "$link" =~ ^https://[^[:space:]]+$ ]]; then
	echo "send: the connector reported a start without a usable session id and link, so a session may have started. Check the host's own list of sessions before sending $item again." >&2
	exit 1
fi

echo "send: started a cloud session for $item; its work comes back on $prefix$item."
echo "  session: $id"
echo "  link:    $link"
