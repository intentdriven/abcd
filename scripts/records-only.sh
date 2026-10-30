#!/usr/bin/env bash
# Whether HEAD is a records-only change (iss-2610251555094324): every path it
# changes against the merge base with origin's default branch lies in the
# record tiers, .abcd/development/ or .abcd/work/, no test names one of them, by
# its path, its file name or its record id, and the working tree matches HEAD. Such a
# change can fail only a Go test that reads the committed record, so `make
# preflight` runs precheck's tests, which read it, in place of the full unit and
# race stages, says so, and the receipt it mints says so too. CI's unit lane
# still runs every test on the pull request.
#
# It classifies fail-closed, as CI's inert-path classifier does: only a diff
# proven to lie in the tiers is records-only, and every other outcome (a path
# outside them, a file name a test names, a working tree that differs from HEAD,
# no origin default branch to diff against, an empty diff, a git failure, an
# aborted script) is the full run.
#
# LIMITS, stated rather than implied. A test that reads the record tiers by
# directory rather than by a path it names (a walk of .abcd/work/issues) is run
# for a records-only change only when precheck runs it: precheck runs the
# packages known to read the committed record tree whole, and the command and
# site packages' audit tests by name. One elsewhere, or added without joining
# precheck, runs only in CI's unit lane, which runs every test before the merge.
#
# Usage:
#   records-only.sh    print "records-only: <reason>" and exit 0, or
#                      "full run: <reason>" and exit 1; a records-only reason
#                      names the merge base's full commit id, so a later reader
#                      can redo the check
set -uo pipefail

verdict=false
reason='the classifier did not reach a verdict'
# The reason can carry a changed path, which a writer chose, so only printable
# ASCII reaches a terminal: a C0 or C1 control (U+009B opens a sequence as ESC [
# does), a bidirectional override or any other non-ASCII byte is dropped.
finish() {
	reason="$(printf '%s' "$reason" | LC_ALL=C tr -cd '\040-\176')"
	if [ "$verdict" = true ]; then
		printf 'records-only: %s\n' "$reason"
		exit 0
	fi
	printf 'full run: %s\n' "$reason"
	exit 1
}
trap finish EXIT

in_tier() {
	case "$1" in
	.abcd/development/* | .abcd/work/*) return 0 ;;
	*) return 1 ;;
	esac
}

classify() {
	local top base full ref changed count path name id status flags
	top="$(git rev-parse --show-toplevel 2>/dev/null)" || {
		reason='this is not a git checkout'
		return 1
	}
	cd "$top" || {
		reason='the checkout root could not be entered'
		return 1
	}
	# The tests run on the working tree, so the commits alone speak for it only
	# when it matches HEAD: an uncommitted edit, or one an index flag hides from
	# git status (skip-worktree S, assume-unchanged in lower case), is unseen by
	# the diff below. Here-strings, not pipes, as in preflight-receipt.sh.
	if ! status="$(git status --porcelain --untracked-files=normal 2>/dev/null)" ||
		! flags="$(git ls-files -v 2>/dev/null)"; then
		reason='the working tree could not be read'
		return 1
	fi
	if [ -n "$status" ] || grep -q '^[Shs] ' <<<"$flags"; then
		reason='the working tree differs from HEAD, and the change is judged by its commits'
		return 1
	fi
	# Origin's default branch as last fetched: origin/HEAD when it is set,
	# else origin/main. It is named by its full ref throughout, so a local
	# branch that happens to be called origin/main never stands in for it.
	full="$(git symbolic-ref -q refs/remotes/origin/HEAD 2>/dev/null)" || full=''
	[ -n "$full" ] || full='refs/remotes/origin/main'
	ref="${full#refs/remotes/}"
	case "$full" in
	refs/remotes/origin/?*) ;;
	*)
		reason="origin/HEAD names ${full}, which is not a branch of origin"
		return 1
		;;
	esac
	if ! git rev-parse -q --verify "${full}^{commit}" >/dev/null 2>&1; then
		reason="there is no ${ref} to diff against"
		return 1
	fi
	if ! base="$(git merge-base HEAD "$full" 2>/dev/null)"; then
		reason="HEAD shares no history with ${ref}"
		return 1
	fi
	# --no-renames: with detection on, a file moved into a tier reports only
	# its destination, and the path it left never reaches the check. The list
	# is NUL-delimited, so a path holding a newline stays one path.
	changed="$(mktemp)" || {
		reason='no temporary file for the changed paths'
		return 1
	}
	if ! git diff --no-renames --name-only -z "$base" HEAD -- >"$changed" 2>/dev/null; then
		rm -f "$changed"
		reason="diffing against the merge base with ${ref} failed"
		return 1
	fi
	count=0
	while IFS= read -r -d '' path; do
		count=$((count + 1))
		if ! in_tier "$path"; then
			rm -f "$changed"
			reason="${path} is outside the record tiers (.abcd/development/, .abcd/work/)"
			return 1
		fi
		# A record a test names is that test's fixture, whether the test names
		# its path or builds the path and names only the file (its name holds
		# its path's last part, so one search finds both), or finds the file by
		# its record id with a glob (itd-121-*.md): a name that starts with an
		# id is searched for by that id and the dash after it, which a citation
		# in a comment (iss-N, or iss-N)) does not carry. git grep exits 1 for
		# no match and above 1 when it could not search.
		# A pattern git grep is given is split at a newline, so a name that
		# holds one cannot be searched for whole: it takes the full run.
		case "$path" in
		*$'\n'*)
			rm -f "$changed"
			reason="${path} holds a newline in its name, so the tests cannot be searched for it whole"
			return 1
			;;
		esac
		name="${path##*/}"
		git grep -q -F -e "$name" -- '*_test.go' 2>/dev/null
		case $? in
		0)
			rm -f "$changed"
			reason="${path} is named by a test (its file name ${name}), so it is that test's fixture"
			return 1
			;;
		1) ;;
		*)
			rm -f "$changed"
			reason="the tests could not be searched for ${path}"
			return 1
			;;
		esac
		# An ADR's file name starts with its bare number (the adr- prefix is
		# its id's, not its name's), which a test globs as <number>-*.md, so
		# a name that starts with twelve or more digits is searched for by
		# that number alone: an id that long is no fixture's by chance.
		id='' want=''
		if [[ "$name" =~ ^([a-z]+-[0-9]+)- ]]; then
			id="${BASH_REMATCH[1]}" want="${BASH_REMATCH[1]}-"
		elif [[ "$name" =~ ^([0-9]{12,})- ]]; then
			id="${BASH_REMATCH[1]}" want="${BASH_REMATCH[1]}"
		fi
		if [ -n "$id" ]; then
			git grep -q -F -e "$want" -- '*_test.go' 2>/dev/null
			case $? in
			0)
				rm -f "$changed"
				reason="${path} is named by a test (its record id ${id}), so it is that test's fixture"
				return 1
				;;
			1) ;;
			*)
				rm -f "$changed"
				reason="the tests could not be searched for ${path}"
				return 1
				;;
			esac
		fi
	done <"$changed"
	rm -f "$changed"
	if [ "$count" -eq 0 ]; then
		reason="the diff against the merge base with ${ref} is empty"
		return 1
	fi
	reason="all ${count} changed paths since the merge base with ${ref}, ${base}, are in the record tiers, and no test names one"
	return 0
}

if classify; then
	verdict=true
fi
