# Site Documentation Structure

This directory contains the project marketing site plus generated monitor docs.

Canonical source for monitor documentation lives in:

- site/content/docs/*.md
- site/content/docs/style.css

Generated HTML output lives in:

- site/docs/

Regenerate docs with:

```bash
make docs
```

The docs build uses scripts/build-docs and writes static HTML pages into site/docs.
Do not hand-edit files under site/docs; it is generated output and ignored by git.

What to edit for normal docs changes:

1. Update markdown under site/content/docs/
2. Run make docs
3. Commit the source markdown and generator changes only

CI runs make docs so docs edits stay reviewable without committing generated HTML.
