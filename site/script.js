document.addEventListener('DOMContentLoaded', () => {
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

    // ── Line Copy button ─────────────────────────
    document.querySelectorAll('.line-copy-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            const text = btn.dataset.copy;
            if (text) {
                navigator.clipboard.writeText(text).then(() => {
                    const originalHTML = btn.innerHTML;
                    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
                    setTimeout(() => btn.innerHTML = originalHTML, 2000);
                });
            }
        });
    });
});
