#!/usr/bin/env bash
# Print origin's forge repository as <owner>/<name> (spc-2610120941441282),
# for the one REST call the hand-back's fallback makes and the one the list
# reads: the tracking issue's comments.
#
# Parsed strictly from origin's configured URL (remote.origin.url, as written,
# before any insteadOf rewrite), and only from a plain github.com form:
#
#   https://github.com/<owner>/<name>[.git]
#   git@github.com:<owner>/<name>[.git]
#   ssh://git@github.com/<owner>/<name>[.git]
#
# Anything else is refused: another host, a port, credentials in the URL, a
# proxy, a path with more segments, an owner or name outside the forge's own
# alphabet. A refusal is never a guess at where a comment would go: the
# report then goes nowhere but the transcript.
#
# Usage: forge-repo.sh   (run from inside the checkout)
# Exit 0 with <owner>/<name> on standard output; 1 refused, naming why.
set -euo pipefail

if [ $# -ne 0 ]; then
	echo "usage: forge-repo.sh" >&2
	exit 2
fi

url="$(git config --get remote.origin.url 2>/dev/null)" || url=""
if [ -z "$url" ]; then
	echo "forge-repo: this checkout has no remote named origin, so it names no forge repository." >&2
	exit 1
fi

OWNER_RE='[A-Za-z0-9]([A-Za-z0-9-]{0,37}[A-Za-z0-9])?'
NAME_RE='[A-Za-z0-9._-]{1,100}'
path=""
if [[ "$url" =~ ^https://github\.com/(.+)$ ]] ||
	[[ "$url" =~ ^git@github\.com:(.+)$ ]] ||
	[[ "$url" =~ ^ssh://git@github\.com/(.+)$ ]]; then
	path="${BASH_REMATCH[1]}"
fi
path="${path%.git}"
if [ -z "$path" ] || ! [[ "$path" =~ ^$OWNER_RE/$NAME_RE$ ]]; then
	echo "forge-repo: origin's URL is not a plain github.com <owner>/<name> URL, so no forge repository is read from it." >&2
	exit 1
fi
case "/$path/" in
*/./* | */../*)
	echo "forge-repo: origin's URL names no repository." >&2
	exit 1
	;;
esac
printf '%s\n' "$path"
