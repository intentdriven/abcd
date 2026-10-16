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
// What this page talks to, and nothing else: its relay (its WebSocket,
// /config, and /auth/token, or /auth/device on a local session's own page),
// which is the page's own origin when the relay serves it, and otherwise the
// one the invite link or the repository's committed relay list names;
// https://api.github.com (without a token for the repository's list of its
// relays, and with the person's own token); and github.com for sign-in and
// pre-filled issues. The GitHub token
// is held in this module's memory only; it is never stored, put in a URL or
// logged, and it reaches a relay only once the repository's committed relay
// list names that relay, the relay serves this page itself, or, where the
// list cannot be read, the person has confirmed it; and only to the relay
// GitHub returned the person to after signing in. GitHub's sign-in code goes
// only to that relay, through the callback the team's app lists, and never
// to a relay the invite link or this tab's storage names. The relay's
// rejoin token, which brings this participant
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
  title: "Brief",
  wrongAddress: "This page opens a project's brief for a live session. Open it from the link your team shared.",
  checking: "Getting the live session ready…",
  notSetUp: "Live sessions are not set up for this project yet. Ask your technical facilitator to set them up.",
  privateNoLink: "This page could not find a live session for this project. If your team runs one, ask the host for the invite link. If not, ask your technical facilitator to set live sessions up.",
  notOurLink: "This invite link does not lead to this project's own live session, so the page will not open it. Ask the host for a new link.",
  cannotCheck: "The page could not reach GitHub to check this project's settings. Try again in a moment.",
  tryAgain: "Try again",
  pageFor: "This page is for",
  trustHeading: "Before you continue",
  trustQuestion: "The page cannot check this invite against the project's own settings, because the project is private.",
  trustOnly: "Only continue if someone on the project's team sent you this link.",
  trustYes: "Someone on the team sent it",
  signInNotSetUp: "Signing in with GitHub is not set up for this project yet. Ask your technical facilitator. You can still join as a guest.",
  signInBusy: "Too many sign-ins are in progress. Wait a few minutes, then try again.",
  framedRefused: "This page does not run inside another page. Open it in its own tab:",
  joinHeading: "Join the live session",
  joinMembers: "On the project's team? Sign in with GitHub to edit the brief directly.",
  joinGuests: "Everyone else: join as a guest with your name, and suggest changes for the team to take up.",
  codeLabel: "Session code",
  nameLabel: "Your name",
  join: "Join as a guest",
  codeBad: "A session code is six letters and numbers. The host can read it out to you.",
  haveCode: "Have a session code?",
  joinWithCode: "Join with the code",
  startHeading: "Start a live session",
  startHint: "Sign in with GitHub to start a session on this brief. You will get a link and a QR code to invite people.",
  signIn: "Sign in with GitHub",
  signingInTo: "You are signing in to work on",
  signInFailed: "Signing in with GitHub did not finish. Try again.",
  signingIn: "Signing in…",
  signedInAs: "Signed in as",
  notMember: "GitHub says you cannot change this project, so you take part as a guest.",
  deviceHeading: "Confirm on GitHub",
  deviceStep1: "Open GitHub's sign-in page:",
  deviceStep2: "Enter this code there, and confirm:",
  deviceOpen: "Open GitHub",
  deviceCopy: "Copy the code",
  deviceCopied: "Code copied.",
  deviceWaiting: "Waiting for you to confirm on GitHub…",
  deviceExpired: "That code has expired. Sign in again for a new one.",
  deviceDenied: "Signing in was cancelled on GitHub.",
  host: "Start the session",
  modeLabel: "When you are done, send the changes back as",
  modePR: "one pull request for the whole session",
  modeIssues: "one GitHub issue per change",
  loadingBrief: "Reading the brief from GitHub…",
  sendingBrief: "Sharing the brief with the session…",
  noBrief: "This repository has no brief at .abcd/development/brief/ on its default branch.",
  githubFailed: "GitHub did not answer the request for the brief",
  connecting: "Connecting…",
  connected: "Live",
  reconnecting: "Connection lost. Reconnecting…",
  cannotConnect: "The live session could not be reached. Try again in a moment, or ask the host.",
  syncing: "Loading the brief…",
  ended: "The session has ended.",
  expired: "The session has ended: a session lasts at most twelve hours. The host can open a new one.",
  codeGone: "That session is no longer open.",
  wrongCode: "That code does not match a live session. Check it with the host.",
  tooManyTries: "Too many wrong codes. Wait a few minutes, then try again.",
  joinPaused: "Joining is paused for a few minutes. Try again shortly.",
  notThisProject: "Live sessions for this project are not set up here. Ask your technical facilitator.",
  signInRefused: "GitHub did not accept the sign-in. Sign in again.",
  inviteHeading: "Invite people",
  inviteHint: "Scan the QR code, or open the link and type the code.",
  newCode: "New code",
  endSession: "End session",
  endConfirm: "End the session for everyone? Edits not yet sent to GitHub are lost.",
  copyLink: "Copy invite link",
  linkCopied: "Invite link copied.",
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
  suggestTooBig: "That suggestion is too long to send. Choose a shorter passage, or a shorter replacement.",
  suggestTooFast: "Your suggestion was not sent: suggestions came too fast. Wait a moment, then send it again.",
  suggestNotSent: "Your suggestion was not sent.",
  suggestWait: "Your last suggestion is still on its way. Send this one when it has arrived.",
  suggestLost: "The connection dropped before your last suggestion was confirmed, so it may not have reached the team. Check Your suggestions, and send it again if it is not there.",
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
  setAside: "Edits that build on a refused edit can never be applied, so this page set them aside to stay in step.",
  metaVoid: "A change to the session's settings was ignored, because only the host sets them and the base, the branch and the repository are set once. It came from",
  someone: "a participant the session could not credit",
  guestCursor: "guest",
  notesHeading: "Warnings",
  pathRefused: "Not sent: this file is not one of the brief's files the session started from:",
  // The two designs (Guided, the default, and Shared), switched with
  // Ctrl+[ and Ctrl+]. A word with {repo}, {owner}, {name} or {n} has the
  // value put in by say(), and is shown with textContent like every other.
  designGuided: "Switched to the guided design",
  designShared: "Switched to the shared design",
  heroTitle: "Talk through the {repo} brief, together",
  heroLede: "This is where the {repo} team reads and edits the project brief at the same time, and invites others to suggest changes. Nothing changes in the project until the team approves it.",
  heroLedeShared: "Talk through the {repo} brief with your team and your guests, live, and change it together.",
  sharedTitle: "{repo} brief",
  invitedTitle: "You're invited to discuss the {repo} brief",
  invitedLede: "The team is going through its brief right now and would like your view.",
  brandSub: "Project brief · {owner}",
  teamHeading: "I'm on the {repo} team",
  codeHint: "Someone on the team gave you six letters and numbers. Type them here to join as a guest.",
  nameHint: "Everyone in the session will see this name next to your suggestions.",
  whatNext: "What happens next",
  nextSignIn: "Sign in or enter a code",
  nextHere: "You are here.",
  nextJoin: "Join the session",
  nextJoinHint: "You see the brief and who else is here.",
  nextWork: "Work on it together",
  nextWorkHint: "Edit, or suggest changes as a guest.",
  nextSend: "Send it for approval",
  nextSendHint: "A team member sends the changes to GitHub.",
  nextInvited: "Once you join, the brief opens in front of you. You see who else is there, with a coloured name flag wherever they are working.",
  nextNotSetUp: "Once live sessions are on, this page shows a button to start one. You will not need to set anything up yourself.",
  nextChose: "Chose to sign in",
  nextChoseHint: "On this computer.",
  nextConfirm: "Confirm on GitHub",
  nextOpens: "The session opens",
  nextOpensHint: "You start or join the session.",
  nextGuest: "When the team is done, a team member sends the changes to GitHub for approval. You can leave whenever you like; your suggestions stay with the team.",
  askHeading: "Want to ask them now?",
  askHint: "Copy this message and send it however you normally would.",
  askMessage: "Hi, could you turn on live brief sessions for {slug}? I would like to go through the brief with the team: {page}",
  askCopy: "Copy the message",
  askCopied: "Message copied.",
  deviceIntro: "Signing in on this computer takes one extra step: you confirm a short code on GitHub's website.",
  deviceNote: "Note this code",
  deviceOpenTitle: "Open GitHub and type it in",
  deviceOpenHint: "GitHub opens in a new tab. Type the code there, then confirm.",
  deviceBack: "Come back here",
  deviceBackHint: "This page moves on by itself. Keep it open.",
  bannerHost: "You're hosting the {repo} brief session",
  bannerMember: "You're editing the {repo} brief with your team",
  bannerGuest: "Welcome, {name}. You're a guest in the {repo} session",
  bannerMemberHelp: "Edit the text directly; everyone sees your changes as you type. Review guests' suggestions on the right. When the team is happy, send it all to GitHub.",
  bannerGuestHelp: "You can read everything. To suggest a change, select some words in the text, then press Suggest a change. A team member applies or declines it.",
  hereCount: "{n} here",
  liveHere: "Live · {n} here",
  documents: "Documents",
  documentsHint: "Pick one to open it.",
  showDocuments: "Show the documents",
  hideDocuments: "Hide the documents",
  reviewPeople: "Review and people",
  suggestFollow: "Suggest and follow",
  showPanel: "Show",
  hidePanel: "Hide",
  close: "Close",
  done: "Done",
  briefArea: "The brief",
  editHelp: "Click into the text to edit it.",
  readHelp: "Select words in the text to suggest a change.",
  readOnly: "Read only: suggest changes instead of editing",
  savedForEveryone: "Saved for everyone as you type.",
  sugReview: "Suggestions to review",
  sugHelp: "Guests cannot edit. Apply a suggestion to put it into the text, or decline it.",
  sugWaiting: "Suggestions waiting",
  sugSub: "Guests suggest; the team decides. Applying puts the new words in the brief for everyone at once.",
  sugShort: "Suggestions",
  mineHelp: "What happened to the changes you have suggested so far.",
  mineSub: "The team sees each one and applies or declines it. You see the outcome here.",
  noMine: "You have not suggested anything yet.",
  wordsWas: "Was",
  wordsNow: "Now",
  wordsSuggested: "Suggested",
  youSelected: "You selected",
  whoHere: "Who's here",
  hereNow: "Here now",
  thisSession: "This session",
  sessionInfoMember: "Changes save for everyone as you type. Guests suggest; the team decides.",
  sessionInfoGuest: "You suggest changes; the team applies or declines them.",
  youShort: "You",
  signedInShort: "Signed in",
  joinedAsGuest: "Joined as a guest",
  inviteSomeone: "Invite someone",
  inviteHelp: "Share the link, or let them scan the code or type it in.",
  invitePeople: "Invite people",
  inviteTitle: "Invite people to this session",
  scanHint: "Scan with a phone",
  joinCode: "Join code",
  copyCode: "Copy code",
  codeCopied: "Code copied.",
  finishHeading: "Finished? Send it to GitHub",
  finishHelp: "Everything from this session goes back to GitHub, naming everyone who took part. Nothing in the project changes until someone on the team accepts it there.",
  submitMore: "Send to GitHub…",
  sendTip: "Turn this session into proposed changes on GitHub",
  suggestTip: "Select words in the brief, then suggest new ones",
  whatChanged: "What changed",
  sendReassure: "Sending only proposes the changes. The brief on GitHub stays exactly as it is until someone on the team accepts them.",
  backToSession: "Back to the session",
  sessionBar: "Session",
  whoIsHere: "Who is here",
});

// ---------------------------------------------------------------------------
// Fixed values.
// ---------------------------------------------------------------------------
const GITHUB_API = "https://api.github.com";
const GITHUB_WEB = "https://github.com";
const GITHUB_AUTHORIZE = "https://github.com/login/oauth/authorize";
const GITHUB_DEVICE = "https://github.com/login/device";
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
const TAG_UPDATE = 0;
const TAG_AWARENESS = 1;
// The relay sends every member's update on as tag 2: the tag, the sender's
// participant number as four big-endian bytes, then the update's bytes as the
// member sent them. The roster's authors map names each number's login.
const TAG_ATTRIBUTED = 2;
// And every awareness message as tag 3, the same way: the tag, the sender's
// participant number, then the message's bytes as the participant sent them.
const TAG_ATTRIBUTED_AWARENESS = 3;
// A relay message is at most 256 KiB; the seed is sent in updates of at most
// SEED_BUDGET bytes of text, and a longer file in pieces of SEED_PIECE UTF-16
// units (three bytes each at most).
const SEED_BUDGET = 160000;
const SEED_PIECE = 50000;
const QUOTE_CONTEXT = 32;
// A suggestion a member's page records lands in the session's log, so each of
// its fields is bounded: the replacement, and the passage it replaces, at
// most MAX_REPLACEMENT characters; the quote's context at most MAX_QUOTE_PART
// each side; a relative position at most MAX_POSITION_JSON as JSON; and all of
// it at most MAX_SUGGEST_BYTES, the relay's own cap on a suggestion.
const MAX_REPLACEMENT = 64000;
const MAX_QUOTE_PART = 200;
const MAX_POSITION_JSON = 256;
const MAX_SUGGEST_BYTES = 70 * 1024;
// The relay numbers a session's suggestions s1, s2, …
const SUGGESTION_ID_RE = /^s[0-9]{1,9}$/;
// Origins of Yjs transactions: what the relay sent, and the host's seeding.
const FROM_RELAY = Symbol("relay");
const FROM_SEED = Symbol("seed");
const KEY_NAME = "abcd.session.name";
const KEY_SIGN_IN = "abcd.session.sign-in";
const KEY_REJOIN = "abcd.session.rejoin";
// The relays a person confirmed, per repository, each with when:
// { "owner/repo": [{ relay, at }] }, in this tab's session storage. A
// confirmation is the person's word that the team sent this link: it lasts
// this tab, across GitHub's sign-in redirect, and lapses after CONFIRM_LIFE,
// the most a session lasts, so a restored tab does not carry it on.
const KEY_TRUSTED = "abcd.session.confirmed-relays";
const CONFIRM_LIFE = 12 * 60 * 60 * 1000;
// Where an earlier page kept confirmations, in local storage and for good.
const LEGACY_TRUSTED = "abcd.session.trusted-relays";
// The web sign-in's callback on the relay: the team's GitHub App lists the
// relay's origin followed by it, and GitHub returns the person there alone.
const CALLBACK_PATH = "/auth/callback";
// The repository's committed list of its relays: {"relays": [origin, …]}.
const RELAYS_FILE = ".abcd/relay/relays.json";
// A relay's rejoin token: 128 random bits, base64url without padding.
const REJOIN_RE = /^[A-Za-z0-9_-]{22}$/;
// A participant's awareness colour: the editor writes it into a style
// attribute, so a remote one is used only when it is exactly this.
const HEX_COLOUR_RE = /^#[0-9a-fA-F]{6}$/;
const COLOURS = ["#2f6fdb", "#c2410c", "#15803d", "#9333ea", "#b91c1c", "#0e7490", "#a16207", "#be185d"];
// The page's two designs, the first the default, and the key that keeps a
// viewer's choice in this browser. The choice is a look, never a secret.
const DESIGNS = ["guided", "shared"];
const KEY_DESIGN = "abcd.session.design";
// The widths at which a sidebar becomes a sheet (a phone) and the right one
// a slide-over (a tablet, or a narrow window), per design.
const PHONE_MAX = 600;
const SLIDE_MAX = { guided: 1240, shared: 1300 };
// The page's icons are inline SVG, drawn from these paths; the namespace is
// a name, never fetched.
const SVG_NS = "http://www.w3.org/2000/svg";

// ---------------------------------------------------------------------------
// Storage: only the display name is remembered; the relays the person
// confirmed are kept for this tab, and the sign-in's PKCE verifier waits in
// session storage across GitHub's redirect. Every access may throw (private
// windows, blocked storage).
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

// trustedRelays reads this tab's confirmations of every repository;
// confirmedRelays lists this repository's that have not lapsed, and
// confirmRelay and forgetRelay change them. Nothing else is kept with them.
function trustedRelays() {
  try {
    const all = JSON.parse(sessionStorage.getItem(KEY_TRUSTED) || "{}");
    return all !== null && typeof all === "object" && !Array.isArray(all) ? all : {};
  } catch { return {}; }
}
function keepTrusted(all) {
  try { sessionStorage.setItem(KEY_TRUSTED, JSON.stringify(all)); } catch { /* storage unavailable */ }
}
function liveConfirmations() {
  const list = trustedRelays()[route.slug];
  const now = Date.now();
  return Array.isArray(list)
    ? list.filter((c) => c && typeof c.relay === "string" && typeof c.at === "number" && c.at <= now && now - c.at < CONFIRM_LIFE)
    : [];
}
function confirmedRelays() {
  return liveConfirmations().map((c) => c.relay);
}
function confirmRelay(relay) {
  const all = trustedRelays();
  all[route.slug] = liveConfirmations().filter((c) => c.relay !== relay).concat([{ relay, at: Date.now() }]).slice(-8);
  keepTrusted(all);
}
function forgetRelay(relay) {
  const all = trustedRelays();
  all[route.slug] = liveConfirmations().filter((c) => c.relay !== relay);
  keepTrusted(all);
}

// ---------------------------------------------------------------------------
// The GitHub token: module memory only, and only ever from a sign-in: nobody
// pastes a token. A token just signed in is held apart until the relay it
// would go to is cleared; only then does it become the token verify sends.
// ---------------------------------------------------------------------------
let githubToken = "";
let heldToken = "";
// tokenRelay is the relay the token was signed in through: the one GitHub
// returned the person to (the web flow), or the one that served this page
// (the device flow). The token is presented to it alone.
let tokenRelay = "";

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
// warn shows a warning that stays on the page, once, as well as a toast.
function warn(text) {
  toast(text);
  if (!notes.includes(text)) notes.push(text);
  renderSide();
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
  hostLogin: "",      // the login that opened the session, from the relay's roster
};
// Attributed updates wait here until a roster on this connection names their
// senders: the relay replays a session's log before it sends the roster.
const held = [];
let rosterSeen = false;
// Client ids seen in an unattributed (tag 0) update, from a relay that does
// not attribute: their text is applied but credited to nobody.
const unattributed = new Set();
const mine = [];              // a guest's own suggestions
const notes = [];             // warnings the page keeps showing
// The guest's suggestion sent and not yet answered. The relay's answer, a
// refusal or a rate limit included, does not say which message it answers,
// so a guest has one suggestion in flight at a time. Every refusal names
// kind "suggest"; one no answer comes for in SUGGEST_TIMEOUT is given up on.
let suggestionInFlight = null;
let suggestTimer = 0;
const SUGGEST_TIMEOUT = 15000;
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
  if (ws !== null || session.over || !session.relay || !relayCleared()) return;
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
  if (githubToken && relayTokenCleared()) {
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
  // No answer comes on a closed connection.
  clearTimeout(suggestTimer);
  if (suggestionInFlight) {
    suggestionInFlight = null;
    toast(WORDS.suggestLost);
  }
  renderTop();
  if (session.over) return;
  // The relay ended the session. Its closed message comes first, unless the
  // close cut a log replay short: then the close's reason says so.
  if (ev.code === 1000 && (ev.reason === "session closed" || ev.reason === "session expired")) {
    endSession(ev.reason === "session expired" ? WORDS.expired : WORDS.ended);
    return;
  }
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
        settlePending();
      } else if (bytes[0] === TAG_ATTRIBUTED_AWARENESS) {
        if (bytes.length < 5) return;
        admitAwareness(new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(1), bytes.subarray(5));
      }
      // An awareness message that names no sender (tag 1) is not applied: the
      // relay sends every one on as tag 3.
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
    case "closed": endSession(msg.reason === "expired" ? WORDS.expired : WORDS.ended); break;
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
    noteRefused(update);
  } else {
    try { Y.applyUpdate(doc, update, FROM_RELAY); } catch { /* an update that does not decode is dropped */ }
  }
  settlePending();
}

// Yjs holds an update that depends on clocks it has not seen as pending, and
// tries it again as each update arrives. Every page refuses the same forged
// update, so nothing can ever supply its clocks, and an update that builds on
// them would wait for ever. refusedFrom is each client id a refused update
// carried, with the lowest clock it carried there.
const refusedFrom = new Map();
function noteRefused(update) {
  let decoded;
  try { decoded = Y.decodeUpdate(update); } catch { return; }
  for (const s of decoded.structs) {
    const { client, clock } = s.id;
    if (!refusedFrom.has(client) || clock < refusedFrom.get(client)) refusedFrom.set(client, clock);
  }
}

// settlePending sets aside, with a note, what Yjs holds pending that waits on
// a refused update's clocks. Yjs keeps its pending structs as one merged
// update, so all of it goes; pending that waits on nothing refused arises only
// when the relay dropped an update for its rate, and the writer's page sends
// its recent updates again after that.
function settlePending() {
  if (refusedFrom.size === 0) return;
  const store = doc.store;
  const waits = (client, clock) => refusedFrom.has(client) && clock >= refusedFrom.get(client);
  let setAside = false;
  if (store.pendingStructs && [...store.pendingStructs.missing].some(([client, clock]) => waits(client, clock))) {
    store.pendingStructs = null;
    setAside = true;
  }
  if (store.pendingDs) {
    let stuck = true;
    try {
      stuck = [...Y.decodeUpdateV2(store.pendingDs).ds.clients].some(([client, ranges]) => ranges.some((r) => waits(client, r.clock + r.len - 1)));
    } catch { stuck = true; }
    if (stuck) {
      store.pendingDs = null;
      setAside = true;
    }
  }
  if (setAside) warn(WORDS.setAside);
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
  if (rejoined && githubToken && relayTokenCleared() && !session.login && session.role !== "member") sendJSON({ type: "verify", token: githubToken });
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
  // The login that opened the session: the relay names it, also while the
  // host is away, and marks the host among the participants.
  const named = session.roster.find((p) => p.host === true && p.role === "member" && typeof p.login === "string" && NAME_RE.test(p.login));
  if (typeof msg.host === "string" && NAME_RE.test(msg.host)) session.hostLogin = msg.host;
  else if (named) session.hostLogin = named.login;
  // A guest's awareness client id is tied to them while they are here.
  for (const num of [...guestAwareness.keys()]) {
    if (!session.roster.some((p) => p.id === "p" + num)) guestAwareness.delete(num);
  }
  rosterSeen = true;
  for (const h of held.splice(0)) admitUpdate(h.num, h.update);
  watchMeta();
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
  if (msg.kind === "suggest" || code === "suggestion-too-big" || code === "too-many-suggestions" || code === "suggestions-full" || code === "bad-suggestion") {
    // This refused the one suggestion in flight: the relay names every
    // refusal of a suggest by its kind, a rate limit's included, so a
    // dropped cursor ("awareness") leaves it alone. A suggestion dropped for
    // coming too fast may be sent again.
    const was = suggestionInFlight;
    suggestionInFlight = null;
    clearTimeout(suggestTimer);
    if (was) toast(code === "rate-limited" ? WORDS.suggestTooFast : WORDS.suggestNotSent + " " + text);
    renderSide();
    return;
  }
  if (code === "join-timeout") {
    // A normal close follows; a session in progress reconnects and rejoins.
    quietClose = true;
    return;
  }
  if (code === "rate-limited" && msg.kind === "update" && session.role === "member") {
    // The relay dropped an update that came too fast, and says so by its
    // kind, so the recent ones go again. A dropped cursor needs nothing: the
    // next one replaces it.
    resendRecent();
    return;
  }
  if (code === "rate-limited") {
    // A dropped cursor needs nothing, as the next one replaces it, and must
    // never read as a failed sign-in or a toast per drag. Only a refused
    // session start is worth saying.
    if (msg.kind === "host") toast(text);
    return;
  }
  if (session.verifying) {
    session.verifying = false;
    githubToken = "";
    toast(plainError(code, text));
    proceed();
    return;
  }
  if (code === "repo-not-served") {
    if (session.everJoined) { endSession(WORDS.notThisProject); return; }
    session.code = "";
    session.pending = "";
    toast(WORDS.notThisProject);
    renderStart();
    return;
  }
  if (code === "paused" && session.everJoined) {
    // Joining by code is paused on the relay for a while: try again later.
    toast(WORDS.joinPaused);
    if (ws) ws.close();
    return;
  }
  if (code === "bad-code" || code === "too-many-tries" || code === "paused") {
    if (session.everJoined) endSession(WORDS.codeGone);
    else {
      // The connection stays open: the person can type the code again, in
      // the code field a link's code had hidden.
      session.code = "";
      linkCode = "";
      toast(plainError(code, text));
      renderStart();
    }
    return;
  }
  toast(plainError(code, text));
}

// plainError says a relay's refusal in the page's own plain words, where
// there are some; the relay's words are for the people who run it.
function plainError(code, text) {
  return {
    "bad-code": WORDS.wrongCode, "too-many-tries": WORDS.tooManyTries, paused: WORDS.joinPaused,
    "verify-failed": WORDS.signInRefused, "repo-not-served": WORDS.notThisProject, "no-session": WORDS.codeGone,
  }[code] || text;
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

// admitAwareness applies an awareness message the relay attributed to
// participant num, only when every client id in it is tied to that
// participant, with each cursor labelled from the roster: a page can claim
// any client id and any name in awareness, so neither is believed. Until a
// roster on this connection names the sender, the message is dropped; the
// others share their cursors again when the roster shows a newcomer.
function admitAwareness(num, body) {
  if (!rosterSeen) return;
  const sender = session.roster.find((p) => p.id === "p" + num);
  if (!sender) return;
  const label = cursorLabel(sender);
  applyAwarenessUpdate(awareness, cleanAwarenessUpdate(body, (client) => (awarenessTied(sender, num, client) ? label : null)), FROM_RELAY);
}

// guestAwareness ties each guest's participant number to the one awareness
// client id its first awareness message named. A guest's page registers no
// client id, so this is the only tie; it holds while the guest is in the
// roster, and a later message naming another client id is refused.
const guestAwareness = new Map();

// awarenessTied says whether participant num, the roster's sender, may speak
// for awareness client id client: a member for a client id the roster
// credits to their login, a guest for the client id tied to them, or, on
// their first message, one nobody is credited with or tied to.
function awarenessTied(sender, num, client) {
  if (client === doc.clientID) return false;
  if (sender.role === "member") return typeof sender.login === "string" && sender.login !== "" && creditedTo(client) === sender.login;
  if (creditedTo(client) !== "") return false;
  const tied = guestAwareness.get(num);
  if (tied !== undefined) return tied === client;
  for (const other of guestAwareness.values()) if (other === client) return false;
  guestAwareness.set(num, client);
  return true;
}

// cursorLabel is the name a participant's cursor carries, from the roster: a
// member's login, or a guest's name as the relay stamped it, marked a guest.
function cursorLabel(sender) {
  if (sender.role === "member" && typeof sender.login === "string" && sender.login) return cleanName(sender.login);
  return cleanName(sender.name) + " (" + WORDS.guestCursor + ")";
}

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
// cleanAwarenessUpdate is bytes cleaned, each state carrying the label
// labelFor gives its client id; labelFor's null, for a client id the sender
// may not speak for, drops the whole message.
function cleanAwarenessUpdate(bytes, labelFor) {
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
    const label = labelFor(client);
    if (label === null) throw new Error("awareness");
    const clean = new TextEncoder().encode(JSON.stringify(cleanAwarenessState(client, JSON.parse(json), label)));
    writeVarUint(out, client);
    writeVarUint(out, clock);
    writeVarUint(out, clean.length);
    for (const b of clean) out.push(b);
  }
  return Uint8Array.from(out);
}
// cleanAwarenessState keeps what the editor reads, a user and a cursor: the
// colours only when each is a six-digit hex colour (otherwise the palette's,
// or the editor's own derived light colour), and the name the roster gives
// the sender, whatever name the state claims.
function cleanAwarenessState(client, state, label) {
  if (state === null || typeof state !== "object" || Array.isArray(state)) return null;
  const u = state.user !== null && typeof state.user === "object" ? state.user : {};
  const user = { name: label, color: HEX_COLOUR_RE.test(u.color) ? u.color : COLOURS[client % COLOURS.length] };
  if (HEX_COLOUR_RE.test(u.colorLight)) user.colorLight = u.colorLight;
  const c = state.cursor !== null && typeof state.cursor === "object" ? state.cursor : {};
  const anchor = cleanPosition(c.anchor);
  const head = cleanPosition(c.head);
  return { user, cursor: anchor && head ? { anchor, head } : null };
}
// cleanPosition keeps a Yjs relative position's own fields, of their types.
// The editor sends a position as Yjs builds it, with null for a field it does
// not use (tname, or item at the end of a text): null is that field's absence.
function cleanPosition(p) {
  if (p === null || typeof p !== "object") return null;
  const id = (v) => (v !== null && typeof v === "object" && Number.isSafeInteger(v.client) && v.client >= 0 &&
    Number.isSafeInteger(v.clock) && v.clock >= 0 ? { client: v.client, clock: v.clock } : null);
  const out = {};
  if (p.item != null) { out.item = id(p.item); if (!out.item) return null; }
  if (p.type != null) { out.type = id(p.type); if (!out.type) return null; }
  if (p.tname != null) { if (typeof p.tname !== "string") return null; out.tname = p.tname; }
  if (p.assoc != null) { if (!Number.isSafeInteger(p.assoc)) return null; out.assoc = p.assoc; }
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
// Sign-in through the team's GitHub App: GitHub's authorize page with PKCE,
// then the relay's exchange; or, on a local session's own page, GitHub's
// device flow through the relay. Nobody pastes a token, and the token stays
// in memory.
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
  // The relay offers GitHub's web flow when it holds the app's secret (a
  // cloud relay), or its device flow when it holds the client id alone (a
  // session on the host's own computer, where no secret may sit;
  // adr-2610161400542916). The device flow is taken only on a page the relay
  // serves itself: every local session serves its own page, so this page on
  // any other origin, abcdesign.app above all, signs in through the web flow
  // alone, and a relay naming the device flow there gets no sign-in.
  const flow = cfg.flow === "device" ? (anchor.state === "self" ? "device" : "") : "web";
  return {
    // Sign-in goes through a GitHub App alone: its user token can do only
    // what both the app (its installations and fine-grained permissions) and
    // the person may, so a relay a stranger's repository names cannot collect
    // a token for the person's other repositories, and GitHub recommends Apps
    // over OAuth apps
    // (docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app,
    // docs.github.com/en/apps/oauth-apps/building-oauth-apps/differences-between-github-apps-and-oauth-apps).
    // A relay that does not say github-app gets no sign-in at all: nobody
    // pastes a token. The page takes the relay's word for the kind: GitHub
    // documents no client-id format that tells a GitHub App's from an OAuth
    // app's, and no public API answers for a client id, so the shape is not
    // checked. The authorize request asks for no scope either way.
    signIn: cfg.app_type === "github-app" && cfg.sign_in === true && /^[A-Za-z0-9._-]{1,100}$/.test(clientID) && (flow === "device" || (flow === "web" && authorize !== "")),
    flow,
    clientID,
    authorize,
  };
}

// startSignIn signs a team member in through the relay's sign-in: GitHub's
// web flow, which leaves the page and comes back, or its device flow, which
// shows a code to confirm on github.com.
async function startSignIn() {
  if (framed()) return;
  if (!session.relay) { toast(WORDS.notSetUp); return; }
  if (!session.everJoined) readStartForm();
  // No sign-in starts through a relay the repository does not name, or, where
  // its list cannot be read, one the person has not confirmed. GitHub then
  // sends the code only to a callback the team's app lists.
  if (!relayClearedFor("signin")) return;
  let cfg;
  try { cfg = await relayConfig(); } catch { toast(WORDS.cannotConnect); return; }
  signInOffered = cfg.signIn;
  if (!cfg.signIn) {
    toast(WORDS.signInNotSetUp);
    renderAll();
    return;
  }
  if (cfg.flow === "device" && anchor.state === "self") { startDeviceSignIn(); return; }
  // GitHub returns the person to the relay's own callback, which the team's
  // app lists; the relay sends them back to this page, naming itself. The
  // state carries this page's address for that, after its random part.
  const state = randomText(16) + "." + b64url(new TextEncoder().encode(pageURL()));
  const verifier = randomText(32);
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  const pending = { state, verifier, relay: session.relay, code: session.code, name: session.name };
  if (!stashSignIn(pending)) {
    toast(WORDS.signInFailed);
    return;
  }
  const q = new URLSearchParams({
    client_id: cfg.clientID,
    redirect_uri: session.relay + CALLBACK_PATH,
    state,
    code_challenge: b64url(new Uint8Array(digest)),
    code_challenge_method: "S256",
    allow_signup: "false",
  });
  // A GitHub App's authorization takes no scope.
  location.assign(cfg.authorize + "?" + q.toString());
}

// finishSignIn handles the return from GitHub, which the relay's callback
// sends on to this page naming itself. The code goes to that relay alone:
// GitHub returned the person to it, through a callback the team's app lists.
// It never goes to a relay the invite link or this tab's storage names, and
// a return that names no relay, or another than the sign-in started with, is
// refused.
async function finishSignIn() {
  const q = new URLSearchParams(location.search);
  if (!q.has("state") || (!q.has("code") && !q.has("error"))) return false;
  const code = q.get("code") || "";
  const state = q.get("state") || "";
  const returnedVia = relayOrigin(q.get("relay"));
  const pending = takeSignIn();
  // The code and state come off the address bar before anything else.
  history.replaceState(null, "", location.pathname + location.hash);
  if (!pending || typeof pending.state !== "string" || pending.state !== state || !code) {
    toast(WORDS.signInFailed);
    return true;
  }
  if (!returnedVia || returnedVia !== relayOrigin(pending.relay)) { toast(WORDS.signInFailed); return true; }
  session.relay = returnedVia;
  session.code = normalizeCode(pending.code);
  session.name = typeof pending.name === "string" ? pending.name : "";
  if (session.code) updateFragment();
  // GitHub's code goes to the relay's exchange only while the relay is still
  // cleared for this repository.
  if (!relayCleared()) { toast(standingRefusal(relayStanding(session.relay))); renderAll(); return true; }
  setStatus(WORDS.signingIn, false);
  try {
    const res = await fetch(session.relay + "/auth/token", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code, code_verifier: pending.verifier, redirect_uri: session.relay + CALLBACK_PATH }),
      credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
    });
    const out = await res.json().catch(() => ({}));
    if (res.status === 429) { setStatus("", false); toast(WORDS.signInBusy); return true; }
    if (!res.ok || typeof out.access_token !== "string" || out.access_token === "") throw new Error("exchange");
    heldToken = out.access_token;
    tokenRelay = returnedVia;
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

// ---------------------------------------------------------------------------
// GitHub's device flow, through the relay: GitHub documents no CORS for its
// device endpoints, and the relay keeps the device code, so the page holds
// only a handle. The person confirms the code on github.com in another tab;
// the page asks the relay after it, no faster than GitHub's interval.
// ---------------------------------------------------------------------------
let device = null; // { handle, relay, timer, until }
async function relayPost(path, body) {
  const res = await fetch(session.relay + path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer",
  });
  const out = await res.json().catch(() => ({}));
  return { status: res.status, out: out && typeof out === "object" ? out : {} };
}

async function startDeviceSignIn() {
  stopDeviceSignIn(true);
  // A local session's own page alone signs in this way (see relayConfig).
  if (anchor.state !== "self") { toast(WORDS.signInNotSetUp); return; }
  let got;
  try { got = await relayPost("/auth/device", {}); } catch { toast(WORDS.cannotConnect); return; }
  const out = got.out;
  if (got.status !== 200 || typeof out.handle !== "string" || !/^[A-Za-z0-9_-]{22}$/.test(out.handle) ||
      typeof out.user_code !== "string" || !/^[A-Z0-9]{4}-[A-Z0-9]{4}$/.test(out.user_code)) {
    // Too many under way, from this address or on the relay: say so, rather
    // than that the sign-in failed or is not set up.
    if (got.status === 429 || out.error === "busy") toast(WORDS.signInBusy);
    else toast(out.error === "device_flow_disabled" || got.status === 503 ? WORDS.signInNotSetUp : WORDS.signInFailed);
    return;
  }
  const life = Math.min(Number(out.expires_in) || 900, 900) * 1000;
  device = { handle: out.handle, relay: session.relay, timer: 0, until: Date.now() + life, interval: Math.max(5, Number(out.interval) || 5) };
  ui.deviceCode.textContent = out.user_code;
  ui.device.hidden = false;
  ui.deviceOpen.focus();
  pollDevice();
}

// stopDeviceSignIn stops asking after a device sign-in; with tell, the
// person stopped it (or left the page), so the relay is told to end it and
// free its place, rather than hold it until GitHub's code expires.
function stopDeviceSignIn(tell) {
  const d = device;
  if (d) clearTimeout(d.timer);
  device = null;
  if (ui.device) ui.device.hidden = true;
  if (tell && d) {
    fetch(d.relay + "/auth/device/cancel", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ handle: d.handle }),
      credentials: "omit", cache: "no-store", referrerPolicy: "no-referrer", keepalive: true,
    }).catch(() => { /* the relay drops it when GitHub's code expires */ });
  }
}

function pollDevice() {
  const d = device;
  if (!d) return;
  d.timer = setTimeout(async () => {
    if (device !== d) return;
    if (Date.now() > d.until || d.relay !== session.relay) { stopDeviceSignIn(); toast(WORDS.deviceExpired); return; }
    let got;
    try { got = await relayPost("/auth/device/token", { handle: d.handle }); } catch { got = null; }
    if (device !== d) return;
    const out = got ? got.out : {};
    if (got && got.status === 200 && typeof out.access_token === "string" && out.access_token !== "") {
      stopDeviceSignIn();
      heldToken = out.access_token;
      tokenRelay = d.relay;
      takeRejoin();
      await presentHeldToken(false);
      return;
    }
    if (!got || got.status >= 500 || out.error === "authorization_pending" || out.error === "slow_down") {
      if (typeof out.interval === "number" && out.interval >= 5 && out.interval <= 60) d.interval = out.interval;
      pollDevice();
      return;
    }
    stopDeviceSignIn();
    toast(out.error === "access_denied" ? WORDS.deviceDenied : out.error === "expired_token" || out.error === "token_expired" ? WORDS.deviceExpired : WORDS.signInFailed);
  }, d.interval * 1000);
}

// presentHeldToken gives the held token to the relay it was signed in
// through, and to no other, once that relay is cleared. With a token, the
// page first asks api.github.com alone for the repository's relay list, which
// a private repository shows only then: a list that does not name the relay
// refuses it, and forgets any confirmation. A relay that serves this page
// itself is the page's own origin and needs no list.
async function presentHeldToken(checked) {
  if (!heldToken) return;
  const relay = session.relay;
  if (!relay || relay !== tokenRelay) { heldToken = ""; return; }
  if (!checked && anchor.state !== "listed" && anchor.state !== "self") {
    const seen = await readRelayList(heldToken);
    // A list the token shows decides. One GitHub would not show for a reason
    // other than its absence refuses the relay until it can be checked.
    if (seen.state === "listed" || seen.state === "unavailable") anchor = seen;
  }
  if (!heldToken || relay !== session.relay) { heldToken = ""; return; }
  const standing = relayStanding(relay);
  if (standing === "refused" || standing === "unavailable") {
    if (standing === "refused") forgetRelay(relay);
    heldToken = "";
    toast(standingRefusal(standing));
    renderAll();
    return;
  }
  if (standing === "unconfirmed") { askTrust("token"); return; }
  if (standing !== "listed" && standing !== "confirmed" && standing !== "self") { heldToken = ""; return; }
  githubToken = heldToken;
  heldToken = "";
  connectSignedIn();
}

// connectSignedIn presents the token: on the open connection when there is
// one, or on a new one.
function connectSignedIn() {
  if (!githubToken || !relayTokenCleared()) { githubToken = ""; return; }
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
// Which relay, and whether it may have anything (security review F1). The
// page never shows or asks for a relay address (adr-2610161400542916). A page
// the relay serves itself takes its own origin. Otherwise a link can name any
// relay in its fragment, so the repository is the trust anchor: it names its
// relays in a committed file the page reads from api.github.com, without a
// token for a public repository. Where that file cannot be read, the person
// confirms, in plain words, that the project's team sent the link, before any
// sign-in, name or suggestion reaches the relay.
// ---------------------------------------------------------------------------
// The anchor: "self" when the relay serves this page; "checking" until GitHub
// answers; "listed" with the relays the file names (none, for a file that
// names none the page can use); "unknown" when GitHub says there is no file it
// will show (404: none, or a private repository); or "unavailable" when GitHub
// did not answer that (a 403, a 429, a 5xx, a network error): then the list
// could not be checked, and nothing reaches the relay, confirmed or not,
// until it can be.
let anchor = { state: "checking", relays: [] };

// selfServed says the relay serves this page itself, a local session: its
// HTML then carries the marker, which the static page never does.
function selfServed() {
  const m = document.querySelector('meta[name="abcd-session-relay"]');
  return m !== null && m.getAttribute("content") === "self";
}

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
    if (res.status === 404) return { state: "unknown", relays: [] };
    if (res.status !== 200) return { state: "unavailable", relays: [] };
    return { state: "listed", relays: relaysIn(await res.json()) };
  } catch {
    return { state: "unavailable", relays: [] };
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

// chooseRelay settles which relay this page uses, once the anchor is read: a
// page the relay serves uses its own origin; otherwise the relay the link
// names, or, with none, the first the repository's list names, or, where the
// list cannot be read, the one the person confirmed for this repository.
function chooseRelay(fromLink) {
  if (anchor.state === "self") return location.origin;
  if (fromLink) return fromLink;
  if (anchor.state === "listed") return anchor.relays[0] || "";
  if (anchor.state === "unknown") return confirmedRelays()[0] || "";
  return "";
}

// relayStanding is what the page may do with a relay: "self", "listed" or
// "confirmed" may have a token, "unconfirmed" must be confirmed first,
// "refused" and "unavailable" get nothing and no confirmation is offered for
// them, "checking" waits for GitHub's answer.
function relayStanding(relay) {
  if (!relay) return "";
  if (anchor.state === "self") return relay === location.origin ? "self" : "refused";
  if (anchor.state === "checking") return "checking";
  if (anchor.state === "unavailable") return "unavailable";
  if (anchor.state === "listed") return anchor.relays.includes(relay) ? "listed" : "refused";
  return confirmedRelays().includes(relay) ? "confirmed" : "unconfirmed";
}
function relayCleared() {
  const standing = relayStanding(session.relay);
  return standing === "self" || standing === "listed" || standing === "confirmed";
}
// relayTokenCleared says the token may go to the relay: it is cleared, and
// it is the relay the token was signed in through. A relay an invite link
// names, which GitHub did not return the person to, never has the token.
function relayTokenCleared() {
  return relayCleared() && session.relay !== "" && session.relay === tokenRelay;
}
// standingRefusal says, in plain words and with no address, why a relay that
// is not cleared gets nothing.
function standingRefusal(standing) {
  if (standing === "checking") return WORDS.checking;
  if (standing === "unavailable") return WORDS.cannotCheck;
  if (standing === "refused") return WORDS.notOurLink;
  return WORDS.notSetUp;
}

// recheckRelayList reads the relay list again after GitHub did not answer,
// so the person can try again without reloading.
let rechecking = false;
async function recheckRelayList() {
  if (rechecking || anchor.state !== "unavailable") return;
  rechecking = true;
  try {
    const seen = await readRelayList("");
    if (anchor.state === "unavailable") anchor = seen;
    if (!session.relay) session.relay = chooseRelay(linkRelay);
  } finally {
    rechecking = false;
    renderAll();
    refreshSignIn();
  }
}

// relayClearedFor reports whether the relay is cleared; when it is not, it
// asks the person to confirm it (for a relay the list cannot speak for) or
// says why it is refused.
function relayClearedFor(action) {
  const standing = relayStanding(session.relay);
  if (standing === "self" || standing === "listed" || standing === "confirmed") return true;
  if (standing === "unconfirmed") askTrust(action);
  else {
    heldToken = "";
    toast(standingRefusal(standing));
    if (standing === "unavailable") recheckRelayList();
  }
  renderAll();
  return false;
}

let trustPending = null;
function askTrust(action) {
  if (framed()) return;
  trustPending = { action, relay: session.relay };
  // The project this page is for, shown so a link to a stranger's project is
  // seen for what it is. No address is shown: the question is whether the
  // project's team sent the link.
  clear(ui.trustOwner);
  ui.trustOwner.append(WORDS.pageFor + " ", el("strong", { text: route.slug }), ".");
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
    renderAll();
    return;
  }
  confirmRelay(p.relay);
  renderAll();
  refreshSignIn();
  if (p.action === "signin") startSignIn();
  else if (p.action === "token") presentHeldToken(true);
  else if (p.action === "join") joinClicked();
  else if (p.action === "host") hostClicked();
}

// ownerNotice names the repository beside the sign-in, prominently: a link
// can open the page for any repository, a stranger's among them, and that
// repository's list then names the stranger's relay.
function ownerNotice() {
  return el("span", { class: "owner" }, WORDS.signingInTo + " ", el("strong", { text: route.slug }), ".");
}

// signInOffered is what the relay's /config said: true for a GitHub App's
// sign-in, false for none, null before a cleared relay was asked. Only false
// takes the sign-in buttons away; before the relay is asked, signing in is
// what confirms it.
let signInOffered = null;
async function refreshSignIn() {
  if (!relayCleared()) return;
  let offered = false;
  try { offered = (await relayConfig()).signIn; } catch { offered = false; }
  signInOffered = offered;
  renderStart();
  renderTop();
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

// inviteFragment is what the invite link carries after #: the code, and the
// relay only when a guest's page could not find the same one itself. A page
// the relay serves is its own relay, and the first relay the repository's
// list names is the one a page finds with no relay in the link.
function inviteFragment() {
  const own = anchor.state === "self" || (anchor.state === "listed" && anchor.relays[0] === session.relay);
  return (own ? "" : "relay=" + encodeURIComponent(session.relay) + "&") + "code=" + session.code;
}
function inviteURL() {
  return location.origin + route.base + "/" + encodeURIComponent(route.owner) + "/" + encodeURIComponent(route.repo) +
    "/brief/#" + inviteFragment();
}
function updateFragment() {
  if (!session.code || !session.relay) return;
  history.replaceState(null, "", location.pathname + "#" + inviteFragment());
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

// sendSuggestion sends a guest's suggestion and reports whether it went.
function sendSuggestion(replacement) {
  if (!view || !currentPath) return false;
  const ytext = files.get(currentPath);
  const sel = view.state.selection.main;
  if (!(ytext instanceof Y.Text) || sel.empty) { toast(WORDS.selectFirst); return false; }
  if (replacement.length > MAX_REPLACEMENT || sel.to - sel.from > MAX_REPLACEMENT) { toast(WORDS.suggestTooBig); return false; }
  const text = ytext.toString();
  const body = {
    path: currentPath,
    anchor: Y.relativePositionToJSON(Y.createRelativePositionFromTypeIndex(ytext, sel.from)),
    head: Y.relativePositionToJSON(Y.createRelativePositionFromTypeIndex(ytext, sel.to)),
    quote: quoteOf(text, sel.from, sel.to),
    replacement,
  };
  // The relay refuses a suggestion over its size; say so here instead.
  if (utf8Length(JSON.stringify(body)) > MAX_SUGGEST_BYTES) { toast(WORDS.suggestTooBig); return false; }
  if (suggestionInFlight) { toast(WORDS.suggestWait); return false; }
  suggestionInFlight = body;
  clearTimeout(suggestTimer);
  suggestTimer = setTimeout(suggestionTimedOut, SUGGEST_TIMEOUT);
  sendJSON({ type: "suggest", body });
  return true;
}

// suggestionTimedOut gives up on a suggestion the relay has not answered:
// the guest is told it may not have reached the team.
function suggestionTimedOut() {
  if (!suggestionInFlight) return;
  suggestionInFlight = null;
  toast(WORDS.suggestLost);
  renderSide();
}

function onSuggested(msg) {
  const body = suggestionInFlight;
  suggestionInFlight = null;
  clearTimeout(suggestTimer);
  if (!body) return;
  mine.push({ id: String(msg.id || ""), body });
  toast(msg.delivered > 0 ? WORDS.suggestionSent : WORDS.noMembers);
  renderSide();
}

// validSuggestion checks a suggestion's fields, each of its type and bounded:
// a path among the brief's files the session started with, a passage and a
// replacement of at most MAX_REPLACEMENT characters, and the passage's
// context at most MAX_QUOTE_PART each side.
function validSuggestion(b) {
  if (!b || typeof b !== "object" || !b.quote || typeof b.quote !== "object") return false;
  if (typeof b.path !== "string" || typeof b.replacement !== "string" || typeof b.quote.exact !== "string" ||
      typeof b.quote.prefix !== "string" || typeof b.quote.suffix !== "string") return false;
  const base = baseSnapshot();
  return b.quote.exact !== "" && b.quote.exact.length <= MAX_REPLACEMENT && b.replacement.length <= MAX_REPLACEMENT &&
    b.quote.prefix.length <= MAX_QUOTE_PART && b.quote.suffix.length <= MAX_QUOTE_PART &&
    briefFile(b.path, baseFilePaths(base));
}

// boundedPosition is a relative position as the page keeps it: its own
// fields, of their types, at most MAX_POSITION_JSON as JSON, or null.
function boundedPosition(p) {
  const clean = cleanPosition(p);
  return clean && JSON.stringify(clean).length <= MAX_POSITION_JSON ? clean : null;
}

// storedSuggestion is what a member's page records of a guest's suggestion:
// the fields it is made of, each validated and bounded, and nothing else; or
// null for one that is not a suggestion, or is larger than the relay passes.
// The relay caps the body alone, so the body alone is measured: the id and
// the name added here must not push a suggestion it passed over the cap.
function storedSuggestion(id, from, b) {
  if (!SUGGESTION_ID_RE.test(id) || !validSuggestion(b)) return null;
  const quote = { exact: b.quote.exact, prefix: b.quote.prefix, suffix: b.quote.suffix };
  const out = {
    id, from: cleanName(from),
    body: { path: b.path, anchor: boundedPosition(b.anchor), head: boundedPosition(b.head), quote, replacement: b.replacement },
  };
  return utf8Length(JSON.stringify(out.body)) > MAX_SUGGEST_BYTES ? null : out;
}

function onSuggest(msg) {
  if (session.role !== "member") return;
  const id = String(msg.id || "");
  const from = typeof msg.from === "string" && msg.from ? msg.from : "Guest";
  const stored = storedSuggestion(id, from, msg.body);
  if (!stored) return;
  // Every member's page sees the suggestion; the first to record it wins, and
  // the others find it there already.
  if (!suggestions.has(id) && !suggestionsDone.has(id)) {
    doc.transact(() => { suggestions.set(id, stored); });
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

// ---------------------------------------------------------------------------
// The session's settings. Any member's page can write the shared meta map,
// for instance to point the pull request at another branch, so a value counts
// only when the Yjs item holding it was written by a client id the roster
// credits to the host's login. The base, the branch and the repository are
// written once, when the host's page seeds the session: the host's first
// write of each counts and every later write is void. The mode may change,
// but only from the host: the host's latest write counts. gc is off, so an
// overwritten value's item keeps its content and its writer.
// ---------------------------------------------------------------------------
const META_ONCE = ["base_snapshot", "base_sha", "branch", "repo"];
const META_KEYS = META_ONCE.concat(["mode"]);

// effectiveMeta is the settings as the host wrote them, and in voided each
// void write as { key, client, clock }. A write whose writer the roster does
// not credit yet (this page's own, before the roster names it) is neither
// counted nor void. Without the host's login, nothing is known.
function effectiveMeta() {
  const out = { voided: [] };
  const host = session.hostLogin;
  if (!host) return out;
  for (const key of META_KEYS) {
    // The key's items, oldest first: each new value's item goes to the right
    // of the one it replaced.
    const items = [];
    for (let item = meta._map.get(key); item; item = item.left) items.unshift(item);
    let chosen = null;
    for (const item of items) {
      const writer = creditedTo(item.id.client);
      if (writer === host && (chosen === null || !META_ONCE.includes(key))) chosen = item;
      else if (writer !== "") out.voided.push({ key, client: item.id.client, clock: item.id.clock });
    }
    if (chosen) out[key] = chosen.content.getContent()[0];
  }
  return out;
}

// watchMeta warns once for each void write to the settings it has not seen.
const voidSeen = new Set();
function watchMeta() {
  for (const v of effectiveMeta().voided) {
    const id = v.client + ":" + v.clock;
    if (voidSeen.has(id)) continue;
    voidSeen.add(id);
    const who = loginOf(v.client);
    warn(WORDS.metaVoid + " " + (who ? "@" + who : WORDS.someone) + ".");
  }
}
meta.observe(() => {
  watchMeta();
  renderSide();
});

let baseCache = { raw: null, snap: null };
function baseSnapshot() {
  const raw = effectiveMeta().base_snapshot;
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
// show formats a guest's name for where the lines go: as it is for the page,
// mdName for an issue or a pull request body.
function creditLines(p, show = (s) => s) {
  const lines = [];
  if (p.authors.length) lines.push("Written by " + p.authors.map((l) => "@" + l).join(", "));
  for (const a of p.approvals) lines.push("Suggested by " + show(a.guest) + " (guest), approved by @" + a.approver);
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

// COMMIT_RE is a commit id as an issue or a pull request names the base: the
// forty hex digits of a SHA-1 commit.
const COMMIT_RE = /^[0-9a-f]{40}$/;

// baseCommit is the base commit as the host seeded it, or "unknown" for
// anything that is not a commit id: it goes into a Markdown body.
function baseCommit() {
  const sha = effectiveMeta().base_sha;
  return typeof sha === "string" && COMMIT_RE.test(sha) ? sha : "unknown";
}

// mdName makes a guest's name inert in a Markdown body: it goes in a code
// span, after trailerText's cleaning (no line breaks, no angle brackets, no
// @). Escaping alone is not enough on GitHub, which autolinks a bare URL or
// domain (evil.example/x, www.example, a full web address) whatever is
// escaped around it; inside a code span nothing is a link, a mention, an
// image or emphasis. The fence is longer than any run of backticks in the
// name, and a name that starts or ends with a backtick is padded with a
// space each side, which the span drops again.
function mdName(s) {
  const name = trailerText(s);
  let longest = 0;
  for (const m of name.matchAll(/`+/g)) longest = Math.max(longest, m[0].length);
  const fence = "`".repeat(longest + 1);
  const pad = name.startsWith("`") || name.endsWith("`") ? " " : "";
  return fence + pad + name + pad + fence;
}

function issueTitle(p) {
  return "Brief: " + relPath(p.path) + ", " + linesLabel(p);
}

function issueBody(p) {
  return [
    "A change to the brief, proposed in a live session.",
    "",
    "- File: `" + p.path + "`",
    "- Base commit: " + baseCommit(),
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
    ...creditLines(p, mdName),
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
  for (const p of pairs) people.add(mdName(p.guest) + " (guest)");
  for (const r of session.roster) {
    if (r.role === "member" && typeof r.login === "string" && NAME_RE.test(r.login)) people.add("@" + r.login);
    else if (r.role !== "member" && typeof r.name === "string" && r.name) people.add(mdName(trailerText(r.name)) + " (guest)");
  }
  const out = [
    "Edits to the brief made together in a live session, from base commit " + baseCommit() + ".",
    "",
    "Participants: " + [...people].join(", "),
    "",
    "Changed passages:",
  ];
  for (const p of list) out.push("- `" + relPath(p.path) + "`, " + linesLabel(p) + ": " + creditLines(p, mdName).join("; "));
  if (pairs.length) {
    out.push("", "Approved guest suggestions:");
    for (const p of pairs) out.push("- " + mdName(p.guest) + " (guest), approved by @" + p.approver);
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
  // The base commit, the branch the pull request goes into and the
  // repository, as the host seeded them; nobody else's write counts.
  const settings = effectiveMeta();
  const baseSha = settings.base_sha;
  const target = settings.branch;
  if (!SHA_RE.test(baseSha || "") || typeof target !== "string" || !target ||
      typeof settings.repo !== "string" || settings.repo.toLowerCase() !== route.slug.toLowerCase()) {
    showSendResult(WORDS.prNoBase, true);
    return;
  }
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
  applyLayout();
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

function closeSend() {
  ui.send.hidden = true;
  applyLayout();
}

function renderSend() {
  if (!ui.send || ui.send.hidden) return;
  if (session.role !== "member" || session.over) { closeSend(); return; }
  const mode = effectiveMeta().mode === "issues" ? "issues" : "pr";
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
      icon(fileIcon(p.path)),
      el("div", { class: "passage-body" },
        el("div", { class: "from", text: relPath(p.path) + " · " + linesLabel(p) }),
        el("div", { class: "credit", text: creditLines(p).join("\n") }),
        el("del", { text: p.before }),
        el("ins", { text: p.after }),
        actions.childElementCount ? actions : null)));
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
    el("button", { type: "button", class: "back", text: WORDS.backToSession, onclick: () => closeSend() }));
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
  ui.filetitle.textContent = prettyName(path) + " · " + relPath(path);
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
// firstFile is the file a session opens on: the brief's press release, the
// file whose name holds "press-release", or else the first file.
function firstFile() {
  const paths = [...files.keys()].sort();
  return paths.find((p) => /press-release/i.test(p.slice(p.lastIndexOf("/") + 1))) || paths[0] || "";
}
files.observe(() => {
  renderFiles();
  if (!currentPath || !files.has(currentPath)) {
    const first = firstFile();
    if (first && session.synced) openFile(first);
  }
});

// ---------------------------------------------------------------------------
// The two designs. Guided (the default) and Shared are one page, one set of
// elements and one set of logic: the design is an attribute the stylesheet
// reads, and a few parts (the documents, the people, the suggestions, the
// invite, the editor) move into the design's own places. Ctrl+[ and Ctrl+]
// switch on every platform, the Mac included; Alt+Shift+D and Alt+Shift+S
// show and hide the two sidebars; Escape closes the overlay on top.
// ---------------------------------------------------------------------------
let design = DESIGNS[0];
// The sidebars: shown or hidden on a wide screen; on a phone, the sheets
// that are open (they start closed). Kept here, outside any screen, so it
// lasts through every change of screen, width and design.
const layout = { left: true, right: true, pleft: false, pright: false, sheet: "", invite: false };

// storedDesign is the design this viewer chose last, or the default.
function storedDesign() {
  const d = recall(KEY_DESIGN);
  return DESIGNS.includes(d) ? d : DESIGNS[0];
}
function switchDesign(step) {
  const i = DESIGNS.indexOf(design);
  design = DESIGNS[(i + step + DESIGNS.length) % DESIGNS.length];
  remember(KEY_DESIGN, design);
  applyLayout();
  toast(design === "shared" ? WORDS.designShared : WORDS.designGuided);
}

// shortcut names what a key press asks for: the design switch on Ctrl+[ and
// Ctrl+] (never Cmd), a sidebar on Alt+Shift+D or Alt+Shift+S (read from the
// key's place, so a Mac's Option characters do not matter), or Escape.
function shortcut(e) {
  if (e.key === "Escape") return "escape";
  if (e.metaKey) return "";
  if (e.ctrlKey && !e.altKey) {
    if (e.code === "BracketLeft" || e.key === "[") return "design-prev";
    if (e.code === "BracketRight" || e.key === "]") return "design-next";
    return "";
  }
  if (e.altKey && e.shiftKey && !e.ctrlKey) {
    if (e.code === "KeyD") return "left";
    if (e.code === "KeyS") return "right";
  }
  return "";
}
function onKey(e) {
  const action = shortcut(e);
  if (!action) return;
  if (action === "escape") {
    if (closeTop()) { e.preventDefault(); e.stopPropagation(); }
    return;
  }
  if ((action === "left" || action === "right") && !inWorkspace()) return;
  e.preventDefault();
  e.stopPropagation();
  if (action === "design-prev") switchDesign(-1);
  else if (action === "design-next") switchDesign(1);
  else toggleSide(action);
}

function viewWidth() {
  return document.documentElement.clientWidth || window.innerWidth || 0;
}
function inWorkspace() {
  return !!ui.work && !ui.work.hidden;
}

// closeTop closes the overlay on top, and says whether there was one: the
// relay confirmation (which is then refused), the Send panel, the invite
// card, the guest's suggestion popover, a phone's sheet, then a slide-over.
function closeTop() {
  if (ui.trust && !ui.trust.hidden) { trustAnswered(false); return true; }
  if (ui.send && !ui.send.hidden) { ui.send.hidden = true; applyLayout(); return true; }
  if (layout.invite) { layout.invite = false; applyLayout(); return true; }
  if (design === "shared" && ui.suggestForm && !ui.suggestForm.hidden) { ui.suggestForm.hidden = true; applyLayout(); return true; }
  if (!inWorkspace()) return false;
  const w = viewWidth();
  if (w <= PHONE_MAX) {
    if (layout.sheet) { layout.sheet = ""; applyLayout(); return true; }
    if (layout.pleft || layout.pright) { layout.pleft = false; layout.pright = false; applyLayout(); return true; }
    return false;
  }
  if (w <= SLIDE_MAX[design] && layout.right) { layout.right = false; applyLayout(); return true; }
  return false;
}

// defaultSheet is the right sidebar's first panel on a guided phone.
function defaultSheet() {
  return session.role === "member" ? "sugs" : "suggest";
}
function toggleSide(side) {
  const phone = viewWidth() <= PHONE_MAX;
  if (phone && design === "shared") {
    if (side === "left") { layout.pleft = !layout.pleft; if (layout.pleft) layout.pright = false; }
    else { layout.pright = !layout.pright; if (layout.pright) layout.pleft = false; }
  } else if (phone && side === "right") {
    layout.sheet = layout.sheet ? "" : defaultSheet();
  } else {
    layout[side] = !layout[side];
  }
  applyLayout();
}
function openSheet(name) {
  layout.sheet = name;
  applyLayout();
}

// sideShown says whether a sidebar shows, at this width and in this design.
function sideShown(side) {
  if (viewWidth() > PHONE_MAX) return layout[side];
  if (design === "shared") return side === "left" ? layout.pleft : layout.pright;
  return side === "left" ? layout.left : !!layout.sheet;
}

// applyLayout writes the design and the sidebars' state where the
// stylesheet reads it, names the toggles for what they would do, and moves
// the shared parts into the design's places.
function applyLayout() {
  document.documentElement.dataset.design = design;
  if (!ui.app) return;
  const L = sideShown("left");
  const R = sideShown("right");
  ui.app.dataset.left = L ? "shown" : "hidden";
  ui.app.dataset.right = R ? "shown" : "hidden";
  ui.app.dataset.sheet = viewWidth() <= PHONE_MAX && design === "guided" ? layout.sheet : "";
  ui.app.dataset.overlay = !ui.send.hidden ? "send" : layout.invite ? "invite" : "";
  const rightName = session.role === "guest" ? WORDS.suggestFollow : WORDS.reviewPeople;
  const sharedRight = session.role === "guest" ? WORDS.mySuggestionsHeading : WORDS.sugShort;
  const leftTip = L ? WORDS.hideDocuments : WORDS.showDocuments;
  const rightTip = (R ? WORDS.hidePanel : WORDS.showPanel) + " " + (design === "shared" ? sharedRight : rightName).toLowerCase();
  for (const b of [ui.gLeft, ui.sLeft]) {
    b.setAttribute("aria-pressed", String(L));
    b.setAttribute("aria-label", leftTip);
  }
  for (const b of [ui.gRight, ui.sRight]) {
    b.setAttribute("aria-pressed", String(R));
    b.setAttribute("aria-label", rightTip);
  }
  ui.gLeft.title = leftTip + " (Alt+Shift+D)";
  ui.gRight.title = rightTip + " (Alt+Shift+S)";
  ui.sLeft.dataset.tip = leftTip;
  ui.sRight.dataset.tip = rightTip;
  ui.gRightLabel.textContent = rightName;
  ui.sRightLabel.textContent = sharedRight;
  ui.gSideTitle.textContent = rightName;
  ui.inviteToggle.setAttribute("aria-expanded", String(layout.invite));
  for (const box of [ui.suggestBox, ui.sugBox, ui.mineBox, ui.rosterBox, ui.inviteBox]) {
    box.classList.toggle("is-open", !!layout.sheet && box.dataset.panel === layout.sheet);
  }
  for (const tab of ui.tabs) tab.setAttribute("aria-pressed", String((tab.dataset.sheet || "") === layout.sheet));
  placeParts();
  renderTop();
  fitTiles();
  // The invite card opens under its button.
  if (layout.invite && design === "shared" && viewWidth() > PHONE_MAX) {
    const r = ui.inviteToggle.getBoundingClientRect();
    ui.sShare.style.setProperty("--share-r", Math.max(12, Math.round(viewWidth() - (r.left + r.width / 2) - 90)) + "px");
  }
}

// placeParts moves the parts both designs show into this design's places.
let placedFor = "";
function placeParts() {
  if (placedFor === design) return;
  placedFor = design;
  if (design === "shared") {
    ui.sDocs.append(ui.files);
    ui.sPeople.append(ui.rosterBox);
    ui.sInsp.append(ui.notesBox, ui.sugBox, ui.mineBox);
    ui.sShare.append(ui.inviteBox);
    ui.sPop.append(ui.suggestBox);
    ui.sDoc.append(ui.page);
  } else {
    ui.gDocs.append(ui.files);
    ui.gSide.append(ui.notesBox, ui.suggestBox, ui.sugBox, ui.mineBox, ui.rosterBox, ui.inviteBox);
    ui.gPaper.append(ui.page);
  }
  if (view) view.requestMeasure();
}

// fitTiles folds the people in the shared team bar into one summary when
// they do not fit whole: a tile is never cut.
function fitTiles() {
  if (!ui.hud) return;
  ui.hud.classList.remove("crowded");
  if (design === "shared" && ui.tiles.scrollWidth > ui.tiles.clientWidth + 1) ui.hud.classList.add("crowded");
}

// ---------------------------------------------------------------------------
// Rendering.
// ---------------------------------------------------------------------------
const ui = {};
let statusText = "";
let statusBad = false;
function setStatus(text, bad) {
  statusText = text;
  statusBad = bad;
  paintStatus();
}
// paintStatus writes the status into each place a design shows it.
function paintStatus() {
  if (!ui.statuses) return;
  for (const s of ui.statuses) {
    s.textContent = statusText;
    s.classList.toggle("bad", statusBad);
  }
  const n = session.roster.length;
  const live = !statusBad && (statusText === WORDS.connected || !statusText);
  ui.liveDot.textContent = live ? say(WORDS.liveHere, { n }) : statusText;
  ui.liveDot.classList.toggle("bad", !live);
  ui.hudStatus.textContent = live ? WORDS.connected : statusText;
  ui.hudLive.classList.toggle("bad", !live);
  ui.hudCount.textContent = say(WORDS.hereCount, { n });
}

// say puts values into a word's {placeholders}; the result is shown with
// textContent like any other text.
function say(word, values) {
  return word.replace(/\{([a-z]+)\}/g, (m, k) => (Object.prototype.hasOwnProperty.call(values, k) ? String(values[k]) : m));
}

// The page's icons, as path data: [d] is stroked, [d, opacity] filled.
const ICONS = Object.freeze({
  doc: ["M7 3h7l5 5v11a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z", "M14 3v5h5M8.5 12.5h7M8.5 16h7"],
  faq: ["M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v8a2.5 2.5 0 0 1-2.5 2.5H11l-4.5 3.5V17A2.5 2.5 0 0 1 4 14.5z", "M10 8.7a2 2 0 1 1 2.6 1.9c-.4.2-.6.5-.6.9v.4M12 14.2h.01"],
  person: ["M12 11.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7z", "M5 20c.7-3.6 3.5-6 7-6s6.3 2.4 7 6"],
  people: ["M9 11.7a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4z", "M3.5 19c.5-3.2 2.7-5 5.5-5s5 1.8 5.5 5", "M16.8 11.8a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z", "M15.6 14.2c2.6-.3 4.6 1.3 5 4.3"],
  noentry: ["M12 20.5a8.5 8.5 0 1 0 0-17 8.5 8.5 0 0 0 0 17z", "M6 6l12 12"],
  panelLeft: [["M5 5.5h4.5v13H5a1.5 1.5 0 0 1-1.5-1.5V7A1.5 1.5 0 0 1 5 5.5z", ".28"], "M6 5.5h12a2.5 2.5 0 0 1 2.5 2.5v8a2.5 2.5 0 0 1-2.5 2.5H6A2.5 2.5 0 0 1 3.5 16V8A2.5 2.5 0 0 1 6 5.5z", "M9.5 5.5v13"],
  panelRight: [["M14.5 5.5H19A1.5 1.5 0 0 1 20.5 7v10a1.5 1.5 0 0 1-1.5 1.5h-4.5z", ".28"], "M6 5.5h12a2.5 2.5 0 0 1 2.5 2.5v8a2.5 2.5 0 0 1-2.5 2.5H6A2.5 2.5 0 0 1 3.5 16V8A2.5 2.5 0 0 1 6 5.5z", "M14.5 5.5v13"],
  invite: ["M10 11.9a3.4 3.4 0 1 0 0-6.8 3.4 3.4 0 0 0 0 6.8z", "M3.5 19.5c.5-3.4 3-5.4 6.5-5.4 1.4 0 2.6.3 3.6.9M18 13.5v6M15 16.5h6"],
  send: ["M20.5 3.5L10.2 13.8", "M20.5 3.5l-6.3 17-4-6.7-6.7-4z"],
  check: ["M5 12.5l4.5 4.5L19 7.5"],
  x: ["M6.5 6.5l11 11M17.5 6.5l-11 11"],
  copy: ["M11 8.5h6a2.5 2.5 0 0 1 2.5 2.5v6a2.5 2.5 0 0 1-2.5 2.5h-6A2.5 2.5 0 0 1 8.5 17v-6A2.5 2.5 0 0 1 11 8.5z", "M15.5 8.5V6.5a2 2 0 0 0-2-2h-7a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h2"],
  pencil: ["M4.5 19.5l1-4.2L15.8 5a2 2 0 0 1 2.9 0l.3.3a2 2 0 0 1 0 2.9L8.7 18.5z", "M13.8 7l3.2 3.2"],
  eye: ["M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12z", "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"],
  bubble: ["M4 6.5A2.5 2.5 0 0 1 6.5 4h11A2.5 2.5 0 0 1 20 6.5v8a2.5 2.5 0 0 1-2.5 2.5H11l-4.5 3.5V17A2.5 2.5 0 0 1 4 14.5z", "M8 9h8M8 12.5h5"],
  right: ["M9 5l7 7-7 7"],
  out: ["M9 5h10v10M19 5L6 18"],
  brief: ["M7 3h7l5 5v11a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z", "M14 3v5h5", "M8.5 13.5l2 2 4-4.5"],
  laptop: ["M6 5h12a1.5 1.5 0 0 1 1.5 1.5v9h-15v-9A1.5 1.5 0 0 1 6 5z", "M2.5 19h19"],
  hand: ["M8 12V6a1.5 1.5 0 0 1 3 0v5M11 11V4.5a1.5 1.5 0 0 1 3 0V11M14 11V6a1.5 1.5 0 0 1 3 0v7c0 4-2.5 7.5-6.5 7.5-2.5 0-4-1.3-5.5-3.5L3.6 14a1.5 1.5 0 0 1 2.4-1.8L8 14.5"],
  clock: ["M12 20.5a8.5 8.5 0 1 0 0-17 8.5 8.5 0 0 0 0 17z", "M12 7.5V12l3 2"],
  keypad: ["M5 6h14a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2z", "M7 10h.01M11 10h.01M15 10h.01M7 14h10"],
  shield: ["M12 3l8 3v6c0 4.5-3.4 8.3-8 9-4.6-.7-8-4.5-8-9V6z", "M8.5 12l2.5 2.5 4.5-5"],
  next: ["M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18z", "M12 8l4 4-4 4M8 12h8"],
  refresh: ["M19.5 12a7.5 7.5 0 1 1-2.2-5.3M19.5 4.5v4h-4"],
  github: [["M12 2a10 10 0 0 0-3.16 19.49c.5.09.68-.22.68-.48v-1.7c-2.78.6-3.37-1.34-3.37-1.34-.45-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.9 1.52 2.34 1.08 2.91.83.09-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.94 0-1.09.39-1.98 1.03-2.68-.1-.25-.45-1.27.1-2.65 0 0 .84-.27 2.75 1.02a9.6 9.6 0 0 1 5 0c1.91-1.29 2.75-1.02 2.75-1.02.55 1.38.2 2.4.1 2.65.64.7 1.03 1.59 1.03 2.68 0 3.84-2.34 4.69-4.57 4.93.36.31.68.92.68 1.85v2.75c0 .27.18.58.69.48A10 10 0 0 0 12 2z", "1"]],
});
function icon(name, cls) {
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("class", cls ? "i " + cls : "i");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  for (const part of ICONS[name] || []) {
    const path = document.createElementNS(SVG_NS, "path");
    if (Array.isArray(part)) {
      path.setAttribute("d", part[0]);
      path.setAttribute("class", "fill");
      path.setAttribute("opacity", part[1]);
    } else {
      path.setAttribute("d", part);
    }
    svg.append(path);
  }
  return svg;
}
// fileIcon picks a document's icon from its name.
function fileIcon(path) {
  const name = path.slice(path.lastIndexOf("/") + 1).toLowerCase();
  if (/faq|question/.test(name)) return "faq";
  if (/who|audience|user|customer|people/.test(name)) return "person";
  if (/scope|non-goal|not-/.test(name)) return "noentry";
  return "doc";
}
// prettyName is a document's name as a reader says it: "01-press-release.md"
// is "Press release".
function prettyName(path) {
  const name = path.slice(path.lastIndexOf("/") + 1);
  const words = name.replace(/\.md$/i, "").replace(/^[0-9]+[-_. ]*/, "").replace(/[-_]+/g, " ").trim();
  if (/^faq$/i.test(words)) return "FAQ";
  return words ? words.charAt(0).toUpperCase() + words.slice(1) : name;
}
// avatar is a person's initial on their colour. The colour comes from the
// page's own list, never from another participant.
function avatar(name, id, cls) {
  const a = el("span", { class: cls ? "av " + cls : "av", "aria-hidden": "true", text: ([...String(name || "").trim()][0] || "?").toUpperCase() });
  const n = Number(String(id || "").replace(/^p/, "")) || 0;
  a.style.setProperty("--pc", COLOURS[n % COLOURS.length]);
  return a;
}
// closeButton is every overlay's Close: an X, top right, named "Close".
function closeButton(onclick) {
  return el("button", { type: "button", class: "close", "aria-label": WORDS.close, "data-tip": WORDS.close, "data-key": "Esc", onclick }, icon("x"));
}
// areaHead is a guided card's numbered title, with the phone sheet's Done.
function areaHead(text, extra) {
  return el("div", { class: "card-head g-only" },
    el("h2", { class: "area-title" }, el("span", { class: "num", "aria-hidden": "true" }), el("span", { text }), extra || null),
    el("button", { type: "button", class: "btn small sheet-close", text: WORDS.done, onclick: () => openSheet("") }));
}
// nextSteps is the guided design's "What happens next" as numbered steps.
function nextSteps(steps) {
  return el("ol", { class: "steps" }, ...steps.map(([title, hint, state]) =>
    el("li", { class: state || "" }, el("div", {}, el("strong", { text: title }), hint ? el("span", { text: hint }) : null))));
}

function build() {
  const app = document.getElementById("app");
  clear(app);
  ui.toast = el("div", { class: "toast", role: "status", "aria-live": "polite", hidden: true });
  if (!route) {
    app.append(el("p", { class: "notice", text: WORDS.wrongAddress }), ui.toast);
    return false;
  }
  ui.app = app;
  app.classList.add("app");
  design = storedDesign();
  document.title = route.slug + " · " + WORDS.title;
  const repo = route.repo;

  // Where the status shows: the guided app bar, the shared team bar's Live
  // and, before a session, under the shared page's title.
  ui.status = el("span", { class: "status g-only" });
  ui.heroStatus = el("p", { class: "status s-only", role: "status" });
  ui.statuses = [ui.status, ui.heroStatus];
  ui.liveDot = el("span", { class: "live-dot", role: "status" });
  ui.hudStatus = el("b");
  ui.hudCount = el("small");
  ui.hudLive = el("div", { class: "hud-live" }, el("span", { class: "dot", "aria-hidden": "true" }), el("span", {}, ui.hudStatus, ui.hudCount));

  ui.signInTop = el("button", { type: "button", class: "btn small", text: WORDS.signIn, onclick: () => startSignIn() });
  ui.submit = el("button", { type: "button", class: "btn primary tb-send tip-r", "aria-label": WORDS.submit, "data-tip": WORDS.sendTip, "data-key": "", onclick: () => submitSession() },
    icon("send"), el("span", { class: "lbl", "aria-hidden": "true", text: WORDS.submit }));
  ui.suggestTop = el("button", { type: "button", class: "btn primary tb-send tip-r", "aria-label": WORDS.suggest, "data-tip": WORDS.suggestTip, "data-key": "", onclick: () => openSuggestForm() },
    icon("pencil"), el("span", { class: "lbl", "aria-hidden": "true", text: WORDS.suggest }));

  // The sidebar toggles: in the guided app bar, and at the two ends of the
  // shared toolbar. Their names and tooltips say what they would do.
  ui.gLeft = el("button", { type: "button", class: "btn small tb-toggle live-only", "aria-keyshortcuts": "Alt+Shift+D", onclick: () => toggleSide("left") },
    icon("panelLeft"), el("span", { class: "tb-label", text: WORDS.documents }));
  ui.gRightLabel = el("span", { class: "tb-label" });
  ui.gRight = el("button", { type: "button", class: "btn small tb-toggle live-only", "aria-keyshortcuts": "Alt+Shift+S", onclick: () => toggleSide("right") },
    icon("panelRight"), ui.gRightLabel);
  ui.sLeft = el("button", { type: "button", class: "tbtn tg-left", "aria-keyshortcuts": "Alt+Shift+D", "data-key": "Alt+Shift+D", onclick: () => toggleSide("left") },
    icon("panelLeft"), el("span", { class: "lbl", "aria-hidden": "true", text: WORDS.documents }));
  ui.sRightLabel = el("span", { class: "lbl", "aria-hidden": "true" });
  ui.rightCount = el("span", { class: "count", "aria-hidden": "true" });
  ui.sRight = el("button", { type: "button", class: "tbtn tg-right", "aria-keyshortcuts": "Alt+Shift+S", "data-key": "Alt+Shift+S", onclick: () => toggleSide("right") },
    icon("panelRight"), ui.sRightLabel, ui.rightCount);

  // The guided app bar.
  ui.gMe = el("div", { class: "me" });
  const bar = el("header", { class: "top g-only" },
    ui.gLeft, el("span", { class: "tb-sep live-only" }),
    el("div", { class: "logo" }, icon("brief")),
    el("div", { class: "names" }, el("div", { class: "project", text: repo }), el("div", { class: "sub", text: say(WORDS.brandSub, { owner: route.owner }) })),
    el("span", { class: "spacer" }), ui.status, ui.gMe,
    el("span", { class: "tb-sep live-only" }), ui.gRight);

  // The start screen. It asks for nothing technical: no relay address and no
  // token (adr-2610161400542916). It shows what the page can do here, in
  // plain words, or why it cannot.
  ui.heroTitleGuided = el("span", { class: "g-only" });
  ui.heroTitleShared = el("span", { class: "s-only" });
  ui.heroTitle = el("h1", { tabindex: "-1" }, ui.heroTitleGuided, ui.heroTitleShared);
  ui.heroLede = el("p", { class: "lede g-only" });
  ui.heroLedeShared = el("p", { class: "lede s-only" });
  ui.hero = el("header", { class: "card hero" },
    el("div", { class: "appicon s-only" }, icon("brief")),
    el("div", { class: "hero-names" }, el("p", { class: "eyebrow s-only", text: route.owner + " / " + repo }), ui.heroTitle),
    ui.heroLede, ui.heroLedeShared, ui.heroStatus);
  ui.notice = el("p");
  ui.retry = el("button", { type: "button", class: "btn primary", hidden: true, text: WORDS.tryAgain, onclick: () => recheckRelayList() });
  ui.noticeCard = el("section", { class: "card notice-card", hidden: true, role: "status" },
    el("div", { class: "illus s-only" }, icon("hand")), ui.notice, el("div", { class: "row" }, ui.retry));
  ui.askMessage = el("div", { class: "quote-box" });
  ui.askCard = el("section", { class: "card ask-card", hidden: true },
    el("h2", { text: WORDS.askHeading }), el("p", { class: "hint", text: WORDS.askHint }), ui.askMessage,
    el("div", { class: "row" }, el("button", { type: "button", class: "btn primary", text: WORDS.askCopy, onclick: () => copyText(ui.askMessage.textContent, WORDS.askCopied) })));
  ui.code = el("input", { id: "code", class: "code-input", autocomplete: "off", spellcheck: "false", maxlength: "12", autocapitalize: "characters", placeholder: "K7M2QX" });
  ui.name = el("input", { id: "name", autocomplete: "nickname", maxlength: "40" });
  ui.mode = el("select", { id: "mode" },
    el("option", { value: "pr", text: WORDS.modePR }), el("option", { value: "issues", text: WORDS.modeIssues }));
  ui.signedIn = el("p", { class: "state" });
  ui.signInNote = el("p", { class: "state" });
  ui.signInStart = el("button", { type: "button", class: "btn primary block", onclick: () => startSignIn() }, icon("github"), WORDS.signIn);
  ui.hostRow = el("div", { class: "host-row", hidden: true },
    el("label", { for: "mode", text: WORDS.modeLabel }), ui.mode,
    el("div", { class: "row" }, el("button", { type: "button", class: "btn primary block", text: WORDS.host, onclick: () => hostClicked() })));
  ui.startCard = el("section", { class: "card choice", hidden: true },
    el("div", { class: "card-ic", "aria-hidden": "true" }, icon("people")),
    el("h2", { text: say(WORDS.teamHeading, { repo }) }), el("p", { text: WORDS.startHint }),
    el("p", { class: "owner" }, ownerNotice()),
    el("div", { class: "choice-act" }, ui.signInStart, ui.hostRow));
  // Joining: with a code from the invite link, members sign in and everyone
  // else gives a name; without one, a smaller form takes a code given aloud.
  ui.signInJoin = el("button", { type: "button", class: "btn primary block", onclick: () => startSignIn() }, icon("github"), WORDS.signIn);
  ui.joinMembers = el("section", { class: "card choice", hidden: true },
    el("div", { class: "card-ic", "aria-hidden": "true" }, icon("pencil")),
    el("h2", { text: say(WORDS.teamHeading, { repo }) }),
    el("p", { text: WORDS.joinMembers }), el("p", { class: "owner" }, ownerNotice()),
    el("div", { class: "choice-act" }, ui.signInJoin));
  ui.joinTitle = el("h2");
  ui.joinGuests = el("p", { text: WORDS.joinGuests });
  ui.codeHint = el("p", { text: WORDS.codeHint });
  ui.codeRow = el("div", { class: "field-group" }, el("label", { for: "code", text: WORDS.codeLabel }), ui.code);
  ui.joinButton = el("button", { type: "submit", class: "btn primary block" });
  ui.joinCard = el("form", { class: "card choice", hidden: true, onsubmit: (e) => { e.preventDefault(); joinClicked(); } },
    el("div", { class: "card-ic", "aria-hidden": "true" }, icon("bubble")),
    ui.joinTitle, ui.joinGuests, ui.codeHint,
    el("div", { class: "choice-form" }, ui.codeRow,
      el("div", { class: "field-group" }, el("label", { for: "name", text: WORDS.nameLabel }), ui.name, el("p", { class: "hint", text: WORDS.nameHint }))),
    el("div", { class: "choice-act" }, ui.joinButton));
  ui.nextStart = el("div", { class: "next g-only", role: "note" });
  ui.start = el("div", { class: "start" },
    el("div", { class: "narrow" }, ui.hero, ui.noticeCard, ui.askCard,
      el("div", { class: "signed" }, ui.signedIn, ui.signInNote),
      el("div", { class: "choices" }, ui.startCard, ui.joinMembers, ui.joinCard),
      ui.nextStart));

  // GitHub's device flow: the code to confirm on github.com, which the page
  // opens in a new tab.
  ui.deviceCode = el("div", { class: "code bigcode", "aria-live": "polite" });
  ui.deviceOpen = el("a", { class: "button primary", href: GITHUB_DEVICE, target: "_blank", rel: "noopener noreferrer", text: WORDS.deviceOpen }, icon("out"));
  ui.device = el("section", { class: "card device", hidden: true, role: "dialog", "aria-labelledby": "device-heading" },
    el("div", { class: "device-head" },
      el("div", { class: "illus s-only" }, icon("laptop")),
      el("div", {}, el("p", { class: "eyebrow s-only", text: route.owner + " / " + repo }), el("h2", { id: "device-heading", text: WORDS.deviceHeading })),
      el("p", { class: "lead", text: WORDS.deviceIntro })),
    el("div", { class: "numbered" },
      el("div", { class: "n" }, el("b", { "aria-hidden": "true", text: "1" }), el("div", {},
        el("h3", { text: WORDS.deviceNote }),
        el("div", { class: "devcode" }, ui.deviceCode,
          el("button", { type: "button", class: "btn small", onclick: () => copyText(ui.deviceCode.textContent, WORDS.deviceCopied) }, icon("copy"), WORDS.deviceCopy)))),
      el("div", { class: "n" }, el("b", { "aria-hidden": "true", text: "2" }), el("div", {},
        el("h3", { text: WORDS.deviceOpenTitle }), el("p", { class: "hint", text: WORDS.deviceOpenHint }),
        el("div", { class: "row" }, ui.deviceOpen))),
      el("div", { class: "n" }, el("b", { "aria-hidden": "true", text: "3" }), el("div", {},
        el("h3", { text: WORDS.deviceBack }),
        el("div", { class: "waiting" }, el("span", { class: "spinner", "aria-hidden": "true" }),
          el("div", {}, el("strong", { class: "state", text: WORDS.deviceWaiting }), el("span", { class: "hint", text: WORDS.deviceBackHint })))))),
    el("div", { class: "row device-cancel" }, el("button", { type: "button", class: "btn", text: WORDS.cancel, onclick: () => stopDeviceSignIn(true) })),
    el("div", { class: "next g-only", role: "note" },
      el("h2", {}, icon("next"), WORDS.whatNext), // g-only: the guided design's alone
      nextSteps([[WORDS.nextChose, WORDS.nextChoseHint, "done"], [WORDS.nextConfirm, WORDS.nextHere, "here"], [WORDS.nextOpens, WORDS.nextOpensHint]])));

  // The workspace's parts, which each design places for itself.
  ui.files = el("div", { class: "docs" });
  ui.filetitle = el("div", { class: "filetitle" });
  ui.editor = el("div", { class: "editor" });
  ui.page = el("div", { class: "page" }, ui.filetitle, ui.editor);
  ui.notesList = el("ul");
  ui.notesBox = el("section", { class: "notes", role: "alert", hidden: true }, el("h2", { text: WORDS.notesHeading }), ui.notesList);
  ui.people = el("ul", { class: "people" });
  ui.rosterBox = el("section", { class: "box area", "data-panel": "people", "aria-label": WORDS.whoHere }, areaHead(WORDS.whoHere), ui.people);
  ui.sugCount = el("span", { class: "count" });
  ui.sugList = el("div", { class: "sugs" });
  ui.sugBox = el("section", { class: "box area", "data-panel": "sugs", "aria-label": WORDS.suggestionsHeading, hidden: true },
    areaHead(WORDS.sugReview, ui.sugCount), el("p", { class: "area-help g-only", text: WORDS.sugHelp }), ui.sugList);
  ui.mineList = el("ul", { class: "my-sugs" });
  ui.mineBox = el("section", { class: "box area", "data-panel": "mine", "aria-label": WORDS.mySuggestionsHeading, hidden: true },
    areaHead(WORDS.mySuggestionsHeading), el("p", { class: "area-help g-only", text: WORDS.mineHelp }), ui.mineList);
  ui.inviteBody = el("div", { class: "invite-body" });
  ui.inviteBox = el("section", { class: "box area invite-box", "data-panel": "invite", "aria-label": WORDS.inviteHeading, hidden: true },
    areaHead(WORDS.inviteSomeone), el("p", { class: "area-help g-only", text: WORDS.inviteHelp }), ui.inviteBody);
  // The guest's suggestion: a card in Guided, a popover in Shared.
  ui.quote = el("div", { class: "quote" });
  ui.replacement = el("textarea", { id: "replacement", maxlength: String(MAX_REPLACEMENT) });
  ui.suggestForm = el("form", { class: "suggest-form", hidden: true, onsubmit: (e) => { e.preventDefault(); if (sendSuggestion(ui.replacement.value)) { ui.suggestForm.hidden = true; applyLayout(); } } },
    el("div", { class: "field-group" }, el("span", { class: "field", text: WORDS.youSelected }), ui.quote),
    el("div", { class: "field-group" }, el("label", { for: "replacement", text: WORDS.replacementLabel }), ui.replacement),
    el("div", { class: "row" },
      el("button", { type: "submit", class: "btn primary", text: WORDS.send }),
      el("button", { type: "button", class: "btn", text: WORDS.cancel, onclick: () => { ui.suggestForm.hidden = true; applyLayout(); } })));
  ui.suggestButton = el("button", { type: "button", class: "btn primary", onclick: () => openSuggestForm() }, icon("pencil"), WORDS.suggest);
  ui.suggestBox = el("section", { class: "box area guest-panel", "data-panel": "suggest", "aria-label": WORDS.suggest, hidden: true },
    areaHead(WORDS.suggest), el("h3", { class: "s-only pop-title", text: WORDS.suggest }),
    el("p", { class: "area-help g-only", text: WORDS.guestHint }),
    el("div", { class: "row g-only" }, ui.suggestButton), ui.suggestForm);

  // The guided workspace: a banner, then the documents, the brief and the
  // numbered areas, and the way to GitHub.
  ui.bannerTitle = el("h1", { tabindex: "-1" });
  ui.bannerHelp = el("p");
  ui.gDocs = el("div", { class: "docs-slot" });
  ui.gPaper = el("div", { class: "paper" });
  ui.readonlyTag = el("div", { class: "readonly-tag" }, icon("eye"), WORDS.readOnly);
  ui.briefHelp = el("p", { class: "area-help" });
  ui.editHint = el("p", { class: "edit-hint" }, icon("check"), WORDS.savedForEveryone);
  ui.gSideTitle = el("h2");
  ui.gSide = el("aside", { class: "side" },
    el("div", { class: "inspector-head" }, ui.gSideTitle,
      el("button", { type: "button", class: "btn small", text: WORDS.close, onclick: () => toggleSide("right") })));
  ui.submitG = el("button", { type: "button", class: "btn primary", onclick: () => submitSession() }, icon("send"), WORDS.submitMore);
  ui.finish = el("section", { class: "card finish full area", "aria-label": WORDS.sendHeading },
    el("div", {}, el("h2", { class: "area-title" }, el("span", { class: "num", "aria-hidden": "true" }), el("span", { text: WORDS.finishHeading })),
      el("p", { class: "area-help", text: WORDS.finishHelp })),
    el("div", { class: "actions" }, ui.submitG));
  ui.nextLive = el("div", { class: "next g-only", role: "note" },
    el("h2", {}, icon("next"), WORDS.whatNext), // g-only: the guided design's alone
    el("p", { class: "one", text: WORDS.nextGuest }));
  ui.tabs = [
    ["", "doc", WORDS.briefArea, "all"], ["sugs", "bubble", WORDS.sugShort, "member"], ["suggest", "pencil", WORDS.suggest, "guest"],
    ["mine", "clock", WORDS.mySuggestionsHeading, "guest"], ["people", "people", WORDS.whoHere, "all"], ["invite", "invite", WORDS.invitePeople, "host"],
  ].map(([sheet, ic, text, who]) => el("button", { type: "button", class: "tab", "data-sheet": sheet, "data-who": who, onclick: () => openSheet(sheet) },
    icon(ic), el("span", { text })));
  ui.tabCount = el("b", { class: "tab-count", "aria-hidden": "true" });
  ui.tabs[1].append(ui.tabCount);
  const guided = el("div", { class: "g-live g-only" },
    el("section", { class: "card banner" }, el("div", {}, ui.bannerTitle, ui.bannerHelp), el("div", { class: "banner-acts" }, ui.liveDot)),
    el("div", { class: "live-grid" },
      el("nav", { class: "card docs-card", "aria-label": WORDS.documents },
        el("h2", { class: "side-title" }, icon("doc"), el("span", { text: WORDS.documents })),
        el("p", { class: "hint", text: WORDS.documentsHint }), ui.gDocs),
      el("section", { class: "card brief-card area", "aria-label": WORDS.briefArea },
        el("div", { class: "card-head" }, el("h2", { class: "area-title" }, el("span", { class: "num", "aria-hidden": "true" }), el("span", { text: WORDS.briefArea }))),
        ui.briefHelp, ui.readonlyTag, ui.gPaper, ui.editHint),
      ui.gSide, ui.finish),
    ui.nextLive,
    el("nav", { class: "tabbar", "aria-label": WORDS.sessionBar }, ...ui.tabs));

  // The shared workspace: a floating toolbar over the document, a sidebar
  // of documents and people, an inspector of suggestions, and overlays.
  ui.sDocs = el("div", { class: "docs-slot" });
  ui.sPeople = el("div", { class: "people-slot" });
  ui.sbInfo = el("p", { class: "sb-info" });
  ui.sFoot = el("div", { class: "sb-foot" });
  ui.sSidebar = el("nav", { class: "sidebar glass", id: "s-docs", "aria-label": WORDS.documents },
    el("div", { class: "sb-head" }, el("span", { class: "appicon sm" }, icon("brief")),
      el("span", {}, el("b", { text: repo }), el("small", { text: say(WORDS.brandSub, { owner: route.owner }) }))),
    el("h2", { class: "sb-ttl", text: repo }),
    closeButton(() => toggleSide("left")),
    el("p", { class: "sb-label", text: WORDS.documents }), ui.sDocs,
    el("p", { class: "sb-label", text: WORDS.hereNow }), ui.sPeople,
    el("p", { class: "sb-label", text: WORDS.thisSession }), ui.sbInfo, ui.sFoot);
  ui.guestBanner = el("p", { class: "banner" }, icon("bubble"), el("span", { text: WORDS.guestHint }));
  ui.sDoc = el("div", { class: "doc-slot" });
  ui.tiles = el("ul", { class: "tiles" });
  ui.hudSum = el("button", { type: "button", class: "hud-sum", onclick: () => { if (!sideShown("left")) toggleSide("left"); } });
  ui.inviteToggle = el("button", { type: "button", class: "ibtn hud-inv tip-r", "aria-label": WORDS.invitePeople, "aria-haspopup": "dialog", "data-tip": WORDS.invitePeople, "data-key": "", onclick: () => { layout.invite = !layout.invite; applyLayout(); } }, icon("invite"));
  ui.hud = el("div", { class: "hud glass", role: "group", "aria-label": WORDS.whoIsHere }, ui.hudLive, ui.tiles, ui.hudSum, ui.inviteToggle);
  ui.inspTitle = el("h2");
  ui.inspCount = el("span", { class: "count" });
  ui.inspSub = el("p", { class: "insp-sub" });
  ui.sInsp = el("div", { class: "insp-list" });
  ui.sPop = el("div", { class: "popover glass-strong", role: "dialog", "aria-label": WORDS.suggest }, closeButton(() => { ui.suggestForm.hidden = true; applyLayout(); }));
  ui.sShare = el("div", { class: "share glass-strong", role: "dialog", "aria-label": WORDS.invitePeople },
    closeButton(() => { layout.invite = false; applyLayout(); }),
    el("div", { class: "share-h" }, el("span", { class: "appicon sm" }, icon("invite")),
      el("div", {}, el("b", { text: WORDS.inviteTitle }), el("small", { text: say(WORDS.brandSub, { owner: route.owner }) }))));
  const shared = el("div", { class: "s-live s-only" },
    ui.sSidebar,
    el("div", { class: "main" }, el("div", { class: "docscroll" }, el("article", { class: "doc" }, ui.guestBanner, ui.sDoc))),
    el("div", { class: "toolbar", role: "toolbar", "aria-label": WORDS.sessionBar },
      el("div", { class: "tcap tc-left glass" }, ui.sLeft), ui.hud, ui.submit, ui.suggestTop,
      el("div", { class: "tcap tc-right glass" }, ui.sRight)),
    el("aside", { class: "inspector glass", id: "s-side", "aria-label": WORDS.suggestionsHeading },
      closeButton(() => toggleSide("right")),
      el("div", { class: "insp-h" }, ui.inspTitle, ui.inspCount), ui.inspSub, ui.sInsp),
    ui.sPop, ui.sShare,
    el("div", { class: "scrim", onclick: () => closeTop() }));

  // Sending the session to GitHub: a screen of its own in Guided, a sheet
  // over the session in Shared.
  ui.sendIntro = el("p", { class: "state" });
  ui.sendList = el("ul", { class: "changes" });
  ui.sendActions = el("div", { class: "row actions" });
  ui.sendResult = el("p", { class: "state", role: "status", "aria-live": "polite" });
  ui.send = el("section", { class: "send glass-strong", hidden: true, role: "dialog", "aria-labelledby": "send-heading" },
    closeButton(() => closeSend()),
    el("div", { class: "send-in card" },
      el("div", { class: "send-h" }, el("span", { class: "appicon sm soft s-only" }, icon("send")),
        el("div", {}, el("h2", { id: "send-heading", text: WORDS.sendHeading }), ui.sendIntro)),
      el("h3", { text: WORDS.whatChanged }), ui.sendList),
    el("div", { class: "send-foot card" },
      el("div", { class: "reassure" }, icon("shield"), el("p", { text: WORDS.sendReassure })),
      ui.sendActions, ui.sendResult));
  ui.work = el("div", { class: "work", hidden: true }, guided, shared, ui.send);

  // The confirmation a relay the repository's list cannot speak for needs
  // before any sign-in, name or suggestion reaches it, shown over the start
  // screen or a session. It names the project and no address.
  ui.trustOwner = el("p", { class: "owner" });
  ui.trustCancel = el("button", { type: "button", class: "btn", text: WORDS.cancel, onclick: () => trustAnswered(false) });
  ui.trust = el("section", { class: "card trust glass-strong", hidden: true, role: "alertdialog", "aria-labelledby": "trust-heading" },
    el("div", { class: "s-only" }, closeButton(() => trustAnswered(false))),
    el("h2", { id: "trust-heading", text: WORDS.trustHeading }),
    ui.trustOwner, el("p", { text: WORDS.trustQuestion }), el("p", { class: "only", text: WORDS.trustOnly }),
    el("div", { class: "row" },
      el("button", { type: "button", class: "btn primary", text: WORDS.trustYes, onclick: () => trustAnswered(true) }), ui.trustCancel));

  app.append(bar, ui.trust, ui.device, ui.start, ui.work, ui.toast);
  window.addEventListener("keydown", onKey, true);
  let resizing = 0;
  window.addEventListener("resize", () => {
    cancelAnimationFrame(resizing);
    resizing = requestAnimationFrame(() => applyLayout());
  });
  if (typeof ResizeObserver === "function") new ResizeObserver(() => fitTiles()).observe(ui.hud);
  applyLayout();
  return true;
}

// readStartForm takes the code and the name from the start screen.
function readStartForm() {
  session.code = linkCode || normalizeCode(ui.code.value);
  session.name = ui.name.value.trim().slice(0, 40);
}

function joinClicked() {
  if (!session.relay) { toast(standingRefusal("")); return; }
  const code = linkCode || normalizeCode(ui.code.value);
  if (!code) { toast(WORDS.codeBad); ui.code.focus(); return; }
  // A guest's name reaches a relay only once the relay is cleared, as a
  // sign-in does: its own origin, listed by the repository, or confirmed.
  if (!relayClearedFor("join")) return;
  session.code = code;
  session.name = ui.name.value.trim().slice(0, 40);
  if (session.name) remember(KEY_NAME, session.name);
  if (socketOpen()) proceed();
  else openSocket();
}

function hostClicked() {
  if (!relayClearedFor("host")) return;
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
  if (viewWidth() <= PHONE_MAX) {
    if (design === "guided") layout.sheet = "suggest";
    else { layout.pleft = false; layout.pright = false; }
  }
  applyLayout();
  ui.replacement.focus();
}

// renderTop shows who this is, the sign-in a guest may still make, and the
// way to GitHub, in each design's place for them.
function renderTop() {
  if (!ui.gMe) return;
  let who = "";
  if (session.login) who = session.login;
  else if (session.inRoom && session.name) who = session.name;
  const me = session.roster.find((p) => p.id === session.myId);
  clear(ui.gMe);
  clear(ui.sFoot);
  if (who) {
    ui.gMe.append(avatar(who, session.myId, "md"), el("span", { class: "who", text: session.login ? WORDS.signedInAs + " " + who : who }), me ? badge(me) : null);
    ui.sFoot.append(avatar(who, session.myId, "sm"),
      el("span", {}, who, el("small", { text: session.role === "guest" ? WORDS.joinedAsGuest : WORDS.signedInShort + (session.host ? " · " + WORDS.badgeHost : "") })));
  }
  ui.signInTop.hidden = !(session.inRoom && session.role === "guest" && !session.over) || signInOffered === false;
  (design === "shared" ? ui.sFoot : ui.gMe).append(ui.signInTop);
  ui.submit.hidden = !(session.synced && session.role === "member" && !session.over);
  ui.finish.hidden = ui.submit.hidden;
  ui.suggestTop.hidden = !(session.synced && session.role === "guest" && !session.over);
  ui.sbInfo.textContent = session.role === "guest" ? WORDS.sessionInfoGuest : WORDS.sessionInfoMember;
  const name = session.name || WORDS.badgeGuest;
  ui.bannerTitle.textContent = session.role === "guest" ? say(WORDS.bannerGuest, { name, repo: route.repo })
    : say(session.host ? WORDS.bannerHost : WORDS.bannerMember, { repo: route.repo });
  ui.bannerHelp.textContent = session.role === "guest" ? WORDS.bannerGuestHelp : WORDS.bannerMemberHelp;
  ui.briefHelp.textContent = session.role === "guest" ? WORDS.readHelp : WORDS.editHelp;
  ui.readonlyTag.hidden = session.role === "member";
  ui.editHint.hidden = session.role !== "member";
  ui.guestBanner.hidden = session.role !== "guest" || session.over;
  ui.nextLive.hidden = session.role !== "guest";
  paintStatus();
}

function renderStart() {
  if (!ui.start) return;
  const inSession = session.everJoined || session.over;
  ui.start.hidden = inSession;
  ui.work.hidden = !inSession || session.seeding;
  ui.app.dataset.screen = ui.work.hidden ? "start" : "work";
  if (inSession) return;
  // What stands in the way, in plain words and with no address: no session
  // set up, a link that is not the project's own, or GitHub out of reach.
  const standing = relayStanding(session.relay);
  let notice = "";
  if (!session.relay) {
    notice = { checking: WORDS.checking, unavailable: WORDS.cannotCheck, unknown: WORDS.privateNoLink }[anchor.state] || WORDS.notSetUp;
  } else if (standing === "refused" || standing === "unavailable" || standing === "checking") {
    notice = standingRefusal(standing);
  }
  ui.notice.textContent = notice;
  ui.noticeCard.hidden = !notice;
  ui.retry.hidden = anchor.state !== "unavailable";
  const notSetUp = notice === WORDS.notSetUp;
  ui.askCard.hidden = !notSetUp;
  ui.askMessage.textContent = say(WORDS.askMessage, { slug: route.slug, page: pageURL() });
  const usable = !notice;
  const invited = !!linkCode;
  const canSignIn = !session.login && signInOffered !== false;
  ui.heroTitleGuided.textContent = say(invited ? WORDS.invitedTitle : WORDS.heroTitle, { repo: route.repo });
  ui.heroTitleShared.textContent = say(invited ? WORDS.invitedTitle : WORDS.sharedTitle, { repo: route.repo });
  ui.heroLede.textContent = invited ? WORDS.invitedLede : say(WORDS.heroLede, { repo: route.repo });
  ui.heroLedeShared.textContent = invited ? WORDS.invitedLede : say(WORDS.heroLedeShared, { repo: route.repo });
  ui.signedIn.textContent = session.login
    ? WORDS.signedInAs + " " + session.login + (session.member ? "" : ". " + WORDS.notMember)
    : "";
  ui.signInNote.textContent = usable && signInOffered === false ? WORDS.signInNotSetUp : "";
  ui.startCard.hidden = !usable || invited;
  ui.signInStart.hidden = !canSignIn;
  ui.hostRow.hidden = !(session.login && session.member);
  ui.joinCard.hidden = !usable;
  ui.joinTitle.textContent = invited ? WORDS.join : WORDS.haveCode;
  ui.joinMembers.hidden = !usable || !invited || !canSignIn;
  ui.joinGuests.hidden = !invited;
  ui.codeHint.hidden = invited;
  ui.codeRow.hidden = invited;
  ui.joinButton.textContent = invited ? WORDS.join : WORDS.joinWithCode;
  // What happens next, in the guided design: the steps from here, or what
  // follows once sessions are set up.
  clear(ui.nextStart);
  ui.nextStart.hidden = !(usable || notSetUp);
  ui.nextStart.append(el("h2", {}, icon("next"), WORDS.whatNext)); // g-only: ui.nextStart is the guided design's alone
  if (notSetUp) ui.nextStart.append(el("p", { class: "one", text: WORDS.nextNotSetUp }));
  else if (invited) ui.nextStart.append(el("p", { class: "one", text: WORDS.nextInvited }));
  else {
    ui.nextStart.append(nextSteps([[WORDS.nextSignIn, WORDS.nextHere, "here"], [WORDS.nextJoin, WORDS.nextJoinHint],
      [WORDS.nextWork, WORDS.nextWorkHint], [WORDS.nextSend, WORDS.nextSendHint]]));
  }
}

function renderFiles() {
  if (!ui.files) return;
  clear(ui.files);
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
      class: "doc-tab",
      title: rel,
      "aria-current": p === currentPath ? "true" : "false",
      onclick: () => { openFile(p); if (viewWidth() <= PHONE_MAX && design === "shared" && layout.pleft) { layout.pleft = false; applyLayout(); } },
    }, icon(fileIcon(p)), el("b", { class: "t", text: prettyName(p) }), el("small", { text: cut < 0 ? rel : rel.slice(cut + 1) }), icon("right", "chev")));
  }
}

function renderGuestBar() {
  if (!ui.suggestBox) return;
  const guest = session.role === "guest" && session.synced && !session.over && !!view;
  ui.suggestBox.hidden = !guest;
  if (!guest && design === "shared") ui.suggestForm.hidden = true;
  if (guest) ui.suggestButton.disabled = view.state.selection.main.empty;
}

function badge(p) {
  if (p.host) return el("span", { class: "badge host", text: WORDS.badgeHost });
  return el("span", { class: p.role === "member" ? "badge member" : "badge guest", text: p.role === "member" ? WORDS.badgeMember : WORDS.badgeGuest });
}

function renderSide() {
  if (!ui.app) return;
  clear(ui.notesList);
  for (const n of notes) ui.notesList.append(el("li", { text: n }));
  ui.notesBox.hidden = notes.length === 0;

  const hosting = session.host && session.code && !session.seeding && !session.over;
  ui.inviteBox.hidden = !hosting;
  ui.inviteToggle.hidden = !hosting;
  if (!hosting) layout.invite = false;
  clear(ui.inviteBody);
  if (hosting) {
    const url = inviteURL();
    let img = null;
    try {
      const qr = qrcode(0, "M");
      qr.addData(url);
      qr.make();
      img = el("img", { class: "qr", src: qr.createDataURL(6, 2), alt: WORDS.inviteHint });
    } catch { img = null; }
    const mode = el("select", { id: "mode-live", onchange: (e) => { doc.transact(() => meta.set("mode", e.target.value)); } },
      el("option", { value: "pr", text: WORDS.modePR }), el("option", { value: "issues", text: WORDS.modeIssues }));
    mode.value = effectiveMeta().mode === "issues" ? "issues" : "pr";
    ui.inviteBody.append(
      el("div", { class: "invite" },
        el("div", { class: "qr-wrap" }, img, el("span", { class: "qr-cap s-only", text: WORDS.scanHint })),
        el("div", { class: "stack" },
          el("span", { class: "hint", text: WORDS.joinCode }),
          el("div", { class: "code joincode", "aria-label": WORDS.codeLabel + ": " + [...session.code].join(" ") },
            ...[...session.code].map((c) => el("span", { "aria-hidden": "true", text: c }))),
          el("p", { class: "code-hint", text: WORDS.inviteHint }),
          el("button", { type: "button", class: "btn small primary", onclick: () => copyLink(url) }, icon("copy"), WORDS.copyLink))),
      el("div", { class: "targets" },
        el("button", { type: "button", class: "btn small", onclick: () => copyText(session.code, WORDS.codeCopied) }, icon("copy"), WORDS.copyCode),
        el("button", { type: "button", class: "btn small", onclick: () => sendJSON({ type: "rotate" }) }, icon("refresh"), WORDS.newCode),
        el("button", { type: "button", class: "btn small danger", onclick: () => { if (confirm(WORDS.endConfirm)) sendJSON({ type: "close" }); } }, icon("x"), WORDS.endSession)),
      el("div", { class: "field-group" }, el("label", { for: "mode-live", text: WORDS.modeLabel }), mode));
  }

  // Who is here: a list in each design, and the tiles of the shared team bar.
  clear(ui.people);
  clear(ui.tiles);
  clear(ui.hudSum);
  const stack = el("span", { class: "stack" });
  for (const p of session.roster) {
    const name = p.name || p.login || "";
    const isMe = p.id === session.myId;
    ui.people.append(el("li", { class: isMe ? "person me" : "person" },
      avatar(name, p.id), el("span", { class: "nm" }, name, isMe ? " " + WORDS.you : ""), badge(p)));
    ui.tiles.append(el("li", { class: isMe ? "tile me" : "tile" }, avatar(name, p.id),
      el("span", { class: "tile-n" }, isMe ? WORDS.youShort : name, badge(p))));
    if (stack.childElementCount < 4) stack.append(avatar(name, p.id, "xs"));
  }
  const here = say(WORDS.hereCount, { n: session.roster.length });
  ui.hudSum.append(stack, el("span", { class: "txt" }, el("b", { text: here })), icon("right"));
  ui.hudSum.setAttribute("aria-label", here + ". " + WORDS.showDocuments);

  // The suggestions waiting for a member, and a guest's own.
  const member = session.role === "member" && session.synced;
  ui.sugBox.hidden = !member;
  clear(ui.sugList);
  let waiting = 0;
  if (member) {
    for (const s of suggestions.values()) {
      if (!s || typeof s.id !== "string" || suggestionsDone.has(s.id) || !validSuggestion(s.body)) continue;
      waiting++;
      ui.sugList.append(el("article", { class: "sug" },
        el("div", { class: "sug-h" }, avatar(s.from, "", "sm guest-av"),
          el("div", { class: "who" }, el("b", {}, WORDS.suggestedBy + " " + s.from, " ", el("span", { class: "badge guest", text: WORDS.badgeGuest })),
            el("small", { text: prettyName(s.body.path) + " · " + s.body.path.replace(BRIEF_PREFIX, "") }))),
        el("div", { class: "diff" },
          el("div", { class: "old" }, el("span", { class: "k" }, el("span", { class: "g-only", text: WORDS.wordsNow }), el("span", { class: "s-only", text: WORDS.wordsWas })), el("del", { text: s.body.quote.exact })),
          el("div", { class: "new" }, el("span", { class: "k" }, el("span", { class: "g-only", text: WORDS.wordsSuggested }), el("span", { class: "s-only", text: WORDS.wordsNow })), el("ins", { text: s.body.replacement }))),
        el("div", { class: "acts" },
          el("button", { type: "button", class: "btn small primary apply", text: WORDS.apply, onclick: () => applySuggestion(s.id) }),
          el("button", { type: "button", class: "btn small decline", text: WORDS.decline, onclick: () => declineSuggestion(s.id) }))));
    }
    if (!waiting) ui.sugList.append(el("p", { class: "state empty", text: WORDS.noSuggestions }));
  }

  const guest = session.role === "guest";
  ui.mineBox.hidden = !guest || !session.synced;
  clear(ui.mineList);
  if (guest) {
    for (const s of mine) {
      const done = suggestionsDone.get(s.id);
      const [cls, ic, word] = done === "applied" ? ["ok", "check", WORDS.applied] : done === "declined" ? ["no", "x", WORDS.declined] : ["wait", "clock", WORDS.waiting];
      ui.mineList.append(el("li", { class: "mine-item" },
        el("div", { class: "mine-h" }, el("b", { text: prettyName(s.body.path || "") }), el("span", { class: "status " + cls }, icon(ic), word)),
        el("del", { text: s.body.quote.exact }),
        el("ins", { text: s.body.replacement })));
    }
    if (!mine.length) ui.mineList.append(el("li", { class: "state empty", text: WORDS.noMine }));
  }

  // The counts and the right sidebar's names, for this role.
  const count = member ? waiting : guest ? mine.length : 0;
  for (const c of [ui.sugCount, ui.inspCount, ui.rightCount, ui.tabCount]) {
    c.textContent = String(count);
    c.hidden = count === 0;
  }
  ui.inspTitle.textContent = guest ? WORDS.mySuggestionsHeading : WORDS.sugWaiting;
  ui.inspSub.textContent = guest ? WORDS.mineSub : WORDS.sugSub;
  for (const tab of ui.tabs) {
    const who = tab.dataset.who;
    tab.hidden = !(who === "all" || (who === "member" && session.role === "member") || (who === "guest" && guest) || (who === "host" && hosting));
  }
  paintStatus();
  fitTiles();
}

async function copyLink(url) {
  await copyText(url, WORDS.linkCopied);
}
async function copyText(text, done) {
  try { await navigator.clipboard.writeText(text); toast(done); } catch { /* clipboard refused */ }
}

function renderAll() {
  if (session.over && ui.send) ui.send.hidden = true;
  renderTop();
  renderStart();
  renderSide();
  renderGuestBar();
  applyLayout();
  if (session.synced && !session.seeding && !view && files.size > 0) {
    openFile(firstFile());
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

// framed says the page is inside another page's frame. A static host cannot
// send frame-ancestors (or X-Frame-Options), so the page refuses a frame
// itself: framed, it could be overlaid to steal a click on the relay's
// confirmation or the sign-in.
function framed() {
  try { return window.top !== window.self; } catch { return true; }
}
// refuseFramed shows only a message and a link that opens the page in its
// own tab: no trust confirmation, no sign-in, no session.
function refuseFramed() {
  const app = document.getElementById("app");
  clear(app);
  app.append(el("p", { class: "notice" }, WORDS.framedRefused + " ",
    el("a", { href: pageURL(), target: "_top", rel: "noopener noreferrer", text: pageURL() })));
}

// The relay and the code an invite link carries in its fragment, which never
// reaches a server.
let linkRelay = "";
let linkCode = "";

async function main() {
  if (addSlash()) return;
  if (framed()) { refuseFramed(); return; }
  if (!build()) return;
  // A confirmation an earlier page kept in local storage, for good, is
  // dropped: one now lasts this tab, and a bounded time.
  try { localStorage.removeItem(LEGACY_TRUSTED); } catch { /* storage unavailable */ }
  // Leaving the page ends a device sign-in under way on the relay too.
  addEventListener("pagehide", () => stopDeviceSignIn(true));
  const frag = new URLSearchParams(location.hash.slice(1));
  linkRelay = relayOrigin(frag.get("relay"));
  linkCode = normalizeCode(frag.get("code"));
  session.code = linkCode;
  session.name = recall(KEY_NAME);
  ui.code.value = session.code;
  ui.name.value = session.name;
  setStatus("", false);
  renderAll();
  // Before anything reaches a relay: is the relay this page's own server,
  // and if not, which relays does the repository name?
  anchor = selfServed() ? { state: "self", relays: [] } : await readRelayList("");
  session.relay = chooseRelay(linkRelay);
  renderAll();
  refreshSignIn();
  if (!(await finishSignIn())) {
    // A reload of a page that was in a session: back in by its rejoin token.
    if (takeRejoin()) openSocket();
  }
  ui.code.value = session.code;
  renderAll();
}
main();
