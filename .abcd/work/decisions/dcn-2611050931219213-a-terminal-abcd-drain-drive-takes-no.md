---
id: dcn-2611050931219213
slug: a-terminal-abcd-drain-drive-takes-no
date: 2026-11-05
---
2026-11-05 — A terminal `abcd drain --drive` takes no off-profile warning: it starts agents of its own from a plain terminal, so it is not a session starting work under the person's own setup. This changes the clause of the itd-2611040344170206 criterion opening "Given a session that is not on a profile" that listed `abcd drain --drive` among the starts that warn.
