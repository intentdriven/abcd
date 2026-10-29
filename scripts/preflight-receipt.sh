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
# unmoved between: then what the gates read is what the push ships. A tracked file
# flagged skip-worktree or assume-unchanged is hidden from `git status`, so its
# local edit would read as clean: while any such flag is set (a sparse checkout
# sets skip-worktree on every file it leaves out), no receipt is minted.
#
# LIMITS, stated rather than implied. The comparison is `git status` at two
# instants, so it cannot see: a file git ignores, which is outside the commit yet
# can be read by a gate (a `go.work` beside go.mod, which .gitignore lists, changes
# every Go gate's module resolution); HEAD moved and moved back, or the tree changed
# and restored, between the two instants; and anything a gate reads from outside
# the checkout. Each passes a receipt for a tree the gates did not read exactly.
#
# The receipt is a file named by the full commit id under the checkout's local
# tier, .abcd/.work.local/preflight-receipts/. It is a local convenience gate, not
# a security boundary: CI re-runs every gate on the pushed commit and is the
# authority, and `git push --no-verify` skips this layer exactly as it always did.
#
# Usage:
#   preflight-receipt.sh state          print "<HEAD> clean|dirty|hidden" for this
#                                       tree (hidden: an index flag hides edits)
#   preflight-receipt.sh mint "<state>" [records-only]
#                                       mint a receipt for HEAD if the tree was
#                                       clean at <state> and is clean at the same
#                                       HEAD now; otherwise say why none is minted.
#                                       The receipt names its scope: full, or
#                                       records-only (the unit and race stages
#                                       skipped, iss-2610251555094324), which
#                                       scripts/records-only.sh must confirm again
#                                       here, and whose receipt carries that
#                                       verdict and the merge base it was taken at
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
	local head status flags
	head="$(git rev-parse --verify --quiet HEAD 2>/dev/null || true)"
	[ -n "$head" ] || head="none"
	status="$(git status --porcelain --untracked-files=normal 2>/dev/null)" || {
		echo "$head dirty"
		return 0
	}
	# `git ls-files -v` tags a skip-worktree entry S and an assume-unchanged one in
	# lower case: either flag hides an edit from the status read above.
	flags="$(git ls-files -v 2>/dev/null)" || {
		echo "$head dirty"
		return 0
	}
	# A here-string, not a pipe: under pipefail an early-exiting `grep -q` can fail
	# the writer with SIGPIPE and turn a match into "no match".
	if grep -q '^[Shs] ' <<<"$flags"; then
		echo "$head hidden"
	elif [ -z "$status" ]; then
		echo "$head clean"
	else
		echo "$head dirty"
	fi
}

mint() {
	local began="$1" scope="$2" began_head began_tree now now_head now_tree verdict base
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
	if [ "$began_tree" = "hidden" ] || [ "$now_tree" = "hidden" ]; then
		echo "preflight: no push receipt minted — a tracked file is flagged skip-worktree or assume-unchanged"
		echo "           (git ls-files -v tags it S or in lower case), which hides its edits from git status,"
		echo "           so the tree these gates read cannot be shown to be the commit. Clear the flag"
		echo "           (git update-index --no-skip-worktree / --no-assume-unchanged <path>) and run"
		echo "           \`make preflight\` again."
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
	if [ "$scope" = records-only ]; then
		# The classifier's line has its control bytes dropped already, and a
		# records-only line names the merge base as a full commit id.
		if ! verdict="$(scripts/records-only.sh)"; then
			echo "preflight: no push receipt minted — the change no longer reads as records-only ($verdict),"
			echo "           and its unit and race stages did not run. Run \`make preflight\` again."
			return 0
		fi
		base="$(sed -n 's/.*, \([0-9a-f]\{40,64\}\), are in the record tiers.*/\1/p' <<<"$verdict")"
		if [ -z "$base" ]; then
			echo "preflight: no push receipt minted — the records-only verdict names no merge base ($verdict)."
			return 0
		fi
	fi
	mkdir -p "$receipts_rel"
	if [ "$scope" = records-only ]; then
		printf 'commit %s\nminted %s\nscope records-only\nverdict %s\nbase %s\n' \
			"$now_head" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$verdict" "$base" >"$receipts_rel/$now_head"
	else
		printf 'commit %s\nminted %s\nscope full\n' "$now_head" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$receipts_rel/$now_head"
	fi
	# Keep the newest $keep receipts. The names are full commit ids, so a plain
	# listing is safe to word-split.
	# shellcheck disable=SC2012
	ls -t "$receipts_rel" | tail -n +"$((keep + 1))" | while IFS= read -r old; do
		rm -f "${receipts_rel:?}/$old"
	done
	if [ "$scope" = records-only ]; then
		echo "preflight: push receipt minted for ${now_head:0:12}, records-only (the unit and race stages were skipped)"
		echo "           — a push of that commit passes the pre-push gate."
		return 0
	fi
	echo "preflight: push receipt minted for ${now_head:0:12} — a push of that commit passes the pre-push gate."
}

# own_receipt <dir>: the worktree at dir holds a receipt for $commit that is
# its own. A local tier git tracks anything in came with a commit, so no
# receipt in it counts: by any letter case of its name, since a filesystem that
# folds case opens .abcd/.Work.Local as the tier, by any other spelling that
# opens the tier's own directory, and with .abcd itself tracked as a link or a
# submodule. A worktree git cannot list counts for nothing either. Git is asked
# about dir alone, never through a repository the hook's environment names.
own_receipt() {
	local listed pairs pair
	[ -f "$1/$receipts_rel/$commit" ] || return 1
	listed=$(env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE -u GIT_COMMON_DIR \
		git -C "$1" -c core.quotePath=false ls-files 2>/dev/null) || return 1
	# Here-strings, not pipes, as in state(): under pipefail a listing longer
	# than a pipe holds would turn grep's match into "no match".
	if grep -qiE '^"?\.abcd(/\.work\.local(/|$)|$)' <<<"$listed"; then
		return 1
	fi
	pairs=$(awk -F/ 'NF > 1 && !seen[$1 "/" $2]++ { print $1 "/" $2 }' <<<"$listed") || return 1
	while IFS= read -r pair; do
		if [ -n "$pair" ] && [ "$1/${pair%%/*}" -ef "$1/.abcd" ] && [ "$1/$pair" -ef "$1/.abcd/.work.local" ]; then
			return 1
		fi
	done <<<"$pairs"
}

check() {
	local commit="$1" wt
	case "$commit" in
	*[!0-9a-f]* | "") return 2 ;;
	esac
	if own_receipt .; then
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
		if own_receipt "$wt"; then
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
	case "$#:${3:-}" in
	2: | 3:records-only) ;;
	*) exit 2 ;;
	esac
	mint "$2" "${3:-full}"
	;;
check)
	[ $# -eq 2 ] || exit 2
	check "$2"
	;;
*)
	echo "usage: preflight-receipt.sh state | mint \"<state>\" [records-only] | check <commit>" >&2
	exit 2
	;;
esac
