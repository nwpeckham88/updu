document.addEventListener('DOMContentLoaded', () => {
    const currentBetaVersion = 'v0.3.2-beta';

    // ── Nav scroll effect ────────────────────────
    const nav = document.querySelector('.nav');
    const scrollThreshold = 50;
    const onScroll = () => {
        nav.classList.toggle('scrolled', window.scrollY > scrollThreshold);
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    onScroll();

    // ── Mobile menu toggle ───────────────────────
    const toggle = document.querySelector('.mobile-toggle');
    const links = document.querySelector('.nav-links');
    toggle?.addEventListener('click', () => {
        links.classList.toggle('open');
        const isOpen = links.classList.contains('open');
        toggle.setAttribute('aria-expanded', isOpen);
    });

    // Close mobile menu on link click
    links?.querySelectorAll('a').forEach(a => {
        a.addEventListener('click', () => links.classList.remove('open'));
    });

    // ── Scroll reveal ────────────────────────────
    // Removed scroll animation as requested

    // ── Quick-start tabs ─────────────────────────
    const tabs = document.querySelectorAll('.quickstart-tab');
    const contents = document.querySelectorAll('.tab-content');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const target = tab.dataset.tab;
            tabs.forEach(t => t.classList.remove('active'));
            contents.forEach(c => c.classList.remove('active'));
            tab.classList.add('active');
            document.getElementById(target)?.classList.add('active');
        });
    });

    // ── Copy button ──────────────────────────────
    document.querySelectorAll('.copy-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const block = btn.closest('.code-block');
            const code = block?.querySelector('code');
            if (code) {
                const text = code.textContent;
                navigator.clipboard.writeText(text).then(() => {
                    const orig = btn.textContent;
                    btn.textContent = 'Copied!';
                    setTimeout(() => btn.textContent = orig, 2000);
                });
            }
        });
    });

    // ── Animated counter ─────────────────────────
    // Removed scroll animation as requested
    const counters = document.querySelectorAll('[data-count]');
    counters.forEach(el => {
        const target = el.dataset.count;
        const prefix = el.dataset.prefix || '';
        const suffix = el.dataset.suffix || '';
        el.textContent = prefix + target + suffix;
    });

    // ── Binary Architecture Selector ─────────────
    const archTabs = document.querySelectorAll('.arch-tab');
    const binaryCode = document.getElementById('binary-code');
    const oidcCheckbox = document.getElementById('oidc-checkbox');

    const archCommands = {
        amd64: 'amd64',
        arm64: 'arm64',
        armv7: 'armv7',
        armv6: 'armv6'
    };

    const renderBinaryCode = () => {
        if (!binaryCode) return;

        const activeTab = document.querySelector('.arch-tab.active');
        const activeArch = activeTab?.dataset.arch || 'amd64';
        const arch = archCommands[activeArch] || 'amd64';
        const includeOidc = Boolean(oidcCheckbox?.checked);
        const suffix = includeOidc ? '-oidc' : '';

        binaryCode.innerHTML = `<span class="comment"># 1. Download the current beta for your architecture</span>
<span class="command">curl</span> <span class="flag">-LO</span> <span class="string">https://github.com/nwpeckham88/updu/releases/download/${currentBetaVersion}/updu-linux-${arch}${suffix}</span>

<span class="comment"># 2. Make executable and run</span>
<span class="command">chmod</span> <span class="flag">+x</span> updu-linux-${arch}${suffix}
<span class="command">./updu-linux-${arch}${suffix}</span>`;
    };

    archTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const arch = tab.dataset.arch;
            if (!arch) return;

            // Update active state
            archTabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            renderBinaryCode();
        });
    });

    oidcCheckbox?.addEventListener('change', renderBinaryCode);
    renderBinaryCode();
});
