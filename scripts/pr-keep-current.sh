#!/usr/bin/env bash
# pr-keep-current.sh — keep auto-merge pull requests eligible for the merge queue.
#
# The default branch's ruleset requires a pull request to be up to date with
# its base before its checks count (strict required status checks), and that
# policy is deliberate: it is what gates a duplicate record id minted by two
# checkouts (iss-172, the invariant). Auto-merge enqueues a pull request only
# when its merge state is CLEAN, so a pull request whose base moved sits BEHIND
# with auto-merge armed and never enters the queue. GitHub does not update the
# branch for it.
#
# This is iss-172's first rung, run by the person or agent who armed auto-merge:
# walk every open pull request with auto-merge enabled, and where it is BEHIND
# and not already in the queue, ask the forge to merge the base into its branch.
# The update commit re-runs the required checks, the pull request turns CLEAN,
# and auto-merge enqueues it. Nothing is force-pushed and nothing is merged by
# this script; it only performs the update the web console offers by hand.
#
# Usage:
#   scripts/pr-keep-current.sh            # one pass
#   scripts/pr-keep-current.sh --watch    # repeat every two minutes until no
#                                         # open pull request has auto-merge
#
# Needs: gh (authenticated), a token allowed to push to the pull-request
# branches. Exits 0 when nothing needed updating.
set -euo pipefail

repo=$(gh repo view --json nameWithOwner -q .nameWithOwner)
interval=${PR_KEEP_CURRENT_INTERVAL:-120}
watch=false
[ "${1:-}" = "--watch" ] && watch=true

pass() {
  local n state queue
  local any=false
  while read -r n; do
    [ -n "$n" ] || continue
    any=true
    # One GraphQL read per pull request: merge state and queue membership.
    read -r state queue < <(gh api graphql \
      -f query="{ repository(owner:\"${repo%/*}\", name:\"${repo#*/}\") { pullRequest(number:$n) { mergeStateStatus mergeQueueEntry { state } } } }" \
      -q '.data.repository.pullRequest | "\(.mergeStateStatus) \(.mergeQueueEntry.state // "none")"')
    if [ "$state" = "BEHIND" ] && [ "$queue" = "none" ]; then
      msg=$(gh api -X PUT "repos/$repo/pulls/$n/update-branch" -f update_method=merge -q .message 2>&1 || true)
      printf '%s PR #%s was BEHIND with auto-merge armed: %s\n' "$(date +%H:%M)" "$n" "$msg"
    fi
  done < <(gh pr list --state open --json number,autoMergeRequest \
             -q '.[] | select(.autoMergeRequest != null) | .number')
  $any
}

if $watch; then
  while pass; do sleep "$interval"; done
  echo "no open pull request has auto-merge armed; done"
else
  pass || true
fi
