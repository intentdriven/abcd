#!/usr/bin/env bash
# The first step of every cloud brief (spc-2610120941441282,
# itd-2610120738075547 C3, decisions 8, 17 and 18). In order, and each check
# stops the session before the next:
#
#   1. the place: the connector's is_remote script must pass. A brief started
#      on the person's own computer stops here, having run nothing else (C3).
#   2. the user: never root; every check runs as the normal user the setup
#      created (decision 8).
#   3. the author: the checkout's author and committer identity must be set and
#      be a person's, judged by the attribution gate's own signals
#      (scripts/check-attribution.sh identity, which reads the one identity
#      list). The setup takes the person's identity from the forge token
#      (decision 17); when it could not, the clone still carries the vendor's
#      default, and no commit could be authored as the person, so nothing
#      starts (decision 18). The refusal names the cause and NEVER the
#      identity it refused: the vendor identity is shown nowhere.
#   4. the machine: scripts/cloud/env-check.sh, whose exit is this script's.
#
# Usage: begin.sh [--connector <name>]   (default: the only connector declared)
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"

connector=""
while [ $# -gt 0 ]; do
	case "$1" in
	--connector)
		[ $# -ge 2 ] || {
			echo "usage: begin.sh [--connector <name>]" >&2
			exit 2
		}
		connector="$2"
		shift 2
		;;
	*)
		echo "usage: begin.sh [--connector <name>]" >&2
		exit 2
		;;
	esac
done

if [ -z "$connector" ]; then
	connector="$(bash "$here/connector.sh" default)" || exit 1
fi
is_remote="$(bash "$here/connector.sh" get "$connector" is_remote)" || exit 1

# 1. The place.
if ! "$is_remote"; then
	echo "begin: STOPPING: this brief was written for a cloud session, and this is not running in the environment the brief was written for. No work was started." >&2
	exit 1
fi

# 2. The user.
if [ "$(id -u)" = 0 ]; then
	echo "begin: STOPPING: refusing to run as root (uid 0); every check runs as the normal user the setup created." >&2
	exit 1
fi

# 3. The author. git's own resolution, with guessing switched off
# (user.useConfigOnly), so a name or address git would make up from the
# machine counts as unset rather than as someone's.
judge_identity() {
	local role="$1" var="$2" line name mail rc=0
	if ! line="$(git -C "$root" -c user.useConfigOnly=true var "$var" 2>/dev/null)"; then
		echo "begin: STOPPING before any work: the checkout's $role identity is not set, so no commit could be authored as the person. The setup takes it from the forge token; see what it reported." >&2
		exit 1
	fi
	# "Name <mail> <epoch> <tz>"
	name="$(printf '%s\n' "$line" | sed -n 's/^\(.*\) <[^<>]*> [0-9][0-9]* [-+][0-9][0-9]*$/\1/p')"
	mail="$(printf '%s\n' "$line" | sed -n 's/^.* <\([^<>]*\)> [0-9][0-9]* [-+][0-9][0-9]*$/\1/p')"
	if [ -z "$name" ] || [ -z "$mail" ]; then
		echo "begin: STOPPING before any work: the checkout's $role identity is not set in full, so no commit could be authored as the person." >&2
		exit 1
	fi
	# The gate prints what it refuses; that output is discarded unread, so the
	# identity never reaches this session's transcript.
	bash "$root/scripts/check-attribution.sh" identity "$role" "$name" "$mail" >/dev/null 2>&1 || rc=$?
	case "$rc" in
	0) ;;
	1)
		echo "begin: STOPPING before any work: the checkout's $role identity is not a person's (the attribution gate refuses it as an AI or machine identity), so no commit could be authored as the person. The setup takes the person's identity from the forge token; see what it reported." >&2
		exit 1
		;;
	*)
		echo "begin: STOPPING before any work: the attribution gate could not judge the checkout's $role identity (exit $rc)." >&2
		exit 1
		;;
	esac
}
judge_identity author GIT_AUTHOR_IDENT
judge_identity committer GIT_COMMITTER_IDENT

# 4. The machine.
exec bash "$here/env-check.sh"
