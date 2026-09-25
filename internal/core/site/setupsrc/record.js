/* abcdev.app — the relationship chart at /record/graph/.
 *
 * Lifted from the clickable prototype in the investigation cluster, which is the
 * behavioural spec, and adapted in exactly two places: the hash router became
 * real URLs (`?focus=<id>` on this page's own query string), and the data came
 * out of a positional blob and into the named fields of `record.json`.
 *
 * It ADDS NO WORDS. Every string it shows is put into the markup by the build
 * out of site-src/ui.json and read back from a data attribute, so the
 * interface-string allowlist stays the complete list of what a reader sees.
 *
 * Both arrangements were computed at build time, so nothing here is a layout
 * engine: the only physics is a glide toward the published position and a
 * collision pass that keeps a little air between neighbours, and the loop idles
 * as soon as it has settled. No analytics, no trackers; the one request it makes
 * is for record.json from this same origin.
 */
(function () {
  'use strict';

  var wrap = document.querySelector('.bwrap');
  if (!wrap || !wrap.dataset.graph) return;
  var W_ = wrap.dataset;
  var stage = wrap.querySelector('.bstage');
  var canvas = document.getElementById('bc');
  var card = document.getElementById('bcard');
  var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var esc = function (s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  };
  var fmtDate = function (d) {
    if (!d) return '';
    try {
      return new Date(d + 'T00:00:00Z').toLocaleDateString(undefined,
        { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' });
    } catch (e) { return d; }
  };
  var fmtShort = function (d) {
    if (!d) return '';
    try {
      return new Date(d + 'T00:00:00Z').toLocaleDateString(undefined,
        { day: 'numeric', month: 'short', timeZone: 'UTC' });
    } catch (e) { return d; }
  };

  /* Colour names only the three durable stores plus the ledger; everything else
     takes ink, because a fourth categorical hue stops being distinguishable
     under colour-vision deficiency in an all-pairs chart. */
  var TYPE_TOKEN = { adr: '--s-adr', intent: '--s-intent', spec: '--s-spec', issue: '--s-issue' };
  /* A discipline is filed under intents by the record and is its own KIND of
     thing to a reader — a rule that holds rather than a change that ships — so
     it is drawn in its own colour and the legend names it in its own right.
     The lifecycle wins over the type, because that is where the record says so. */
  var DISCIPLINE = 'disciplines';
  /* A record's KIND is what the chart colours it by, filters it by and names it
     in the card — one answer, used everywhere, so a purple bubble is never
     labelled 'intent' and never hides behind an 'intents' chip. A discipline is
     filed under intents by the record and is its own kind to a reader. */
  var kindOf = function (n) { return n.status === DISCIPLINE ? DISCIPLINE : n.type; };
  var tok = function (t, state) {
    if (state === DISCIPLINE) return '--s-discipline';
    return TYPE_TOKEN[t] || '--s-neutral';
  };
  /* GitHub's state palette: green in play, purple done, grey draft or set aside. */
  var SOLID = ['accepted', 'shipped', 'closed', 'resolved', 'active'];
  var RING = ['planned', 'open', 'proposed', 'disciplines'];
  var DASH = ['drafts', 'draft'];
  var has = function (a, v) { return a.indexOf(v) >= 0; };
  var STATUS_TONE = function (s) {
    if (has(['open', 'planned', 'proposed', 'active'], s)) return 'open';
    if (has(['accepted', 'shipped', 'closed', 'resolved'], s)) return 'done';
    if (has(DASH, s)) return 'draft';
    if (has(['superseded', 'wontfix'], s)) return 'notplanned';
    return 'plain';
  };
  var SEV_TONE = { critical: 'critical', major: 'high', minor: 'moderate', nitpick: 'low' };
  var DIRECTED = ['supersedes', 'implements', 'builds_on'];
  var REL_INV = {
    blocked_by: W_.relBlockedBy, supersedes: W_.relSupersedes,
    implements: W_.relImplements, builds_on: W_.relBuildsOn
  };
  var relWord = function (rel) { return String(rel).replace(/_/g, ' '); };

  var G = null;

  fetch(wrap.dataset.graph, { credentials: 'same-origin' })
    .then(function (r) { return r.json(); })
    .then(start)
    .catch(function () {
      /* No data, no chart. The list twin under the stage is already in the
         markup and reaches every record, so the page still works. */
      offerList();
    });

  /* offerList removes the chart and opens the accessible twin in its place. It
     is what a reader who asked for stillness gets, and what everyone gets if the
     data cannot be read. */
  function offerList() {
    if (stage) stage.remove();
    var list = wrap.querySelector('.blist');
    if (list) list.open = true;
  }

  function start(D) {
    if (reduced) { offerList(); return; }
    if (!D || !D.nodes || !D.nodes.length) { offerList(); return; }

    var L = D.layout, CR = L.coil_radius || 1;
    var index = {};
    var nodes = D.nodes.map(function (n, i) {
      index[n.id] = i;
      var coil = L.coil[i] || { x: 0, y: 0 }, links = L.links[i] || { x: 0, y: 0 };
      return {
        i: i, id: n.id, type: n.type, status: n.lifecycle || n.status || '', title: n.title,
        path: n.path, kind: n.kind, sev: n.severity, date: n.date,
        created: (n.dates && n.dates.created) || n.date,
        entered: (n.dates && n.dates.entered) || n.date,
        last: (n.dates && n.dates.touched) || n.date,
        hx: { date: coil.x * CR, links: links.x * CR },
        hy: { date: coil.y * CR, links: links.y * CR },
        r: L.radius[i], x: 0, y: 0, vx: 0, vy: 0, a: 1, ta: 1, s: 1
      };
    });
    var N = nodes.length;

    /* Two records may be joined by more than one relation — a spec that
       implements an intent and is also related to it. Each neighbour therefore
       holds a LIST: keeping one entry per pair let the last edge read overwrite
       every earlier one, and the card silently showed a single relation where
       the record states two. */
    var adj = nodes.map(function () { return new Map(); });
    var join = function (from, to, rel, out) {
      if (!adj[from].has(to)) adj[from].set(to, []);
      var rels = adj[from].get(to);
      for (var i = 0; i < rels.length; i++) {
        if (rels[i].rel === rel && rels[i].out === out) return;
      }
      rels.push({ rel: rel, out: out });
    };
    D.edges.forEach(function (e) {
      var a = index[e.from], b = index[e.to];
      if (a === undefined || b === undefined) return;
      join(a, b, e.rel, true);
      join(b, a, e.rel, false);
    });
    var madj = nodes.map(function () { return new Set(); });
    (D.mentions || []).forEach(function (e) {
      var a = index[e.from], b = index[e.to];
      if (a === undefined || b === undefined) return;
      madj[a].add(b); madj[b].add(a);
    });
    /* A reference whose target has left the tree. It is listed in the card as
       the absence it is — never drawn at a position nothing occupies. */
    var stubs = nodes.map(function () { return []; });
    ((D.health && D.health.unresolved) || []).forEach(function (u) {
      var a = index[u.from];
      if (a !== undefined) stubs[a].push(u);
    });

    var typeOn = {};
    nodes.forEach(function (n) { typeOn[kindOf(n)] = true; });
    var arr = 'date', useMent = false, focus = -1, hover = -1;
    var W = 0, H = 0, dpr = 1, narrow = false, need = true, energy = 1;
    var css = function () { return getComputedStyle(document.documentElement); };
    var col = function (v) { return css().getPropertyValue(v).trim(); };
    var on = function (n) { return typeOn[kindOf(n)]; };
    /* A record whose store grades it by neither a lifecycle directory nor a
       status field declares no state, and is drawn solid because it has none —
       not faded, which is what "set aside" looks like. */
    var cls = function (n) {
      if (!n.status) return 'solid';
      return has(SOLID, n.status) ? 'solid' : has(RING, n.status) ? 'ring' : has(DASH, n.status) ? 'dash' : 'fade';
    };

    /* view: screen = T + k · world. T is the stage geometry plus the viewer's
       pan; k is the fit times their zoom. */
    var view = { k: 1, fit: 1, zoom: 1, tx: 0, ty: 0, px: 0, py: 0 };
    var base = { x: 0, y: 0 }, CY0 = 0;
    function baseTarget() {
      var x = narrow
        ? (focus >= 0 ? W / 2 - nodes[focus].hx[arr] * view.k : W / 2)
        : (focus >= 0 ? W / 2 + 190 : W / 2);
      var y = (narrow && focus >= 0)
        ? (40 + 0.22 * H) - nodes[focus].hy[arr] * view.k
        : CY0;
      return { x: x, y: y };
    }
    function size() {
      dpr = window.devicePixelRatio || 1;
      W = stage.clientWidth; H = stage.clientHeight;
      canvas.width = W * dpr; canvas.height = H * dpr;
      narrow = W < 700;
      var R = narrow ? Math.min(W / 2 - 10, (H - 100 - 14) / 2) : Math.min(H / 2 - 18, W / 2 - 200);
      CY0 = narrow ? 100 + R : H / 2;
      view.fit = Math.min(1.15, R / CR); view.k = view.fit * view.zoom;
      if (!base.x) base = baseTarget();
      need = true;
    }
    var sx = function (wx) { return view.tx + wx * view.k; };
    var sy = function (wy) { return view.ty + wy * view.k; };
    var toWorld = function (x, y) { return [(x - view.tx) / view.k, (y - view.ty) / view.k]; };
    function zoomAt(factor, x, y) {
      var z = Math.max(0.5, Math.min(8, view.zoom * factor));
      var f = z / view.zoom;
      if (f === 1) return;
      var w = toWorld(x, y);
      view.zoom = z; view.k = view.fit * view.zoom;
      view.px = x - w[0] * view.k - base.x;
      view.py = y - w[1] * view.k - base.y;
      view.tx = base.x + view.px; view.ty = base.y + view.py;
      need = true;
    }
    function resetView() { view.zoom = 1; view.k = view.fit; view.px = 0; view.py = 0; need = true; }

    /* focus: a browsing history of the records looked at. Back and forward walk
       it, and a fresh pick drops the forward trail — the way a browser does. */
    var neigh = new Map(), hist = [], hpos = -1;
    function setFocus(i, nav) {
      focus = i; neigh = new Map();
      if (i >= 0 && !nav && hist[hpos] !== i) { hist.splice(hpos + 1); hist.push(i); hpos = hist.length - 1; }
      if (i >= 0) {
        adj[i].forEach(function (rels, j) { neigh.set(j, rels.slice()); });
        if (useMent) madj[i].forEach(function (j) { if (!neigh.has(j)) neigh.set(j, [{ rel: 'mention', out: true }]); });
      }
      renderCard(); syncHist();
      nodes.forEach(function (n) {
        n.ta = i < 0 ? (on(n) ? 1 : 0.05) : n.i === i ? 1 : neigh.has(n.i) ? 0.55 : 0.06;
      });
      need = true;
    }

    /* physics, in world units: bubbles glide home, collisions keep them apart,
       and the loop idles once nothing is moving. */
    var cell = 24, grid = new Map();
    function step() {
      var bt = baseTarget();
      base.x += (bt.x - base.x) * 0.1; base.y += (bt.y - base.y) * 0.1;
      view.tx = base.x + view.px; view.ty = base.y + view.py;
      energy = 0;
      for (var q = 0; q < N; q++) {
        var n = nodes[q];
        var k = n.i === focus ? 0.1 : 0.04;
        n.vx += (n.hx[arr] - n.x) * k; n.vy += (n.hy[arr] - n.y) * k;
        n.vx *= 0.72; n.vy *= 0.72;
        n.x += n.vx; n.y += n.vy;
        var da = n.ta - n.a; n.a += da * 0.15;
        /* The focused bubble grows into its own padding only; the highlight is
           a ring, so nothing around it ever moves. */
        var ts = n.i === focus ? Math.min(1.35, (n.r + 2) / n.r) : 1;
        var ds = ts - n.s; n.s += ds * 0.2;
        energy = Math.max(energy, Math.abs(n.vx), Math.abs(n.vy), Math.abs(da) * 4, Math.abs(ds) * 20);
      }
      for (var pass = 0; pass < 2; pass++) {
        grid.clear();
        for (var a = 0; a < N; a++) {
          var m = nodes[a];
          if (m.a < 0.2) continue;
          var key = ((m.x / cell) | 0) + ',' + ((m.y / cell) | 0);
          if (!grid.has(key)) grid.set(key, []);
          grid.get(key).push(m);
        }
        for (var b = 0; b < N; b++) {
          var u = nodes[b];
          if (u.a < 0.2) continue;
          var gx = (u.x / cell) | 0, gy = (u.y / cell) | 0;
          for (var ox = -1; ox <= 1; ox++) for (var oy = -1; oy <= 1; oy++) {
            var bucket = grid.get((gx + ox) + ',' + (gy + oy));
            if (!bucket) continue;
            for (var c = 0; c < bucket.length; c++) {
              var v = bucket[c];
              if (v.i <= u.i) continue;
              var dx = v.x - u.x, dy = v.y - u.y, d = Math.hypot(dx, dy);
              /* The LAYOUT owns spacing; this only stops a true overlap. The
                 extra padding that used to sit here was the renderer asking
                 for more room than the published positions give, so on an
                 arrangement packed tighter than that padding every frame
                 pushed bubbles apart and every frame pulled them home — a
                 loop that never idled and never stopped drawing. */
              var min = u.r * u.s + v.r * v.s;
              if (d >= min) continue;
              if (d < 0.01) { dx = 0.1; dy = 0.1; d = 0.14; }
              var push = (min - d) / d * 0.5;
              u.x -= dx * push; u.y -= dy * push; v.x += dx * push; v.y += dy * push;
              energy = Math.max(energy, min - d);
            }
          }
        }
      }
      energy = Math.max(energy, Math.abs(bt.x - base.x), Math.abs(bt.y - base.y));
    }

    /* One line is drawn per PAIR however many relations join it, so the line
       takes a representative: the directed one where there is one, because the
       arrowhead is the thing the line has to get right. */
    function pickRel(rels) {
      for (var i = 0; i < rels.length; i++) {
        if (has(DIRECTED, rels[i].rel)) return rels[i];
      }
      return rels[0];
    }
    function drawLinks(ctx, f, map, alpha, heads) {
      var ink2 = col('--ink-2'), rule = col('--rule-2'), k = view.k;
      map.forEach(function (rels, j) {
        var m = pickRel(rels);
        var n = nodes[j], dir = heads && has(DIRECTED, m.rel);
        var fx = sx(f.x), fy = sy(f.y), nx = sx(n.x), ny = sy(n.y);
        ctx.strokeStyle = ctx.fillStyle = m.rel === 'mention' ? rule : ink2;
        ctx.globalAlpha = alpha * (m.rel === 'mention' ? 0.5 : 0.9);
        ctx.lineWidth = dir ? 1.4 : 1.2;
        if (!dir) { ctx.beginPath(); ctx.moveTo(fx, fy); ctx.lineTo(nx, ny); ctx.stroke(); return; }
        /* The arrow runs from the record whose frontmatter names the other. */
        var from = m.out ? [fx, fy] : [nx, ny];
        var to = m.out ? [nx, ny, n] : [fx, fy, f];
        var dx = to[0] - from[0], dy = to[1] - from[1], d = Math.hypot(dx, dy) || 1;
        var ux = dx / d, uy = dy / d;
        var tr = to[2].r * to[2].s * k + 2.5;
        var tipx = to[0] - ux * tr, tipy = to[1] - uy * tr, Ln = 8, Wd = 3.6;
        ctx.beginPath(); ctx.moveTo(from[0], from[1]);
        ctx.lineTo(tipx - ux * Ln * 0.6, tipy - uy * Ln * 0.6); ctx.stroke();
        ctx.beginPath(); ctx.moveTo(tipx, tipy);
        ctx.lineTo(tipx - ux * Ln - uy * Wd, tipy - uy * Ln + ux * Wd);
        ctx.lineTo(tipx - ux * Ln + uy * Wd, tipy - uy * Ln - ux * Wd);
        ctx.closePath(); ctx.fill();
      });
    }

    /* month zones: concentric bands under the coil, a boundary at the first
       record of each month, the way fare zones are drawn on a tube map. */
    function zones() {
      var bounds = (L.months || []).map(function (m) {
        var n = nodes[m.node];
        if (!n) return null;
        return { m: m.month, r: Math.max(0, Math.hypot(n.hx.date, n.hy.date) - n.r - 2) };
      }).filter(Boolean);
      bounds.push({ m: null, r: CR + 8 });
      var out = [];
      for (var i = 0; i < bounds.length - 1; i++) {
        out.push({ label: bounds[i].m || '', r0: bounds[i].r, r1: bounds[i + 1].r });
      }
      var merged = [];
      out.forEach(function (z) {
        var last = merged[merged.length - 1];
        if (last && (z.r1 - z.r0) * view.k < 18) { last.label += '–' + z.label; last.r1 = z.r1; }
        else merged.push({ label: z.label, r0: z.r0, r1: z.r1 });
      });
      return merged;
    }

    /* Labels step aside rather than overprint: one that would sit on a label
       already placed is skipped. */
    var placed = [];
    function fits(x, y, w, h) {
      for (var i = 0; i < placed.length; i++) {
        var p = placed[i];
        if (x < p.x + p.w && x + w > p.x && y < p.y + p.h && y + h > p.y) return false;
      }
      placed.push({ x: x, y: y, w: w, h: h });
      return true;
    }
    function label(ctx, n, txt, bold, side, alpha) {
      ctx.font = (bold ? 650 : 500) + ' ' + (bold ? 12 : 11) + 'px "Bricolage Grotesque", sans-serif';
      var w = ctx.measureText(txt).width, h = 14, rr = n.r * n.s * view.k;
      var x = side === 'left' ? sx(n.x) - rr - 6 - w : sx(n.x) + rr + 6, y = sy(n.y);
      if (!fits(x, y - h / 2, w, h)) return false;
      ctx.textAlign = 'left'; ctx.globalAlpha = alpha; ctx.lineWidth = bold ? 5 : 3;
      ctx.strokeText(txt, x, y); ctx.fillText(txt, x, y);
      return true;
    }

    function draw() {
      var ctx = canvas.getContext('2d');
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0); ctx.clearRect(0, 0, W, H);
      var surface = col('--surface'), ink = col('--ink'), ink2 = col('--ink-2'),
        ink3 = col('--ink-3'), rule = col('--rule'), zone = col('--surface-2');
      var k = view.k, cx = sx(0), cy = sy(0), dim = focus >= 0 ? 0.5 : 1;
      placed = [];
      if (arr === 'date') {
        var zs = zones();
        zs.forEach(function (z, i) {
          if (i % 2 === 0) return;
          ctx.globalAlpha = 0.7 * dim; ctx.fillStyle = zone;
          ctx.beginPath(); ctx.arc(cx, cy, z.r1 * k, 0, Math.PI * 2);
          ctx.arc(cx, cy, z.r0 * k, 0, Math.PI * 2, true); ctx.fill();
        });
        ctx.globalAlpha = dim; ctx.lineWidth = 1; ctx.strokeStyle = rule;
        zs.forEach(function (z, i) {
          if (!i) return;
          ctx.beginPath(); ctx.arc(cx, cy, z.r0 * k, 0, Math.PI * 2); ctx.stroke();
        });
        ctx.globalAlpha = 0.5 * dim; ctx.strokeStyle = col('--rule-2'); ctx.beginPath();
        (L.order || []).forEach(function (i, j) {
          var n = nodes[i];
          if (!n) return;
          if (j) ctx.lineTo(sx(n.x), sy(n.y)); else ctx.moveTo(sx(n.x), sy(n.y));
        });
        ctx.stroke();
      } else {
        ctx.lineWidth = 1; ctx.strokeStyle = ink2; ctx.globalAlpha = 0.45 * dim; ctx.beginPath();
        D.edges.forEach(function (e) {
          var a = nodes[index[e.from]], b = nodes[index[e.to]];
          if (!a || !b || !on(a) || !on(b)) return;
          ctx.moveTo(sx(a.x), sy(a.y)); ctx.lineTo(sx(b.x), sy(b.y));
        });
        ctx.stroke();
      }
      if (focus >= 0) drawLinks(ctx, nodes[focus], neigh, 1, true);
      else if (hover >= 0) drawLinks(ctx, nodes[hover], adj[hover], 0.7, true);

      for (var q = 0; q < N; q++) {
        var n = nodes[q];
        if (n.a < 0.02) continue;
        var c = col(tok(n.type, n.status)), st = cls(n);
        var r = n.r * n.s * k, x = sx(n.x), y = sy(n.y);
        if (x < -r || y < -r || x > W + r || y > H + r) continue;
        ctx.globalAlpha = n.a * (st === 'fade' ? 0.5 : 1);
        ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2);
        if (st === 'solid' || st === 'fade') {
          ctx.fillStyle = c; ctx.fill(); ctx.lineWidth = 1.2; ctx.strokeStyle = surface; ctx.stroke();
        } else {
          ctx.fillStyle = surface; ctx.fill();
          ctx.lineWidth = Math.min(2, r * 0.5); ctx.strokeStyle = c;
          if (st === 'dash') ctx.setLineDash([3, 2.5]);
          ctx.stroke(); ctx.setLineDash([]);
        }
      }
      var ring = function (n, colour, wdt, gap) {
        ctx.globalAlpha = 1; ctx.lineWidth = wdt; ctx.strokeStyle = colour;
        ctx.beginPath(); ctx.arc(sx(n.x), sy(n.y), n.r * n.s * k + gap, 0, Math.PI * 2); ctx.stroke();
      };
      if (hover >= 0 && hover !== focus) ring(nodes[hover], col('--accent'), 2, 3);
      if (focus >= 0) { ring(nodes[focus], surface, 4, 3.5); ring(nodes[focus], ink, 2.2, 3.5); }

      if (arr === 'date') {
        ctx.font = '600 10.5px "IBM Plex Mono", monospace';
        ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
        ctx.lineWidth = 4; ctx.strokeStyle = surface; ctx.fillStyle = ink3; ctx.globalAlpha = dim;
        zones().forEach(function (z) {
          if (!z.label) return;
          var y = cy - ((z.r0 + z.r1) / 2) * k, w = ctx.measureText(z.label).width;
          if (fits(cx - w / 2, y - 7, w, 14)) { ctx.strokeText(z.label, cx, y); ctx.fillText(z.label, cx, y); }
        });
      }
      ctx.textBaseline = 'middle'; ctx.strokeStyle = surface; ctx.fillStyle = ink;
      if (focus >= 0) {
        var f = nodes[focus];
        label(ctx, f, f.id + ' · ' + fmtShort(f.date), true, sx(f.x) > W * 0.72 ? 'left' : 'right', 1);
        var ns = [];
        neigh.forEach(function (rels, j) { ns.push({ n: nodes[j], rels: rels }); });
        ns.sort(function (a, b) {
          return Math.hypot(a.n.x - f.x, a.n.y - f.y) - Math.hypot(b.n.x - f.x, b.n.y - f.y);
        });
        ns.forEach(function (e) {
          ctx.fillStyle = ink;
          label(ctx, e.n, e.n.id, false, sx(e.n.x) < sx(f.x) - 8 ? 'left' : 'right', 0.8);
          var d = Math.hypot(sx(e.n.x) - sx(f.x), sy(e.n.y) - sy(f.y));
          if (d < (f.r * f.s + e.n.r * e.n.s) * k + 50) return;
          var txt = e.rels.map(function (m) { return relWord(m.rel); }).join(' · ');
          ctx.font = '400 9.5px "IBM Plex Mono", monospace';
          var w = ctx.measureText(txt).width;
          var mx = sx(f.x) + (sx(e.n.x) - sx(f.x)) * 0.55, my = sy(f.y) + (sy(e.n.y) - sy(f.y)) * 0.55;
          if (!fits(mx - w / 2, my - 6, w, 12)) return;
          ctx.textAlign = 'center'; ctx.globalAlpha = 0.9; ctx.fillStyle = ink3; ctx.lineWidth = 4;
          ctx.strokeText(txt, mx, my); ctx.fillText(txt, mx, my);
        });
      }
      if (hover >= 0 && hover !== focus && !neigh.has(hover)) {
        var h = nodes[hover];
        ctx.fillStyle = ink;
        label(ctx, h, h.id + ' · ' + fmtShort(h.date) + ' · ' + h.title.slice(0, 44) +
          (h.title.length > 44 ? '…' : ''), false, sx(h.x) > W * 0.6 ? 'left' : 'right', 1);
      }
      ctx.globalAlpha = 1;
    }

    /* standby: the one piece of feedback a reader gets while the chart is
       working, raised for every relayout they wait on and not the first alone.
       It is the element the page was served with, so it comes down by the class
       the first paint has always used, and is hidden rather than removed.

       Raising it and doing the work in the same task paints nothing: the frame
       callbacks run before the paint, so the loop would draw the new layout
       before the browser ever showed the indicator. `raise` therefore hands the
       work to the second of two animation frames, which leaves the browser a
       whole frame in which to put the indicator on screen. Work that must stay
       inside the gesture that asked for it — a full-screen request, which the
       browser grants to a live user gesture only — is passed no work function
       and runs at the call site; there the transition is itself the wait.

       It comes down on the first frame drawn after the bubbles stop moving, or
       at HOLD, whichever is sooner. The cap is what makes the promise keepable:
       an arrangement whose published homes overlap never falls under the idle
       threshold at all — the glide home and the collision push trade places
       forever — and an indicator that waited for a stillness that never comes
       would sit over the chart for the rest of the session. The floor is the
       same clock read the other way: before it, a drawn frame takes the
       indicator down at once, which is how the first paint clears it. */
    var HOLD = 1000;
    var sb = document.getElementById('bstandby');
    var standby = true, held = 0;
    function raise(work) {
      if (sb) sb.classList.remove('done');
      requestAnimationFrame(function () {
        requestAnimationFrame(function () {
          standby = true;
          held = performance.now() + HOLD;
          /* a relayout the loop sleeps through — a stage that resized to the
             same size — draws no frame to come down on */
          setTimeout(function () { lower(true); }, HOLD);
          if (work) work();
        });
      });
    }
    function lower(over) {
      if (!standby) return;
      if (!over && performance.now() < held) return;
      standby = false;
      if (sb) sb.classList.add('done');
    }

    function frame() {
      if (!canvas.isConnected) return;
      if (need || energy > 0.02) {
        step(); draw();
        need = energy > 0.02;
        lower(!need);
      }
      G.raf = requestAnimationFrame(frame);
    }

    /* interaction: tap focuses, drag pans, wheel and pinch zoom. */
    function pick(x, y) {
      var best = -1, bd = 1e9, w = toWorld(x, y);
      for (var q = 0; q < N; q++) {
        var n = nodes[q];
        if (n.a < 0.3) continue;
        var d = Math.hypot(n.x - w[0], n.y - w[1]);
        var hit = Math.max(10 / view.k, n.r * n.s + 4 / view.k);
        if (d < hit && d < bd) { bd = d; best = n.i; }
      }
      return best;
    }
    var pos = function (e) {
      var r = canvas.getBoundingClientRect();
      return [e.clientX - r.left, e.clientY - r.top];
    };
    var ptrs = new Map(), dragging = false, pinch0 = 0, zoom0 = 1, pinchMid = [0, 0];
    canvas.addEventListener('pointerdown', function (e) {
      canvas.setPointerCapture(e.pointerId);
      var p = pos(e);
      ptrs.set(e.pointerId, { x: p[0], y: p[1], sx: p[0], sy: p[1] });
      dragging = false;
      if (ptrs.size === 2) {
        var v = Array.from(ptrs.values());
        pinch0 = Math.hypot(v[0].x - v[1].x, v[0].y - v[1].y);
        zoom0 = view.zoom;
        pinchMid = [(v[0].x + v[1].x) / 2, (v[0].y + v[1].y) / 2];
      }
    });
    canvas.addEventListener('pointermove', function (e) {
      var p = pos(e);
      if (!ptrs.has(e.pointerId)) {
        if (e.pointerType !== 'touch') {
          var h = pick(p[0], p[1]);
          if (h !== hover) { hover = h; need = true; }
          canvas.style.cursor = h >= 0 ? 'pointer' : 'grab';
        }
        return;
      }
      var q = ptrs.get(e.pointerId);
      if (ptrs.size === 2) {
        q.x = p[0]; q.y = p[1];
        var v = Array.from(ptrs.values());
        var d = Math.hypot(v[0].x - v[1].x, v[0].y - v[1].y);
        zoomAt((d / pinch0) * zoom0 / view.zoom, pinchMid[0], pinchMid[1]);
        dragging = true;
        return;
      }
      var dx = p[0] - q.x, dy = p[1] - q.y;
      if (!dragging && Math.hypot(p[0] - q.sx, p[1] - q.sy) > 4) {
        dragging = true; canvas.style.cursor = 'grabbing';
      }
      if (dragging) { view.px += dx; view.py += dy; need = true; }
      q.x = p[0]; q.y = p[1];
    });
    var up = function (e) {
      var q = ptrs.get(e.pointerId);
      ptrs.delete(e.pointerId);
      canvas.style.cursor = 'grab';
      if (!q || dragging) { if (!ptrs.size) dragging = false; return; }
      var h = pick(q.x, q.y);
      if (h >= 0) setFocus(h);
      else if (focus >= 0) setFocus(-1);
    };
    canvas.addEventListener('pointerup', up);
    canvas.addEventListener('pointercancel', up);
    canvas.addEventListener('pointerleave', function () { if (hover >= 0) { hover = -1; need = true; } });
    canvas.addEventListener('wheel', function (e) {
      e.preventDefault();
      var p = pos(e);
      zoomAt(Math.exp(-e.deltaY * 0.0015), p[0], p[1]);
    }, { passive: false });
    canvas.addEventListener('dblclick', function (e) { var p = pos(e); zoomAt(1.6, p[0], p[1]); });
    $('#bzin').addEventListener('click', function () { zoomAt(1.4, W / 2, H / 2); });
    $('#bzout').addEventListener('click', function () { zoomAt(1 / 1.4, W / 2, H / 2); });
    $('#bzreset').addEventListener('click', resetView);

    /* the card's date continuum: every date of the record on one line that runs
       the whole project, and the same line spoken as one sentence. */
    var span = [dayNum((L.span && L.span[0]) || ''), dayNum((L.span && L.span[1]) || '')];
    function dayNum(d) { return d ? Date.parse(d + 'T00:00:00Z') : 0; }
    function continuum(n) {
      /* A record corpus none of whose entries can be dated has no axis to draw
         one on. Drawing it anyway would walk a month at a time from 1970. */
      if (!span[0] || !span[1]) return '';
      var Wd = 320, x0 = 8, x1 = Wd - 8;
      var lo = span[0], hi = span[1] > span[0] ? span[1] : span[0] + 1;
      var X = function (d) { return x0 + (dayNum(d) - lo) / (hi - lo) * (x1 - x0); };
      var ticks = '';
      for (var t = new Date(lo); t.getTime() <= hi; t = new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth() + 1, 1))) {
        var iso = t.toISOString().slice(0, 10), tx = X(iso).toFixed(1);
        ticks += '<line x1="' + tx + '" y1="20" x2="' + tx + '" y2="25" class="tick"/>' +
          '<text x="' + tx + '" y="35" class="tlab">' +
          esc(t.toLocaleDateString(undefined, { month: 'short', timeZone: 'UTC' })) + '</text>';
      }
      var marks = [], spoken = [];
      var add = function (d, kind, txt, klass) {
        if (!d) return;
        var x = X(d), anchor = x < 60 ? 'start' : x > Wd - 60 ? 'end' : 'middle';
        var shape = kind === 'diamond'
          ? '<path d="M' + x.toFixed(1) + ' 13l6 7-6 7-6-7z"/>'
          : kind === 'ring'
            ? '<circle cx="' + x.toFixed(1) + '" cy="20" r="5.5"/>'
            : '<circle cx="' + x.toFixed(1) + '" cy="20" r="' + (kind === 'small' ? 3.5 : 5) + '"/>';
        marks.push('<g class="mark ' + esc(klass) + '" tabindex="0"><title>' + esc(txt) + '</title>' +
          '<circle class="hit" cx="' + x.toFixed(1) + '" cy="20" r="10"/>' + shape +
          '<text x="' + x.toFixed(1) + '" y="8" text-anchor="' + anchor + '">' + esc(txt) + '</text></g>');
        spoken.push(txt);
      };
      var tone = STATUS_TONE(n.status);
      var alive = (n.created && n.last)
        ? '<line x1="' + X(n.created).toFixed(1) + '" y1="20" x2="' + X(n.last).toFixed(1) + '" y2="20" class="alive"/>'
        : '';
      /* Each date is named by the record's own field name — the same words the
         record's page prints in its frontmatter table — so the line reads out
         as four named dates rather than as four bare ones. The middle mark
         takes the record's state instead, which is what that date means. */
      if (n.date && n.date !== n.created) add(n.date, 'diamond', W_.fDate + ' ' + fmtDate(n.date), 'dated ' + n.type);
      add(n.created, 'dot', W_.fCreated + ' ' + fmtDate(n.created), 'tree');
      if (n.entered && n.entered !== n.created) add(n.entered, 'ring', n.status + ' ' + fmtDate(n.entered), 'since ' + tone);
      if (n.last && n.last !== n.created && n.last !== n.entered) add(n.last, 'small', W_.fTouched + ' ' + fmtDate(n.last), 'last');
      var sr = spoken.join(' · ');
      return '<svg class="continuum" viewBox="0 0 ' + Wd + ' 40" role="img" aria-label="' + esc(sr) + '">' +
        '<line x1="' + x0 + '" y1="20" x2="' + x1 + '" y2="20" class="base"/>' +
        ticks + alive + marks.join('') + '</svg><span class="sr">' + esc(sr) + '</span>';
    }

    function renderCard() {
      card.getAnimations().forEach(function (a) { a.cancel(); });
      if (focus < 0) {
        if (!card.hidden) {
          var a = card.animate(
            [{ opacity: 1, transform: 'none' }, { opacity: 0, transform: 'translateY(10px)' }],
            { duration: 160, easing: 'ease-in', fill: 'forwards' });
          a.onfinish = function () { a.cancel(); card.hidden = true; };
        }
        return;
      }
      var wasHidden = card.hidden;
      var oldH = wasHidden ? 0 : card.getBoundingClientRect().height;
      var n = nodes[focus];
      /* One row per RELATION, not per neighbour: a record joined to another by
         two relations states two facts, and the card says both. */
      var links = [];
      neigh.forEach(function (rels, j) {
        rels.forEach(function (m) {
          if (m.rel === 'mention') return;
          links.push({ n: nodes[j], rel: m.rel, out: m.out });
        });
      });
      links.sort(function (a, b) {
        return (a.n.date || '').localeCompare(b.n.date || '') ||
          a.n.id.localeCompare(b.n.id, undefined, { numeric: true }) ||
          a.rel.localeCompare(b.rel);
      });
      var ment = madj[focus].size;
      var mine = stubs[focus] || [];
      card.hidden = false;

      /* The type pill's colour is set through the CSSOM after the markup is in
         place, never as a style="" attribute built out of data: an attribute is
         the one thing here a content policy has to make an allowance for, and
         this is the only one the script would have written. */
      var pills = '<span class="pill type"><i></i>' + esc(kindOf(n)) + '</span>' +
        (n.status ? '<span class="pill ' + STATUS_TONE(n.status) + '">' + esc(n.status) + '</span>' : '') +
        (n.sev ? '<span class="pill ' + (SEV_TONE[n.sev] || 'plain') + '">' + esc(n.sev) + '</span>' : '') +
        (n.kind && n.kind !== 'null' ? '<span class="pill plain">' + esc(n.kind) + '</span>' : '') +
        '<span class="mono id">' + esc(n.id) + '</span>';

      var rows = links.map(function (l) {
        var rel = l.out || !REL_INV[l.rel] ? relWord(l.rel) : REL_INV[l.rel];
        return '<li data-j="' + l.n.i + '"><span class="id">' + esc(l.n.id) + '</span>' +
          '<span>' + esc(l.n.title.slice(0, 56)) + '</span>' +
          '<span class="rel">' + esc(rel) + ' · ' + esc(fmtShort(l.n.date)) + '</span></li>';
      }).join('');
      /* A target that has left the tree is listed as absent, dashed and
         unclickable — never as a link to nothing. */
      rows += mine.map(function (u) {
        return '<li class="stubrow"><span class="id stub">' + esc(u.to) + '</span>' +
          '<span class="muted">' + esc(W_.notintree) + '</span>' +
          '<span class="rel">' + esc(relWord(u.rel)) + '</span></li>';
      }).join('');
      if (!rows) rows = '<li><span class="muted small">' + esc(W_.nolinks) + '</span></li>';

      var count = links.length + mine.length;
      var summary = count + ' ' + esc(W_.linked) + (ment ? ' · ' + ment + ' ' + esc(W_.mentions) : '');

      var foot = '';
      if (W_.blob) {
        foot += '<a class="gh" href="' + esc(W_.blob + n.path) + '">' + esc(W_.open) + ' ↗</a>';
      }
      if (W_.commits && n.last) {
        /* The last-touched date LINKS the record's commit history, so an
           amendment is traceable from the card rather than merely visible. */
        foot += '<a class="gh" href="' + esc(W_.commits + n.path) + '">' + esc(W_.history) +
          ' · ' + esc(fmtShort(n.last)) + ' ↗</a>';
      }

      card.innerHTML =
        '<button class="x" aria-label="' + esc(W_.close) + '">×</button>' +
        '<div class="head"><div class="t">' + esc(n.title) + '</div>' +
        '<div class="pills">' + pills + '</div>' + continuum(n) + '</div>' +
        '<div class="body"><details class="linked' + (count ? '' : ' empty') + '">' +
        '<summary' + (count ? '' : ' aria-disabled="true" tabindex="-1"') + '>' + summary + '</summary>' +
        '<ul>' + rows + '</ul></details></div>' +
        '<div class="foot"><div class="ghs">' + foot + '</div>' +
        '<div class="nav" role="group" aria-label="' + esc(W_.navlabel) + '">' +
        '<button class="hb" id="bback" aria-label="' + esc(W_.back) + '" disabled>‹</button>' +
        '<button class="hb" id="bfwd" aria-label="' + esc(W_.forward) + '" disabled>›</button></div></div>';

      var typePill = $('.pill.type', card);
      if (typePill) typePill.style.setProperty('--c', 'var(' + tok(n.type, n.status) + ')');
      $$('li[data-j]', card).forEach(function (li) {
        li.addEventListener('click', function () { setFocus(+li.dataset.j); });
      });
      $('.x', card).addEventListener('click', function () { setFocus(-1); });
      $('#bback', card).addEventListener('click', function () { go(-1); });
      $('#bfwd', card).addEventListener('click', function () { go(1); });

      if (wasHidden) {
        card.animate([{ opacity: 0, transform: 'translateY(12px)' }, { opacity: 1, transform: 'none' }],
          { duration: 240, easing: 'cubic-bezier(.2,.7,.2,1)' });
      } else {
        var newH = card.getBoundingClientRect().height;
        if (Math.abs(newH - oldH) > 2) {
          card.style.overflow = 'hidden';
          var an = card.animate([{ height: oldH + 'px' }, { height: newH + 'px' }],
            { duration: 240, easing: 'cubic-bezier(.2,.7,.2,1)' });
          an.onfinish = function () { card.style.overflow = ''; };
        }
      }
    }

    function syncHist() {
      var hb = $('#bback', card), hf = $('#bfwd', card);
      if (!hb) return;
      hb.disabled = !(hpos > 0);
      hf.disabled = !(hpos < hist.length - 1);
    }
    function go(d) {
      var p = hpos + d;
      if (p < 0 || p >= hist.length) return;
      hpos = p;
      setFocus(hist[p], true);
    }
    document.addEventListener('keydown', function (e) {
      if (!canvas.isConnected || !e.altKey || focus < 0) return;
      if (e.key === 'ArrowLeft') { e.preventDefault(); go(-1); }
      else if (e.key === 'ArrowRight') { e.preventDefault(); go(1); }
    });

    /* the explicit two-state arrangement control */
    $$('#barr button').forEach(function (b) {
      b.addEventListener('click', function () {
        $$('#barr button').forEach(function (x) { x.setAttribute('aria-checked', String(x === b)); });
        /* every bubble leaves for a new home, so this is the long wait */
        raise(function () { arr = b.dataset.arr; need = true; });
      });
    });

    /* search */
    var sInput = $('#bsearch'), hits = $('#bhits');
    sInput.addEventListener('input', function () {
      var q = sInput.value.trim().toLowerCase();
      if (!q) { hits.style.display = 'none'; return; }
      var found = nodes.filter(function (n) {
        return (n.id + ' ' + n.title).toLowerCase().indexOf(q) >= 0;
      }).slice(0, 12);
      hits.innerHTML = found.map(function (n) {
        return '<div data-j="' + n.i + '"><span class="id">' + esc(n.id) + '</span>' +
          '<span>' + esc(n.title.slice(0, 60)) + '</span></div>';
      }).join('');
      hits.style.display = 'block';
      $$('div[data-j]', hits).forEach(function (d) {
        d.addEventListener('click', function () {
          setFocus(+d.dataset.j); hits.style.display = 'none'; sInput.value = '';
        });
      });
    });
    sInput.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') { var f = $('div[data-j]', hits); if (f) f.click(); }
      if (e.key === 'Escape') hits.style.display = 'none';
    });

    /* full screen: the Fullscreen API where it exists, a fixed overlay where it
       does not (iPhone Safari has no element fullscreen). */
    var fsb = $('#bfs');
    var fsOn = function () { return document.fullscreenElement === stage || stage.classList.contains('fs'); };
    function setFs(want) {
      /* the stage changes size under every bubble, so the chart re-settles
         either way in and out; the request itself stays in this task, because
         the browser grants full screen to a live gesture only */
      raise();
      if (want) {
        if (stage.requestFullscreen) stage.requestFullscreen().catch(function () { stage.classList.add('fs'); });
        else stage.classList.add('fs');
      } else {
        if (document.fullscreenElement) document.exitFullscreen();
        stage.classList.remove('fs');
      }
    }
    function syncFs() {
      var o = fsOn();
      fsb.setAttribute('aria-pressed', String(o));
      fsb.textContent = o ? '✕' : '⤢';
      fsb.setAttribute('aria-label', o ? W_.fsOff : W_.fsOn);
      document.documentElement.classList.toggle('bfs-open', o);
    }
    fsb.addEventListener('click', function () { setFs(!fsOn()); setTimeout(syncFs, 50); });
    /* the browser's own way out of full screen never passes through setFs */
    document.addEventListener('fullscreenchange', function () { raise(); syncFs(); });
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && stage.classList.contains('fs')) { setFs(false); syncFs(); }
    });

    /* filters */
    var fb = $('#bfilters-btn'), fp = $('#bfilters');
    fb.addEventListener('click', function () {
      var open = fp.hidden;
      fp.hidden = !open;
      fb.setAttribute('aria-expanded', String(open));
    });
    $$('#typechips button').forEach(function (b) {
      b.addEventListener('click', function () {
        typeOn[b.dataset.t] = !typeOn[b.dataset.t];
        b.classList.toggle('on'); b.classList.toggle('off');
        setFocus(focus);
      });
    });
    var gm = $('#gmentions');
    if (gm) gm.addEventListener('change', function (e) { useMent = e.target.checked; setFocus(focus); });

    /* lifecycle */
    G = { raf: 0, ro: null };
    size();
    base = baseTarget(); view.tx = base.x; view.ty = base.y;
    nodes.forEach(function (n) { n.x = n.hx[arr]; n.y = n.hy[arr]; });
    /* A stage that changes size by a LOT has been thrown into or out of full
       screen, and the chart re-settles either way. Raising here rather than only
       at the click covers every route out — the button, Escape, the browser's
       own chrome — and it covers the LEAVING transition, where the browser's
       animation runs first and a hold started at the click has already expired
       by the time the chart begins to move. A drag of the window edge changes
       the size by a little and is left alone, so the indicator never strobes. */
    var lastW = 0, lastH = 0;
    var ro = new ResizeObserver(function () {
      if (!canvas.isConnected) { ro.disconnect(); return; }
      var r = stage.getBoundingClientRect();
      var big = lastW > 0 && (Math.abs(r.width - lastW) > lastW * 0.25 ||
        Math.abs(r.height - lastH) > lastH * 0.25);
      lastW = r.width; lastH = r.height;
      if (big) raise();
      size();
    });
    ro.observe(stage); G.ro = ro;
    G.raf = requestAnimationFrame(frame);

    var wanted = new URLSearchParams(location.search).get('focus');
    var fi = -1;
    if (wanted && index[wanted] !== undefined) fi = index[wanted];
    setFocus(fi);
  }
})();
