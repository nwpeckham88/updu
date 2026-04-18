(function () {
  'use strict';

  // ---- Theme toggle ----
  var STORAGE_KEY = 'updu-theme';
  var root = document.documentElement;
  var toggle = document.getElementById('theme-toggle');

  function applyTheme(theme) {
    root.setAttribute('data-theme', theme);
    if (toggle) toggle.setAttribute('aria-pressed', theme === 'dark' ? 'true' : 'false');
  }

  if (toggle) {
    var current = root.getAttribute('data-theme') || 'light';
    toggle.setAttribute('aria-pressed', current === 'dark' ? 'true' : 'false');

    toggle.addEventListener('click', function () {
      var next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      applyTheme(next);
      try { localStorage.setItem(STORAGE_KEY, next); } catch (e) {}
    });
  }

  // Track OS preference if user hasn't picked one
  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    if (!stored && window.matchMedia) {
      var mq = window.matchMedia('(prefers-color-scheme: dark)');
      mq.addEventListener && mq.addEventListener('change', function (e) {
        if (!localStorage.getItem(STORAGE_KEY)) {
          applyTheme(e.matches ? 'dark' : 'light');
        }
      });
    }
  } catch (e) {}

  // ---- Copy-to-clipboard ----
  document.querySelectorAll('[data-copy]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var block = btn.closest('.codeblock');
      if (!block) return;
      var code = block.querySelector('code');
      if (!code) return;
      var text = code.innerText.trim();

      var done = function () {
        var original = btn.textContent;
        btn.textContent = 'copied';
        btn.classList.add('copied');
        setTimeout(function () {
          btn.textContent = original || 'copy';
          btn.classList.remove('copied');
        }, 1400);
      };

      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(function () { fallbackCopy(text, done); });
      } else {
        fallbackCopy(text, done);
      }
    });
  });

  function fallbackCopy(text, cb) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'absolute';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch (e) {}
    document.body.removeChild(ta);
    cb && cb();
  }
})();
