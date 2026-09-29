#!/usr/bin/env bash
# Dependency-bump re-authoring (itd-2609221842494980), scaffolded by
# `abcd launch scaffold --dependency-reauthor`.
#
# A bot-opened dependency bump is re-authored as the repository owner when, and
# only when, it is inside the BOUND: the pull request was opened by a bot the
# repository declares, from a branch in this repository under that bot's prefix
# for an ecosystem the repository declares, carrying exactly one commit, authored
# by that bot, whose diff touches nothing but that ecosystem's manifest and lock
# files. Anything else is left exactly as it is, and the run names the clause the
# commit failed. The attribution gate is not involved and not changed: the
# re-authored commit passes it because a person is its author and committer.
#
# The declaration is .abcd/config/dependency-reauthor.conf, read from the
# checkout the workflow made of the pull request's BASE, never from the branch
# under judgement:
#
#   owner_name=<the person whose authorship the re-authored commit carries>
#   owner_email=<that person's address>
#   ecosystem=<bot login> <branch prefix> <file name> [<file name>...]
#
# The owner is the person's to set; while either value is empty an in-bound bump
# is REFUSED, never re-authored as anybody else.
#
# The push is made with a GitHub App installation token minted here from two
# secrets the person creates: DEPENDENCY_REAUTHOR_APP_ID and
# DEPENDENCY_REAUTHOR_APP_KEY (the App's private key). A missing secret REFUSES
# the in-bound bump; the workflow's own GITHUB_TOKEN is never used to push, and a
# push by it would start no checks anyway. The App only pushes: it is neither
# the author nor the committer of anything.
#
# Usage:
#   dependency-reauthor.sh check   the bound alone; writes nothing, needs no secret
#   dependency-reauthor.sh run     the bound, then an in-bound bump is re-authored
#                                  and pushed back to its branch
#
# Inputs come from the environment, which the workflow fills from the event;
# nothing the pull request's author wrote is ever spliced into this script:
#   PR_AUTHOR HEAD_REF HEAD_SHA BASE_SHA HEAD_REPO GITHUB_REPOSITORY
#   DEPENDENCY_REAUTHOR_APP_ID DEPENDENCY_REAUTHOR_APP_KEY   (run, in bound only)
#   REAUTHOR_CONF      default .abcd/config/dependency-reauthor.conf
#   REAUTHOR_WORKFLOW  the workflow named in the message (default
#                      .github/workflows/dependency-reauthor.yml)
#   GITHUB_API_URL GITHUB_SERVER_URL GITHUB_STEP_SUMMARY   (set by the runner)
#   REAUTHOR_REMOTE    where the push goes (default the repository on the forge)
#
# Exit 0: re-authored, or left alone with the reason printed.
# Exit 1: refused — the declaration is malformed, the owner or a secret is
#         missing, or minting or pushing failed. Nothing was pushed.
# Exit 2: usage.
set -euo pipefail
# No pathname expansion anywhere: rows are word-split on purpose, and a bot login
# such as dependabot[bot] is a bracket expression to the globber.
set -f

CONF="${REAUTHOR_CONF:-.abcd/config/dependency-reauthor.conf}"
WORKFLOW="${REAUTHOR_WORKFLOW:-.github/workflows/dependency-reauthor.yml}"

say() { printf 'dependency-reauthor: %s\n' "$*"; }
refuse() {
	printf '::error::dependency-reauthor: refused: %s\n' "$*" >&2
	exit 1
}
# leave_alone reports the clause the commit failed and ends the run green:
# a pull request outside the bound is not a fault, it is a person's to land.
leave_alone() {
	say "left alone ($1): $2"
	if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
		printf 'dependency-reauthor: left alone (%s): %s\n' "$1" "$2" >>"$GITHUB_STEP_SUMMARY"
	fi
	exit 0
}

mode="${1:-}"
case "$mode" in
check | run) ;;
*)
	echo "usage: dependency-reauthor.sh check|run" >&2
	exit 2
	;;
esac

# --- The declaration ---------------------------------------------------------
[ -f "$CONF" ] || refuse "no declaration at $CONF: the repository has not opted in"
owner_name=""
owner_email=""
rows=""
lineno=0
while IFS= read -r line || [ -n "$line" ]; do
	lineno=$((lineno + 1))
	line="${line%%#*}"
	line="$(printf '%s' "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
	[ -n "$line" ] || continue
	case "$line" in
	*=*) ;;
	*) refuse "$CONF line $lineno is not key=value" ;;
	esac
	key="$(printf '%s' "${line%%=*}" | sed 's/[[:space:]]*$//')"
	val="$(printf '%s' "${line#*=}" | sed 's/^[[:space:]]*//')"
	case "$key" in
	owner_name) owner_name="$val" ;;
	owner_email) owner_email="$val" ;;
	ecosystem)
		# shellcheck disable=SC2086 # word-splitting the row is the point
		set -- $val
		[ $# -ge 3 ] || refuse "$CONF line $lineno: an ecosystem row is <bot> <branch prefix> <file>..."
		rows="$rows$val
"
		;;
	*) refuse "$CONF line $lineno: unknown key '$key'" ;;
	esac
done <"$CONF"

# --- The bound -----------------------------------------------------------------
: "${PR_AUTHOR:?PR_AUTHOR is required}" "${HEAD_REF:?HEAD_REF is required}"
: "${HEAD_SHA:?HEAD_SHA is required}" "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_REPO:?HEAD_REPO is required}" "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

bot=""
prefix=""
files=""
known_bot=0
while IFS= read -r row; do
	[ -n "$row" ] || continue
	# shellcheck disable=SC2086
	set -- $row
	[ "$1" = "$PR_AUTHOR" ] || continue
	known_bot=1
	case "$HEAD_REF" in
	"$2"*)
		bot="$1"
		prefix="$2"
		shift 2
		files=" $* "
		break
		;;
	esac
done <<EOF
$rows
EOF

[ "$known_bot" -eq 1 ] || leave_alone author "the pull request's author $PR_AUTHOR is not a bot this repository declares"
[ "$HEAD_REPO" = "$GITHUB_REPOSITORY" ] ||
	leave_alone head-repo "the branch lives in $HEAD_REPO, not in $GITHUB_REPOSITORY"
[ -n "$bot" ] || leave_alone ecosystem "the branch $HEAD_REF matches no ecosystem this repository declares for $PR_AUTHOR"
printf '%s' "$HEAD_REF" | grep -Eq '^[A-Za-z0-9._/+-]+$' ||
	leave_alone branch "the branch name carries a character outside [A-Za-z0-9._/+-]"
for sha in "$HEAD_SHA" "$BASE_SHA"; do
	printf '%s' "$sha" | grep -Eq '^([0-9a-f]{40}|[0-9a-f]{64})$' || refuse "'$sha' is not a full commit id"
	git cat-file -e "$sha^{commit}" 2>/dev/null || refuse "commit $sha is not in this checkout"
done

fork="$(git merge-base "$BASE_SHA" "$HEAD_SHA")" || refuse "no merge base between $BASE_SHA and $HEAD_SHA"
count="$(git rev-list --count "$fork..$HEAD_SHA")"
[ "$count" -eq 1 ] || leave_alone commits "the branch carries $count commits; a bump is one commit"
author="$(git show -s --format='%an' "$HEAD_SHA")"
[ "$author" = "$bot" ] || leave_alone commit-author "the head commit is authored by $author, not by $bot"

diff="$(git diff --raw --no-renames --no-abbrev "$fork" "$HEAD_SHA")" ||
	refuse "could not diff $fork..$HEAD_SHA"
changed=0
while IFS= read -r raw; do
	[ -n "$raw" ] || continue
	# :<old mode> <new mode> <old sha> <new sha> <status>\t<path>
	meta="${raw%%	*}"
	path="${raw#*	}"
	# shellcheck disable=SC2086
	set -- $meta
	newmode="$2"
	status="$5"
	case "$status" in
	M | A) ;;
	*) leave_alone diff "$path is $status in the diff; a bump only modifies or adds" ;;
	esac
	case "$newmode" in
	100644 | 100755) ;;
	*) leave_alone diff "$path has mode $newmode; a bump changes regular files" ;;
	esac
	base_name="${path##*/}"
	case "$files" in
	*" $base_name "*) ;;
	*) leave_alone diff "$path is not one of the declared files for $prefix:$files" ;;
	esac
	changed=$((changed + 1))
done <<EOF
$diff
EOF
[ "$changed" -gt 0 ] || leave_alone diff "the commit changes no file"

subject="$(git show -s --format='%s' "$HEAD_SHA")"
say "in bound: $bot, $prefix, $changed file(s), \"$subject\""
[ "$mode" = run ] || exit 0

# --- The owner and the secrets: refuse, never fall back -----------------------
[ -n "$owner_name" ] || refuse "owner_name is empty in $CONF; the person whose authorship this workflow asserts sets it"
[ -n "$owner_email" ] || refuse "owner_email is empty in $CONF; the person whose authorship this workflow asserts sets it"
case "$owner_email" in
*@*) ;;
*) refuse "owner_email in $CONF is not an address" ;;
esac
[ -n "${DEPENDENCY_REAUTHOR_APP_ID:-}" ] ||
	refuse "the secret DEPENDENCY_REAUTHOR_APP_ID is not set (a Dependabot secret: the GitHub App's id)"
[ -n "${DEPENDENCY_REAUTHOR_APP_KEY:-}" ] ||
	refuse "the secret DEPENDENCY_REAUTHOR_APP_KEY is not set (a Dependabot secret: the GitHub App's private key)"
printf '%s' "$DEPENDENCY_REAUTHOR_APP_ID" | grep -Eq '^[A-Za-z0-9._-]+$' ||
	refuse "DEPENDENCY_REAUTHOR_APP_ID is not an App id"
printf '%s' "$GITHUB_REPOSITORY" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$' ||
	refuse "GITHUB_REPOSITORY '$GITHUB_REPOSITORY' is not owner/name"
for tool in openssl curl jq; do
	command -v "$tool" >/dev/null 2>&1 || refuse "$tool is not on PATH"
done

# --- The re-authored commit ------------------------------------------------------
# The same tree and parent as the bot's commit; the person as author AND
# committer (`git commit --amend --reset-author`, made without a working tree).
msgfile="$(mktemp "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/reauthor-msg.XXXXXX")"
trap '/bin/rm -f "$msgfile"' EXIT
{
	printf '%s\n\n' "$subject"
	printf 'Proposed by %s as %s; re-authored by %s under the bound\n' "$bot" "$HEAD_SHA" "$WORKFLOW"
	printf 'declared in %s (%s).\n\n' "$CONF" "$prefix"
	printf 'Assisted-by: None\n'
} >"$msgfile"
new="$(GIT_AUTHOR_NAME="$owner_name" GIT_AUTHOR_EMAIL="$owner_email" \
	GIT_COMMITTER_NAME="$owner_name" GIT_COMMITTER_EMAIL="$owner_email" \
	git commit-tree "$HEAD_SHA^{tree}" -p "$fork" -F "$msgfile")" || refuse "could not compose the re-authored commit"

# --- The App installation token ------------------------------------------------
api="${GITHUB_API_URL:-https://api.github.com}"
b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }
now="$(date +%s)"
header="$(printf '{"alg":"RS256","typ":"JWT"}' | b64url)"
claims="$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' "$((now - 60))" "$((now + 540))" "$DEPENDENCY_REAUTHOR_APP_ID" | b64url)"
signature="$(printf '%s.%s' "$header" "$claims" |
	openssl dgst -sha256 -sign <(printf '%s\n' "$DEPENDENCY_REAUTHOR_APP_KEY") -binary | b64url)" ||
	refuse "could not sign with DEPENDENCY_REAUTHOR_APP_KEY (is it the App's PEM private key?)"
[ -n "$signature" ] || refuse "could not sign with DEPENDENCY_REAUTHOR_APP_KEY (is it the App's PEM private key?)"
jwt="$header.$claims.$signature"

# call <method> <path> <credential> [body]: the credential reaches curl through a
# header file on a descriptor, never through its argument list.
call() {
	local method="$1" path="$2" cred="$3" body="${4:-}"
	if [ -n "$body" ]; then
		curl -fsS --max-time 30 -X "$method" -H 'Accept: application/vnd.github+json' \
			-H @<(printf 'Authorization: Bearer %s\n' "$cred") --data "$body" "$api$path"
	else
		curl -fsS --max-time 30 -X "$method" -H 'Accept: application/vnd.github+json' \
			-H @<(printf 'Authorization: Bearer %s\n' "$cred") "$api$path"
	fi
}

installation="$(call GET "/repos/$GITHUB_REPOSITORY/installation" "$jwt" | jq -r '.id // empty')" || true
printf '%s' "$installation" | grep -Eq '^[0-9]+$' ||
	refuse "the App named by DEPENDENCY_REAUTHOR_APP_ID is not installed on $GITHUB_REPOSITORY"
scope="$(jq -cn --arg name "${GITHUB_REPOSITORY#*/}" '{repositories: [$name], permissions: {contents: "write"}}')"
token="$(call POST "/app/installations/$installation/access_tokens" "$jwt" "$scope" | jq -r '.token // empty')" || true
[ -n "$token" ] || refuse "the App could not mint an installation token for $GITHUB_REPOSITORY"
if [ -n "${GITHUB_ACTIONS:-}" ]; then
	echo "::add-mask::$token"
fi
trap '/bin/rm -f "$msgfile"; call DELETE /installation/token "$token" >/dev/null 2>&1 || true' EXIT

# --- The push --------------------------------------------------------------------
# Only when the branch still holds the bot's commit (the lease): a bump the bot
# rebased meanwhile is judged again on its own run, not overwritten here. The
# token reaches git through a credential helper reading the environment.
remote="${REAUTHOR_REMOTE:-${GITHUB_SERVER_URL:-https://github.com}/$GITHUB_REPOSITORY.git}"
# shellcheck disable=SC2016 # the helper expands the variable, not this shell
REAUTHOR_TOKEN="$token" git -c credential.helper= \
	-c credential.helper='!f() { test "$1" = get || exit 0; echo username=x-access-token; echo "password=$REAUTHOR_TOKEN"; }; f' \
	push --force-with-lease="refs/heads/$HEAD_REF:$HEAD_SHA" "$remote" "$new:refs/heads/$HEAD_REF" ||
	refuse "the push of $new to $HEAD_REF was refused; nothing was re-authored"

# --- The record ------------------------------------------------------------------
record="re-authored $bot bump \"$subject\" on $HEAD_REF: $HEAD_SHA -> $new (author and committer $owner_name)"
say "$record"
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
	printf 'dependency-reauthor: %s\n' "$record" >>"$GITHUB_STEP_SUMMARY"
fi
