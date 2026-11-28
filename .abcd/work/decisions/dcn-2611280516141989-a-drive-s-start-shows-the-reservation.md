---
id: dcn-2611280516141989
slug: a-drive-s-start-shows-the-reservation
date: 2026-11-28
---
2026-11-28 — A drive's start shows the reservation ceiling on its holds line as this checkout last fetched it, and asks origin for it nowhere before the yes (spc-2611251810508418 step 3). Grounds: a start given --until asks origin nothing; asking for the ceiling alone would add a network read and a fetch to that start's window before the question, the window a first interrupt must still end. The line is only shown, and every act that adds a reservation asks origin itself, so a figure that has gone stale since the last fetch changes what the yes reads and nothing that is enforced; the line names its source so that it does not read as origin's own answer. To reverse: ask origin through claim.AskOriginDefault before the consent and read the figure at the tip it names, as the default length is read when no --until is given.
