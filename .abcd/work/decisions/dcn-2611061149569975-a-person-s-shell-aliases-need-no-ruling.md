---
id: dcn-2611061149569975
slug: a-person-s-shell-aliases-need-no-ruling
date: 2026-11-06
---
2026-11-06 — A person's shell aliases need no ruling before a drive. That is known from the code: a stage's shell home is its own folder, emptied at each launch and given only the git configuration the drive writes (`internal/core/runner/wall.go:335`), and a launch carries an allow-list of variables. It has not yet been seen in a real run. The hazard stands in a host session, whose shell reads the person's own profile (iss-2608301715040589, iss-2611040337241546).
