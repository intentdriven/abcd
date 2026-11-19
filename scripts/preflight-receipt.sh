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
#                                       verdict and the merge base it was taken at.
#                                       A full receipt needs both test stages'
#                                       passing reports for this commit's tree,
#                                       a reused result counting for that exact
#                                       tree, and records how the stages ran (reuse,
#                                       cut, fallback or mixed) and how many packages
#                                       each reused. Whether or not it mints, its
#                                       last lines then give the run's time from
#                                       the clock ABCD_PREFLIGHT_CLOCK names, when
#                                       make preflight named one (clock_report)
#   preflight-receipt.sh check <commit> exit 0 when a receipt for <commit> exists
#                                       in any worktree of this repository
#   preflight-receipt.sh clock new      print a new clock's path for a run starting
#                                       now (make preflight names it in
#                                       ABCD_PREFLIGHT_CLOCK); writes nothing
#   preflight-receipt.sh clock opened   note the load gate's wait, from the run's
#                                       start to now, once per run
#   preflight-receipt.sh clock mark <name>
#   preflight-receipt.sh clock since <name>
#                                       note the start of a wait, then the wait
#                                       from that mark to now (the race gate's)
#                                       Each clock form but new writes nothing when
#                                       no clock is named, and a clock it cannot
#                                       write is said on stderr and fails nothing.
#
# Exit 0 on success; `check` exits 1 when there is no receipt; 2 on a usage fault.
set -euo pipefail

# Every git here, and scripts/records-only.sh, which mint runs, reads with
# replace refs, graft files and the commit-graph turned off, as .githooks/pre-push
# does: no refs/replace/* ref stands another commit, tree or blob in for one HEAD
# stores, no graft file gives it other parents, and no commit-graph file, a
# cache git reads a commit's tree and parents from in place of the commit,
# claims another tree for it. None of the three changes the state or the tree
# the stage reports are checked against. A tree or blob object's content is not
# checked against its id here. The graft path is the one
# internal/gitutil uses (noGraftFile): it names nothing, so git reads no graft
# file, and git prints no hint for it as it does for /dev/null. core.commitGraph
# is set false as the last entry of GIT_CONFIG_PARAMETERS, after any entries the
# caller's environment carries, and the last entry wins; GIT_TEST_COMMIT_GRAPH,
# which loads a graph whatever the setting says, is off.
export GIT_NO_REPLACE_OBJECTS=1
export GIT_GRAFT_FILE=/dev/null/none
export GIT_CONFIG_PARAMETERS="${GIT_CONFIG_PARAMETERS:+$GIT_CONFIG_PARAMETERS }'core.commitgraph=false'"
export GIT_TEST_COMMIT_GRAPH=0

cd "$(git rev-parse --show-toplevel)"

receipts_rel=".abcd/.work.local/preflight-receipts"
# Each test stage's report, one file per go test command it ran, by commit
# (internal/testreuse): what a full receipt reads its reuse counts from, and how
# it knows the stages checked this exact tree.
reuse_rel=".abcd/.work.local/preflight-reuse"
# The clock of each run (iss-2611190439574731; internal/testreuse/clock.go): one
# file per run, named <start>-<pid> with the run's start in Unix seconds, on which
# this script notes the waits at the load gate and the race gate, and each test
# stage its checking time and its own waits.
clock_rel=".abcd/.work.local/preflight-clock"
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

# stage_counts <commit> <tree> <stage>: print "<reused> <packages> <mode>" summed
# over the stage's reports, each of which must be a pass on this exact tree; a
# stage that ran two ways reads as mixed. Print nothing, and return 1, when there is
# none or one is not: a run over a working tree's edits names no tree at all.
stage_counts() {
	local commit="$1" tree="$2" stage="$3" report reused=0 total=0 found=false mode="" r_tree r_result r_reused r_packages r_mode
	for report in "$reuse_rel/$commit/$stage-"*; do
		[ -f "$report" ] || continue
		found=true
		r_tree="$(sed -n 's/^tree //p' "$report")"
		r_result="$(sed -n 's/^result //p' "$report")"
		r_reused="$(sed -n 's/^reused //p' "$report")"
		r_packages="$(sed -n 's/^packages //p' "$report")"
		r_mode="$(sed -n 's/^mode //p' "$report")"
		if [ "$r_tree" != "$tree" ] || [ "$r_result" != pass ] || [ "$r_mode" = dirty ]; then
			return 1
		fi
		if [ -z "$mode" ]; then
			mode="$r_mode"
		elif [ "$mode" != "$r_mode" ]; then
			mode=mixed
		fi
		case "$r_reused$r_packages" in
		"" | *[!0-9]*) return 1 ;;
		esac
		reused=$((reused + r_reused))
		total=$((total + r_packages))
	done
	[ "$found" = true ] || return 1
	echo "$reused $total $mode"
}

mint() {
	local began="$1" scope="$2" began_head began_tree now now_head now_tree verdict base tree unit race unit_reused unit_total unit_mode race_reused race_total race_mode tests
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
		echo "           Where \`git status\` reads clean, a replacement ref or a commit-graph that changes"
		echo "           how HEAD reads may be the cause: these gates read HEAD as stored."
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
	if [ "$scope" != records-only ]; then
		tree="$(git rev-parse 'HEAD^{tree}')"
		if ! unit="$(stage_counts "$now_head" "$tree" unit)" || ! race="$(stage_counts "$now_head" "$tree" race)"; then
			echo "preflight: no push receipt minted — the test stages left no passing report for this commit's"
			echo "           exact tree (${tree:0:12}), so nothing shows they checked what a push ships."
			echo "           Run \`make preflight\` again."
			return 0
		fi
		unit_reused="${unit%% *}"
		unit_mode="${unit##* }"
		unit_total="${unit#* }"
		unit_total="${unit_total% *}"
		race_reused="${race%% *}"
		race_mode="${race##* }"
		race_total="${race#* }"
		race_total="${race_total% *}"
		tests="$unit_mode"
		[ "$unit_mode" = "$race_mode" ] || tests=mixed
	fi
	mkdir -p "$receipts_rel"
	if [ "$scope" = records-only ]; then
		printf 'commit %s\nminted %s\nscope records-only\nverdict %s\nbase %s\n' "$now_head" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$verdict" "$base" >"$receipts_rel/$now_head"
	else
		printf 'commit %s\nminted %s\nscope full\ntests %s\nreuse unit %s of %s\nreuse race %s of %s\n' "$now_head" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$tests" "$unit_reused" "$unit_total" "$race_reused" "$race_total" >"$receipts_rel/$now_head"
	fi
	# Keep the newest $keep receipts. The names are full commit ids, so a plain
	# listing is safe to word-split.
	# shellcheck disable=SC2012
	ls -t "$receipts_rel" | tail -n +"$((keep + 1))" | while IFS= read -r old; do
		rm -f "${receipts_rel:?}/$old"
	done
	if [ -d "$reuse_rel" ]; then
		# shellcheck disable=SC2012
		ls -t "$reuse_rel" | tail -n +"$((keep + 1))" | while IFS= read -r old; do
			rm -rf "${reuse_rel:?}/$old"
		done
	fi
	if [ "$scope" = records-only ]; then
		echo "preflight: push receipt minted for ${now_head:0:12}, records-only (the unit and race stages were skipped)"
		echo "           — a push of that commit passes the pre-push gate."
		return 0
	fi
	echo "preflight: push receipt minted for ${now_head:0:12} — a push of that commit passes the pre-push gate."
	echo "           the unit stage reused $unit_reused of $unit_total packages and the race stage $race_reused of $race_total ($tests)."
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

# clock_report: the run's time, on its last lines, beside the reuse counts. One
# line per stage, `preflight: stage <name>: <time>`, the gates first (the lint
# gates, scans, eval lanes, build and vet: the checked time the test stages do
# not account for), then the test stages the clock names, in the order they ran;
# then `preflight: checked in <time> (waited <time>: load gate <time>, race gate
# <time>, check lock <time>)`. The checked time is the whole run's, from its
# start to now, less every wait the clock holds, whose total the parenthesis
# gives and then, one field per kind, where it was spent: the load gate's wait,
# the race gate's (before the race stage and a stage's own pass again) and a test
# stage's wait for the check directory's lock. A kind that waited no whole second
# is left out, and a run that waited for none says `(waited 0s)`; each field is
# rounded to the second and the total is their sum. A time is whole seconds in
# the form 42s, 3m12s, 41m07s or 1h02m03s, which Go's time.ParseDuration reads
# (internal/testreuse's formatClock prints the same). With no clock named (mint run by hand)
# it says nothing; a clock no stage or gate wrote is a run that waited for nothing
# and ran no test stage, as the records-only rule does. The report never costs the
# receipt, which mint has already written: a line that is not a note of a number
# of seconds no greater than the run's whole time is skipped, and a clock whose
# name gives no start, that is not a file or cannot be read gives one line saying
# no time is reported, and returns 0 (TestAGarbledClockNeverCostsTheReceipt).
clock_report() {
	local clock="${ABCD_PREFLIGHT_CLOCK:-}" base started now notes=/dev/null
	[ -n "$clock" ] || return 0
	base="${clock##*/}"
	started="${base%%-*}"
	if ! unix_seconds "$started"; then
		echo "preflight: no time reported — the run's clock ($base) does not name its start."
		return 0
	fi
	if [ -e "$clock" ] || [ -L "$clock" ]; then
		if ! [ -f "$clock" ] || ! [ -r "$clock" ]; then
			echo "preflight: no time reported — the run's clock ($base) is not a file this account can read."
			return 0
		fi
		notes="$clock"
	fi
	now="$(date +%s)"
	awk -v total="$((now - started))" -v full="$1" '
		function fmt(s,   h, m) {
			h = int(s / 3600); m = int((s % 3600) / 60); s = s % 60
			if (h > 0) return sprintf("%dh%02dm%02ds", h, m, s)
			if (m > 0) return sprintf("%dm%02ds", m, s)
			return sprintf("%ds", s)
		}
		function part(k, name,   s) {
			s = int(waited[k] + 0.5)
			if (s <= 0) return
			w += s; parts = parts sep name " " fmt(s); sep = ", "
		}
		NF == 3 && $2 ~ /^[a-z]+$/ && $3 ~ /^[0-9]+(\.[0-9]+)?$/ && length($3) <= 16 && $3 + 0 <= total + 1 {
			if ($1 == "wait") { if (!($2 in waited)) kinds[++m] = $2; waited[$2] += $3 }
			else if ($1 == "stage") { if (!($2 in took)) order[++n] = $2; took[$2] += $3 }
		}
		END {
			label["load"] = "load gate"; label["race"] = "race gate"; label["lock"] = "check lock"
			part("load", label["load"]); part("race", label["race"]); part("lock", label["lock"])
			for (i = 1; i <= m; i++) if (!(kinds[i] in label)) part(kinds[i], kinds[i])
			c = total - w; if (c < 0) c = 0
			for (i = 1; i <= n; i++) { st[i] = int(took[order[i]] + 0.5); sum += st[i] }
			if (full == "full") { g = c - sum; if (g < 0) g = 0; printf "preflight: stage gates: %s\n", fmt(g) }
			for (i = 1; i <= n; i++) printf "preflight: stage %s: %s\n", order[i], fmt(st[i])
			printf "preflight: checked in %s (waited %s%s)\n", fmt(c), fmt(w), (parts == "" ? "" : ": " parts)
		}' "$notes" 2>/dev/null || echo "preflight: no time reported — the run's clock ($base) could not be read."
	# Keep the newest $keep clocks, as the receipts. The names are <start>-<pid>,
	# safe to word-split.
	if [ -d "$clock_rel" ]; then
		# shellcheck disable=SC2012
		# A directory this account cannot list prunes nothing and fails nothing.
		ls -t "$clock_rel" 2>/dev/null | tail -n +"$((keep + 1))" | while IFS= read -r old; do
			rm -f "${clock_rel:?}/$old" 2>/dev/null || true
		done || true
	fi
}

# unix_seconds <value>: whether value is a time in Unix seconds the clock may do
# shell arithmetic on: digits with no leading zero, which the shell would read as
# octal, and no more than twelve of them. A run's start, from its clock's name,
# and a mark read back from the clock both pass it before any arithmetic.
unix_seconds() {
	case "$1" in
	"" | *[!0-9]* | 0* | ?????????????*) return 1 ;;
	esac
}

# clock_start <clock>: the run's start in Unix seconds, from its clock's name, by
# unix_seconds' rule.
clock_start() {
	local base="${1##*/}"
	base="${base%%-*}"
	unix_seconds "$base" || return 1
	printf '%s\n' "$base"
}

# clock_note <line>: append one line to the run's clock. A clock is a report,
# never a gate, so one that cannot be written is said and fails nothing.
clock_note() {
	local clock="$ABCD_PREFLIGHT_CLOCK"
	if ! { mkdir -p "${clock%/*}" && printf '%s\n' "$1" >>"$clock"; } 2>/dev/null; then
		echo "preflight: the run's clock (${clock##*/}) was not written, so its last lines will count this wait as checking." >&2
	fi
}

# clock <form> [<name>]: make preflight's clock, which clock_report reads. new
# prints the path of a clock for a run starting now, under this checkout's local
# tier; the rest note a wait on the clock ABCD_PREFLIGHT_CLOCK names, and nothing
# when none is named (outside make preflight), it is not absolute, or its name
# gives no start. opened notes the load gate's wait, `wait load <seconds>`, from
# the run's start to now: the stages' make runs it as it starts, so the wait is
# the whole time before them (the gate's compile, its queue and its settling),
# and a clock that holds one already is left as it is, so a make started inside
# the stages notes nothing. mark notes the start of a wait, and since the wait
# from the last mark of that name to now, as `wait <name> <seconds>`, or nothing
# when that mark is not Unix seconds by unix_seconds' rule.
clock() {
	local clock="${ABCD_PREFLIGHT_CLOCK:-}" start now at
	if [ "$1" = new ]; then
		printf '%s/%s/%s-%s\n' "$PWD" "$clock_rel" "$(date +%s)" "$$"
		return 0
	fi
	case "$clock" in
	/*) ;;
	*) return 0 ;;
	esac
	start="$(clock_start "$clock")" || return 0
	now="$(date +%s)"
	case "$1" in
	opened)
		if [ -f "$clock" ] && grep -q '^wait load ' "$clock" 2>/dev/null; then
			return 0
		fi
		clock_note "wait load $((now - start))"
		;;
	mark)
		clock_note "mark $2 $now"
		;;
	since)
		# The last mark of that name, refused unless it is Unix seconds by the
		# start's rule; a refused mark notes nothing.
		at="$(awk -v n="$2" '$1 == "mark" && $2 == n { t = $3 } END { print t }' "$clock" 2>/dev/null || true)"
		unix_seconds "$at" || return 0
		clock_note "wait $2 $((now - at))"
		;;
	esac
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
	clock_report "${3:-full}"
	;;
check)
	[ $# -eq 2 ] || exit 2
	check "$2"
	;;
clock)
	case "$#:${2:-}" in
	2:new | 2:opened) ;;
	3:mark | 3:since)
		case "$3" in
		"" | *[!a-z]*) exit 2 ;;
		esac
		;;
	*) exit 2 ;;
	esac
	clock "$2" "${3:-}"
	;;
*)
	echo "usage: preflight-receipt.sh state | mint \"<state>\" [records-only] | check <commit> | clock new|opened | clock mark|since <name>" >&2
	exit 2
	;;
esac
