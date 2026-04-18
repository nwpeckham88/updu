// updu marketing — small interactions
(function () {
  'use strict';

  var RELEASES_URL = 'https://github.com/nwpeckham88/updu/releases';
  var RELEASE_DOWNLOADS_URL = RELEASES_URL + '/latest/download/';
  var DEFAULT_TARGET = 'updu-linux-amd64';
  var BUILD_TARGETS = {
    'updu-linux-amd64': { label: 'Linux x86_64' },
    'updu-linux-arm64': { label: 'Linux ARM64' },
    'updu-linux-armv7': { label: 'Linux ARMv7' },
    'updu-linux-armv6': { label: 'Linux ARMv6' }
  };

  function buildAssetUrl(target) {
    return RELEASE_DOWNLOADS_URL + target;
  }

  function buildInstallCommand(target) {
    return 'curl -LO ' + buildAssetUrl(target) + '\n'
      + 'chmod +x ' + target + '\n'
      + './' + target;
  }

  function detectOS(value) {
    if (!value) return 'unknown';
    if (/iphone|ipad|ipod/.test(value)) return 'ios';
    if (/android/.test(value)) return 'android';
    if (/win/.test(value)) return 'windows';
    if (/mac|darwin/.test(value)) return 'macos';
    if (/linux|x11/.test(value)) return 'linux';
    return 'unknown';
  }

  function detectArchitecture(value) {
    if (!value) return '';
    if (/armv?6\b|armv6l/.test(value)) return 'armv6';
    if (/armv?7\b|armv7l/.test(value)) return 'armv7';
    if (/aarch64|arm64/.test(value)) return 'arm64';
    if (/x86_64|amd64|wow64|win64|x64|intel64/.test(value)) return 'amd64';
    if (/\bi[3-6]86\b|\bx86\b/.test(value)) return 'x86';
    if (/\barm\b/.test(value)) return 'arm';
    return '';
  }

  function buildPlatformInfo(os, architecture, usedClientHints, assumedArmVariant) {
    var info = {
      os: os || 'unknown',
      architecture: architecture || '',
      architectureKnown: false,
      target: DEFAULT_TARGET,
      supportedNative: os === 'linux',
      usedClientHints: !!usedClientHints,
      assumedArmVariant: !!assumedArmVariant
    };

    if (architecture === 'amd64') {
      info.architectureKnown = true;
      info.target = 'updu-linux-amd64';
    } else if (architecture === 'arm64') {
      info.architectureKnown = true;
      info.target = 'updu-linux-arm64';
    } else if (architecture === 'armv7') {
      info.architectureKnown = true;
      info.target = 'updu-linux-armv7';
    } else if (architecture === 'armv6') {
      info.architectureKnown = true;
      info.target = 'updu-linux-armv6';
    }

    return info;
  }

  function detectPlatformInfo() {
    var sources = [];
    if (navigator.userAgentData && navigator.userAgentData.platform) {
      sources.push(String(navigator.userAgentData.platform).toLowerCase());
    }
    if (navigator.platform) {
      sources.push(String(navigator.platform).toLowerCase());
    }
    if (navigator.userAgent) {
      sources.push(String(navigator.userAgent).toLowerCase());
    }

    var source = sources.join(' ');
    return buildPlatformInfo(detectOS(source), detectArchitecture(source), false);
  }

  function refinePlatformInfo(initialInfo) {
    if (!navigator.userAgentData || typeof navigator.userAgentData.getHighEntropyValues !== 'function') {
      return Promise.resolve(initialInfo);
    }

    return navigator.userAgentData.getHighEntropyValues(['architecture', 'bitness', 'platform']).then(function (values) {
      var os = detectOS(String(values.platform || initialInfo.os).toLowerCase()) || initialInfo.os;
      var architecture = initialInfo.architecture;
      var clientHintArch = String(values.architecture || '').toLowerCase();
      var bitness = String(values.bitness || '').toLowerCase();

      if (clientHintArch === 'arm') {
        architecture = bitness === '64' ? 'arm64' : (architecture === 'armv6' ? 'armv6' : 'armv7');
      } else if (clientHintArch === 'x86') {
        architecture = bitness === '64' ? 'amd64' : architecture;
      } else if (clientHintArch) {
        architecture = detectArchitecture(clientHintArch + ' ' + bitness) || architecture;
      }

      return buildPlatformInfo(os, architecture, true, clientHintArch === 'arm' && bitness !== '64' && initialInfo.architecture !== 'armv6' && initialInfo.architecture !== 'armv7');
    }).catch(function () {
      return initialInfo;
    });
  }

  function getInstallNotice(info) {
    if (info.os === 'linux') {
      if (info.assumedArmVariant) {
        return 'Detected 32-bit ARM Linux. Browsers cannot reliably distinguish ARMv6 from ARMv7, so ARMv7 is selected by default. Switch to ARMv6 below for Raspberry Pi Zero and similar boards.';
      }
      if (info.architectureKnown) {
        return 'Detected ' + BUILD_TARGETS[info.target].label + '. The Binary tab is preselected below.';
      }
      return 'Detected Linux, but browsers rarely expose CPU details. Defaulting to Linux x86_64 until you pick a different target. If this machine is ARM-based, choose ARM64, ARMv7, or ARMv6 below.';
    }

    if (info.os === 'macos') {
      return 'Detected macOS. updu currently publishes native Linux binaries only, so Docker is selected by default.';
    }

    if (info.os === 'windows') {
      return 'Detected Windows. updu currently publishes native Linux binaries only, so Docker is selected by default.';
    }

    if (info.os === 'android' || info.os === 'ios') {
      return 'Detected a mobile browser. updu is typically installed on a Linux host, so the Binary tab stays configurable for that target.';
    }

    return 'Could not identify your OS. Defaulting to Linux x86_64; change the target manually if needed.';
  }

  function getBinaryNote(info, manualSelection) {
    if (manualSelection) {
      return 'Manual Linux target selected. Switch it again if you are downloading for a different machine.';
    }

    if (info.assumedArmVariant) {
      return '32-bit ARM detected. ARMv7 is selected as the safer default, but Raspberry Pi Zero and older ARMv6 boards should switch to ARMv6 manually.';
    }

    if (info.os === 'linux' && info.architectureKnown) {
      return 'Best guess from your browser. Change it if this browser is not running on the Linux machine you plan to install updu on.';
    }

    if (info.os === 'linux') {
      return 'Detected Linux, but browsers rarely expose CPU details. Defaulting to x86_64 until you choose another Linux target. ARM systems should switch to the matching target manually.';
    }

    return 'updu publishes native binaries for Linux only. Use this tab when you need a build for a VM, remote host, or another Linux machine.';
  }

  function setInstallNotice(text) {
    var installDetected = document.querySelector('[data-install-detected]');
    if (installDetected) {
      installDetected.textContent = text;
    }
  }

  function getDirectDownloadLabel(target, mode) {
    var label = BUILD_TARGETS[target] ? BUILD_TARGETS[target].label : target;
    if (mode === 'short') {
      return 'Download ' + label.replace(/^Linux\s+/, '');
    }
    return 'Download ' + label;
  }

  function syncReleaseLinks(target, allowDirectDownload) {
    document.querySelectorAll('[data-release-download]').forEach(function (link) {
      if (allowDirectDownload) {
        link.href = buildAssetUrl(target);
        link.textContent = getDirectDownloadLabel(target, link.dataset.directLabelMode);
        link.title = 'Download ' + target;
        link.setAttribute('aria-label', 'Download ' + target);
      } else {
        link.href = RELEASES_URL;
        link.textContent = link.dataset.browseLabel || 'Browse releases';
        link.title = 'Browse updu releases';
        link.setAttribute('aria-label', 'Browse updu releases');
      }
    });
  }

  function setBinaryTarget(target, note, allowDirectDownload) {
    var resolvedTarget = BUILD_TARGETS[target] ? target : DEFAULT_TARGET;
    var config = BUILD_TARGETS[resolvedTarget];
    var installCommand = document.querySelector('[data-install-command]');
    var selectedTargetLabel = document.querySelector('[data-selected-target-label]');
    var selectedTargetNote = document.querySelector('[data-selected-target-note]');
    var binaryDownloadLink = document.querySelector('[data-binary-download-link]');

    document.querySelectorAll('[data-build-target]').forEach(function (control) {
      var active = control.dataset.buildTarget === resolvedTarget;
      control.classList.toggle('is-active', active);
      control.setAttribute('aria-pressed', active ? 'true' : 'false');
    });

    if (selectedTargetLabel) {
      selectedTargetLabel.textContent = config.label;
    }

    if (selectedTargetNote) {
      selectedTargetNote.textContent = note;
    }

    if (installCommand) {
      installCommand.textContent = buildInstallCommand(resolvedTarget);
    }

    if (binaryDownloadLink) {
      binaryDownloadLink.href = buildAssetUrl(resolvedTarget);
      binaryDownloadLink.textContent = 'Download ' + resolvedTarget;
    }

    syncReleaseLinks(resolvedTarget, allowDirectDownload);
  }

  function activateInstallTab(name, shouldFocus) {
    var button = document.querySelector('[data-tabs] [data-tab="' + name + '"]');
    if (button) {
      button.click();
      if (shouldFocus) {
        button.focus();
      }
    }
  }

  // Year in footer
  document.querySelectorAll('[data-year]').forEach(function (el) {
    el.textContent = String(new Date().getFullYear());
  });

  // Theme toggle
  var toggle = document.querySelector('[data-theme-toggle]');
  if (toggle) {
    toggle.addEventListener('click', function () {
      var current = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
      var next = current === 'dark' ? 'light' : 'dark';
      document.documentElement.dataset.theme = next;
      try { localStorage.setItem('updu-marketing-theme', next); } catch (e) {}
    });
  }

  // Tabs (install methods)
  document.querySelectorAll('[data-tabs]').forEach(function (tabs) {
    var buttons = tabs.querySelectorAll('[role="tab"]');
    var panels = tabs.querySelectorAll('[data-panel]');

    function activate(name) {
      buttons.forEach(function (btn) {
        var active = btn.dataset.tab === name;
        btn.setAttribute('aria-selected', active ? 'true' : 'false');
        btn.tabIndex = active ? 0 : -1;
      });
      panels.forEach(function (panel) {
        var active = panel.dataset.panel === name;
        panel.hidden = !active;
      });
    }

    buttons.forEach(function (btn, i) {
      btn.addEventListener('click', function () { activate(btn.dataset.tab); });
      btn.addEventListener('keydown', function (e) {
        if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
          e.preventDefault();
          var dir = e.key === 'ArrowRight' ? 1 : -1;
          var next = buttons[(i + dir + buttons.length) % buttons.length];
          next.focus();
          activate(next.dataset.tab);
        }
      });
    });
  });

  // Build target selection
  var detectedPlatform = detectPlatformInfo();
  var manualTargetSelected = false;

  setInstallNotice(getInstallNotice(detectedPlatform));
  setBinaryTarget(detectedPlatform.target, getBinaryNote(detectedPlatform, false), detectedPlatform.supportedNative);

  if (detectedPlatform.os === 'macos' || detectedPlatform.os === 'windows') {
    activateInstallTab('docker', false);
  }

  document.querySelectorAll('[data-build-target]').forEach(function (control) {
    control.addEventListener('click', function (event) {
      event.preventDefault();
      manualTargetSelected = true;
      setBinaryTarget(control.dataset.buildTarget, getBinaryNote(detectedPlatform, true), true);
      setInstallNotice('Using a manually selected Linux target. Switch tabs if you prefer Docker or Compose instead.');
      activateInstallTab('binary', true);
    });
  });

  refinePlatformInfo(detectedPlatform).then(function (refinedInfo) {
    if (manualTargetSelected) return;
    if (refinedInfo.target === detectedPlatform.target && refinedInfo.os === detectedPlatform.os) return;

    detectedPlatform = refinedInfo;
    setInstallNotice(getInstallNotice(refinedInfo));
    setBinaryTarget(refinedInfo.target, getBinaryNote(refinedInfo, false), refinedInfo.supportedNative);
    if (refinedInfo.os === 'macos' || refinedInfo.os === 'windows') {
      activateInstallTab('docker', false);
    }
  });

  // Copy buttons
  document.querySelectorAll('[data-copy]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var panel = btn.closest('.tab-panel');
      if (!panel) return;
      var code = panel.querySelector('code');
      if (!code) return;
      var text = code.innerText;
      var done = function () {
        var original = btn.textContent;
        btn.textContent = 'Copied';
        btn.classList.add('is-copied');
        setTimeout(function () {
          btn.textContent = original;
          btn.classList.remove('is-copied');
        }, 1400);
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(function () { fallback(text, done); });
      } else {
        fallback(text, done);
      }
    });
  });

  function fallback(text, cb) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'absolute';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    var copied = false;
    try { copied = document.execCommand('copy'); } catch (e) {}
    document.body.removeChild(ta);
    if (copied) {
      cb();
    }
  }

  // Animated counters
  var prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var counters = document.querySelectorAll('[data-count]');
  if (counters.length && 'IntersectionObserver' in window) {
    var seen = new WeakSet();
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        var el = entry.target;
        if (seen.has(el)) return;
        seen.add(el);
        var target = parseInt(el.dataset.count, 10) || 0;
        if (prefersReduced) { el.textContent = String(target); return; }
        var start = performance.now();
        var duration = 900;
        function step(now) {
          var p = Math.min(1, (now - start) / duration);
          var eased = 1 - Math.pow(1 - p, 3);
          el.textContent = String(Math.round(target * eased));
          if (p < 1) requestAnimationFrame(step);
        }
        requestAnimationFrame(step);
      });
    }, { threshold: 0.4 });
    counters.forEach(function (c) { io.observe(c); });
  }

  // Reveal on scroll
  if ('IntersectionObserver' in window && !prefersReduced) {
    var revealTargets = document.querySelectorAll('.feature, .chip, .hero-panel, .section-head, .install-grid > *');
    revealTargets.forEach(function (el) { el.classList.add('reveal'); });
    var ro = new IntersectionObserver(function (entries, obs) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add('is-visible');
          obs.unobserve(entry.target);
        }
      });
    }, { threshold: 0.12 });
    revealTargets.forEach(function (el) { ro.observe(el); });
  }
})();
