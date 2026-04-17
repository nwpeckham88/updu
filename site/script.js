document.addEventListener('DOMContentLoaded', () => {
  const currentReleaseVersion = 'v0.5.1';
  const currentReleaseHref = `https://github.com/nwpeckham88/updu/releases/tag/${currentReleaseVersion}`;

  document.querySelectorAll('[data-release-version]').forEach((element) => {
    element.textContent = currentReleaseVersion;
  });

  document.querySelectorAll('[data-release-link]').forEach((element) => {
    if (element instanceof HTMLAnchorElement) {
      element.href = currentReleaseHref;
    }
  });

  const nav = document.getElementById('nav');
  const navLinks = document.getElementById('nav-links');
  const mobileToggle = document.getElementById('mobile-toggle');
  const shellRegions = Array.from(document.querySelectorAll('main, .footer'));
  const supportsInert = 'inert' in HTMLElement.prototype;
  const getMobileNavLinks = () => {
    if (!navLinks) {
      return [];
    }

    return Array.from(navLinks.querySelectorAll('a')).filter((link) => !link.hasAttribute('hidden'));
  };

  const setScrolledState = () => {
    nav?.classList.toggle('scrolled', window.scrollY > 12);
  };

  setScrolledState();
  window.addEventListener('scroll', setScrolledState, { passive: true });

  const setMobileMenuState = (isOpen) => {
    if (!navLinks || !mobileToggle) {
      return;
    }

    navLinks.classList.toggle('open', isOpen);
    mobileToggle.setAttribute('aria-expanded', String(isOpen));

    shellRegions.forEach((region) => {
      if (region instanceof HTMLElement && supportsInert) {
        region.inert = isOpen;
      }
    });

    if (isOpen) {
      getMobileNavLinks()[0]?.focus();
    }
  };

  const closeMobileMenu = ({ returnFocus = false } = {}) => {
    setMobileMenuState(false);

    if (returnFocus) {
      mobileToggle?.focus();
    }
  };

  mobileToggle?.addEventListener('click', () => {
    if (!navLinks || !mobileToggle) {
      return;
    }

    const isOpen = !navLinks.classList.contains('open');
    setMobileMenuState(isOpen);
  });

  window.addEventListener('resize', () => {
    if (window.innerWidth > 820) {
      closeMobileMenu();
    }
  });

  navLinks?.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', closeMobileMenu);
  });

  document.addEventListener('click', (event) => {
    if (!navLinks || !mobileToggle) {
      return;
    }

    const target = event.target;
    if (!(target instanceof Node)) {
      return;
    }

    if (!navLinks.contains(target) && !mobileToggle.contains(target)) {
      closeMobileMenu();
    }
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closeMobileMenu({ returnFocus: true });
      return;
    }

    if (event.key !== 'Tab' || !navLinks?.classList.contains('open')) {
      return;
    }

    const focusableLinks = getMobileNavLinks();
    if (!focusableLinks.length) {
      return;
    }

    const firstLink = focusableLinks[0];
    const lastLink = focusableLinks[focusableLinks.length - 1];
    const activeElement = document.activeElement;

    if (event.shiftKey && activeElement === firstLink) {
      event.preventDefault();
      lastLink.focus();
    } else if (!event.shiftKey && activeElement === lastLink) {
      event.preventDefault();
      firstLink.focus();
    }
  });

  document.querySelectorAll('.copy-btn').forEach((button) => {
    button.addEventListener('click', async () => {
      const code = button.closest('.code-block')?.querySelector('code');
      if (!code) {
        return;
      }

      const originalLabel = button.textContent ?? 'Copy';
      const codeText = code.textContent?.trim();

      if (!codeText) {
        button.textContent = 'No code';
        window.setTimeout(() => {
          button.textContent = originalLabel;
        }, 1600);
        return;
      }

      try {
        await navigator.clipboard.writeText(codeText);
        button.textContent = 'Copied';
      } catch {
        button.textContent = 'Failed';
      }

      window.setTimeout(() => {
        button.textContent = originalLabel;
      }, 1600);
    });
  });

  const archTabs = Array.from(document.querySelectorAll('.arch-tab'));
  const oidcCheckbox = document.getElementById('oidc-checkbox');
  const binaryCode = document.getElementById('binary-code');

  const architectureMap = {
    amd64: 'amd64',
    arm64: 'arm64',
    armv7: 'armv7',
    armv6: 'armv6',
  };

  const setActiveArchitectureTab = (activeTab) => {
    archTabs.forEach((item) => {
      const isActive = item === activeTab;
      item.classList.toggle('active', isActive);
      item.setAttribute('aria-checked', String(isActive));
      item.tabIndex = isActive ? 0 : -1;
    });
  };

  const renderBinaryCode = () => {
    if (!binaryCode) {
      return;
    }

    let activeTab = archTabs.find((tab) => tab.classList.contains('active'));
    if (!activeTab) {
      activeTab = archTabs[0];
      if (!activeTab) {
        binaryCode.textContent = 'Architecture selection unavailable.';
        return;
      }

      setActiveArchitectureTab(activeTab);
    }

    const activeArch = architectureMap[activeTab.dataset.arch ?? ''];
    if (!activeArch) {
      binaryCode.textContent = 'Architecture selection unavailable.';
      return;
    }

    const suffix = oidcCheckbox instanceof HTMLInputElement && oidcCheckbox.checked ? '-oidc' : '';

    binaryCode.textContent = [
      `curl -fsSLO https://github.com/nwpeckham88/updu/releases/download/${currentReleaseVersion}/updu-linux-${activeArch}${suffix} \\`,
      `  && chmod +x updu-linux-${activeArch}${suffix} \\`,
      `  && ./updu-linux-${activeArch}${suffix}`,
    ].join('\n');
  };

  const activateArchitectureTab = (tab, { focus = false } = {}) => {
    setActiveArchitectureTab(tab);
    renderBinaryCode();

    if (focus) {
      tab.focus();
    }
  };

  archTabs.forEach((tab, index) => {
    tab.addEventListener('click', () => {
      activateArchitectureTab(tab);
    });

    tab.addEventListener('keydown', (event) => {
      let nextIndex = index;

      if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
        nextIndex = (index + 1) % archTabs.length;
      } else if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
        nextIndex = (index - 1 + archTabs.length) % archTabs.length;
      } else if (event.key === 'Home') {
        nextIndex = 0;
      } else if (event.key === 'End') {
        nextIndex = archTabs.length - 1;
      } else {
        return;
      }

      event.preventDefault();
      const nextTab = archTabs[nextIndex];
      if (!nextTab) {
        return;
      }

      activateArchitectureTab(nextTab, { focus: true });
    });
  });

  oidcCheckbox?.addEventListener('change', renderBinaryCode);
  renderBinaryCode();
});
