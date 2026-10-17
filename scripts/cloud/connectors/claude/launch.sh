#!/usr/bin/env bash
# Start one hosted cloud session of this host (spc-2610120941441282 step 9,
# itd-2610120738075547 C10 and C5). Everything specific to how this host
# starts a session, and to how its failures show up, lives here; send.sh
# renders the outcome without naming the host (C12).
#
# The host's documented launch, read from its documentation on 2026-10-12 (the
# intent's "What the host offers today"):
#
#   claude -p "<prompt>" --environment <env-id> --ref <branch> --output-format json
#
# which returns the session's id and link. The answer's exact JSON fields and
# the wording of its failures are not documented; the shapes read below are
# the best reading of the host's usual JSON result and error lines, and the
# spec's step 12 proves them in a real session. A failure this script cannot
# place is a refusal (exit 7), never a success.
#
# Usage: launch.sh <environment-id> <ref> <brief-file>
#
# The contract with send.sh, the same for every connector:
#   exit 0  started; standard output is two lines, `id=<session id>` and
#           `link=<session link>`, and nothing else
#   exit 2  misuse
#   exit 3  the host's command-line tool is not installed here
#   exit 4  no login
#   exit 5  the login has expired (the person renews it by hand)
#   exit 6  the host does not know the environment
#   exit 7  the host refused the start for another reason (its own first line
#           of explanation on standard error)
#   exit 8  the host answered without naming a session, so one may have
#           started
# It never retries, never starts anything else in place of a refused start,
# and writes nothing but a temporary file it removes.
set -euo pipefail

if [ $# -ne 3 ] || [ -z "$1" ] || [ -z "$2" ] || [ ! -f "$3" ]; then
	echo "usage: launch.sh <environment-id> <ref> <brief-file>" >&2
	exit 2
fi
env_id="$1"
ref="$2"
brief_file="$3"

command -v claude >/dev/null 2>&1 || exit 3

prompt="$(cat "$brief_file")"
errf="$(mktemp "${TMPDIR:-/tmp}/abcd-launch.XXXXXX")"
trap 'rm -f -- "$errf"' EXIT

rc=0
out="$(claude -p "$prompt" --environment "$env_id" --ref "$ref" --output-format json 2>"$errf")" || rc=$?
err="$(cat "$errf")"

failed=0
[ "$rc" -eq 0 ] || failed=1
# Matched on the answer with its whitespace removed, so a line break inside
# the pair, as pretty-printed JSON has, cannot hide it (iss-2610170635209097).
case "$(printf '%s' "$out" | tr -d '[:space:]')" in
*'"is_error":true'*) failed=1 ;;
esac

if [ "$failed" -eq 1 ]; then
	said="$(printf '%s\n%s\n' "$err" "$out" | tr '[:upper:]' '[:lower:]')"
	case "$said" in
	*expired*) exit 5 ;;
	*"not logged in"* | *"/login"* | *"log in"* | *"invalid api key"* | *unauthori[sz]ed* | *"authentication"*) exit 4 ;;
	esac
	case "$said" in
	*environment*)
		case "$said" in
		*"not found"* | *unknown* | *invalid* | *"does not exist"* | *"no such"*) exit 6 ;;
		esac
		;;
	esac
	first="$(printf '%s\n%s\n' "$err" "$out" | grep -m 1 . || true)"
	echo "launch: the host refused the start (exit $rc): ${first:-it gave no reason}" >&2
	exit 7
fi

id="$(printf '%s\n' "$out" | sed -n 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([A-Za-z0-9_-]*\)".*/\1/p' | head -n 1)"
link=""
if [ -n "$id" ]; then
	link="$(printf '%s\n' "$out" | grep -oE 'https://[^"[:space:]\\]+' | grep -F -- "$id" | head -n 1 || true)"
fi
if [ -z "$id" ] || [ -z "$link" ]; then
	exit 8
fi
printf 'id=%s\nlink=%s\n' "$id" "$link"
