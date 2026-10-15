#!/usr/bin/env bash
# The required check that keeps a cloud session's unchecked work off the
# default branch (spc-2610120941441282 step 7, itd-2610120738075547 C7,
# adr-2610120937435856 decision 4, brief invariant 23).
#
# A cloud session pushes its work, checked or not, to a work-in-progress
# branch, and the pre-push hook admits that push without a receipt. The hook is
# not the boundary (`--no-verify` skips it); this check is, required by the
# default branch's ruleset and run by ci.yml's wip-guard job on every pull
# request and every merge-queue entry. It refuses a change when:
#
#   - its source branch holds the path segment `abcd-wip` anywhere. The
#     segment, not any one host's prefix, so the check names no host: every
#     connector's prefix ends in that segment (connector.sh refuses one that
#     does not). A longer word (`abcd-wipe`) or the word inside a segment is
#     not the segment.
#   - its diff <base>...<head> adds a file under the cloud report directory. A
#     report travels on its work-in-progress branch and is left out by the
#     person's collect, so one arriving on the default branch is a branch that
#     did not go through the collect. A rename into the directory counts as an
#     add; editing or removing a report already there does not.
#
# The directory is named here, not read from scripts/cloud/cloud.conf in the
# tree being judged, so a change cannot move the guard by editing the setting;
# check-wip-source-cases.sh holds the two equal.
#
# Usage: check-wip-source.sh <source-ref> <base> <head>
#
# Exit 0 clean, 1 refused, 2 a usage or repository fault (an empty source, a
# base or head that resolves to nothing): a check that could not judge has
# cleared nothing.
set -euo pipefail

REPORT_DIR='.abcd/work/cloud-reports'

if [ $# -ne 3 ] || [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
	echo "usage: check-wip-source.sh <source-ref> <base> <head>   (none of them empty)" >&2
	exit 2
fi
source_ref="${1#refs/heads/}"
base="$2"
head="$3"

for ref in "$base" "$head"; do
	if ! git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
		echo "check-wip-source: cannot judge the diff: '$ref' names no commit here." >&2
		exit 2
	fi
done

refused=0

case "/$source_ref/" in
*/abcd-wip/*)
	echo "check-wip-source: REFUSED — the source branch '$source_ref' is a work-in-progress branch" >&2
	echo "  (it holds the segment abcd-wip). Its work reaches the default branch only through the" >&2
	echo "  person's own collect, which re-runs preflight on their computer and lands a new branch" >&2
	echo "  (adr-2610120937435856)." >&2
	refused=1
	;;
esac

# A file moved into the directory is an add like any other: the pathspec
# leaves its old path out of the pairing, and --no-renames keeps it so
# whatever the configuration says about rename detection.
if ! added="$(git diff --no-renames --name-only --diff-filter=A "$base...$head" -- "$REPORT_DIR/")"; then
	echo "check-wip-source: cannot judge the diff $base...$head." >&2
	exit 2
fi
if [ -n "$added" ]; then
	echo "check-wip-source: REFUSED — the change adds a cloud report under $REPORT_DIR/:" >&2
	printf '%s\n' "$added" | sed 's/^/  /' >&2
	echo "  A report travels on its work-in-progress branch and the person's collect leaves it out" >&2
	echo "  (adr-2610120937435856)." >&2
	refused=1
fi

if [ "$refused" -ne 0 ]; then
	exit 1
fi
echo "check-wip-source: clean — '$source_ref' is no work-in-progress branch and adds no cloud report"
