#!/usr/bin/env bash
# The preflight receipt: how `make preflight` tells the pre-push hook that the
# gates passed on exactly the commit being pushed, so the hook never runs the
# preflight itself (iss-2608290810036869, iss-2608210738378295).
#
# WHY A RECEIPT. The pre-push hook used to run the whole preflight, and git opens
# the push's connection BEFORE it runs the hook: a ten-minute preflight outlasted
# the transport's idle timeout, the server closed the connection, and the push
# died while the preflight's own passing output scrolled past the failure — a
# push that reported success and moved nothing. The ruling (M16, 2026-09-23) is
# check before connect: the preflight runs to completion before the push opens
# its connection. So the preflight runs first, as its own command, and mints a
# receipt; the hook only reads receipts, which takes milliseconds.
#
# WHY "ON A CLEAN TREE". The gates read the WORKING TREE. A tree that differs from
# HEAD — a staged rename whose follow-up edit is unstaged, an untracked record —
# can pass every gate while the committed tree fails CI, because CI checks out the
# commit. A receipt is minted only when the tree matched HEAD (no staged, unstaged
# or untracked change) when the preflight began AND when it ended, with HEAD
# unmoved between: then what the gates read is what the push ships. Files git
# ignores are outside that comparison, as they are outside the commit.
#
# The receipt is a file named by the full commit id under the checkout's local
# tier, .abcd/.work.local/preflight-receipts/. It is a local convenience gate, not
# a security boundary: CI re-runs every gate on the pushed commit and is the
# authority, and `git push --no-verify` skips this layer exactly as it always did.
#
# Usage:
#   preflight-receipt.sh state          print "<HEAD> clean|dirty" for this tree
#   preflight-receipt.sh mint "<state>" mint a receipt for HEAD if the tree was
#                                       clean at <state> and is clean at the same
#                                       HEAD now; otherwise say why none is minted
#   preflight-receipt.sh check <commit> exit 0 when a receipt for <commit> exists
#                                       in any worktree of this repository
#
# Exit 0 on success; `check` exits 1 when there is no receipt; 2 on a usage fault.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

receipts_rel=".abcd/.work.local/preflight-receipts"
# How many receipts a checkout keeps. Each is a few bytes; the cap only stops the
# directory growing without bound across months of preflights.
keep=50

state() {
	local head status
	head="$(git rev-parse --verify --quiet HEAD 2>/dev/null || true)"
	[ -n "$head" ] || head="none"
	status="$(git status --porcelain --untracked-files=normal 2>/dev/null)" || {
		echo "$head dirty"
		return 0
	}
	if [ -z "$status" ]; then
		echo "$head clean"
	else
		echo "$head dirty"
	fi
}

mint() {
	local began="$1" began_head began_tree now now_head now_tree
	began_head="${began%% *}"
	began_tree="${began##* }"
	now="$(state)"
	now_head="${now%% *}"
	now_tree="${now##* }"
	if [ -z "$began" ]; then
		echo "preflight: no push receipt minted — the tree's state when the run began was not recorded"
		echo "           (the receipt is minted only by \`make preflight\` named on the command line)."
		return 0
	fi
	if [ "$began_head" = "none" ] || [ "$now_head" = "none" ]; then
		echo "preflight: no push receipt minted — there is no commit to vouch for."
		return 0
	fi
	if [ "$began_tree" != "clean" ] || [ "$now_tree" != "clean" ]; then
		echo "preflight: no push receipt minted — the working tree differed from HEAD (staged, unstaged"
		echo "           or untracked changes), so these gates did not read the tree a push ships."
		echo "           Commit or set aside the changes and run \`make preflight\` again before pushing."
		return 0
	fi
	if [ "$began_head" != "$now_head" ]; then
		echo "preflight: no push receipt minted — HEAD moved during the run (${began_head:0:12} -> ${now_head:0:12}),"
		echo "           so no single commit is what these gates read. Run \`make preflight\` again."
		return 0
	fi
	mkdir -p "$receipts_rel"
	printf 'commit %s\nminted %s\n' "$now_head" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$receipts_rel/$now_head"
	# Keep the newest $keep receipts. The names are full commit ids, so a plain
	# listing is safe to word-split.
	# shellcheck disable=SC2012
	ls -t "$receipts_rel" | tail -n +"$((keep + 1))" | while IFS= read -r old; do
		rm -f "$receipts_rel/$old"
	done
	echo "preflight: push receipt minted for ${now_head:0:12} — a push of that commit passes the pre-push gate."
}

check() {
	local commit="$1" wt
	case "$commit" in
	*[!0-9a-f]* | "") return 2 ;;
	esac
	if [ -f "$receipts_rel/$commit" ]; then
		return 0
	fi
	# A commit's receipt is valid in whichever worktree of this repository it was
	# minted in: the same commit is the same tracked tree everywhere, and the
	# receipt says the gates passed on it with nothing uncommitted beside it. This
	# is what lets a branch preflighted in its own worktree be pushed from the
	# primary checkout. Only worktrees git itself lists are read.
	while IFS= read -r wt; do
		case "$wt" in
		"worktree "*) wt="${wt#worktree }" ;;
		*) continue ;;
		esac
		if [ -f "$wt/$receipts_rel/$commit" ]; then
			return 0
		fi
	done <<<"$(git worktree list --porcelain 2>/dev/null || true)"
	return 1
}

case "${1:-}" in
state)
	[ $# -eq 1 ] || exit 2
	state
	;;
mint)
	[ $# -eq 2 ] || exit 2
	mint "$2"
	;;
check)
	[ $# -eq 2 ] || exit 2
	check "$2"
	;;
*)
	echo "usage: preflight-receipt.sh state | mint \"<state>\" | check <commit>" >&2
	exit 2
	;;
esac
