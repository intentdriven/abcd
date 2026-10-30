#!/usr/bin/env python3
"""Build data.js for the abcdev.app prototype from record.json (extracted from the repo).

Mirrors what an `abcd site build` step would do: read the record, derive counts,
precompute a deterministic force-directed layout, and emit one JSON blob.
"""
import json, math, os, re, random
import numpy as np

rec = json.load(open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'record.json')))
nodes = rec['nodes']
edges = rec['edges']
mentions = rec['mentions']

# --- index ---------------------------------------------------------------
idx = {n['id']: i for i, n in enumerate(nodes)}
N = len(nodes)

# distinct links: the spec link is recorded from both ends (intent `spec_id` <-> spec `implements`),
# and some `related_*` pairs are listed in both files; keep one edge each. Direction is kept for
# builds_on / supersedes / implements (source file -> target), related is symmetric.
seen = {}
links = []
for e in edges:
    s, t, r = idx[e['s']], idx[e['t']], e['r']
    if r == 'spec':
        s, t, r = t, s, 'implements'          # spec -> intent, same link as the spec's own `implements`
    key = (r, min(s, t), max(s, t)) if r in ('related', 'implements') else (r, s, t)
    if key in seen:
        continue
    seen[key] = True
    links.append((s, t, r))
print('stored references', len(edges), 'distinct links', len(links))
# degree (typed + mention), used for node radius
deg = np.zeros(N)
typed_pairs = []
for s, t, r in links:
    typed_pairs.append((s, t))
    deg[s] += 1; deg[t] += 1
mention_pairs = []
typed_set = set((min(a, b), max(a, b)) for a, b in typed_pairs)
for m in mentions:
    s, t = idx[m['s']], idx[m['t']]
    key = (min(s, t), max(s, t))
    if key in typed_set:
        continue
    mention_pairs.append((s, t))
    deg[s] += 0.35; deg[t] += 0.35

# --- layout: the coil ---------------------------------------------------------------------
# Records are placed one by one in date order (frontmatter `date` where a record carries one,
# otherwise the day its file first appeared in git; same-day ties by type, then id). The first
# sits at the centre; each next bubble goes beside the previous one — a little further round,
# as close to the centre as the bubbles already placed allow — so the sequence winds outward
# like a snail shell. Deterministic, no simulation: the picture is the same on every build.
dates = json.load(open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'dates.json')))   # path -> [created, entered current dir, last touched] from git log
TYPE_ORDER = {'adr': 0, 'principle': 1, 'intent': 2, 'spec': 3, 'issue': 4, 'rfc': 5, 'phase': 6}
def id_num(s):
    m = re.search(r'(\d+)', s); return int(m.group(1)) if m else 0
eff_date = [n.get('date') or dates[n['path']][0] for n in nodes]
order = sorted(range(N), key=lambda i: (eff_date[i], TYPE_ORDER.get(nodes[i]['type'], 9), id_num(nodes[i]['id'])))
# bubble radii in reference pixels (the renderer's formula at its desktop scale)
REF_SCALE = 0.86
br = np.array([((3 if n['type'] == 'issue' else 4.5) + min(math.sqrt(deg[i]), 4) * (1.6 if n['type'] == 'issue' else 3.2)) for i, n in enumerate(nodes)]) * REF_SCALE
GAP = 4.5                        # breathing room between neighbours on the coil (reference pixels)
pos = np.zeros((N, 2)); pr = np.zeros(N)
placed = []                      # indices in placement order
phi = 0.0                        # unwrapped angle of the last bubble
for k, i in enumerate(order):
    r = br[i]
    if k == 0:
        pos[i] = (0.0, 0.0); pr[i] = r; placed.append(i); continue
    prev = placed[-1]
    px, py = pos[prev]; prho = math.hypot(px, py)
    need = pr[prev] + r + GAP
    clear = math.pi / 2 if prho < need else math.asin(need / prho)    # the ray must pass the previous bubble
    phi += clear
    ux, uy = math.cos(phi), math.sin(phi)
    floor = max(0.0, prho - 0.35 * need)                              # only a shallow dip inward: keeps the coil a path
    # forbidden intervals of rho along the ray, one per placed bubble
    P = pos[placed]; Rr = pr[placed] + r + GAP
    proj = P[:, 0] * ux + P[:, 1] * uy
    perp2 = (P[:, 0] ** 2 + P[:, 1] ** 2) - proj ** 2
    hit = perp2 < Rr ** 2
    half = np.sqrt(np.maximum(Rr ** 2 - perp2, 0.0))
    lo = (proj - half)[hit]; hi = (proj + half)[hit]
    rho = floor
    if len(lo):
        o = np.argsort(lo); lo = lo[o]; hi = hi[o]
        changed = True
        while changed:
            changed = False
            for a_, b_ in zip(lo, hi):
                if a_ <= rho < b_:
                    rho = b_; changed = True
    pos[i] = (rho * ux, rho * uy); pr[i] = r; placed.append(i)
rho_max = float(max(math.hypot(*pos[i]) + pr[i] for i in range(N)))
# sanity: overlaps after placement
Dm = pos[:, None, :] - pos[None, :, :]; dist = np.linalg.norm(Dm, axis=2); np.fill_diagonal(dist, 1e9)
min_d = pr[:, None] + pr[None, :]
overlaps = int((dist < min_d - 0.01).sum() // 2)
# month markers: the first record of each month in placement order
months = []; seen_m = set()
for i in order:
    m = eff_date[i][:7]
    if m not in seen_m:
        seen_m.add(m); months.append([m, i])
pos = pos / rho_max             # unit disk; the renderer scales by its own disk radius

# --- second arrangement: by links --------------------------------------------------------------
# A force layout over the typed links only (body mentions excluded: with them it is a hairball).
# Connected work forms islands; records with no typed cross-reference sit on the rim, in date order,
# so a bubble travels a short, readable path when the viewer switches arrangement.
import networkx as nx
Gt = nx.Graph(); Gt.add_nodes_from(range(N))
for s, t in typed_pairs:
    Gt.add_edge(s, t)
tdeg = dict(Gt.degree())
for s, t, d in Gt.edges(data=True):
    d['weight'] = 1.0 / math.sqrt(max(tdeg[s], 1) * max(tdeg[t], 1)) * 3
comps = sorted(nx.connected_components(Gt), key=lambda c: (-len(c), min(c)))
fpos = np.zeros((N, 2))
big = [c for c in comps if len(c) >= 3]
small = [c for c in comps if len(c) == 2]
iso = [next(iter(c)) for c in comps if len(c) == 1]
# islands of 3+: the largest in the middle, the others beside it
rng = np.random.default_rng(11)
island_centres = [(0.0, 0.0)] + [(math.cos(2 * math.pi * j / max(len(big) - 1, 1) - math.pi / 2 + 0.8) * 0.80, math.sin(2 * math.pi * j / max(len(big) - 1, 1) - math.pi / 2 + 0.8) * 0.80) for j in range(len(big) - 1)]
for j, c in enumerate(big):
    Hc = Gt.subgraph(c)
    init = {i: rng.normal(0, 0.3, 2) for i in c}
    sp = nx.spring_layout(Hc, k=1.6 / math.sqrt(len(c)), pos=init, iterations=300, weight='weight', seed=11, scale=1.0)
    P = np.array([sp[i] for i in c]); P -= P.mean(axis=0)
    rad = np.linalg.norm(P, axis=1); ang = np.arctan2(P[:, 1], P[:, 0])
    rad = np.minimum(rad / np.percentile(rad, 97), 1.0) ** 0.5           # decompress the dense middle
    P = np.stack([np.cos(ang) * rad, np.sin(ang) * rad], axis=1)
    scale = 0.72 if j == 0 else 0.09
    cx, cy = island_centres[j]
    for q, i in enumerate(c):
        fpos[i] = (cx + P[q][0] * scale, cy + P[q][1] * scale)
# pairs: a ring just outside the main island
for j, c in enumerate(small):
    a = 2 * math.pi * j / max(len(small), 1) - math.pi / 2 + 0.2
    cx, cy = math.cos(a) * 0.80, math.sin(a) * 0.80
    m = sorted(c)
    fpos[m[0]] = (cx - math.sin(a) * 0.012, cy + math.cos(a) * 0.012)
    fpos[m[1]] = (cx + math.sin(a) * 0.012, cy - math.cos(a) * 0.012)
# unlinked records: the rim, in date order, two rows, spaced by bubble size so nothing overlaps
iso.sort(key=lambda i: (eff_date[i], TYPE_ORDER.get(nodes[i]['type'], 9), id_num(nodes[i]['id'])))
rows = {0: [], 1: []}
for j, i in enumerate(iso):
    rows[j % 2].append(i)
for row, mem in rows.items():
    rr = 0.905 if row == 0 else 0.97
    widths = np.array([2 * br[i] + GAP for i in mem]); total = widths.sum()
    cum = np.concatenate([[0], np.cumsum(widths)[:-1]]) + widths / 2
    for i, c in zip(mem, cum):
        a = 2 * math.pi * c / total - math.pi / 2
        fpos[i] = (math.cos(a) * rr, math.sin(a) * rr)
links_meta = {'islands': [len(c) for c in big], 'pairs': len(small), 'unlinked': len(iso)}
print(links_meta)
span = [min(eff_date), max(max(dates[n['path']][2] for n in nodes), max(eff_date))]
layout_meta = {'span': span, 'overlaps': overlaps, 'isolated': int(sum(1 for i in range(N) if deg[i] == 0)), 'coil_radius': round(rho_max, 1),
               'ref_scale': REF_SCALE, 'months': months, 'order': order, 'date_range': [min(eff_date), max(eff_date)], 'links': links_meta}
print({k: v for k, v in layout_meta.items() if k != 'order'})

# --- counts ------------------------------------------------------------------
from collections import Counter, defaultdict
by = defaultdict(Counter)
for n in nodes:
    by[n['type']][n['status']] += 1
counts = {t: dict(c) for t, c in by.items()}

# --- adr timeline ---------------------------------------------------------------
adrs = sorted([n for n in nodes if n['type'] == 'adr' and n['date']], key=lambda n: n['date'])
supersedes = [(nodes[s]['id'], nodes[t]['id']) for s, t, r in links if r == 'supersedes']
dangling = rec.get('dangling', [])

# --- releases (newest first in CHANGELOG) ---------------------------------------------
releases = list(reversed(rec['releases']))

# --- references from ACKNOWLEDGEMENTS.md ---------------------------------------------
ack = open(os.path.join(os.environ.get('ABCD_ORIGINALS_DIR', '_originals'), 'ACKNOWLEDGEMENTS.md'), encoding='utf-8').read()
refsec = ack.split('## References & sources', 1)[1]
entries = re.split(r'\n(?=\d+\. )', refsec.strip())
refs = []
for en in entries:
    m = re.match(r'(\d+)\. (.*)', en, re.S)
    if not m:
        continue
    body = ' '.join(line.strip() for line in m.group(2).splitlines())
    # markdown -> html (minimal)
    body = re.sub(r'\[([^\]]+)\]\(([^)]+)\)', r'<a href="\2">\1</a>', body)
    body = re.sub(r'<(https?://[^>]+)>', r'<a href="\1">\1</a>', body)
    body = re.sub(r'\*([^*]+)\*', r'<em>\1</em>', body)
    refs.append({'n': int(m.group(1)), 'html': body})

# inspirations (bullets) — title only
insp_sec = ack.split('## Inspirations', 1)[1].split('## References & sources', 1)[0]
insp = []
for b in re.split(r'\n(?=- )', insp_sec.strip()):
    m = re.match(r'- \*\*(.+?)\*\*', b)
    if m:
        insp.append(m.group(1))

# --- contributors / AI attribution (from git on the maintainer's machine, 21 Aug 2026) ---
contrib = {
    'commits': 1188,
    'first_commit': '2026-07-06',
    'humans': [
        {'name': 'The product thinker', 'handle': 'product-thinker', 'commits': 1185},
    ],
    'bots': [
        {'name': 'dependabot[bot]', 'commits': 2},
        {'name': 'Claude <noreply@anthropic.com> (author field; pre-policy)', 'commits': 1},
    ],
    'assisted': 822,
    'by_agent': [
        ['Claude:claude-fable-5', 425],
        ['Claude:claude-opus-4-8', 209],
        ['Claude:claude-opus-5', 170],
        ['Claude:claude-sonnet-5', 9],
        ['Claude (no model)', 8],
        ['None (human-only, declared)', 1],
    ],
}

out = {
    'generated_at': '2026-08-21',
    'commit_note': 'main @ 7a2eec6 (after git pull, 21 Aug 2026)',
    'nodes': [[n['id'], n['type'], n['status'], n['title'][:90], round(float(pos[i][0]), 4), round(float(pos[i][1]), 4), round(float(deg[i]), 1),
               n.get('kind', ''), n.get('severity', ''), n['path'], eff_date[i], *dates[n['path']], round(float(fpos[i][0]), 4), round(float(fpos[i][1]), 4)] for i, n in enumerate(nodes)],
    'edges': [[s, t, r] for s, t, r in links],
    'mentions': [[s, t] for s, t in mention_pairs],
    'counts': counts,
    'adrs': [[n['id'], n['date'], n['status'], n['title']] for n in adrs],
    'supersedes': supersedes,
    'dangling': dangling,
    'releases': releases,
    'refs': refs,
    'inspirations': insp,
    'contrib': contrib,
    'layout': layout_meta,
}
js = 'window.ABCD_DATA=' + json.dumps(out, separators=(',', ':'), ensure_ascii=False) + ';'
open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'data.js'), 'w', encoding='utf-8').write(js)
print('nodes', N, 'typed', len(typed_pairs), 'mentions', len(mention_pairs), 'refs', len(refs), 'insp', len(insp), 'bytes', len(js))
print(json.dumps(counts, indent=1))
print('adr date range', adrs[0]['date'], adrs[-1]['date'])
