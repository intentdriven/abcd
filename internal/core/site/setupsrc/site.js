/* abcdev.app — the landing page's only script.
 *
 * It adds no words. Every string it shows is already in the markup, put there
 * by the build out of site-src/ui.json, and read back from a data attribute —
 * so the interface-string allowlist stays the complete list of what a reader
 * sees, script included.
 *
 * No analytics, no trackers, no network requests of any kind (adr-38, adr-48).
 * Nothing is stored: the starred install tab is decided from the browser's own
 * platform string at render time and forgotten again.
 */
(function () {
  'use strict';

  /* Install tabs. Keyboard behaviour is the ARIA tabs pattern: arrows move,
     Home and End jump, and the selected tab is the only one in the tab order. */
  function wireTabs(tabs) {
    var list = tabs.querySelector('[role=tablist]');
    if (!list) return;
    var buttons = Array.prototype.slice.call(list.querySelectorAll('[role=tab]'));
    if (!buttons.length) return;

    function select(btn, focus) {
      buttons.forEach(function (b) {
        var on = b === btn;
        b.setAttribute('aria-selected', on ? 'true' : 'false');
        b.tabIndex = on ? 0 : -1;
        var panel = document.getElementById(b.getAttribute('aria-controls'));
        if (panel) panel.hidden = !on;
      });
      if (focus) btn.focus();
    }

    buttons.forEach(function (btn, i) {
      btn.addEventListener('click', function () { select(btn, false); });
      btn.addEventListener('keydown', function (e) {
        var next = null;
        if (e.key === 'ArrowRight' || e.key === 'ArrowDown') next = buttons[(i + 1) % buttons.length];
        else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') next = buttons[(i - 1 + buttons.length) % buttons.length];
        else if (e.key === 'Home') next = buttons[0];
        else if (e.key === 'End') next = buttons[buttons.length - 1];
        if (!next) return;
        e.preventDefault();
        select(next, true);
      });
    });

    /* A star marks the tab matching the reader's own system. The title text is
       a ui.json string the build put on the tablist. */
    var title = list.getAttribute('data-mine-title') || '';
    var mine = matchPlatform(buttons);
    if (mine) {
      mine.classList.add('mine');
      if (title) mine.title = title;
      select(mine, false);
    }
  }

  /* matchPlatform picks the tab whose LABEL names the reader's platform. The
     labels are section headings from the install page, so the match is by the
     page's own words rather than by a table of platform names kept here. */
  function matchPlatform(buttons) {
    var ua = ((navigator.userAgentData && navigator.userAgentData.platform) ||
              navigator.platform || navigator.userAgent || '').toLowerCase();
    var wants = null;
    if (ua.indexOf('mac') >= 0 || ua.indexOf('iphone') >= 0 || ua.indexOf('ipad') >= 0) wants = ['macos', 'mac os', 'darwin'];
    else if (ua.indexOf('win') >= 0) wants = ['windows'];
    else if (ua.indexOf('linux') >= 0 || ua.indexOf('android') >= 0 || ua.indexOf('x11') >= 0) wants = ['linux'];
    if (!wants) return null;
    for (var i = 0; i < buttons.length; i++) {
      var label = (buttons[i].textContent || '').trim().toLowerCase();
      for (var j = 0; j < wants.length; j++) {
        if (label.indexOf(wants[j]) >= 0) return buttons[i];
      }
    }
    return null;
  }

  /* Copy buttons. The payload and both labels ride in the markup. */
  function wireCopy(btn) {
    btn.addEventListener('click', function () {
      var text = btn.getAttribute('data-copy') || '';
      var done = btn.getAttribute('data-copied');
      var was = btn.textContent;
      function ok() {
        if (!done) return;
        btn.textContent = done;
        window.setTimeout(function () { btn.textContent = was; }, 1200);
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(ok, function () {});
        return;
      }
      /* No clipboard API: select the command so the reader can copy it the way
         they always could. Better than a button that silently does nothing. */
      var pre = btn.parentNode.querySelector('code');
      if (!pre || !window.getSelection) return;
      var range = document.createRange();
      range.selectNodeContents(pre);
      var sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(range);
    });
  }

  document.querySelectorAll('.tabs').forEach(wireTabs);
  document.querySelectorAll('.cmd .copy').forEach(wireCopy);
})();
