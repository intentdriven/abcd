---
id: dcn-2611012113432054
slug: when-a-build-agent-abcd-hands-to-a
date: 2026-11-01
---
2026-11-01 — When a build agent abcd hands to a runner fails to start, the work waits for the person's API key instead of falling back to the session that started the build: the lane pauses and says why, and nothing is billed to the session's subscription. Each later step retries the paused role once on the runner, so the work resumes as soon as the key works, and a retry can also be asked for by hand. The product thinker ruled this on 2026-11-01.
