---
id: dcn-2611260613242267
slug: while-the-person-is-away-a-prompt-they
date: 2026-11-26
---
2026-11-26 — While the person is away, a prompt they type opens a 15-minute window in which that session reads them as present at the stop check and at the question check (iss-2611260602137306, the person's pick of option A on 2026-11-26). A prompt counts as the person's unless it matches a wake prompt the session's stop checks have seen (kept hashed, newest 32) or opens with one of the host's own envelopes (task-notification, cross-session message, usage-limit resumption, continued-session summary); the host's UserPromptSubmit payload names no origin (code.claude.com/docs/en/hooks), so these are the documented fields that tell them apart. A wake not yet seen reads as the person's, which costs one unheld window, never a stop held against the person. abcd guard question, which names no session, reads any session of the checkout in its window. Reverse by removing the Conversing check from autonomy.Judge and awayAtQuestion and the NotePrompt call from the prompt router.
