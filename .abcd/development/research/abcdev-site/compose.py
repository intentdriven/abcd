#!/usr/bin/env python3
"""Compose the landing page and the docs index for the abcdev.app prototype.

Rule: every sentence rendered comes from a file in the repository (or the migration
drafts that will be files in the repository), selected by path + heading via
.abcd/site.json. This script carries layout only. Interface strings (nav labels,
button labels, tile captions) come from site-src/ui.json and nothing else.
"""
import json, re, html, base64, os
import markdown

MIG = os.environ.get('ABCD_MIG_DIR', 'readme-migration')
UP = os.environ.get('ABCD_ORIGINALS_DIR', '_originals')          # staged originals (docs/README.md, commands.md, logo)
ASSETS = os.environ.get('ABCD_SITE_ASSETS_DIR', 'site-assets')
manifest = json.load(open(f'{MIG}/.abcd/site.json'))
UI = json.load(open(f'{MIG}/site-src/ui.json'))
DATA = json.loads(open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'data.js')).read()[len('window.ABCD_DATA='):-1])

def b64(p, mime='image/webp'):
    return f'data:{mime};base64,' + base64.b64encode(open(p, 'rb').read()).decode()
IMG = {   # raster assets committed under docs/assets/img/ (the build optimises them to WebP; the prototype embeds that output)
    '../assets/img/intro.png': (b64(f'{ASSETS}/img1.webp'), 1024, 530),
    '../assets/img/role-product-thinker.png': (b64(f'{ASSETS}/cast-thinker.webp'), 203, 203),
    '../assets/img/role-facilitator.png': (b64(f'{ASSETS}/cast-facilitator.webp'), 203, 203),
    '../assets/img/role-ai-agents.png': (b64(f'{ASSETS}/cast-agents.webp'), 203, 203),
}
def svg_asset(key):
    """SVG assets committed under docs/assets/img/ are inlined, so their var(--token, fallback) colours follow the site theme."""
    p = f'{MIG}/docs/assets/img/' + key.rsplit('/', 1)[1]
    t = open(p, encoding='utf-8').read()
    t = re.sub(r'\s(width|height)="\d+"', '', t, count=2)          # size comes from CSS
    return t

def read(path):
    """Resolve a repo path to the migration draft, a saved source excerpt, or the staged original."""
    cands = [f'{MIG}/{path}', f'{MIG}/_sources/' + path.replace('/', '-').replace('.abcd-development-', 'development-'), f'{UP}/{path}']
    for c in cands:
        if os.path.exists(c):
            return open(c, encoding='utf-8').read(), c
    raise FileNotFoundError(path)

# ---------- markdown structure ----------
def strip_frontmatter(t):
    if t.startswith('---'):
        end = t.find('\n---', 3)
        if end > 0:
            return t[end + 4:]
    return t

def sections(md):
    """Split markdown into [{level, title, anchor, body}] honouring fenced code."""
    out = []; cur = {'level': 0, 'title': '', 'anchor': '', 'body': []}
    fence = False
    for line in md.splitlines():
        if line.startswith('```'):
            fence = not fence
        m = None if fence else re.match(r'^(#{1,6})\s+(.*)$', line)
        if m:
            out.append(cur)
            title = m.group(2).strip()
            cur = {'level': len(m.group(1)), 'title': title, 'anchor': slug(title), 'body': []}
        else:
            cur['body'].append(line)
    out.append(cur)
    for s in out:
        s['body'] = '\n'.join(s['body']).strip('\n')
    return [s for s in out if s['level'] or s['body']]

def slug(t):
    t = re.sub(r'[`*_]', '', t).lower()
    t = re.sub(r'[^a-z0-9]+', '-', t).strip('-')
    return t

def blocks(md):
    """Top-level markdown blocks: paragraphs, tables, fences, images."""
    out = []; buf = []; fence = False
    for line in md.splitlines():
        if line.startswith('```'):
            fence = not fence; buf.append(line)
            if not fence:
                out.append('\n'.join(buf)); buf = []
            continue
        if fence:
            buf.append(line); continue
        if not line.strip():
            if buf: out.append('\n'.join(buf)); buf = []
        else:
            buf.append(line)
    if buf: out.append('\n'.join(buf))
    return out

MD = lambda s: markdown.markdown(s, extensions=['tables', 'fenced_code'])

def fix_html(h, src):
    """Rewrite docs-relative links and images for the prototype's hash router; decorate code blocks."""
    def img(m):
        attrs = m.group(1)
        srcm = re.search(r'src="([^"]+)"', attrs); altm = re.search(r'alt="([^"]*)"', attrs)
        key = srcm.group(1); alt = altm.group(1) if altm else ''
        if key in IMG:
            uri, w, hgt = IMG[key]
            return f'<img src="{uri}" alt="{alt}" width="{w}" height="{hgt}" loading="lazy">'
        if key.endswith('.svg'):
            name = key.rsplit('/', 1)[1][:-4]
            return f'<span class="svgasset {name}" data-asset="docs/assets/img/{name}.svg">{svg_asset(key)}</span>'
        return m.group(0)
    h = re.sub(r'<img ([^>]*)/?>', img, h)
    def link(m):
        href = m.group(1)
        if href.startswith(('http://', 'https://', '#', 'mailto:')):
            return m.group(0)
        if href.endswith('.md') or '.md#' in href:
            return m.group(0).replace(f'href="{href}"', 'href="#/docs/"')
        if href.startswith('../../docs/') or href == 'docs/README.md':
            return m.group(0).replace(f'href="{href}"', 'href="#/docs/"')
        return m.group(0)
    h = re.sub(r'<a href="([^"]+)"', link, h)
    # code blocks -> copy-able command blocks (button label is a UI string)
    def pre(m):
        code = m.group(2)
        raw = html.unescape(re.sub(r'<[^>]+>', '', code))
        return f'<div class="cmd"><pre><code{m.group(1)}>{code}</code></pre><button class="copy" data-copy="{html.escape(raw, quote=True)}">{UI["copy"]}</button></div>'
    h = re.sub(r'<pre><code([^>]*)>([\s\S]*?)</code></pre>', pre, h)
    return h

def src_attr(path, anchor=''):
    return f' data-src="{html.escape(path + ("#" + anchor if anchor else ""))}"'

# ---------- identity ----------
ident_md, _ = read(manifest['identity']['file'].replace('.abcd/development/brief/01-product/README.md', 'identity-block.md'))
identity = {}
for m in re.finditer(r'- \*\*(\w+):\*\*\s*(.+?)(?=\n- \*\*|\Z)', ident_md, re.S):
    identity[m.group(1).lower()] = ' '.join(m.group(2).split())
IDENT_SRC = manifest['identity']['file'] + '#identity-canonical'

# ---------- hero ----------
home = manifest['home']
rat_md, _ = read(home['hero']['page']); rat_md = strip_frontmatter(rat_md)
rs = sections(rat_md)
h1 = rs[0]; rbl = blocks(h1['body'])
lede = next(b for b in rbl if not b.startswith('!['))
figure = next(b for b in rbl if b.startswith('!['))
figure_html = fix_html(MD(figure), home['hero']['page'])
hero = f'''
<section class="hero">
  <div class="wrap">
    <div class="grid">
      <div>
        <p class="eyebrow"{src_attr(IDENT_SRC)}>{html.escape(identity["title"])}</p>
        <h1{src_attr(home["hero"]["page"], "")}>{html.escape(h1["title"])}</h1>
        <div class="lede"{src_attr(home["hero"]["page"], h1["anchor"])}>{fix_html(MD(lede), home["hero"]["page"])}</div>
      </div>
      <div class="hero-aside">
        <p class="tagline"{src_attr(IDENT_SRC)}>{html.escape(identity["tagline"])}</p>
        <p class="pitch"{src_attr(IDENT_SRC)}>{html.escape(identity["pitch"])}</p>
        <div class="getit">
          <span class="eyebrow">{UI["latest_release"]}</span>
          <span class="dl"><a href="https://github.com/Partnermedia/abcd/releases/latest/download/abcd-darwin-arm64" title="abcd-darwin-arm64">{html.escape(UI["platform"]["darwin-arm64"])}</a> · <a href="https://github.com/Partnermedia/abcd/releases/latest/download/abcd-darwin-amd64" title="abcd-darwin-amd64">{html.escape(UI["platform"]["darwin-amd64"].split(" · ")[1])}</a></span>
          <span class="dl"><a href="https://github.com/Partnermedia/abcd/releases/latest/download/abcd-linux-arm64" title="abcd-linux-arm64">{html.escape(UI["platform"]["linux-arm64"])}</a> · <a href="https://github.com/Partnermedia/abcd/releases/latest/download/abcd-linux-amd64" title="abcd-linux-amd64">{html.escape(UI["platform"]["linux-amd64"].split(" · ")[1])}</a></span>
          <a class="more" href="#/#install">{UI["cta_install"]} ↓</a>
        </div>
      </div>
    </div>
    <aside class="note" data-n="1"><b>Nothing on this page is written for the website.</b> Turn this toggle on and every block shows its source as <code>path#heading</code>. The eyebrow, tagline and pitch are the brief's <i>Identity (canonical)</i> block — the same three lines <code>positioning.json</code> already renders into README and the plugin manifest. The headline and the paragraph are the H1 and first paragraph of <code>docs/explanation/rationale.md</code>, which is README § Purpose moved verbatim. Button labels, nav and tile captions are interface strings from <code>site-src/ui.json</code>; that file is the complete list of words the generator is allowed to add.<span class="src">rule: .abcd/site.json → checks.every_text_node_has_source</span></aside>
    <figure class="comic"{src_attr(home["hero"]["page"], h1["anchor"])}>{figure_html}</figure>
    <aside class="note" data-n="2"><b>Every picture is a committed asset under <code>docs/assets/img/</code></b> and is referenced from a docs page like any other image: the cartoon (<code>intro.png</code>, rationale.md), the three role portraits (roles.md), the artefact icons (artefacts.md), the process loop (process.md). The build optimises rasters (WebP/AVIF, a 1200×630 OpenGraph crop) and inlines SVGs; it never draws pictures of its own.<span class="src">docs/assets/img/</span></aside>
  </div>
</section>'''

# ---------- chapters ----------
def cards_from_h2(page, secs):
    intro = ''.join(fix_html(MD(b), page) for b in blocks(secs[0]['body']))
    cards = []
    for s in secs[1:]:
        bl = blocks(s['body']); portrait = ''
        if bl and bl[0].startswith('!['):
            portrait = fix_html(MD(bl[0]), page).replace('<p>', '').replace('</p>', '').replace('<img ', '<img class="portrait-lg" ')
            bl = bl[1:]
        body = ''.join(fix_html(MD(b), page) for b in bl)
        cards.append(f'<div class="card"{src_attr(page, s["anchor"])}>{portrait}<h3>{html.escape(s["title"])}</h3>{body}</div>')
    return f'<div class="prose lede-2"{src_attr(page, secs[0]["anchor"])}>{intro}</div><div class="cards two">{"".join(cards)}</div>'

ROLE_PORTRAIT = {'Product thinker': '../assets/img/role-product-thinker.png', 'Facilitator': '../assets/img/role-facilitator.png'}
def lead_in_cards(page, secs):
    out = []; cards = []; icon = ''
    for b in blocks(secs[0]['body']):
        if b.lstrip().startswith('|'):
            if cards: out.append(f'<div class="cards">{"".join(cards)}</div>'); cards = []
            th = MD(b)
            for label, key in ROLE_PORTRAIT.items():   # the role portraits (committed assets) sit above the column labels
                uri, w, hgt = IMG[key]
                th = th.replace(f'<th>{label}</th>', f'<th><img class="th-portrait" src="{uri}" alt="" width="{w}" height="{hgt}"><span>{label}</span></th>')
            out.append(f'<div class="tablewrap"{src_attr(page, secs[0]["anchor"])}>{th}</div>')
            continue
        if b.startswith('!['):                        # an image line before a lead-in paragraph is that card's icon
            icon = fix_html(MD(b), page).replace('<p>', '').replace('</p>', ''); continue
        m = re.match(r'^((?:\S+\s+){0,2}?)\*\*([^*]+)\*\*', b)
        if m:
            title = (m.group(1) + m.group(2)).strip()
            rest = b[m.end():].lstrip(' ')
            cards.append(f'<div class="card"{src_attr(page, secs[0]["anchor"])}>{icon and f"<div class=icon>{icon}</div>"}<h3>{html.escape(title)}</h3>{fix_html(MD(rest), page)}</div>'); icon = ''
        else:
            if cards: out.append(f'<div class="cards">{"".join(cards)}</div>'); cards = []
            out.append(f'<div class="prose"{src_attr(page, secs[0]["anchor"])}>{fix_html(MD(b), page)}</div>')
    if cards: out.append(f'<div class="cards">{"".join(cards)}</div>')
    return ''.join(out)

def prose(page, secs):
    """Prose sections; the page's first image block is lifted out and returned separately as the figure."""
    out = []; figure = ''
    for s in secs:
        if s['level'] > 1:
            out.append(f'<h3 class="subh"{src_attr(page, s["anchor"])}>{html.escape(s["title"])}</h3>')
        bl = blocks(s['body'])
        if not figure:
            for b in bl:
                if b.startswith('!['):
                    figure = f'<figure class="card loopfig"{src_attr(page, s["anchor"] or slug(s["title"]))}>{fix_html(MD(b), page).replace("<p>", "").replace("</p>", "")}</figure>'
                    bl = [x for x in bl if x != b]; break
        out.append(f'<div class="prose"{src_attr(page, s["anchor"] or slug(s["title"]))}>{"".join(fix_html(MD(b), page) for b in bl)}</div>')
    return ''.join(out), figure

def feature_block(spec):
    md_, path = read('itd-100.md')  # prototype: the real build picks the newest shipped intent whose audit verdict is MET
    secs = sections(strip_frontmatter(md_))
    pr = next(s for s in secs if s['title'] == 'Press Release')
    ac = next(s for s in secs if s['title'] == 'Acceptance Criteria')
    quote = MD(pr['body'])                      # the blockquote, verbatim
    crit = blocks(ac['body'])[0].lstrip('- ').strip()
    crit_html = MD(crit)
    rel = 'v0.4.1'                              # from CHANGELOG.md: "(itd-100)" under ## [0.4.1] - 2026-07-28
    src = '.abcd/development/intents/shipped/itd-100-alice-reads-one-page-and-knows-exactly.md'
    return f'''<div class="quote"{src_attr(src, "press-release")}>
        <div class="pr"><span><a href="https://github.com/Partnermedia/abcd/blob/main/{src}">itd-100</a> · shipped · {rel}</span><span>{UI["from_the_record"]}</span></div>
        {quote}
        <div class="crit"{src_attr(src, "acceptance-criteria")}>{crit_html}</div>
      </div>
      <aside class="note" data-n="4"><b>The "testimonial" is a real intent.</b> The press release and its first acceptance criterion are quoted verbatim from the shipped intent itd-100 (which delivered <code>docs/reference/terminology.md</code> in v0.4.1). The build picks the newest shipped intent whose audit verdict is MET; "Alice" is a persona from <code>personas.json</code>, never a real person.<span class="src">.abcd/development/intents/shipped/ · personas.json</span></aside>'''



def install(page, secs, ch):
    """Tab row: Plugin on the left (the product thinker's path); the CLI group on the right, labelled,
    made of the CLI section's H3s (macOS · Linux · Windows) and Build. The CLI lead paragraph opens each
    OS panel, the 'Afterwards' section closes it. A star marks the tab that matches the visitor's system."""
    intro = ''.join(fix_html(MD(b), page) for b in blocks(secs[0]['body']))
    h2 = [s for s in secs if s['level'] == 2]
    lead = next(s for s in h2 if s['title'] == ch['lead'])
    li = secs.index(lead)
    h3 = []
    for s in secs[li + 1:]:
        if s['level'] == 2: break
        if s['level'] == 3: h3.append(s)
    after = next((s for s in h3 if s['title'] == ch.get('after')), None)
    os_tabs = [s for s in h3 if s is not after]
    left = [s for s in h2 if s['title'] in ch.get('left', [])]
    group = os_tabs + [s for s in h2 if s is not lead and s not in left]
    tabs = [(s, 'left') for s in left] + [(s, 'os') for s in os_tabs] + [(s, 'h2') for s in group if s not in os_tabs]
    def btn(s, i): return f'<button role="tab" id="tab-{s["anchor"]}" aria-selected="{"true" if i == 0 else "false"}" aria-controls="panel-{s["anchor"]}" tabindex="{0 if i == 0 else -1}"{src_attr(page, s["anchor"])}>{html.escape(s["title"])}</button>'
    order = [s for s, _ in tabs]
    row = ''.join(btn(s, order.index(s)) for s in left) + f'<div class="tabgroup"><span class="grp">{html.escape(UI["cli_group"])}</span>' + ''.join(btn(s, order.index(s)) for s in group) + '</div>'
    rb = lambda bl: ''.join(fix_html(MD(x), page) for x in bl)
    panels = []
    for i, (s, kind) in enumerate(tabs):
        bl = blocks(s['body'])
        code = any(b.startswith('```') for b in bl)
        head = f'<div class="prose small tabprose lead"{src_attr(page, lead["anchor"])}>{rb(blocks(lead["body"]))}</div>' if kind == 'os' and code else ''
        extra = ''
        if kind == 'os' and code and after:
            abl = blocks(after['body'])
            extra = f'<div class="prose small tabprose after"{src_attr(page, after["anchor"])}><h4>{html.escape(after["title"])}</h4>{rb(abl[:2])}<details class="more"><summary>{UI["more"]}</summary>{rb(abl[2:])}</details></div>'
        body = rb(bl[:4]) + (f'<details class="more"><summary>{UI["more"]}</summary>{rb(bl[4:])}</details>' if len(bl) > 4 else '')   # page order; long sections fold after their first blocks
        panels.append(f'<div role="tabpanel" id="panel-{s["anchor"]}" aria-labelledby="tab-{s["anchor"]}"{"" if i == 0 else " hidden"}{src_attr(page, s["anchor"])}>{head}<div class="prose small tabbody">{body}</div>{extra}</div>')
    return f'''<div class="stack installwrap" style="--stack:16px">
        <div class="prose"{src_attr(page, secs[0]["anchor"])}>{intro}</div>
        <div class="tabs install"><div role="tablist" aria-label="{html.escape(secs[0]["title"])}" data-mine-title="{html.escape(UI["matches_system"])}">{row}</div>{"".join(panels)}</div>
        <p class="small muted"><a href="https://github.com/Partnermedia/abcd/releases/latest">{UI["latest_release"]}</a> · <a href="https://github.com/Partnermedia/abcd/releases/latest/download/checksums.txt">checksums.txt</a> · <a href="https://github.com/Partnermedia/abcd/blob/main/CHANGELOG.md">CHANGELOG.md</a> · <a href="https://github.com/Partnermedia/abcd/releases">{UI["all_releases"]}</a></p>
    </div>
    <aside class="note" data-n="7"><b>The install text is README § Install, moved to <code>docs/how-to/install.md</code>.</b> The tab row is that page's structure, arranged for the audience: <i>Plugin</i> on the left — the product thinker's path, the page's first option — and the command-line group on the right under its own label: the CLI section's H3s macOS, Linux, Windows, then Build. The tab matching the visitor's system gets a star (detected in the browser, nothing stored). Each OS panel opens with the CLI lead paragraph, shows its command and prose, and closes with the page's <i>Afterwards</i> section (PATH, older installs, inspecting before running; the tail folded). The per-OS commands are the universal one-liner specialised — same URLs, same checksum step, only <code>uname -s</code> resolved — and a test should assert exactly that, so the three cannot drift; README keeps the universal form. The Windows tab is a sentence in install.md, not site copy: when a Windows build ships, the page changes and the tab follows.<span class="src">docs/how-to/install.md · .github/workflows/release.yml</span></aside>'''

chapters_html = []
CHAPTER_NOTES = {
    'a': '<aside class="note" data-n="3"><b>Chapters a·b·c·d are README\'s own table of contents</b> — Roles, Artefacts, Process, Install — now four pages under <code>docs/</code>. Each chapter heading is the page\'s H1; the cards are its H2 sections (roles) or its bold lead-in paragraphs (artefacts). Layout is the only thing the generator adds.<span class="src">.abcd/site.json → home.chapters</span></aside>',
    'c': '<aside class="note" data-n="5"><b>Commands are pinned.</b> Every <code>abcd …</code> snippet on the site is a fenced block from a docs page, and the build checks each against the generated CLI reference, so a renamed verb fails the build rather than shipping stale copy. The loop figure is a committed asset, <code>docs/assets/img/process-loop.svg</code>, referenced from process.md like any other image — so GitHub and the docs page show the same picture; the site inlines it so its <code>var(--token, fallback)</code> colours follow the theme. A check asserts that every label in the figure is a phrase on the page.<span class="src">docs/explanation/process.md · docs/assets/img/process-loop.svg</span></aside>',
}
for ch in home['chapters']:
    page = ch['page']; md_, _ = read(page); md_ = strip_frontmatter(md_)
    secs = sections(md_)
    title = secs[0]['title']
    if ch['layout'] == 'cards-from-h2':
        body = cards_from_h2(page, secs)
    elif ch['layout'] == 'lead-in-cards':
        body = lead_in_cards(page, secs) + feature_block(ch.get('feature'))
    elif ch['layout'] == 'prose':
        text, figure = prose(page, secs)
        body = f'<div class="loop"><div>{text}</div>{figure}</div>'
    elif ch['layout'] == 'install':
        body = install(page, secs, ch)
    anchor = slug(title)
    chapters_html.append(f'''
<section class="chapter" id="{anchor}">
  <div class="wrap">
    <div class="head"><div class="letter {ch["letter"]}" aria-hidden="true">{ch["letter"]}</div><h2{src_attr(page)}>{html.escape(title)}</h2></div>
    <div class="body stack" style="--stack:22px">{body}{CHAPTER_NOTES.get(ch["letter"], "")}</div>
  </div>
</section>''')

# ---------- record page title (used by the docs nav) ----------
rec_md, _ = read('development-README.md')
rsecs = sections(rec_md); rtitle = rsecs[0]['title']

footer = f'''
<footer class="site-foot"><div class="wrap">
  <span>© 2026 REPPL · MIT</span>
  <a href="https://github.com/Partnermedia/abcd">GitHub</a>
  <a href="https://github.com/Partnermedia/abcd/blob/main/SECURITY.md">SECURITY.md</a>
  <a href="https://github.com/Partnermedia/abcd/blob/main/ACKNOWLEDGEMENTS.md">ACKNOWLEDGEMENTS.md</a>
  <a href="#/references/">{UI["nav_references"]}</a>
  <a href="https://github.com/Partnermedia/abcd/blob/main/CITATION.cff">CITATION.cff</a>
  <span style="margin-left:auto" class="mono small">abcdev.app · {UI["beta"].lower()} · v0.6.1 · main@7a2eec6</span>
</div></footer>'''

home_html = '<div class="page">' + hero + ''.join(chapters_html) + footer + '</div>'

# ---------- docs index ----------
docs_md = open(f'{UP}/docs/README.md', encoding='utf-8').read()
dsecs = sections(docs_md)
docs_body = fix_html(MD(dsecs[0]['body']), 'docs/README.md').replace('href="https://github.com/Partnermedia/abcd/tree/main/.abcd/development/"', 'href="#/record/"')
cmd_md = open(f'{UP}/docs/reference/cli/commands.md', encoding='utf-8').read()
csecs = sections(cmd_md)
cli_intro = ''.join(fix_html(MD(b), 'docs/reference/cli/commands.md') for b in blocks(csecs[0]['body'])[:2])
verbs = [s['title'].strip('`') for s in csecs if s['level'] in (2, 3)]
verbs_html = ''.join(f'<a href="#/docs/">{html.escape(v)}</a>' for v in verbs)
docs_html = f'''<div class="page wrap docs">
  <nav class="docs-nav" aria-label="Documentation">
    <div class="search" role="search"><span>{UI["search_docs"]}</span><kbd>/</kbd></div>
    <ul><li><a href="#/docs/">{html.escape(dsecs[0]["title"])}</a></li>
      <li><a href="#/docs/">Tutorials</a></li>
      <li><a href="#/docs/">How-to guides</a><ul><li><a href="#/docs/">Install</a></li></ul></li>
      <li><a href="#/docs/">Reference</a><ul><li><a href="#/docs/">Terminology crosswalk</a></li><li><a href="#/docs/">CLI command reference</a></li></ul></li>
      <li><a href="#/docs/">Explanation</a><ul><li><a href="#/docs/">Who abcd is for</a></li><li><a href="#/docs/">Roles</a></li><li><a href="#/docs/">Artefacts</a></li><li><a href="#/docs/">Process</a></li></ul></li>
      <li><a href="#/record/">{html.escape(rtitle)}</a></li></ul>
  </nav>
  <article class="docs-main stack" style="--stack:20px">
    <h1{src_attr("docs/README.md")}>{html.escape(dsecs[0]["title"])}</h1>
    <div class="prose"{src_attr("docs/README.md", "documentation")}>{docs_body}</div>
    <aside class="note" data-n="9"><b>/docs/ stays MkDocs Material for now.</b> Same <code>docs/</code> tree, <code>site_url: https://abcdev.app/docs/</code>, theme overrides so header and type match the rest of the site, <code>content.code.copy</code>, <code>repo_url</code>/<code>edit_uri</code>, social cards, <code>mkdocs-llmstxt</code>; old root URLs 301 via <code>_redirects</code>. The nav entries here are page titles from <code>docs/**</code> (the four explanation pages and the install how-to are the README migration). The SSG decision is a separate ADR before Material's maintenance window closes (~Nov 2026).<span class="src">mkdocs.yml · docs/README.md · readme-migration/</span></aside>
    <div class="stack" style="--stack:10px">
      <h2 style="font-size:1.4rem"{src_attr("docs/reference/cli/commands.md")}>{html.escape(csecs[0]["title"])}</h2>
      <div class="prose small muted"{src_attr("docs/reference/cli/commands.md")}>{cli_intro}</div>
      <div class="verbs">{verbs_html}</div>
    </div>
  </article>
</div>'''

open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'home.html'), 'w', encoding='utf-8').write(home_html)
open(os.path.join(os.environ.get('ABCD_SITE_DATA_DIR', 'site-data'), 'docs.html'), 'w', encoding='utf-8').write(docs_html)
print('home', len(home_html), 'docs', len(docs_html))
# loop-figure label check: every label in the committed SVG must be a phrase on the page
proc_md, _ = read('docs/explanation/process.md')
svg_txt = html.unescape(re.sub(r'<[^>]+>', ' ', open(f'{MIG}/docs/assets/img/process-loop.svg', encoding='utf-8').read().split('</defs>', 1)[1]))
for lab in ['the brief', 'intent', 'building', 'verdict', 'writes the verdict onto the intent', 'shipping closes the loop twice', 'turns the intent into engineering work', 'grades each acceptance bullet', 'updates the brief', 'abcd intent "<one-line idea>"']:
    assert lab in proc_md.lower(), lab
    assert all(w in svg_txt.lower() for w in lab.replace('"', ' ').split()), lab
print('loop labels ok')
