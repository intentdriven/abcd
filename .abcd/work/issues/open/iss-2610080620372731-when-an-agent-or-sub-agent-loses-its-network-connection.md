---
schema_version: 1
id: "iss-2610080620372731"
slug: "when-an-agent-or-sub-agent-loses-its-network-connection"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the guard-refusal session of 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "agents and sub-agents in autonomous runs (internal/core/implement)"
remedy: "On any lost connection (the host's model calls, a sub-agent, or a tool's network call such as git push or gh), a lane keeps doing offline work and, at a network step, waits on one shared probe recorded in the run state and run log: retry at 1, 5 and 10 minutes, then hourly for up to 8 hours, then stop the run and notify the product thinker; a lane whose agent died restarts as a fresh agent from its last commit, with uncommitted edits saved aside for review."
---

When an agent or sub-agent loses its network connection mid-task, the work stalls or fails instead of waiting the outage out, and this keeps happening on the product thinker's machine. An agent that notices a lost connection should pause and retry on a widening schedule: wait one minute and try again, then five minutes, then ten, then an hour. Sibling agents should also learn of the outage without depending on each other over the network, since they may have lost the connection too: a channel on the machine itself, such as the run state and run log abcd already shares between sessions, would let one agent record the outage and the others hold off instead of each failing on its own.

## The product thinker's decisions (interview of 2026-10-09)

- **What it covers:** every kind of lost connection, because which one strikes first is not yet known: the host's own model calls stalling, a sub-agent coming back failed mid-task, and a tool's network call (git push, gh, the merge-queue call, a download) failing. The first lab run records which kinds actually occur.
- **After the hour:** retries continue hourly for up to 8 hours after the hourly stage begins, so an outage at 23:00 is still retried in the morning; then the run stops cleanly and reports what was done and what is left.
- **Who waits:** lanes carry on with offline work (editing, tests); a lane that reaches a network step waits on one shared probe recorded in the run state and run log, instead of each lane retrying on its own.
- **A lane whose agent died:** when the network returns, a fresh agent starts from the lane's last commit; uncommitted edits are saved aside for review, never built on.
- **Telling the product thinker:** a notification only when the 8-hour limit is reached and the run stops; shorter outages appear in the end-of-run report with how long each lasted and what was retried.

### Further decisions (2026-10-09, while building)

- **One outage, two services:** the model service (Claude) being down is the same issue as the network being down. Each probe tests only what failed: the network by the git remote, the model service by a tiny canary agent the lead starts on the quickest, cheapest tier and reports with `implement outage probe --model ok|fail`.
- **A usage limit is not an outage:** a usage or rate limit (a 429, "usage limit") is reported as it is and never waited out as a lost connection.
- **Q1, an agent cut off mid-task:** it does not commit its edits. It stops and hands back with a final `NETWORK: <cmd>` line and no receipt; its uncommitted edits are saved aside for review through `implement step --restart <lane> --yielded "NETWORK: <cmd>"`, and the fresh agent starts from the last commit.
- **Q2, a lead whose own turn died:** every run arms a standing wake-up every 20 minutes for its whole duration, so a lead whose turn died to an outage still follows the schedule.
- **Q3, proving the model service back:** only the canary agent proves it, never the lead's own turn running. The canary runs only when `implement outage` shows a probe is due (read `next_probe_at` first), never once per lane.

Proof belongs in a lab that cuts the network during a live run and shows each of the three kinds waited out, the shared probe holding the other lanes, a dead lane restarted from its last commit, and the give-up path ending in one notification.
