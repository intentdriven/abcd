---
schema_version: 1
id: "iss-2610080621450495"
slug: "consider-a-collection-of-agents-for-abcd-s-autonomous-runs"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "the guard-refusal session of 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "autonomous runs: the implement loop and the drain (internal/core/implement)"
remedy: "Run a lab comparing abcd's current in-session lanes with a collection of agents, each worker an independent terminal session in a store worktree, supervised by a zero-token shell watcher that wakes the lead only on a worker event; measure the lead's context and token use, wall-clock time, and how each survives a lost connection and a restart, then decide whether the implement loop and drain adopt it."
---

Consider running abcd's autonomous runs as a collection of agents, a shape seen in an outside open-source agent tool: the person talks to one lead agent; the lead hands each task to a worker running as its own terminal session in its own git worktree; and a plain shell watcher, which costs no model tokens, sleeps on the workers and wakes the lead only when one finishes, fails or needs a decision. Today abcd's lanes run as sub-agents inside the host session, and the lead keeps checking on them itself (re-arming watchers on pull requests, polling queue state), which spends its context on waiting. Independent worker sessions would also contain a lost connection or a crash to one worker instead of the whole run, and state kept on disk lets every worker resume after a restart. The same tool can run workers on another machine over SSH, which is worth weighing later.

Two personal-agent products that large AI vendors launched in September 2026 point the same way on three counts and differ on one. Both keep working after the person leaves and come back only when something changes or needs approval, which is the watcher's job here. Both run each agent in its own isolated computer, a cloud virtual machine where this proposal uses a worktree and a terminal session. Both gate consequential actions outside the agent itself: one runs a separate gatekeeper agent on the same machine, kept apart from the worker at the system level, through which every outward action must pass; the other checks such actions against the person's rules before deciding whether they need approval. Both also show the person an activity view or audit trail of everything done and planned. They differ in shape: each is one persistent agent per person that juggles several projects, with people and agents meeting on shared pages, not a lead dispatching a collection of parallel workers, and they run in the vendor's cloud, not on the person's machine.

Three of their choices are worth carrying into the lab. A gatekeeper that sits outside the worker, as abcd's guard already does per tool call, could extend to every outward act a worker makes: a push, a pull request, a comment. An activity view of what each worker has done and plans next is the natural first page for the product thinker's dashboard. And a worker that runs away from the person's machine keeps going when that machine loses its connection, which bears on iss-2610080620372731. The broader coordination survey that iss-2608230943533581 asks for is where this comparison belongs once it runs.
