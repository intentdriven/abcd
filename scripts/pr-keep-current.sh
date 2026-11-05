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
# and not already in the queue, withdraw its arming and ask the forge to merge
# the base into its branch, pinned to the head it read. An update that moves the
# head leaves it unarmed: `abcd land` arms the new head once a review receipt
# names it. An update that fails leaves the head where it was, and the arming
# goes back on at that head. Nothing is force-pushed and nothing is merged by this
# script; it only performs the update the web console offers by hand.
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
settle=${PR_KEEP_CURRENT_SETTLE:-30}
watch=false
[ "${1:-}" = "--watch" ] && watch=true

# moved_head waits up to $settle seconds for pull request $1's head to leave $2
# (the forge's update is asynchronous), and prints the head it then reads.
moved_head() {
  local now waited=0
  while :; do
    now=$(gh pr view "$1" --json headRefOid -q .headRefOid 2>/dev/null || true)
    if [ -n "$now" ] && [ "$now" != "$2" ]; then break; fi
    [ "$waited" -lt "$settle" ] || break
    sleep 5
    waited=$((waited + 5))
  done
  printf '%s' "${now:-$2}"
}

pass() {
  local n state queue head method flag msg now
  local any=false
  while read -r n; do
    [ -n "$n" ] || continue
    any=true
    # One GraphQL read per pull request: merge state, queue membership, and the
    # head and merge method an update is pinned to and an arming put back by.
    read -r state queue head method < <(gh api graphql \
      -f query="{ repository(owner:\"${repo%/*}\", name:\"${repo#*/}\") { pullRequest(number:$n) { mergeStateStatus headRefOid mergeQueueEntry { state } autoMergeRequest { mergeMethod } } } }" \
      -q '.data.repository.pullRequest | "\(.mergeStateStatus) \(.mergeQueueEntry.state // "none") \(.headRefOid) \(.autoMergeRequest.mergeMethod // "MERGE")"')
    if [ "$state" = "BEHIND" ] && [ "$queue" = "none" ]; then
      if ! gh pr merge "$n" --disable-auto >/dev/null 2>&1; then
        printf '%s PR #%s was BEHIND, and its arming could NOT be withdrawn, so it was not updated\n' "$(date +%H:%M)" "$n"
        continue
      fi
      if msg=$(gh api -X PUT "repos/$repo/pulls/$n/update-branch" -f update_method=merge -f expected_head_sha="$head" -q .message 2>&1); then
        now=$(moved_head "$n" "$head")
        if [ "$now" != "$head" ]; then
          printf '%s PR #%s was BEHIND and is updated to %s; it is unarmed, and abcd land arms it once a review receipt names that head\n' "$(date +%H:%M)" "$n" "${now:0:12}"
        else
          printf '%s PR #%s was BEHIND: the forge took the update (%s), its head has not moved yet, and it is unarmed; abcd land arms it once a review receipt names its head\n' "$(date +%H:%M)" "$n" "$msg"
        fi
      else
        flag="--$(printf '%s' "$method" | tr '[:upper:]' '[:lower:]')"
        if gh pr merge "$n" --auto "$flag" "--match-head-commit=$head" >/dev/null 2>&1; then
          printf '%s PR #%s was BEHIND and the update failed (%s); its head is still %s, and its arming is re-armed there\n' "$(date +%H:%M)" "$n" "$msg" "${head:0:12}"
        else
          printf '%s PR #%s was BEHIND and the update failed (%s); its arming could NOT be put back; arm it with: gh pr merge %s --auto %s --match-head-commit=%s\n' "$(date +%H:%M)" "$n" "$msg" "$n" "$flag" "$head"
        fi
      fi
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
