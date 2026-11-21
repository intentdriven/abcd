---
id: dcn-2611202018381471
slug: the-judge-accepts-a-sticky-directory
date: 2026-11-20
---
2026-11-20 — The judge accepts a sticky directory that others can write only over an entry this account or root owns, and still holds the sticky directory's own owner to this account or root (itd-2611041815429986, step 1). Grounds: in a sticky directory only the entry's owner, the directory's owner or root may rename or remove an entry, and the needs-step1 Opus review accepted the rule on that ground; a directory directly holding the copy gets no sticky allowance, since Admit refuses a group- or other-writable immediate directory first. Consequence: a Linuxbrew prefix owned by its own dedicated uid is refused as a foreign owner and can only be waived; step 2's surface text must say so. Refs itd-2611041815429986.
