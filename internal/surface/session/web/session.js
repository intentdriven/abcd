// The brief session page (itd-2610091016262335, spc-2610141709430146 steps 3
// and 4).
//
// A team member hosts a live session on the team's own relay; everyone else
// joins by the QR code or the six-character code. Team members edit the brief
// directly and their edits are credited to them; a guest proposes a
// replacement for a passage, which a team member applies or declines. The
// session goes back to GitHub as one pre-filled issue per changed passage, or
// as one pull request made with the submitter's own token.
//
// What this page talks to, and nothing else: the relay the person names (its
// WebSocket, /config and /auth/token), https://api.github.com (without a
// token for the repository's list of its relays, and with the person's own
// token), and github.com for sign-in and pre-filled issues. The GitHub token
// is held in this module's memory only; it is never stored, put in a URL or
// logged, and it reaches a relay only once the repository's committed relay
// list names that relay or, where the list cannot be read, the person has
// confirmed it. The relay's rejoin token, which brings this participant
// back to the session after a dropped connection or a reload, is not a GitHub
// token: it is kept in memory and in this tab's session storage.
// Every piece of text from the brief, the relay or another participant is
// rendered with textContent.
import {
  Y, Awareness, encodeAwarenessUpdate, applyAwarenessUpdate, removeAwarenessStates,
  yCollab, yUndoManagerKeymap, EditorView, basicSetup, EditorState, keymap, markdown, qrcode,
} from "./vendor/collab.js";

// ---------------------------------------------------------------------------
// The words the page shows: one table, so they can be reviewed in one place.
// ---------------------------------------------------------------------------
const WORDS = Object.freeze({
  title: "Brief session",
  wrongAddress: "This page opens a project's brief for a live session. Open it at /<owner>/<repo>/brief/.",
  relayLabel: "Relay address",
  relayHint: "The https address of your team's relay, or of the host's computer on Tailscale.",
  relayBad: "Give the relay's https address (plain http only for localhost).",
  relayChecking: "Checking which relays this repository names…",
  relayListed: "This repository names this relay in .abcd/relay/relays.json.",
  relayConfirmed: "You confirmed earlier that your team runs this relay:",
  relayUnconfirmed: "This repository's relay list could not be read without signing in, so the page cannot check this relay for you. Before any sign-in reaches it, you will be asked to confirm it:",
  relayRefused: "This repository names the relays its team runs in .abcd/relay/relays.json, and this one is not among them, so the page sends it no sign-in and does not connect to it:",
  relayNoneUsable: "This repository's .abcd/relay/relays.json names no relay the page can use, so the page sends no sign-in to this one and does not connect to it:",
  trustHeading: "Check the relay",
  trustOnly: "Only continue if your team runs this relay:",
  trustWhy: "Signing in sends your GitHub sign-in to this relay. This repository does not name its relays where the page can read them, so only you can tell whether this is your team's. A link from someone else can name any relay.",
  trustYes: "My team runs this relay",
  oauthNote: "This relay signs in through an OAuth app, which asks GitHub for access to every repository you can reach. A GitHub App would be narrower: ask your team to register one.",
  joinHeading: "Join a session",
  joinHint: "Type the code the host's screen shows, or scan its QR code.",
  codeLabel: "Join code",
  nameLabel: "Your name",
  join: "Join",
  codeBad: "A join code is six letters and digits.",
  teamHeading: "Team members",
  teamHint: "Sign in with GitHub to edit directly and to host a session. Without signing in you join as a guest and suggest changes for a team member to approve.",
  signIn: "Sign in with GitHub",
  pasteToken: "Paste a GitHub token instead",
  tokenLabel: "Personal access token",
  useToken: "Use this token",
  signInUnavailable: "This relay has no GitHub sign-in set up. Paste a personal access token instead.",
  signInFailed: "GitHub sign-in did not complete. Try again, or paste a personal access token.",
  signingIn: "Signing in…",
  signedInAs: "Signed in as",
  notMember: "GitHub says you cannot change this repository, so you take part as a guest.",
  hostHeading: "Host a session",
  host: "Host a session",
  modeLabel: "Send the result back as",
  modePR: "one pull request",
  modeIssues: "one issue per change",
  loadingBrief: "Reading the brief from GitHub…",
  sendingBrief: "Sharing the brief with the session…",
  noBrief: "This repository has no brief at .abcd/development/brief/ on its default branch.",
  githubFailed: "GitHub did not answer the request for the brief",
  connecting: "Connecting…",
  connected: "Connected",
  reconnecting: "Connection lost. Reconnecting…",
  cannotConnect: "The relay could not be reached.",
  syncing: "Loading the brief…",
  ended: "The session has ended.",
  codeGone: "That session is no longer open on this relay.",
  inviteHeading: "Invite",
  inviteHint: "Scan to join, or type the code.",
  newCode: "New code",
  endSession: "End session",
  endConfirm: "End the session for everyone? Edits not yet sent to GitHub are lost.",
  copyLink: "Copy link",
  linkCopied: "Link copied.",
  filesHeading: "Brief",
  noFile: "Choose a file.",
  rosterHeading: "In this session",
  badgeHost: "host",
  badgeMember: "team member",
  badgeGuest: "guest",
  you: "(you)",
  suggestionsHeading: "Suggestions",
  noSuggestions: "No suggestions waiting.",
  suggestedBy: "Suggested by",
  apply: "Apply",
  decline: "Decline",
  applied: "applied",
  declined: "declined",
  waiting: "waiting for a team member",
  cannotApply: "That passage has changed since it was suggested, so the suggestion cannot be applied.",
  guestHint: "You are a guest. Select a passage to suggest a change.",
  suggest: "Suggest a change",
  selectFirst: "Select a passage first.",
  replacementLabel: "Replace it with",
  send: "Send suggestion",
  cancel: "Cancel",
  suggestionSent: "Suggestion sent to the team members.",
  noMembers: "No team member is connected to receive it.",
  mySuggestionsHeading: "Your suggestions",
  submit: "Send to GitHub",
  sendHeading: "Send to GitHub",
  sendIntroPR: "The session goes back as one pull request, with every credited team member named on its commit.",
  sendIntroIssues: "Each changed passage goes back as its own issue. GitHub opens the issue for you to check before you submit it.",
  sendNothing: "Nothing has changed since the session began.",
  sendRefresh: "Refresh",
  sendClose: "Close",
  sendAsIssue: "Send as issue",
  tooLongForIssue: "Too long for an issue, send as a pull request",
  sendAsPR: "Send as a pull request",
  openPR: "Open the pull request",
  prNeedsSignIn: "Only a team member signed in with GitHub can open the pull request.",
  linesAt: "lines",
  lineAt: "line",
  creditUnknown: "A participant the session could not credit",
  prWorking: "Opening the pull request…",
  prOpened: "Pull request opened:",
  passageMoved: "That passage has changed since the panel showed it. Check it again, then send.",
  prNoBase: "This session does not know which commit it started from, so it cannot open a pull request.",
  ghUnreachable: "GitHub could not be reached. Check your connection and try again.",
  ghSignIn: "GitHub did not accept your sign-in. Sign in again, then send.",
  ghScope: "GitHub refused: your sign-in may not create branches, commits or pull requests in this repository. Sign in again with access to its contents and pull requests.",
  ghNotFound: "GitHub could not find the repository, the branch or a team member, or your sign-in cannot see them.",
  ghStale: "The new branch moved on GitHub before the commit was made. Try again.",
  ghRefTaken: "GitHub would not create the session's branch; one by that name may already exist. Try again.",
  ghPRRefused: "The commit is on the branch, but GitHub would not open the pull request. Open it on GitHub from branch",
  ghOther: "GitHub did not complete the request. Try again in a moment.",
  forgedUpdate: "An edit was not applied, because it claimed to be another participant's. It came from",
  pathRefused: "Not sent: this file is not one of the brief's files the session started from:",
});

// ---------------------------------------------------------------------------
// Fixed values.
// ---------------------------------------------------------------------------
const GITHUB_API = "https://api.github.com";
const GITHUB_WEB = "https://github.com";
const GITHUB_AUTHORIZE = "https://github.com/login/oauth/authorize";
// GitHub answers 414 to a pre-filled issue link over its server limit and does
// not document the number; the GitHub CLI stops at 8,192 bytes. The link is
// measured as GitHub receives it, encoded, in bytes, against this margin
// (research note 2026-10-14-github-roles-admission-and-issue-links.md).
const ISSUE_URL_LIMIT = 7600;
const COMMIT_HEADLINE = "Edit the brief in a live session";
// Passages closer than this many unchanged lines are one passage.
const PASSAGE_JOIN_LINES = 1;
const BRIEF_DIR = [".abcd", "development", "brief"];
const BRIEF_PREFIX = BRIEF_DIR.join("/") + "/";
const SIGN_IN_SCOPE = "repo";
const TAG_UPDATE = 0;
const TAG_AWARENESS = 1;
// The relay sends every member's update on as tag 2: the tag, the sender's
// participant number as four big-endian bytes, then the update's bytes as the
// member sent them. The roster's authors map names each number's login.
const TAG_ATTRIBUTED = 2;
// A relay message is at most 256 KiB; the seed is sent in updates of at most
// SEED_BUDGET bytes of text, and a longer file in pieces of SEED_PIECE UTF-16
// units (three bytes each at most).
const SEED_BUDGET = 160000;
const SEED_PIECE = 50000;
const QUOTE_CONTEXT = 32;
const MAX_REPLACEMENT = 64000;
// Origins of Yjs transactions: what the relay sent, and the host's seeding.
const FROM_RELAY = Symbol("relay");
const FROM_SEED = Symbol("seed");
const KEY_RELAY = "abcd.session.relay";
const KEY_NAME = "abcd.session.name";
const KEY_SIGN_IN = "abcd.session.sign-in";
const KEY_REJOIN = "abcd.session.rejoin";
// The relays a person confirmed, per repository: { "owner/repo": [origin] }.
const KEY_TRUSTED = "abcd.session.trusted-relays";
// The repository's committed list of its relays: {"relays": [origin, …]}.
const RELAYS_FILE = ".abcd/relay/relays.json";
// A relay's rejoin token: 128 random bits, base64url without padding.
const REJOIN_RE = /^[A-Za-z0-9_-]{22}$/;
// A participant's awareness colour: the editor writes it into a style
// attribute, so a remote one is used only when it is exactly this.
const HEX_COLOUR_RE = /^#[0-9a-fA-F]{6}$/;
const COLOURS = ["#2f6fdb", "#c2410c", "#15803d", "#9333ea", "#b91c1c", "#0e7490", "#a16207", "#be185d"];

// ---------------------------------------------------------------------------
// Storage: only the relay address and the display name are remembered, and
// the sign-in's PKCE verifier waits in session storage across GitHub's
// redirect. Every access may throw (private windows, blocked storage).
// ---------------------------------------------------------------------------
function recall(key) {
  try { return localStorage.getItem(key) || ""; } catch { return ""; }
}
function remember(key, value) {
  try { localStorage.setItem(key, value); } catch { /* storage unavailable */ }
}
function stashSignIn(pending) {
  try { sessionStorage.setItem(KEY_SIGN_IN, JSON.stringify(pending)); return true; } catch { return false; }
}
function takeSignIn() {
  try {
    const raw = sessionStorage.getItem(KEY_SIGN_IN);
    sessionStorage.removeItem(KEY_SIGN_IN);
    return raw ? JSON.parse(raw) : null;
  } catch { return null; }
}

// trustedRelays reads the confirmed relays of every repository; confirmRelay
// and forgetRelay change this repository's. Nothing else is kept with them.
function trustedRelays() {
  try {
    const all = JSON.parse(recall(KEY_TRUSTED) || "{}");
    return all !== null && typeof all === "object" && !Array.isArray(all) ? all : {};
  } catch { return {}; }
}
function confirmedRelays() {
  const list = trustedRelays()[route.slug];
  return Array.isArray(list) ? list.filter((r) => typeof r === "string") : [];
}
function confirmRelay(relay) {
  const all = trustedRelays();
  all[route.slug] = confirmedRelays().filter((r) => r !== relay).concat([relay]).slice(-8);
  remember(KEY_TRUSTED, JSON.stringify(all));
}
function forgetRelay(relay) {
  const all = trustedRelays();
  all[route.slug] = confirmedRelays().filter((r) => r !== relay);
  remember(KEY_TRUSTED, JSON.stringify(all));
}

// ---------------------------------------------------------------------------
// The GitHub token: module memory only. A token just signed in or pasted is
// held apart until the relay it would go to is cleared; only then does it
// become the token verify sends.
// ---------------------------------------------------------------------------
let githubToken = "";
let heldToken = "";

// ---------------------------------------------------------------------------
// The rejoin token: the relay gives one to each participant, and presenting
// it brings the same participant back without the join code, which the host
// may have changed since. It is kept in memory, and in this tab's session
// storage with the relay and repository it belongs to, so a reload rejoins.
// ---------------------------------------------------------------------------
let rejoinToken = "";
function keepRejoin(token) {
  if (typeof token !== "string" || !REJOIN_RE.test(token)) return;
  rejoinToken = token;
  try { sessionStorage.setItem(KEY_REJOIN, JSON.stringify({ relay: session.relay, repo: route.slug, token })); } catch { /* storage unavailable */ }
}
function forgetRejoin() {
  rejoinToken = "";
  try { sessionStorage.removeItem(KEY_REJOIN); } catch { /* storage unavailable */ }
}
// takeRejoin reads a token kept by this tab before a reload, for this
// repository: it sets the relay it belongs to and reports whether there is one.
function takeRejoin() {
  let kept = null;
  try { kept = JSON.parse(sessionStorage.getItem(KEY_REJOIN) || "null"); } catch { kept = null; }
  if (!kept || kept.repo !== route.slug || typeof kept.token !== "string" || !REJOIN_RE.test(kept.token)) return false;
  const relay = relayOrigin(kept.relay);
  if (!relay || (session.relay && relay !== session.relay)) return false;
  session.relay = relay;
  rejoinToken = kept.token;
  return true;
}

// ---------------------------------------------------------------------------
// Small DOM helpers. Text always goes in through textContent.
// ---------------------------------------------------------------------------
function el(tag, props = {}, ...children) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(props)) {
    if (v === undefined || v === null || v === false) continue;
    if (k === "class") node.className = v;
    else if (k === "text") node.textContent = v;
    else if (k.startsWith("on")) node.addEventListener(k.slice(2), v);
    else node.setAttribute(k, v === true ? "" : String(v));
  }
  for (const c of children) {
    if (c === null || c === undefined || c === false) continue;
    node.append(typeof c === "string" ? document.createTextNode(c) : c);
  }
  return node;
}
function clear(node) {
  while (node.firstChild) node.removeChild(node.firstChild);
}
let toastTimer = 0;
function toast(text) {
  const t = ui.toast;
  t.textContent = text;
  t.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { t.hidden = true; }, 6000);
}

// ---------------------------------------------------------------------------
// Where the page is: /<owner>/<repo>/brief/, under any base path.
// ---------------------------------------------------------------------------
const NAME_RE = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$/;
const REPO_RE = /^[A-Za-z0-9._-]{1,100}$/;
function readRoute() {
  const m = /^(.*)\/([^/]+)\/([^/]+)\/brief\/?$/.exec(location.pathname);
  if (!m) return null;
  let owner, repo;
  try { owner = decodeURIComponent(m[2]); repo = decodeURIComponent(m[3]); } catch { return null; }
  if (!NAME_RE.test(owner) || !REPO_RE.test(repo) || repo === "." || repo === "..") return null;
  return { base: m[1], owner, repo, slug: owner + "/" + repo };
}
function pageURL() {
  return location.origin + location.pathname;
}

// relayOrigin reads a relay address: https, or http on this computer.
function relayOrigin(input) {
  const s = String(input || "").trim();
  if (!s) return "";
  let u;
  try { u = new URL(/^[a-z][a-z0-9+.-]*:\/\//i.test(s) ? s : "https://" + s); } catch { return ""; }
  if (u.username || u.password) return "";
  const local = u.hostname === "localhost" || u.hostname === "127.0.0.1";
  if (u.protocol === "https:" || (u.protocol === "http:" && local)) return u.origin;
  return "";
}
const CODE_ALPHABET = "ABCDEFGHJKMNPQRSTUVWXYZ23456789";
function normalizeCode(s) {
  const c = String(s || "").toUpperCase().replace(/[\s-]/g, "");
  if (c.length !== 6) return "";
  for (const ch of c) if (!CODE_ALPHABET.includes(ch)) return "";
  return c;
}

// ---------------------------------------------------------------------------
// The shared document. gc stays off so every change can be credited.
// ---------------------------------------------------------------------------
const doc = new Y.Doc({ gc: false });
const files = doc.getMap("files");
const meta = doc.getMap("meta");
const approvals = doc.getArray("approvals");
// A guest's suggestion, recorded by the first team member whose page saw it,
// so a member who joins later sees it too; suggestions_done holds its answer.
const suggestions = doc.getMap("suggestions");
const suggestionsDone = doc.getMap("suggestions_done");
// Yjs records who inserted each piece of text but not who deleted it: each
// member's page records its own deletions here, under its Yjs client id.
const deletions = new Y.PermanentUserData(doc, doc.getMap("deletions"));
const awareness = new Awareness(doc);

// ---------------------------------------------------------------------------
// Page state.
// ---------------------------------------------------------------------------
const route = readRoute();
const session = {
  relay: "",
  code: "",
  name: "",
  login: "",
  member: false,      // GitHub says this login may push
  role: "",           // "member" or "guest", once in the session
  host: false,
  myId: "",
  inRoom: false,      // joined or hosted on the current socket
  rejoining: false,   // a rejoin token was presented on the current socket
  everJoined: false,  // reconnect only once a session was reached
  seeding: false,
  synced: false,
  over: false,
  verifying: false,
  pending: "",        // "host" when the host button waits for a socket
  roster: [],
  credits: {},        // Yjs client id -> login, from the relay's roster
  authors: {},        // participant number -> login, from the relay's roster
};
// Attributed updates wait here until a roster on this connection names their
// senders: the relay replays a session's log before it sends the roster.
const held = [];
let rosterSeen = false;
// Client ids seen in an unattributed (tag 0) update, from a relay that does
// not attribute: their text is applied but credited to nobody.
const unattributed = new Set();
const mine = [];              // a guest's own suggestions
const awaitingAck = [];       // suggestions sent, waiting for "suggested"
let ws = null;
let retries = 0;
let retryTimer = 0;
let outbox = [];              // a member's updates made while disconnected
// A member's updates sent in the last few seconds: the relay drops an update
// that comes too fast and says only "rate-limited", so they are all sent
// again, merged, a moment later. Yjs applies an update it already has as
// nothing, so sending one twice is harmless.
let recentUpdates = [];
let resendTimer = 0;
const RESEND_WINDOW = 5000;
let quietClose = false;       // the relay closed an idle connection normally

// ---------------------------------------------------------------------------
// The relay connection.
// ---------------------------------------------------------------------------
function socketOpen() {
  return ws !== null && ws.readyState === WebSocket.OPEN;
}
function sendJSON(msg) {
  if (socketOpen()) ws.send(JSON.stringify(msg));
}
function sendBinary(tag, bytes) {
  const frame = new Uint8Array(bytes.length + 1);
  frame[0] = tag;
  frame.set(bytes, 1);
  if (socketOpen() && session.inRoom) {
    ws.send(frame);
    return true;
  }
  return false;
}

function openSocket() {
  if (ws !== null || session.over || !session.relay || relayStanding(session.relay) === "refused") return;
  setStatus(session.everJoined ? WORDS.reconnecting : WORDS.connecting, session.everJoined);
  const url = session.relay.replace(/^http/, "ws") + "/session";
  const sock = new WebSocket(url);
  sock.binaryType = "arraybuffer";
  rosterSeen = false;
  ws = sock;
  sock.onopen = () => { if (ws === sock) afterOpen(); };
  sock.onmessage = (ev) => { if (ws === sock) receive(ev.data); };
  sock.onclose = (ev) => { if (ws === sock) socketClosed(ev); };
}

function afterOpen() {
  if (rejoinToken) {
    session.rejoining = true;
    sendJSON({ type: "rejoin", token: rejoinToken });
    return;
  }
  if (githubToken && relayCleared()) {
    session.verifying = true;
    sendJSON({ type: "verify", token: githubToken, repo: route.slug });
    return;
  }
  proceed();
}

// proceed joins or hosts once the connection is open and any sign-in checked.
function proceed() {
  if (session.code) {
    sendJSON({ type: "join", code: session.code, name: session.name });
  } else if (session.pending === "host") {
    session.pending = "";
    sendJSON({ type: "host", repo: route.slug });
  }
  renderStart();
}

function socketClosed(ev) {
  ws = null;
  session.inRoom = false;
  session.verifying = false;
  session.rejoining = false;
  const quiet = quietClose;
  quietClose = false;
  renderTop();
  if (session.over) return;
  // The relay closes a connection that joined nothing in time; the next
  // Join or Host opens another.
  if (quiet && !session.everJoined) {
    setStatus("", false);
    renderStart();
    return;
  }
  if (ev.code === 1008) {
    endSession(ev.reason ? ev.reason : WORDS.cannotConnect);
    return;
  }
  if (!session.everJoined) {
    setStatus(WORDS.cannotConnect, true);
    renderStart();
    return;
  }
  setStatus(WORDS.reconnecting, true);
  const delay = Math.min(30000, 1000 * 2 ** retries) * (0.5 + Math.random() / 2);
  retries += 1;
  clearTimeout(retryTimer);
  retryTimer = setTimeout(openSocket, delay);
}

function endSession(text) {
  session.over = true;
  session.inRoom = false;
  forgetRejoin();
  clearTimeout(retryTimer);
  if (ws) {
    const sock = ws;
    ws = null;
    try { sock.close(); } catch { /* already closed */ }
  }
  setStatus(text, true);
  toast(text);
  renderAll();
}

function receive(data) {
  if (data instanceof ArrayBuffer) {
    const bytes = new Uint8Array(data);
    if (bytes.length === 0) return;
    const body = bytes.subarray(1);
    try {
      if (bytes[0] === TAG_ATTRIBUTED) {
        if (bytes.length < 5) return;
        const num = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(1);
        const update = bytes.slice(5);
        if (!rosterSeen) held.push({ num, update });
        else admitUpdate(num, update);
      } else if (bytes[0] === TAG_UPDATE) {
        // An older relay's update, which names no sender: applied, credited
        // to nobody.
        noteUnattributed(body);
        Y.applyUpdate(doc, body, FROM_RELAY);
      } else if (bytes[0] === TAG_AWARENESS) applyAwarenessUpdate(awareness, cleanAwarenessUpdate(body), FROM_RELAY);
    } catch {
      // A frame that does not decode is dropped; the relay forwards only
      // what a participant's page sent.
    }
    return;
  }
  let msg;
  try { msg = JSON.parse(data); } catch { return; }
  if (!msg || typeof msg.type !== "string") return;
  switch (msg.type) {
    case "verified": onVerified(msg); break;
    case "hosted": onHosted(msg); break;
    case "joined": onJoined(msg); break;
    case "synced": onSynced(); break;
    case "roster": onRoster(msg); break;
    case "suggest": onSuggest(msg); break;
    case "suggested": onSuggested(msg); break;
    case "code": onCode(msg); break;
    case "closed": endSession(WORDS.ended); break;
    case "error": onError(msg); break;
    default: break;
  }
}

// forgedUpdate says whether an attributed update holds anything its sender
// may not have written: a struct whose client id the roster does not credit
// to the sender's login, or a sender the roster does not name. Its deletions
// are not checked: a member may delete anyone's text, and who deleted it is
// credited from the deletion records, checked in verifiedDeleters.
function forgedUpdate(num, update) {
  const login = session.authors[String(num)];
  if (typeof login !== "string" || login === "") return true;
  let decoded;
  try { decoded = Y.decodeUpdate(update); } catch { return true; }
  for (const s of decoded.structs) {
    if (creditedTo(s.id.client) !== login) return true;
  }
  return false;
}

// admitUpdate applies an attributed update, or drops all of it and says
// whose it was (security review F6).
function admitUpdate(num, update) {
  if (forgedUpdate(num, update)) {
    const who = session.authors[String(num)];
    toast(WORDS.forgedUpdate + " " + (typeof who === "string" && NAME_RE.test(who) ? "@" + who : "participant p" + num) + ".");
    return;
  }
  try { Y.applyUpdate(doc, update, FROM_RELAY); } catch { /* an update that does not decode is dropped */ }
}

function noteUnattributed(update) {
  for (const s of Y.decodeUpdate(update).structs) {
    if (s.id.client !== doc.clientID) unattributed.add(s.id.client);
  }
}

function onVerified(msg) {
  const wasVerifying = session.verifying;
  session.verifying = false;
  session.login = typeof msg.login === "string" ? msg.login : "";
  session.member = msg.member === true;
  if (session.inRoom) {
    // A participant who joined by code and then signed in.
    if (session.member) becomeMember();
    else toast(WORDS.notMember);
    renderAll();
    return;
  }
  if (!session.member && !session.code) toast(WORDS.notMember);
  if (wasVerifying) proceed();
  renderAll();
}

function onHosted(msg) {
  keepRejoin(msg.rejoin);
  session.code = normalizeCode(msg.code);
  session.myId = String(msg.id || "");
  session.host = true;
  session.role = "member";
  session.inRoom = true;
  session.everJoined = true;
  session.seeding = true;
  retries = 0;
  setStatus(WORDS.connected, false);
  sendJSON({ type: "register", client: doc.clientID });
  renderAll();
  hostBrief().catch((err) => {
    // A session that cannot be seeded is closed again: nobody has its code.
    sendJSON({ type: "close" });
    session.seeding = false;
    endSession(err instanceof Error && err.message ? err.message : WORDS.githubFailed);
  });
}

function onJoined(msg) {
  keepRejoin(msg.rejoin);
  const rejoined = session.rejoining;
  session.rejoining = false;
  session.myId = String(msg.id || "");
  session.role = msg.role === "member" ? "member" : "guest";
  session.host = msg.host === true;
  if (typeof msg.name === "string" && session.role === "guest") session.name = msg.name;
  session.inRoom = true;
  session.everJoined = true;
  retries = 0;
  setStatus(session.synced ? WORDS.connected : WORDS.syncing, false);
  // Back by the rejoin token after signing in on this page: the relay has not
  // seen this sign-in yet.
  if (rejoined && githubToken && relayCleared() && !session.login && session.role !== "member") sendJSON({ type: "verify", token: githubToken });
  renderAll();
}

function onSynced() {
  if (session.seeding) return; // the host's own fresh session: the seed follows
  session.synced = true;
  setStatus(WORDS.connected, false);
  if (session.role === "member") becomeMember();
  shareAwareness();
  renderAll();
}

function becomeMember() {
  const was = session.role;
  session.role = "member";
  sendJSON({ type: "register", client: doc.clientID });
  trackDeletions();
  const queued = outbox;
  outbox = [];
  for (const frame of queued) {
    if (!sendBinary(TAG_UPDATE, frame)) outbox.push(frame);
    else keepRecent(frame);
  }
  if (was !== "member") openFile(currentPath);
}

function onRoster(msg) {
  const before = new Set(session.roster.map((p) => p.id));
  session.roster = Array.isArray(msg.participants) ? msg.participants.filter((p) => p && typeof p.id === "string") : [];
  session.credits = msg.clients && typeof msg.clients === "object" ? msg.clients : {};
  session.authors = msg.authors && typeof msg.authors === "object" ? msg.authors : {};
  rosterSeen = true;
  for (const h of held.splice(0)) admitUpdate(h.num, h.update);
  for (const p of session.roster) {
    if (p.id === session.myId) {
      session.host = p.host === true;
      if (p.role === "member" && session.role !== "member" && session.synced) becomeMember();
    }
  }
  // A newcomer sees the others' cursors once they say where they are.
  if (session.roster.some((p) => !before.has(p.id))) shareAwareness();
  renderSide();
  renderTop();
}

function onCode(msg) {
  const code = normalizeCode(msg.code);
  if (code) {
    session.code = code;
    updateFragment();
    renderSide();
  }
}

function onError(msg) {
  const code = typeof msg.code === "string" ? msg.code : "";
  const text = typeof msg.message === "string" ? msg.message : code;
  if (session.rejoining && code === "bad-rejoin") {
    // The session no longer knows this token: sign in and join by the code
    // instead, on the same connection.
    session.rejoining = false;
    forgetRejoin();
    afterOpen();
    return;
  }
  if (code === "rejoined-elsewhere" || code === "too-many-tries") forgetRejoin();
  if (code === "join-timeout") {
    // A normal close follows; a session in progress reconnects and rejoins.
    quietClose = true;
    return;
  }
  if (code === "rate-limited" && session.role === "member" && !session.verifying) {
    // The relay dropped a message that came too fast: perhaps an update, so
    // the recent ones go again.
    resendRecent();
    return;
  }
  if (session.verifying) {
    session.verifying = false;
    githubToken = "";
    toast(text);
    proceed();
    return;
  }
  if (code === "repo-not-served") {
    if (session.everJoined) { endSession(text); return; }
    session.code = "";
    session.pending = "";
    toast(text);
    renderStart();
    return;
  }
  if (code === "paused" && session.everJoined) {
    // Joining by code is paused on the relay for a while: try again later.
    toast(text);
    if (ws) ws.close();
    return;
  }
  if (code === "bad-code" || code === "too-many-tries" || code === "paused") {
    if (session.everJoined) endSession(WORDS.codeGone);
    else {
      // The connection stays open: the person can type the code again.
      session.code = "";
      toast(text);
      renderStart();
    }
    return;
  }
  toast(text);
}

// ---------------------------------------------------------------------------
// Yjs updates and awareness on the wire.
// ---------------------------------------------------------------------------
doc.on("update", (update, origin) => {
  if (origin === FROM_RELAY) return;
  if (session.role !== "member") return; // a guest's text never changes locally
  if (!sendBinary(TAG_UPDATE, update)) outbox.push(update);
  else keepRecent(update);
});
function keepRecent(update) {
  const now = Date.now();
  recentUpdates = recentUpdates.filter((u) => now - u.at < RESEND_WINDOW);
  recentUpdates.push({ at: now, update });
}
// resendRecent sends this page's recent updates again, merged into one,
// after the relay dropped one of them for coming too fast.
function resendRecent() {
  clearTimeout(resendTimer);
  resendTimer = setTimeout(() => {
    const now = Date.now();
    const recent = recentUpdates.filter((u) => now - u.at < RESEND_WINDOW * 2).map((u) => u.update);
    recentUpdates = [];
    if (recent.length === 0 || session.role !== "member") return;
    const merged = Y.mergeUpdates(recent);
    if (sendBinary(TAG_UPDATE, merged)) keepRecent(merged);
    else outbox.push(merged);
  }, 1500);
}

function shareAwareness() {
  if (awareness.getLocalState() === null) return;
  sendBinary(TAG_AWARENESS, encodeAwarenessUpdate(awareness, [doc.clientID]));
}
awareness.on("update", ({ added, updated, removed }, origin) => {
  if (origin === FROM_RELAY) return;
  const changed = added.concat(updated, removed);
  if (!changed.includes(doc.clientID)) return;
  sendBinary(TAG_AWARENESS, encodeAwarenessUpdate(awareness, [doc.clientID]));
});

// A remote awareness update is decoded, cleaned and encoded again before it
// is applied, so the editor never renders what another page chose for a
// colour or a name (security review F7). The encoding is y-protocols': a
// count, then per client its id, its clock and its state as JSON, numbers as
// lib0 variable-length unsigned integers and the JSON as a length and UTF-8.
function readVarUint(bytes, at) {
  let num = 0;
  let mult = 1;
  for (;;) {
    if (at.i >= bytes.length || mult > 2 ** 49) throw new Error("awareness");
    const b = bytes[at.i++];
    num += (b & 0x7f) * mult;
    if (b < 0x80) return num;
    mult *= 128;
  }
}
function writeVarUint(out, num) {
  while (num > 0x7f) {
    out.push(0x80 | (num & 0x7f));
    num = Math.floor(num / 128);
  }
  out.push(num);
}
function cleanAwarenessUpdate(bytes) {
  const at = { i: 0 };
  const out = [];
  const count = readVarUint(bytes, at);
  writeVarUint(out, count);
  for (let n = 0; n < count; n++) {
    const client = readVarUint(bytes, at);
    const clock = readVarUint(bytes, at);
    const len = readVarUint(bytes, at);
    if (at.i + len > bytes.length) throw new Error("awareness");
    const json = new TextDecoder("utf-8", { fatal: true }).decode(bytes.subarray(at.i, at.i + len));
    at.i += len;
    const clean = new TextEncoder().encode(JSON.stringify(cleanAwarenessState(client, JSON.parse(json))));
    writeVarUint(out, client);
    writeVarUint(out, clock);
    writeVarUint(out, clean.length);
    for (const b of clean) out.push(b);
  }
  return Uint8Array.from(out);
}
// cleanAwarenessState keeps what the editor reads, a user and a cursor: the
// colours only when each is a six-digit hex colour (otherwise the palette's,
// or the editor's own derived light colour), and the name cleaned.
function cleanAwarenessState(client, state) {
  if (state === null || typeof state !== "object" || Array.isArray(state)) return null;
  const u = state.user !== null && typeof state.user === "object" ? state.user : {};
  const user = { name: cleanName(u.name), color: HEX_COLOUR_RE.test(u.color) ? u.color : COLOURS[client % COLOURS.length] };
  if (HEX_COLOUR_RE.test(u.colorLight)) user.colorLight = u.colorLight;
  const c = state.cursor !== null && typeof state.cursor === "object" ? state.cursor : {};
  const anchor = cleanPosition(c.anchor);
  const head = cleanPosition(c.head);
  return { user, cursor: anchor && head ? { anchor, head } : null };
}
// cleanPosition keeps a Yjs relative position's own fields, of their types.
function cleanPosition(p) {
  if (p === null || typeof p !== "object") return null;
  const id = (v) => (v !== null && typeof v === "object" && Number.isSafeInteger(v.client) && v.client >= 0 &&
    Number.isSafeInteger(v.clock) && v.clock >= 0 ? { client: v.client, clock: v.clock } : null);
  const out = {};
  if (p.item !== undefined) { out.item = id(p.item); if (!out.item) return null; }
  if (p.type !== undefined) { out.type = id(p.type); if (!out.type) return null; }
  if (p.tname !== undefined) { if (typeof p.tname !== "string") return null; out.tname = p.tname; }
  if (p.assoc !== undefined) { if (!Number.isSafeInteger(p.assoc)) return null; out.assoc = p.assoc; }
  return out;
}
// cleanName is a participant's name as the editor shows it: no control or
// direction characters, at most 40 characters.
function cleanName(name) {
  const s = (typeof name === "string" ? name : "").replace(/[\u0000-\u001f\u007f-\u009f\u200e\u200f\u2028-\u202e\u2066-\u2069]/g, "").trim();
  return [...s].slice(0, 40).join("") || "Guest";
}
window.addEventListener("pagehide", () => {
  removeAwarenessStates(awareness, [doc.clientID], "window unload");
});
function setAwarenessUser() {
  const name = session.role === "member" && session.login ? session.login : (session.name || "Guest");
  awareness.setLocalStateField("user", { name, color: COLOURS[doc.clientID % COLOURS.length] });
}

// ---------------------------------------------------------------------------
// Sign-in: GitHub's authorize page with PKCE, then the relay's exchange; or a
// pasted personal access token. Either way the token stays in memory.
// ---------------------------------------------------------------------------
function b64url(bytes) {
  let s = "";
  for (const b of bytes) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
function randomText(n) {
  return b64url(crypto.getRandomValues(new Uint8Array(n)));
}
function isLoopbackOrigin(origin) {
  try {
    const h = new URL(origin).hostname;
    return h === "localhost" || h === "127.0.0.1";
  } catch { return false; }
}

async function relayConfig() {
  const res = await fetch(session.relay + "/config", {
    method: "GET", credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
  });
  if (!res.ok) throw new Error(WORDS.cannotConnect);
  const cfg = await res.json();
  const clientID = typeof cfg.client_id === "string" ? cfg.client_id : "";
  let authorize = typeof cfg.authorize_url === "string" ? cfg.authorize_url : "";
  // Sign-in goes to github.com alone; a loopback relay may name a local
  // stand-in for testing.
  if (authorize !== GITHUB_AUTHORIZE && !(isLoopbackOrigin(session.relay) && isLoopbackOrigin(authorize))) authorize = "";
  return {
    signIn: cfg.sign_in === true && /^[A-Za-z0-9._-]{1,100}$/.test(clientID) && authorize !== "",
    clientID,
    authorize,
    // The relay says whether its sign-in app is a GitHub App; a relay that
    // does not say is taken to run an OAuth app.
    appType: cfg.app_type === "github-app" ? "github-app" : "oauth-app",
  };
}

async function startSignIn() {
  if (!readRelayInput()) return;
  if (!session.everJoined) {
    session.code = normalizeCode(ui.code.value);
    session.name = ui.name.value.trim().slice(0, 40);
  }
  // No sign-in starts through a relay the repository does not name, or, where
  // its list cannot be read, one the person has not confirmed.
  if (!relayClearedFor("signin")) return;
  let cfg;
  try { cfg = await relayConfig(); } catch { toast(WORDS.cannotConnect); return; }
  showSignInNote(cfg);
  if (!cfg.signIn) {
    toast(WORDS.signInUnavailable);
    showTokenForm(true);
    return;
  }
  const state = randomText(16);
  const verifier = randomText(32);
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  const pending = { state, verifier, relay: session.relay, code: session.code, name: session.name };
  if (!stashSignIn(pending)) {
    toast(WORDS.signInFailed);
    showTokenForm(true);
    return;
  }
  const q = new URLSearchParams({
    client_id: cfg.clientID,
    redirect_uri: pageURL(),
    state,
    code_challenge: b64url(new Uint8Array(digest)),
    code_challenge_method: "S256",
    allow_signup: "false",
  });
  // A GitHub App takes no scope: its user token can do only what both the app
  // (its installations and fine-grained permissions) and the person may, and
  // GitHub recommends Apps over OAuth apps
  // (docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app,
  // docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps).
  // An OAuth app needs the repo scope to open the pull request, which reaches
  // every repository the person can.
  if (cfg.appType !== "github-app") q.set("scope", SIGN_IN_SCOPE);
  location.assign(cfg.authorize + "?" + q.toString());
}

// finishSignIn handles GitHub's redirect back to this page.
async function finishSignIn() {
  const q = new URLSearchParams(location.search);
  if (!q.has("state") || (!q.has("code") && !q.has("error"))) return false;
  const code = q.get("code") || "";
  const state = q.get("state") || "";
  const pending = takeSignIn();
  // The code and state come off the address bar before anything else.
  history.replaceState(null, "", location.pathname + location.hash);
  if (!pending || typeof pending.state !== "string" || pending.state !== state || !code) {
    toast(WORDS.signInFailed);
    return true;
  }
  session.relay = relayOrigin(pending.relay);
  session.code = normalizeCode(pending.code);
  session.name = typeof pending.name === "string" ? pending.name : "";
  if (session.code) updateFragment();
  if (!session.relay) { toast(WORDS.signInFailed); return true; }
  // GitHub's code goes to the relay's exchange only while the relay is still
  // cleared for this repository.
  if (!relayCleared()) { toast(refusalWords(session.relay)); renderRelay(); return true; }
  setStatus(WORDS.signingIn, false);
  try {
    const res = await fetch(session.relay + "/auth/token", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code, code_verifier: pending.verifier, redirect_uri: pageURL() }),
      credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
    });
    const out = await res.json().catch(() => ({}));
    if (!res.ok || typeof out.access_token !== "string" || out.access_token === "") throw new Error("exchange");
    heldToken = out.access_token;
  } catch {
    setStatus("", false);
    toast(WORDS.signInFailed);
    return true;
  }
  // Signed in from inside a session: back in as the same participant.
  takeRejoin();
  await presentHeldToken(false);
  return true;
}

function useToken(value) {
  const t = String(value || "").trim();
  if (!t) return;
  if (!readRelayInput()) return;
  heldToken = t;
  presentHeldToken(false);
}

// presentHeldToken gives the held token to the relay, once the relay is
// cleared. With a token, the page first asks api.github.com alone for the
// repository's relay list, which a private repository shows only then: a list
// that does not name the relay refuses it, and forgets any confirmation.
async function presentHeldToken(checked) {
  if (!heldToken) return;
  const relay = session.relay;
  if (!checked && anchor.state !== "listed") {
    const seen = await readRelayList(heldToken);
    if (seen.state === "listed") anchor = seen;
  }
  if (!heldToken || relay !== session.relay) { heldToken = ""; return; }
  const standing = relayStanding(relay);
  if (standing === "refused") {
    forgetRelay(relay);
    heldToken = "";
    toast(refusalWords(relay));
    renderRelay();
    return;
  }
  if (standing === "unconfirmed") { askTrust("token"); return; }
  if (standing !== "listed" && standing !== "confirmed") { heldToken = ""; return; }
  githubToken = heldToken;
  heldToken = "";
  connectSignedIn();
}

// connectSignedIn presents the token: on the open connection when there is
// one, or on a new one.
function connectSignedIn() {
  if (!githubToken || !relayCleared()) { githubToken = ""; return; }
  setStatus(WORDS.signingIn, false);
  if (socketOpen()) {
    session.verifying = !session.inRoom;
    sendJSON(session.inRoom ? { type: "verify", token: githubToken } : { type: "verify", token: githubToken, repo: route.slug });
  } else {
    openSocket();
  }
  renderAll();
}

// ---------------------------------------------------------------------------
// Which relays may have a token (security review F1). A link can name any
// relay in its fragment, so the repository is the trust anchor: it names its
// relays in a committed file the page reads from api.github.com, without a
// token for a public repository. Where that file cannot be read, the person
// confirms the relay before any sign-in reaches it.
// ---------------------------------------------------------------------------
// The anchor: "checking" until GitHub answers, "listed" with the relays the
// file names (none, for a file that names none the page can use), or
// "unknown" when the file could not be read.
let anchor = { state: "checking", relays: [] };

// readRelayList reads the repository's relay list from its default branch,
// with the token given, or none.
async function readRelayList(token) {
  const headers = { Accept: "application/vnd.github+json" };
  if (token) headers.Authorization = "Bearer " + token;
  const stop = new AbortController();
  const timer = setTimeout(() => stop.abort(), 10000);
  try {
    const res = await fetch(GITHUB_API + repoPath() + "/contents/" + RELAYS_FILE, {
      headers, signal: stop.signal, credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
    });
    if (res.status !== 200) return { state: "unknown", relays: [] };
    return { state: "listed", relays: relaysIn(await res.json()) };
  } catch {
    return { state: "unknown", relays: [] };
  } finally {
    clearTimeout(timer);
  }
}

// relaysIn reads the relays a contents answer names: each exactly an origin,
// https (or http on this computer, as the page takes a local relay). A file
// that does not parse names none, so it refuses every relay.
function relaysIn(body) {
  try {
    if (!body || body.type !== "file" || body.encoding !== "base64" || typeof body.content !== "string" || body.content.length > 65536) return [];
    const list = JSON.parse(decodeBlob(body.content));
    if (!list || !Array.isArray(list.relays)) return [];
    const out = [];
    for (const r of list.relays.slice(0, 32)) {
      if (typeof r !== "string") continue;
      let u;
      try { u = new URL(r); } catch { continue; }
      const origin = relayOrigin(r);
      if (origin && origin === u.origin && u.pathname === "/" && !u.search && !u.hash) out.push(origin);
    }
    return out;
  } catch { return []; }
}

// relayStanding is what the page may do with a relay: "listed" or "confirmed"
// may have a token, "unconfirmed" must be confirmed first, "refused" gets
// nothing, "checking" waits for GitHub's answer.
function relayStanding(relay) {
  if (!relay) return "";
  if (anchor.state === "checking") return "checking";
  if (anchor.state === "listed") return anchor.relays.includes(relay) ? "listed" : "refused";
  return confirmedRelays().includes(relay) ? "confirmed" : "unconfirmed";
}
function relayCleared() {
  const standing = relayStanding(session.relay);
  return standing === "listed" || standing === "confirmed";
}
function refusalWords(relay) {
  return (anchor.state === "listed" && anchor.relays.length === 0 ? WORDS.relayNoneUsable : WORDS.relayRefused) + " " + relay;
}

// relayClearedFor reports whether the relay is cleared; when it is not, it
// asks the person to confirm it (for a relay the list cannot speak for) or
// says why it is refused.
function relayClearedFor(action) {
  const standing = relayStanding(session.relay);
  if (standing === "listed" || standing === "confirmed") return true;
  if (standing === "unconfirmed") askTrust(action);
  else {
    heldToken = "";
    toast(standing === "checking" ? WORDS.relayChecking : refusalWords(session.relay));
  }
  renderRelay();
  return false;
}

let trustPending = null;
function askTrust(action) {
  trustPending = { action, relay: session.relay };
  ui.trustOrigin.textContent = session.relay;
  ui.trust.hidden = false;
  ui.trustCancel.focus();
}
function trustAnswered(yes) {
  const p = trustPending;
  trustPending = null;
  ui.trust.hidden = true;
  if (!p) return;
  if (!yes || p.relay !== session.relay || relayStanding(p.relay) !== "unconfirmed") {
    heldToken = "";
    renderRelay();
    return;
  }
  const relay = p.relay;
  confirmRelay(relay);
  if (relayCleared()) remember(KEY_RELAY, relay);
  renderRelay();
  if (p.action === "signin") startSignIn();
  else if (p.action === "token") presentHeldToken(true);
}

// renderRelay shows where the relay stands, with its origin, on the relay
// card, which is never hidden for a relay that is not cleared.
function renderRelay() {
  if (!ui.relayStanding) return;
  const relay = session.everJoined ? session.relay : relayOrigin(ui.relay.value);
  const standing = relayStanding(relay);
  const words = {
    checking: WORDS.relayChecking, listed: WORDS.relayListed, confirmed: WORDS.relayConfirmed,
    unconfirmed: WORDS.relayUnconfirmed, refused: relay ? refusalWords("").trim() : "",
  }[standing] || "";
  ui.relayStandingText.textContent = words;
  ui.relayStandingOrigin.textContent = standing && standing !== "checking" && standing !== "listed" ? relay : "";
  ui.relayStanding.className = standing === "refused" ? "standing bad" : "standing";
  ui.relayStanding.hidden = !words;
  if (!relayCleared()) ui.relayCard.hidden = false;
}

// showSignInNote says, for a relay that signs in through an OAuth app, that
// a GitHub App would be narrower.
function showSignInNote(cfg) {
  ui.signInNote.textContent = cfg && cfg.signIn && cfg.appType !== "github-app" ? WORDS.oauthNote : "";
}
async function refreshSignInNote() {
  showSignInNote(null);
  if (!relayCleared()) return;
  try { showSignInNote(await relayConfig()); } catch { /* the note waits for sign-in */ }
}

// ---------------------------------------------------------------------------
// Hosting: read the brief through GitHub's API, seed the document once, then
// show the invite.
// ---------------------------------------------------------------------------
async function gh(path) {
  const res = await fetch(GITHUB_API + path, {
    headers: { Accept: "application/vnd.github+json", Authorization: "Bearer " + githubToken },
    credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
  });
  if (!res.ok) throw new Error(WORDS.githubFailed + " (" + res.status + ").");
  return res.json();
}
const SHA_RE = /^[0-9a-f]{40,64}$/;
function repoPath() {
  return "/repos/" + encodeURIComponent(route.owner) + "/" + encodeURIComponent(route.repo);
}
async function subtree(sha, name, recursive) {
  if (!SHA_RE.test(sha)) throw new Error(WORDS.noBrief);
  const t = await gh(repoPath() + "/git/trees/" + sha + (recursive ? "?recursive=1" : ""));
  if (!t || !Array.isArray(t.tree)) throw new Error(WORDS.noBrief);
  if (recursive && t.truncated) throw new Error(WORDS.noBrief);
  if (name === null) return t;
  const entry = t.tree.find((e) => e && e.path === name && e.type === "tree");
  if (!entry) throw new Error(WORDS.noBrief);
  return entry.sha;
}
function decodeBlob(b64) {
  const bin = atob(String(b64).replace(/\s/g, ""));
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}
async function loadBrief() {
  const repo = await gh(repoPath());
  const branch = typeof repo.default_branch === "string" ? repo.default_branch : "";
  if (!branch) throw new Error(WORDS.noBrief);
  const head = await gh(repoPath() + "/branches/" + encodeURIComponent(branch));
  const sha = head && head.commit && head.commit.sha;
  const rootTree = head && head.commit && head.commit.commit && head.commit.commit.tree && head.commit.commit.tree.sha;
  if (!SHA_RE.test(sha || "") || !SHA_RE.test(rootTree || "")) throw new Error(WORDS.noBrief);
  let treeSha = rootTree;
  for (const name of BRIEF_DIR) treeSha = await subtree(treeSha, name, false);
  const brief = await subtree(treeSha, null, true);
  const blobs = brief.tree.filter((e) => e && e.type === "blob" && typeof e.path === "string" && e.path.endsWith(".md") && SHA_RE.test(e.sha || ""));
  if (blobs.length === 0) throw new Error(WORDS.noBrief);
  const out = new Array(blobs.length);
  let next = 0;
  async function worker() {
    while (next < blobs.length) {
      const i = next++;
      const b = await gh(repoPath() + "/git/blobs/" + blobs[i].sha);
      if (!b || b.encoding !== "base64") throw new Error(WORDS.noBrief);
      out[i] = { path: BRIEF_PREFIX + blobs[i].path, text: decodeBlob(b.content) };
    }
  }
  await Promise.all([worker(), worker(), worker(), worker(), worker(), worker()]);
  out.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));
  return { sha, branch, files: out };
}

function utf8Length(s) {
  return new TextEncoder().encode(s).length;
}
function pieces(text, size) {
  const out = [];
  let i = 0;
  while (i < text.length) {
    let end = Math.min(text.length, i + size);
    // Never split a surrogate pair.
    if (end < text.length && /[\uD800-\uDBFF]/.test(text[end - 1])) end -= 1;
    out.push(text.slice(i, end));
    i = end;
  }
  return out;
}

// seed fills the document from the brief, exactly once. Each transaction is
// one update the relay logs, kept under its message size.
function seed(brief, mode) {
  if (files.size > 0 || meta.has("base_sha")) return;
  let batch = [];
  let bytes = 0;
  const flush = () => {
    if (batch.length === 0) return;
    const work = batch;
    batch = [];
    bytes = 0;
    doc.transact(() => { for (const f of work) files.set(f.path, new Y.Text(f.text)); }, FROM_SEED);
  };
  for (const f of brief.files) {
    const size = utf8Length(f.text);
    if (size <= SEED_BUDGET) {
      if (bytes + size > SEED_BUDGET) flush();
      batch.push(f);
      bytes += size;
      continue;
    }
    flush();
    const text = new Y.Text();
    doc.transact(() => { files.set(f.path, text); }, FROM_SEED);
    for (const p of pieces(f.text, SEED_PIECE)) doc.transact(() => { text.insert(text.length, p); }, FROM_SEED);
  }
  flush();
  const base = Y.encodeSnapshot(Y.snapshot(doc));
  doc.transact(() => {
    meta.set("repo", route.slug);
    meta.set("base_sha", brief.sha);
    meta.set("branch", brief.branch);
    meta.set("mode", mode === "issues" ? "issues" : "pr");
    meta.set("base_snapshot", base);
  }, FROM_SEED);
}

async function drained() {
  for (let i = 0; i < 600 && socketOpen() && ws.bufferedAmount > 0; i++) {
    await new Promise((r) => setTimeout(r, 50));
  }
}

async function hostBrief() {
  setStatus(WORDS.loadingBrief, false);
  const brief = await loadBrief();
  setStatus(WORDS.sendingBrief, false);
  seed(brief, ui.mode ? ui.mode.value : "pr");
  trackDeletions();
  await drained();
  session.seeding = false;
  session.synced = true;
  updateFragment();
  setStatus(WORDS.connected, false);
  renderAll();
}

function inviteURL() {
  return location.origin + route.base + "/" + encodeURIComponent(route.owner) + "/" + encodeURIComponent(route.repo) +
    "/brief/#relay=" + encodeURIComponent(session.relay) + "&code=" + session.code;
}
function updateFragment() {
  if (!session.code || !session.relay) return;
  history.replaceState(null, "", location.pathname + "#relay=" + encodeURIComponent(session.relay) + "&code=" + session.code);
}

// ---------------------------------------------------------------------------
// Guests' suggestions, and the members' answers to them.
// ---------------------------------------------------------------------------
function quoteOf(text, from, to) {
  return {
    exact: text.slice(from, to),
    prefix: text.slice(Math.max(0, from - QUOTE_CONTEXT), from),
    suffix: text.slice(to, to + QUOTE_CONTEXT),
  };
}

function sendSuggestion(replacement) {
  if (!view || !currentPath) return;
  const ytext = files.get(currentPath);
  const sel = view.state.selection.main;
  if (!(ytext instanceof Y.Text) || sel.empty) { toast(WORDS.selectFirst); return; }
  if (replacement.length > MAX_REPLACEMENT) return;
  const text = ytext.toString();
  const body = {
    path: currentPath,
    anchor: Y.relativePositionToJSON(Y.createRelativePositionFromTypeIndex(ytext, sel.from)),
    head: Y.relativePositionToJSON(Y.createRelativePositionFromTypeIndex(ytext, sel.to)),
    quote: quoteOf(text, sel.from, sel.to),
    replacement,
  };
  awaitingAck.push(body);
  sendJSON({ type: "suggest", body });
}

function onSuggested(msg) {
  const body = awaitingAck.shift();
  if (!body) return;
  mine.push({ id: String(msg.id || ""), body });
  toast(msg.delivered > 0 ? WORDS.suggestionSent : WORDS.noMembers);
  renderSide();
}

function validSuggestion(b) {
  return b && typeof b === "object" && typeof b.path === "string" && typeof b.replacement === "string" &&
    b.replacement.length <= MAX_REPLACEMENT && b.quote && typeof b.quote === "object" &&
    typeof b.quote.exact === "string" && b.quote.exact !== "" &&
    typeof (b.quote.prefix ?? "") === "string" && typeof (b.quote.suffix ?? "") === "string";
}

// storedSuggestion keeps the fields a suggestion is made of, and nothing else.
function storedSuggestion(id, from, b) {
  const quote = { exact: b.quote.exact, prefix: b.quote.prefix || "", suffix: b.quote.suffix || "" };
  return { id, from, body: { path: b.path, anchor: b.anchor ?? null, head: b.head ?? null, quote, replacement: b.replacement } };
}

function onSuggest(msg) {
  if (session.role !== "member") return;
  const id = String(msg.id || "");
  if (!id || !validSuggestion(msg.body)) return;
  const from = typeof msg.from === "string" && msg.from ? msg.from : "Guest";
  // Every member's page sees the suggestion; the first to record it wins, and
  // the others find it there already.
  if (!suggestions.has(id) && !suggestionsDone.has(id)) {
    doc.transact(() => { suggestions.set(id, storedSuggestion(id, from, msg.body)); });
  }
  renderSide();
}

// findQuote places a W3C TextQuoteSelector: the occurrence of exact whose
// surroundings match the prefix and suffix best, or -1 when that is not one.
function findQuote(text, q) {
  const at = [];
  for (let i = text.indexOf(q.exact); i !== -1; i = text.indexOf(q.exact, i + 1)) at.push(i);
  if (at.length === 0) return -1;
  if (at.length === 1) return at[0];
  const prefix = q.prefix || "";
  const suffix = q.suffix || "";
  const score = (i) => {
    let s = 0;
    const before = text.slice(Math.max(0, i - prefix.length), i);
    const after = text.slice(i + q.exact.length, i + q.exact.length + suffix.length);
    while (s < prefix.length && before[before.length - 1 - s] === prefix[prefix.length - 1 - s]) s++;
    let t = 0;
    while (t < suffix.length && after[t] === suffix[t]) t++;
    return s + t;
  };
  const scored = at.map((i) => [score(i), i]).sort((a, b) => b[0] - a[0]);
  return scored[0][0] > scored[1][0] ? scored[0][1] : -1;
}

// locate finds the suggested passage now: its relative positions first, then
// its quote; null when neither still points at the quoted text.
function locate(body) {
  const ytext = files.get(body.path);
  if (!(ytext instanceof Y.Text)) return null;
  const text = ytext.toString();
  try {
    const a = Y.createAbsolutePositionFromRelativePosition(Y.createRelativePositionFromJSON(body.anchor), doc);
    const h = Y.createAbsolutePositionFromRelativePosition(Y.createRelativePositionFromJSON(body.head), doc);
    if (a && h && a.type === ytext && h.type === ytext) {
      const from = Math.min(a.index, h.index);
      const to = Math.max(a.index, h.index);
      if (text.slice(from, to) === body.quote.exact) return { ytext, from, to };
    }
  } catch { /* fall through to the quote */ }
  const at = findQuote(text, body.quote);
  return at < 0 ? null : { ytext, from: at, to: at + body.quote.exact.length };
}

function applySuggestion(id) {
  const s = suggestions.get(id);
  if (!s || !validSuggestion(s.body) || session.role !== "member" || suggestionsDone.has(id)) return;
  const place = locate(s.body);
  if (!place) { toast(WORDS.cannotApply); return; }
  const replacement = s.body.replacement;
  doc.transact((tr) => {
    const before = place.ytext.toString().slice(place.from, place.to);
    place.ytext.delete(place.from, place.to - place.from);
    // The ids of the text the approval removed and inserted, so the credit
    // can tell an approved change from the approver's own typing.
    const del = [];
    tr.deleteSet.clients.forEach((items, client) => { for (const d of items) del.push([client, d.clock, d.len]); });
    const clock = Y.getState(doc.store, doc.clientID);
    place.ytext.insert(place.from, replacement);
    approvals.push([{
      id,
      path: s.body.path,
      guest: typeof s.from === "string" ? s.from : "Guest",
      approver: session.login,
      client: doc.clientID,
      ins: replacement ? [doc.clientID, clock, replacement.length] : null,
      del,
      before,
      after: replacement,
      at: new Date().toISOString(),
    }]);
    suggestionsDone.set(id, "applied");
  });
  renderSide();
}

function declineSuggestion(id) {
  if (session.role !== "member") return;
  if (!suggestionsDone.has(id)) doc.transact(() => { suggestionsDone.set(id, "declined"); });
  renderSide();
}
suggestions.observe(() => renderSide());
suggestionsDone.observe(() => renderSide());

// ---------------------------------------------------------------------------
// Credit: what changed since the session began, and who changed it.
//
// The base is the snapshot the host took right after seeding. Y.Text's
// toDelta(now, base) marks each piece of text added or removed since then,
// with the Yjs item it belongs to; the item's client id names who inserted
// it, and the members' recorded deletions name who removed it. The relay's
// roster maps each member's client ids to their GitHub login.
// ---------------------------------------------------------------------------
let deletionsTracked = false;
function trackDeletions() {
  if (deletionsTracked) return;
  deletionsTracked = true;
  deletions.setUserMapping(doc, doc.clientID, String(doc.clientID), { filter: (tr) => touchesBrief(tr) });
}
function touchesBrief(tr) {
  const texts = new Set(files.values());
  for (const type of tr.changed.keys()) if (texts.has(type)) return true;
  return false;
}

let baseCache = { raw: null, snap: null };
function baseSnapshot() {
  const raw = meta.get("base_snapshot");
  if (!(raw instanceof Uint8Array)) return null;
  if (baseCache.raw !== raw) {
    try { baseCache = { raw, snap: Y.decodeSnapshot(raw) }; } catch { return null; }
  }
  return baseCache.snap;
}

// creditedTo is the login the roster credits a Yjs client id to, or "" for
// one it does not, or one an unattributed update carried.
function creditedTo(client) {
  if (unattributed.has(Number(client))) return "";
  const login = session.credits[String(client)];
  return typeof login === "string" ? login : "";
}
function loginOf(client) {
  const login = creditedTo(client);
  return NAME_RE.test(login) ? login : "";
}

// verifiedApprovals is the approvals each written by its own approver: the
// Yjs client that wrote the entry is the one it names, credited to the login
// it names, and the text it says it inserted is that client's. Any member's
// page can push to the array, so an entry naming someone else counts for
// nobody (security review F6).
function verifiedApprovals() {
  const out = [];
  for (let item = approvals._start; item !== null; item = item.right) {
    if (item.deleted) continue;
    const writer = item.id.client;
    for (const a of item.content.getContent()) {
      if (!a || typeof a !== "object" || a.client !== writer || !loginOf(writer) || loginOf(writer) !== a.approver) continue;
      if (a.ins !== null && !(Array.isArray(a.ins) && a.ins[0] === writer)) continue;
      out.push(a);
    }
  }
  return out;
}

// verifiedDeleters is the members' deletion records each written by the
// member it is filed under: the record's key is a client id, and every Yjs
// client that wrote the record's entry and its contents must be credited to
// that client's login.
function verifiedDeleters() {
  const out = [];
  const users = doc.getMap("deletions");
  for (const [desc, ds] of deletions.dss) {
    const login = loginOf(desc);
    const entry = users._map.get(desc);
    const user = users.get(desc);
    if (!login || !entry || !(user instanceof Y.Map)) continue;
    const writers = [entry.id.client];
    for (const key of ["ids", "ds"]) {
      const list = user.get(key);
      if (!(list instanceof Y.Array)) continue;
      for (let item = list._start; item !== null; item = item.right) writers.push(item.id.client);
    }
    if (writers.every((c) => loginOf(c) === login)) out.push([desc, ds]);
  }
  return out;
}

// changesOf lists a file's text as runs: "same" since the base, "add" or
// "del", each changed run one Yjs item with its client and clock.
function changesOf(ytext, base) {
  const delta = ytext.toDelta(Y.snapshot(doc), base, (type, id) => ({ type, client: id.client, clock: id.clock }));
  const ops = [];
  for (const op of delta) {
    if (typeof op.insert !== "string") continue;
    const ch = op.attributes && op.attributes.ychange;
    if (!ch) ops.push({ kind: "same", text: op.insert });
    else ops.push({ kind: ch.type === "added" ? "add" : "del", text: op.insert, client: ch.client, clock: ch.clock });
  }
  return ops;
}

function newlines(s) {
  let n = 0;
  for (let i = s.indexOf("\n"); i !== -1; i = s.indexOf("\n", i + 1)) n++;
  return n;
}

function overlaps(clock, len, from, n) {
  return clock < from + n && from < clock + len;
}

// approvalKey tells approvals apart: each approved guest change by its own
// id, so two changes by the same guest and approver stay two.
function approvalKey(a) {
  return typeof a.id === "string" && a.id ? a.id : a;
}

// approvalOf names the approval a changed run belongs to, if any.
function approvalOf(path, op, list) {
  for (const a of list) {
    if (!a || a.path !== path) continue;
    if (op.kind === "add" && Array.isArray(a.ins) && a.ins[0] === op.client && overlaps(op.clock, op.text.length, a.ins[1], a.ins[2])) return a;
    if (op.kind === "del" && Array.isArray(a.del) && a.del.some((d) => Array.isArray(d) && d[0] === op.client && overlaps(op.clock, op.text.length, d[1], d[2]))) return a;
  }
  return null;
}

// deletersOf names the members whose pages recorded deleting a removed run.
function deletersOf(op) {
  const out = [];
  for (const [client, ds] of verifiedDeleters()) {
    const ranges = ds.clients.get(op.client) || [];
    if (ranges.some((r) => overlaps(op.clock, op.text.length, r.clock, r.len))) out.push(client);
  }
  return out;
}

// makePassage turns the changed runs ops[i..j] into a passage: whole lines
// before and after, where they sit in the base, and who to credit.
function makePassage(path, ops, offsets, i, j, approvalList) {
  const prev = i > 0 ? ops[i - 1].text : "";
  const left = prev.slice(prev.lastIndexOf("\n") + 1);
  let before = left;
  let after = left;
  const authors = new Set();
  const approved = new Map();
  let unknown = false;
  for (let k = i; k <= j; k++) {
    const op = ops[k];
    if (op.kind !== "del") after += op.text;
    if (op.kind !== "add") before += op.text;
    if (op.kind === "same") continue;
    const a = approvalOf(path, op, approvalList);
    if (a) {
      const approver = loginOf(a.client) || (typeof a.approver === "string" ? a.approver : "");
      const guest = trailerText(a.guest);
      approved.set(approvalKey(a), { key: approvalKey(a), guest, approver });
      continue;
    }
    const who = op.kind === "add" ? [op.client] : deletersOf(op);
    let named = false;
    for (const c of who) {
      const login = loginOf(c);
      if (login) { authors.add(login); named = true; }
    }
    if (!named) unknown = true;
  }
  const lineEnded = (s) => s === "" || s.endsWith("\n");
  if (j + 1 < ops.length && !(lineEnded(before) && lineEnded(after))) {
    const next = ops[j + 1].text;
    const cut = next.indexOf("\n");
    const right = cut < 0 ? next : next.slice(0, cut);
    before += right;
    after += right;
  }
  const baseStart = offsets[i].base - left.length;
  return {
    path,
    baseStart,
    baseEnd: baseStart + before.length,
    before,
    after,
    authors: [...authors].sort(),
    approvals: [...approved.values()],
    unknown,
  };
}

// baseFilePaths names the files the session started with: the files map's
// keys as the base snapshot saw them, or null without a base.
function baseFilePaths(base) {
  if (!base) return null;
  try { return new Set(Object.keys(Y.typeMapGetAllSnapshot(files, base))); } catch { return null; }
}

// briefFile says whether a path may go back to GitHub: a Markdown file under
// the brief, with no ".." and no backslash, that the session started with.
// The paths are the shared files map's keys, which any team member's page
// can write, so nothing else is shown or sent (security review F9).
function briefFile(path, baseFiles) {
  return typeof path === "string" && path.startsWith(BRIEF_PREFIX) && path.endsWith(".md") &&
    !path.includes("..") && !path.includes("\\") && baseFiles instanceof Set && baseFiles.has(path);
}

// shownPath is a refused path as the page names it: one line, bounded.
function shownPath(path) {
  return String(path).replace(/[\u0000-\u001f\u007f-\u009f\u2028\u2029]/g, "?").slice(0, 200);
}

// analyse reads every file's changes since the base, as passages. A path
// outside the brief is left out, and named in the result's refused list.
function analyse() {
  const base = baseSnapshot();
  const out = [];
  out.refused = [];
  if (!base) return out;
  const baseFiles = baseFilePaths(base);
  const refused = out.refused;
  const approvalList = verifiedApprovals();
  for (const path of [...files.keys()].sort()) {
    if (!briefFile(path, baseFiles)) {
      refused.push(path);
      continue;
    }
    const ytext = files.get(path);
    if (!(ytext instanceof Y.Text)) continue;
    const ops = changesOf(ytext, base);
    if (!ops.some((op) => op.kind !== "same")) continue;
    const offsets = [];
    let b = 0;
    let c = 0;
    for (const op of ops) {
      offsets.push({ base: b, cur: c });
      if (op.kind !== "add") b += op.text.length;
      if (op.kind !== "del") c += op.text.length;
    }
    const baseText = ops.filter((op) => op.kind !== "add").map((op) => op.text).join("");
    const passages = [];
    let i = 0;
    while (i < ops.length) {
      if (ops[i].kind === "same") { i++; continue; }
      let j = i;
      let k = i + 1;
      for (;;) {
        while (k < ops.length && ops[k].kind !== "same") { j = k; k++; }
        // ops[k] is unchanged text; one more changed run after it, within
        // PASSAGE_JOIN_LINES unchanged lines, joins this passage.
        if (k + 1 < ops.length && newlines(ops[k].text) <= PASSAGE_JOIN_LINES + 1) { j = k + 1; k += 2; continue; }
        break;
      }
      const p = makePassage(path, ops, offsets, i, j, approvalList);
      p.lineFrom = 1 + newlines(baseText.slice(0, p.baseStart));
      p.lineTo = p.lineFrom + Math.max(0, newlines(p.before.replace(/\n$/, "")));
      passages.push(p);
      i = j + 1;
    }
    const current = ops.filter((op) => op.kind !== "del").map((op) => op.text).join("");
    if (current !== baseText) out.push({ path, base: baseText, current, passages });
  }
  return out;
}

function relPath(path) {
  return path.startsWith(BRIEF_PREFIX) ? path.slice(BRIEF_PREFIX.length) : path;
}
function linesLabel(p) {
  return p.lineTo > p.lineFrom ? WORDS.linesAt + " " + p.lineFrom + "–" + p.lineTo : WORDS.lineAt + " " + p.lineFrom;
}

// creditLines says who a passage is credited to, one line each.
function creditLines(p) {
  const lines = [];
  if (p.authors.length) lines.push("Written by " + p.authors.map((l) => "@" + l).join(", "));
  for (const a of p.approvals) lines.push("Suggested by " + a.guest + " (guest), approved by @" + a.approver);
  if (p.unknown && lines.length === 0) lines.push(WORDS.creditUnknown);
  return lines;
}

// ---------------------------------------------------------------------------
// The issues route: one pre-filled GitHub issue link per passage.
// ---------------------------------------------------------------------------
// fenced wraps text in a code fence longer than any run of backticks in it.
function fenced(text) {
  let longest = 0;
  for (const m of text.matchAll(/`+/g)) longest = Math.max(longest, m[0].length);
  const fence = "`".repeat(Math.max(3, longest + 1));
  return fence + "markdown\n" + text + (text === "" || text.endsWith("\n") ? "" : "\n") + fence;
}

// ISSUE_MARKER ends every issue a session sends, so the team's agent can find
// them without a label: a pre-filled label is refused for anyone who may not
// triage the repository, and the repository may not have one.
const ISSUE_MARKER = "Sent from an abcdesign.app brief session.";

function issueTitle(p) {
  return "Brief: " + relPath(p.path) + ", " + linesLabel(p);
}

function issueBody(p) {
  return [
    "A change to the brief, proposed in a live session.",
    "",
    "- File: `" + p.path + "`",
    "- Base commit: " + meta.get("base_sha"),
    "- At the base commit: " + linesLabel(p),
    "",
    "Before:",
    "",
    fenced(p.before),
    "",
    "After:",
    "",
    fenced(p.after),
    "",
    ...creditLines(p),
    "",
    ISSUE_MARKER,
  ].join("\n");
}

// issueLink builds the passage's pre-filled issue link and says whether it
// fits: measured whole, as GitHub receives it, after encoding, in bytes.
function issueLink(p) {
  const url = GITHUB_WEB + "/" + encodeURIComponent(route.owner) + "/" + encodeURIComponent(route.repo) +
    "/issues/new?title=" + encodeURIComponent(issueTitle(p)) + "&body=" + encodeURIComponent(issueBody(p));
  return { url, fits: utf8Length(url) <= ISSUE_URL_LIMIT };
}

// ---------------------------------------------------------------------------
// The pull request route: a branch at the base commit, one commit made as the
// submitter, crediting everyone else, and the pull request.
// ---------------------------------------------------------------------------
// utf8Base64 encodes text as GitHub's API wants file contents: the UTF-8
// bytes, in base64.
function utf8Base64(text) {
  const bytes = new TextEncoder().encode(text);
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binary);
}

function branchName() {
  const d = new Date();
  const two = (n) => String(n).padStart(2, "0");
  let tail = "";
  for (const b of crypto.getRandomValues(new Uint8Array(6))) tail += (b % 36).toString(36);
  return "abcdesign/" + d.getUTCFullYear() + two(d.getUTCMonth() + 1) + two(d.getUTCDate()) + "-" + tail;
}

// trailerText makes a guest's display name safe for one trailer line, an
// issue body and a pull request body: no line breaks or angle brackets, and
// "@" written "(at)", so a name cannot mention anyone on GitHub. The page
// itself shows the name as the guest gave it.
function trailerText(s) {
  const t = String(s || "").replace(/[\u0000-\u001f\u007f-\u009f\u2028\u2029<>]/g, " ").replace(/@/g, "(at)").replace(/\s+/g, " ").trim().slice(0, 60);
  return t || "Guest";
}

function noreply(login, ids) {
  return login + " <" + ids.get(login.toLowerCase()) + "+" + login + "@users.noreply.github.com>";
}

// composeTrailers credits everyone but the submitter, who is the commit's
// author: a co-author line per credited member, and for each approved guest
// change the guest's line with its approver's line directly beneath.
function composeTrailers(submitter, members, pairs, ids) {
  const lines = [];
  for (const login of members) {
    if (login.toLowerCase() !== submitter.toLowerCase()) lines.push("Co-authored-by: " + noreply(login, ids));
  }
  for (const p of pairs) lines.push("Suggested-by: " + p.guest + " (guest)", "Approved-by: " + noreply(p.approver, ids));
  return lines;
}

// applyPassages puts some of a file's passages onto its base text.
function applyPassages(base, list) {
  let text = base;
  for (const p of [...list].sort((a, b) => b.baseStart - a.baseStart)) {
    text = text.slice(0, p.baseStart) + p.after + text.slice(p.baseEnd);
  }
  return text;
}

async function ghSend(method, path, body) {
  let res;
  try {
    res = await fetch(GITHUB_API + path, {
      method,
      headers: { Accept: "application/vnd.github+json", "Content-Type": "application/json", Authorization: "Bearer " + githubToken },
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
    });
  } catch {
    return { status: 0, json: null };
  }
  const json = await res.json().catch(() => null);
  return { status: res.status, json };
}

// GitHubTrouble is a failed step, already in words the person can act on.
class GitHubTrouble extends Error {}

function trouble(step, status, errors) {
  const types = Array.isArray(errors) ? errors.map((e) => (e && typeof e.type === "string" ? e.type : "")) : [];
  if (status === 0) return new GitHubTrouble(WORDS.ghUnreachable);
  if (status === 401) return new GitHubTrouble(WORDS.ghSignIn);
  if (status === 403 || types.includes("FORBIDDEN") || types.includes("INSUFFICIENT_SCOPES")) return new GitHubTrouble(WORDS.ghScope);
  if (status === 404 || types.includes("NOT_FOUND")) return new GitHubTrouble(WORDS.ghNotFound);
  if (step === "commit" && (status === 422 || types.includes("STALE_DATA"))) return new GitHubTrouble(WORDS.ghStale);
  if (step === "ref" && status === 422) return new GitHubTrouble(WORDS.ghRefTaken);
  return new GitHubTrouble(WORDS.ghOther);
}

function prBody(list, members, pairs, changed) {
  const people = new Set(members.map((l) => "@" + l));
  for (const p of pairs) people.add(p.guest + " (guest)");
  for (const r of session.roster) {
    if (r.role === "member" && typeof r.login === "string" && NAME_RE.test(r.login)) people.add("@" + r.login);
    else if (r.role !== "member" && typeof r.name === "string" && r.name) people.add(trailerText(r.name) + " (guest)");
  }
  const out = [
    "Edits to the brief made together in a live session, from base commit " + meta.get("base_sha") + ".",
    "",
    "Participants: " + [...people].join(", "),
    "",
    "Changed passages:",
  ];
  for (const p of list) out.push("- `" + relPath(p.path) + "`, " + linesLabel(p) + ": " + creditLines(p).join("; "));
  if (pairs.length) {
    out.push("", "Approved guest suggestions:");
    for (const p of pairs) out.push("- " + p.guest + " (guest), approved by @" + p.approver);
  }
  out.push("", "Changed files:", ...changed.map((f) => "- `" + f + "`"));
  return out.join("\n");
}

let sending = false;
// openPullRequest sends the session as one pull request: every passage, or
// (one too long for an issue) the single passage shown, read afresh so what
// is sent is what the panel shows.
async function openPullRequest(only) {
  if (sending) return;
  const analysis = analyse();
  const all = analysis.flatMap((f) => f.passages);
  const list = only ? all.filter((p) => p.path === only.path && p.before === only.before && p.after === only.after) : all;
  if (list.length === 0) {
    showSendResult(only ? WORDS.passageMoved : WORDS.sendNothing, true);
    renderSend();
    return;
  }
  if (!githubToken || session.role !== "member" || !session.login) { showSendResult(WORDS.prNeedsSignIn, true); return; }
  const baseSha = meta.get("base_sha");
  const target = meta.get("branch");
  if (!SHA_RE.test(baseSha || "") || typeof target !== "string" || !target) { showSendResult(WORDS.prNoBase, true); return; }
  sending = true;
  renderSend();
  showSendResult(WORDS.prWorking, false);
  let branch = "";
  let refMade = false;
  try {
    // The files: each changed file with these passages put on its base text.
    const additions = [];
    const changed = [];
    const baseFiles = baseFilePaths(baseSnapshot());
    for (const f of analysis) {
      const mine = list.filter((p) => p.path === f.path);
      if (mine.length === 0) continue;
      const text = mine.length === f.passages.length ? f.current : applyPassages(f.base, mine);
      if (text === f.base) continue;
      if (!briefFile(f.path, baseFiles)) throw new GitHubTrouble(WORDS.pathRefused + " " + shownPath(f.path));
      additions.push({ path: f.path, contents: utf8Base64(text) });
      changed.push(f.path);
    }
    if (additions.length === 0) { showSendResult(WORDS.sendNothing, true); return; }

    // Who to credit, and their GitHub ids for the noreply addresses.
    const credited = new Map();
    const pairs = new Map();
    for (const p of list) {
      for (const l of p.authors) credited.set(l.toLowerCase(), l);
      for (const a of p.approvals) {
        if (!NAME_RE.test(a.approver)) continue;
        credited.set(a.approver.toLowerCase(), a.approver);
        const guest = trailerText(a.guest);
        // One pair per approval, though it spans several passages.
        pairs.set(a.key, { guest, approver: a.approver });
      }
    }
    const members = [...credited.values()].sort((a, b) => a.localeCompare(b));
    const ids = new Map();
    for (const login of members) {
      if (login.toLowerCase() === session.login.toLowerCase() && ![...pairs.values()].some((p) => p.approver.toLowerCase() === login.toLowerCase())) continue;
      const r = await ghSend("GET", "/users/" + encodeURIComponent(login));
      if (r.status !== 200 || !r.json || !Number.isSafeInteger(r.json.id)) throw trouble("users", r.status);
      ids.set(login.toLowerCase(), r.json.id);
    }
    const trailers = composeTrailers(session.login, members, [...pairs.values()], ids);

    // 1. The branch, at the commit the session started from.
    branch = branchName();
    const ref = await ghSend("POST", repoPath() + "/git/refs", { ref: "refs/heads/" + branch, sha: baseSha });
    if (ref.status !== 201) throw trouble("ref", ref.status);
    refMade = true;

    // 2. One commit on it, made as the submitter.
    const message = { headline: COMMIT_HEADLINE, body: "Changed files:\n" + changed.map((f) => "- " + f).join("\n") + (trailers.length ? "\n\n" + trailers.join("\n") : "") };
    const commit = await ghSend("POST", "/graphql", {
      query: "mutation($input: CreateCommitOnBranchInput!) { createCommitOnBranch(input: $input) { commit { oid url } } }",
      variables: {
        input: {
          branch: { repositoryNameWithOwner: route.slug, branchName: branch },
          message,
          expectedHeadOid: baseSha,
          fileChanges: { additions },
        },
      },
    });
    const made = commit.json && commit.json.data && commit.json.data.createCommitOnBranch && commit.json.data.createCommitOnBranch.commit;
    if (commit.status !== 200 || !made || (commit.json.errors && commit.json.errors.length)) {
      throw trouble("commit", commit.status === 200 && !made ? 422 : commit.status, commit.json && commit.json.errors);
    }

    // 3. The pull request into the branch the session read.
    const pr = await ghSend("POST", repoPath() + "/pulls", {
      title: COMMIT_HEADLINE, head: branch, base: target, body: prBody(list, members, [...pairs.values()], changed),
    });
    if (pr.status !== 201 || !pr.json || typeof pr.json.html_url !== "string") {
      refMade = false; // the commit is worth keeping: the person can open the pull request by hand
      throw new GitHubTrouble(WORDS.ghPRRefused + " " + branch + ".");
    }
    refMade = false;
    showSendResult(WORDS.prOpened, false, pr.json.html_url);
  } catch (err) {
    if (refMade) {
      // The branch holds nothing yet: take it away again.
      await ghSend("DELETE", repoPath() + "/git/refs/heads/" + branch.split("/").map(encodeURIComponent).join("/"));
    }
    showSendResult(err instanceof GitHubTrouble ? err.message : WORDS.ghOther, true);
  } finally {
    sending = false;
    renderSend();
  }
}

// ---------------------------------------------------------------------------
// The "Send to GitHub" panel: the passages, their credit and the route's
// actions. It reads meta.mode, the approvals, the base snapshot and the
// roster's credits.
// ---------------------------------------------------------------------------
function submitSession() {
  ui.send.hidden = !ui.send.hidden;
  if (!ui.send.hidden) {
    showSendResult("", false);
    renderSend();
  }
}

function showSendResult(text, bad, href) {
  clear(ui.sendResult);
  ui.sendResult.className = bad ? "state bad" : "state";
  if (text) ui.sendResult.append(text);
  if (href) {
    let ok = false;
    try { ok = new URL(href).origin === new URL(GITHUB_WEB).origin; } catch { ok = false; }
    ui.sendResult.append(" ", ok ? el("a", { href, target: "_blank", rel: "noopener noreferrer", text: href }) : href);
  }
}

let sendTimer = 0;
doc.on("update", () => {
  if (!ui.send || ui.send.hidden) return;
  clearTimeout(sendTimer);
  sendTimer = setTimeout(() => renderSend(), 600);
});

function renderSend() {
  if (!ui.send || ui.send.hidden) return;
  if (session.role !== "member" || session.over) { ui.send.hidden = true; return; }
  const mode = meta.get("mode") === "issues" ? "issues" : "pr";
  const analysis = analyse();
  const all = analysis.flatMap((f) => f.passages);
  ui.sendIntro.textContent = mode === "issues" ? WORDS.sendIntroIssues : WORDS.sendIntroPR;
  clear(ui.sendList);
  clear(ui.sendActions);
  for (const path of analysis.refused) {
    ui.sendList.append(el("li", { class: "state bad", text: WORDS.pathRefused + " " + shownPath(path) }));
  }
  if (all.length === 0) {
    ui.sendList.append(el("li", { class: "state", text: WORDS.sendNothing }));
  }
  for (const p of all) {
    const actions = el("div", { class: "row" });
    if (mode === "issues") {
      const link = issueLink(p);
      if (link.fits) {
        actions.append(el("a", { class: "button", href: link.url, target: "_blank", rel: "noopener noreferrer", text: WORDS.sendAsIssue }));
      } else {
        actions.append(el("span", { class: "state", text: WORDS.tooLongForIssue }),
          el("button", { type: "button", disabled: sending, text: WORDS.sendAsPR, onclick: () => openPullRequest(p) }));
      }
    }
    ui.sendList.append(el("li", { class: "passage" },
      el("div", { class: "from", text: relPath(p.path) + " · " + linesLabel(p) }),
      el("div", { class: "credit", text: creditLines(p).join("\n") }),
      el("del", { text: p.before }),
      el("ins", { text: p.after }),
      actions.childElementCount ? actions : null));
  }
  if (mode === "pr") {
    ui.sendActions.append(el("button", {
      type: "button", class: "primary", disabled: sending || all.length === 0 || !githubToken,
      text: WORDS.openPR, onclick: () => openPullRequest(null),
    }));
    if (!githubToken) ui.sendActions.append(el("span", { class: "state", text: WORDS.prNeedsSignIn }));
  }
  ui.sendActions.append(
    el("button", { type: "button", text: WORDS.sendRefresh, onclick: () => renderSend() }),
    el("button", { type: "button", text: WORDS.sendClose, onclick: () => { ui.send.hidden = true; } }));
}


// ---------------------------------------------------------------------------
// The editor.
// ---------------------------------------------------------------------------
let view = null;
let currentPath = "";
function openFile(path) {
  if (view) { view.destroy(); view = null; }
  clear(ui.editor);
  awareness.setLocalStateField("cursor", null);
  setAwarenessUser();
  const ytext = path ? files.get(path) : null;
  if (!(ytext instanceof Y.Text)) {
    currentPath = "";
    ui.filetitle.textContent = WORDS.noFile;
    renderFiles();
    renderGuestBar();
    return;
  }
  currentPath = path;
  ui.filetitle.textContent = path;
  const extensions = [
    keymap.of([...yUndoManagerKeymap]),
    basicSetup,
    markdown(),
    EditorView.lineWrapping,
    yCollab(ytext, awareness),
  ];
  if (session.role !== "member") {
    // A guest reads and selects; the text changes only through a member.
    extensions.push(EditorState.readOnly.of(true), EditorView.contentAttributes.of({ inputmode: "none" }));
    extensions.push(EditorView.updateListener.of((u) => { if (u.selectionSet) renderGuestBar(); }));
  }
  view = new EditorView({ state: EditorState.create({ doc: ytext.toString(), extensions }), parent: ui.editor });
  renderFiles();
  renderGuestBar();
}
files.observe(() => {
  renderFiles();
  if (!currentPath || !files.has(currentPath)) {
    const first = [...files.keys()].sort()[0];
    if (first && session.synced) openFile(first);
  }
});

// ---------------------------------------------------------------------------
// Rendering.
// ---------------------------------------------------------------------------
const ui = {};
let statusText = "";
let statusBad = false;
function setStatus(text, bad) {
  statusText = text;
  statusBad = bad;
  if (ui.status) {
    ui.status.textContent = text;
    ui.status.className = bad ? "status bad" : "status";
  }
}

function build() {
  const app = document.getElementById("app");
  clear(app);
  ui.toast = el("div", { class: "toast", role: "status", "aria-live": "polite", hidden: true });
  if (!route) {
    app.append(el("p", { class: "notice", text: WORDS.wrongAddress }), ui.toast);
    return false;
  }
  document.title = route.slug + " · " + WORDS.title;
  ui.status = el("span", { class: "status" });
  ui.who = el("span", { class: "who" });
  ui.signInTop = el("button", { type: "button", text: WORDS.signIn, onclick: () => startSignIn() });
  ui.submit = el("button", { type: "button", class: "primary", text: WORDS.submit, onclick: () => submitSession() });
  const bar = el("header", { class: "bar" },
    el("h1", { text: route.slug + " · " + WORDS.title }), ui.status, el("span", { class: "spacer" }),
    ui.who, ui.signInTop, ui.submit);

  // The start screen.
  ui.relay = el("input", { id: "relay", type: "url", inputmode: "url", autocomplete: "off", spellcheck: "false", placeholder: "https://" });
  ui.code = el("input", { id: "code", autocomplete: "off", spellcheck: "false", maxlength: "12", autocapitalize: "characters" });
  ui.name = el("input", { id: "name", autocomplete: "nickname", maxlength: "40" });
  ui.mode = el("select", { id: "mode" },
    el("option", { value: "pr", text: WORDS.modePR }), el("option", { value: "issues", text: WORDS.modeIssues }));
  ui.token = el("input", { id: "token", type: "password", autocomplete: "off", spellcheck: "false" });
  ui.relay.addEventListener("input", () => renderRelay());
  ui.relayStandingText = el("span");
  ui.relayStandingOrigin = el("code", { class: "origin" });
  ui.relayStanding = el("p", { class: "standing", hidden: true }, ui.relayStandingText, " ", ui.relayStandingOrigin);
  ui.relayCard = el("section", { class: "card" },
    el("label", { for: "relay", text: WORDS.relayLabel }), ui.relay, el("p", { text: WORDS.relayHint }), ui.relayStanding);
  ui.joinCard = el("form", { class: "card", onsubmit: (e) => { e.preventDefault(); joinClicked(); } },
    el("h2", { text: WORDS.joinHeading }), el("p", { text: WORDS.joinHint }),
    el("label", { for: "code", text: WORDS.codeLabel }), ui.code,
    el("label", { for: "name", text: WORDS.nameLabel }), ui.name,
    el("div", { class: "row" }, el("button", { type: "submit", class: "primary", text: WORDS.join })));
  ui.signedIn = el("p");
  ui.signInButton = el("button", { type: "button", class: "primary", text: WORDS.signIn, onclick: () => startSignIn() });
  ui.tokenToggle = el("button", { type: "button", text: WORDS.pasteToken, onclick: () => showTokenForm(true) });
  ui.tokenForm = el("form", { hidden: true, onsubmit: (e) => { e.preventDefault(); const v = ui.token.value; ui.token.value = ""; useToken(v); } },
    el("label", { for: "token", text: WORDS.tokenLabel }), ui.token,
    el("div", { class: "row" }, el("button", { type: "submit", text: WORDS.useToken })));
  ui.hostRow = el("div", { hidden: true },
    el("h2", { text: WORDS.hostHeading }),
    el("label", { for: "mode", text: WORDS.modeLabel }), ui.mode,
    el("div", { class: "row" }, el("button", { type: "button", class: "primary", text: WORDS.host, onclick: () => hostClicked() })));
  ui.signInNote = el("p", { class: "state" });
  ui.teamCard = el("section", { class: "card" },
    el("h2", { text: WORDS.teamHeading }), el("p", { text: WORDS.teamHint }), ui.signedIn,
    el("div", { class: "row" }, ui.signInButton, ui.tokenToggle), ui.signInNote, ui.tokenForm, ui.hostRow);
  ui.start = el("div", { class: "start" }, ui.relayCard, ui.joinCard, ui.teamCard);

  // The workspace.
  ui.files = el("nav", { class: "files", "aria-label": WORDS.filesHeading });
  ui.filetitle = el("div", { class: "filetitle" });
  ui.editor = el("div", { class: "editor" });
  ui.quote = el("div", { class: "quote" });
  ui.replacement = el("textarea", { id: "replacement", maxlength: String(MAX_REPLACEMENT) });
  ui.suggestForm = el("form", { hidden: true, onsubmit: (e) => { e.preventDefault(); sendSuggestion(ui.replacement.value); ui.suggestForm.hidden = true; } },
    ui.quote, el("label", { for: "replacement", text: WORDS.replacementLabel }), ui.replacement,
    el("div", { class: "row" },
      el("button", { type: "submit", class: "primary", text: WORDS.send }),
      el("button", { type: "button", text: WORDS.cancel, onclick: () => { ui.suggestForm.hidden = true; } })));
  ui.suggestButton = el("button", { type: "button", text: WORDS.suggest, onclick: () => openSuggestForm() });
  ui.guestbar = el("div", { class: "guestbar", hidden: true },
    el("p", { class: "state", text: WORDS.guestHint }), el("div", { class: "row" }, ui.suggestButton), ui.suggestForm);
  ui.side = el("aside", { class: "side" });
  ui.sendIntro = el("p", { class: "state" });
  ui.sendList = el("ul");
  ui.sendActions = el("div", { class: "row" });
  ui.sendResult = el("p", { class: "state", role: "status", "aria-live": "polite" });
  ui.send = el("section", { class: "send", hidden: true, "aria-label": WORDS.sendHeading },
    el("h2", { text: WORDS.sendHeading }), ui.sendIntro, ui.sendList, ui.sendActions, ui.sendResult);
  ui.work = el("div", { class: "work", hidden: true },
    ui.files, el("section", { class: "center" }, ui.send, ui.filetitle, ui.editor, ui.guestbar), ui.side);

  // The confirmation a relay the repository's list cannot speak for needs
  // before any sign-in reaches it, shown over the start screen or a session.
  ui.trustOrigin = el("code", { class: "origin" });
  ui.trustCancel = el("button", { type: "button", text: WORDS.cancel, onclick: () => trustAnswered(false) });
  ui.trust = el("section", { class: "card trust", hidden: true, role: "alertdialog", "aria-labelledby": "trust-heading" },
    el("h2", { id: "trust-heading", text: WORDS.trustHeading }),
    el("p", { class: "only", text: WORDS.trustOnly }), ui.trustOrigin,
    el("p", { text: WORDS.trustWhy }),
    el("div", { class: "row" },
      el("button", { type: "button", class: "primary", text: WORDS.trustYes, onclick: () => trustAnswered(true) }), ui.trustCancel));

  app.append(bar, ui.trust, ui.start, ui.work, ui.toast);
  return true;
}

function showTokenForm(show) {
  ui.tokenForm.hidden = !show;
  if (show) ui.token.focus();
}

function readRelayInput() {
  if (session.inRoom || session.everJoined) return true;
  const relay = relayOrigin(ui.relay.value);
  if (!relay) { toast(WORDS.relayBad); ui.relay.focus(); return false; }
  if (relay !== session.relay) {
    if (ws) { const sock = ws; ws = null; sock.close(); }
    // A sign-in was cleared for the relay it was made with, never this one.
    if (githubToken || heldToken) { githubToken = ""; heldToken = ""; session.login = ""; session.member = false; }
  }
  session.relay = relay;
  renderRelay();
  const standing = relayStanding(relay);
  if (standing === "refused" || standing === "checking") {
    toast(standing === "checking" ? WORDS.relayChecking : refusalWords(relay));
    return false;
  }
  // Only a cleared relay is remembered: never one a link named and nobody
  // confirmed.
  if (relayCleared()) remember(KEY_RELAY, relay);
  return true;
}

function joinClicked() {
  if (!readRelayInput()) return;
  const code = normalizeCode(ui.code.value);
  if (!code) { toast(WORDS.codeBad); ui.code.focus(); return; }
  session.code = code;
  session.name = ui.name.value.trim().slice(0, 40);
  if (session.name) remember(KEY_NAME, session.name);
  if (socketOpen()) proceed();
  else openSocket();
}

function hostClicked() {
  if (!readRelayInput()) return;
  session.code = "";
  if (socketOpen() && session.member) sendJSON({ type: "host", repo: route.slug });
  else { session.pending = "host"; openSocket(); }
}

function openSuggestForm() {
  if (!view) return;
  const sel = view.state.selection.main;
  if (sel.empty) { toast(WORDS.selectFirst); return; }
  const text = view.state.sliceDoc(sel.from, sel.to);
  ui.quote.textContent = text;
  ui.replacement.value = text;
  ui.suggestForm.hidden = false;
  ui.replacement.focus();
}

function renderTop() {
  if (!ui.who) return;
  let who = "";
  if (session.login) who = WORDS.signedInAs + " " + session.login;
  else if (session.inRoom && session.name) who = session.name;
  ui.who.textContent = who;
  ui.signInTop.hidden = !(session.inRoom && session.role === "guest" && !session.over);
  ui.submit.hidden = !(session.synced && session.role === "member" && !session.over);
  ui.status.textContent = statusText;
  ui.status.className = statusBad ? "status bad" : "status";
}

function renderStart() {
  if (!ui.start) return;
  const inSession = session.everJoined || session.over;
  ui.start.hidden = inSession;
  ui.work.hidden = !inSession || session.seeding;
  if (inSession) return;
  ui.signedIn.textContent = session.login
    ? WORDS.signedInAs + " " + session.login + (session.member ? "" : ". " + WORDS.notMember)
    : "";
  ui.signInButton.hidden = !!session.login;
  ui.tokenToggle.hidden = !!session.login;
  ui.hostRow.hidden = !(session.login && session.member);
}

function renderFiles() {
  if (!ui.files) return;
  clear(ui.files);
  ui.files.append(el("h2", { text: WORDS.filesHeading }));
  const paths = [...files.keys()].sort();
  let folder = null;
  for (const p of paths) {
    const rel = p.startsWith(BRIEF_PREFIX) ? p.slice(BRIEF_PREFIX.length) : p;
    const cut = rel.lastIndexOf("/");
    const dir = cut < 0 ? "" : rel.slice(0, cut);
    if (dir !== folder) {
      folder = dir;
      if (dir) ui.files.append(el("div", { class: "folder", text: dir + "/" }));
    }
    ui.files.append(el("button", {
      type: "button",
      text: cut < 0 ? rel : rel.slice(cut + 1),
      "aria-current": p === currentPath ? "true" : "false",
      onclick: () => openFile(p),
    }));
  }
}

function renderGuestBar() {
  if (!ui.guestbar) return;
  const guest = session.role === "guest" && session.synced && !session.over && !!view;
  ui.guestbar.hidden = !guest;
  if (guest) ui.suggestButton.disabled = view.state.selection.main.empty;
}

function badge(p) {
  if (p.host) return el("span", { class: "badge host", text: WORDS.badgeHost });
  return el("span", { class: "badge", text: p.role === "member" ? WORDS.badgeMember : WORDS.badgeGuest });
}

function renderSide() {
  if (!ui.side) return;
  clear(ui.side);
  if (session.host && session.code && !session.seeding && !session.over) {
    const url = inviteURL();
    let img = null;
    try {
      const qr = qrcode(0, "M");
      qr.addData(url);
      qr.make();
      img = el("img", { src: qr.createDataURL(6, 2), alt: WORDS.inviteHint });
    } catch { img = null; }
    const mode = el("select", { id: "mode-live", onchange: (e) => { doc.transact(() => meta.set("mode", e.target.value)); } },
      el("option", { value: "pr", text: WORDS.modePR }), el("option", { value: "issues", text: WORDS.modeIssues }));
    mode.value = meta.get("mode") === "issues" ? "issues" : "pr";
    ui.side.append(el("section", { class: "invite" },
      el("h2", { text: WORDS.inviteHeading }), img,
      el("div", { class: "code", text: session.code, "aria-label": WORDS.codeLabel }),
      el("p", { class: "state", text: WORDS.inviteHint }),
      el("div", { class: "link", text: url }),
      el("div", { class: "row" },
        el("button", { type: "button", text: WORDS.copyLink, onclick: () => copyLink(url) }),
        el("button", { type: "button", text: WORDS.newCode, onclick: () => sendJSON({ type: "rotate" }) }),
        el("button", { type: "button", text: WORDS.endSession, onclick: () => { if (confirm(WORDS.endConfirm)) sendJSON({ type: "close" }); } })),
      el("label", { for: "mode-live", text: WORDS.modeLabel }), mode));
  }

  const list = el("ul");
  for (const p of session.roster) {
    list.append(el("li", {}, p.name || p.login || "", badge(p), p.id === session.myId ? " " + WORDS.you : ""));
  }
  ui.side.append(el("section", { class: "roster" }, el("h2", { text: WORDS.rosterHeading }), list));

  if (session.role === "member" && session.synced) {
    const items = el("ul");
    for (const s of suggestions.values()) {
      if (!s || typeof s.id !== "string" || suggestionsDone.has(s.id) || !validSuggestion(s.body)) continue;
      items.append(el("li", { class: "suggestion" },
        el("div", { class: "from", text: WORDS.suggestedBy + " " + s.from + " · " + s.body.path.replace(BRIEF_PREFIX, "") }),
        el("del", { text: s.body.quote.exact }),
        el("ins", { text: s.body.replacement }),
        el("div", { class: "row" },
          el("button", { type: "button", class: "primary", text: WORDS.apply, onclick: () => applySuggestion(s.id) }),
          el("button", { type: "button", text: WORDS.decline, onclick: () => declineSuggestion(s.id) }))));
    }
    ui.side.append(el("section", { class: "suggestions" }, el("h2", { text: WORDS.suggestionsHeading }),
      items.childElementCount ? items : el("p", { class: "state", text: WORDS.noSuggestions })));
  }

  if (session.role === "guest" && mine.length) {
    const items = el("ul");
    for (const s of mine) {
      const done = suggestionsDone.get(s.id);
      items.append(el("li", { class: "suggestion" },
        el("del", { text: s.body.quote.exact }),
        el("ins", { text: s.body.replacement }),
        el("div", { class: "state", text: done === "applied" ? WORDS.applied : done === "declined" ? WORDS.declined : WORDS.waiting })));
    }
    ui.side.append(el("section", { class: "suggestions" }, el("h2", { text: WORDS.mySuggestionsHeading }), items));
  }
}

async function copyLink(url) {
  try { await navigator.clipboard.writeText(url); toast(WORDS.linkCopied); } catch { /* clipboard refused */ }
}

function renderAll() {
  if (session.over && ui.send) ui.send.hidden = true;
  renderTop();
  renderStart();
  renderSide();
  renderGuestBar();
  if (session.synced && !session.seeding && !view && files.size > 0) {
    openFile([...files.keys()].sort()[0]);
  } else if (view && session.over) {
    view.destroy();
    view = null;
  }
}

// ---------------------------------------------------------------------------
// Start.
// ---------------------------------------------------------------------------
// addSlash sends /<owner>/<repo>/brief on to /<owner>/<repo>/brief/, the
// address the page's links and the invite are made from.
function addSlash() {
  if (!/\/[^/]+\/[^/]+\/brief$/.test(location.pathname)) return false;
  location.replace(location.pathname + "/" + location.search + location.hash);
  return true;
}

async function main() {
  if (addSlash()) return;
  if (!build()) return;
  const frag = new URLSearchParams(location.hash.slice(1));
  session.relay = relayOrigin(frag.get("relay")) || relayOrigin(recall(KEY_RELAY));
  session.code = normalizeCode(frag.get("code"));
  session.name = recall(KEY_NAME);
  ui.relay.value = session.relay;
  ui.code.value = session.code;
  ui.name.value = session.name;
  setStatus("", false);
  renderAll();
  renderRelay();
  // Before anything reaches a relay: which relays does the repository name?
  anchor = await readRelayList("");
  renderRelay();
  if (frag.get("relay") && session.relay && relayCleared()) ui.relayCard.hidden = true;
  refreshSignInNote();
  if (!(await finishSignIn())) {
    // A reload of a page that was in a session: back in by its rejoin token.
    if (takeRejoin()) openSocket();
  }
  ui.relay.value = session.relay;
  ui.code.value = session.code;
  renderAll();
}
main();
