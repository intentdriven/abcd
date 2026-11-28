---
schema_version: 1
id: "iss-2611281134242570"
slug: "the-name-abcd-is-already-used-by-other"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "away run 2026-11-28"
origin: researcher-authored
production_mode: hand-written
remedy: "Decide whether to keep the name: record the overlap in the naming section of the brief, make every abcd doctor/ahoy check of the PATH copy say which program answered when 'abcd --version' does not identify as this one, and leave a rename as a separate product decision for the person."
---

The name abcd is already used by other projects: libAtoms/abcd (https://github.com/libAtoms/abcd), a Python database for atomistic data, installs a console command named abcd, and the PyPI package name abcd belongs to an unrelated build CLI (https://pypi.org/project/abcd/). A person with either installed can have a different abcd first on PATH, and a search for the name finds them first.
