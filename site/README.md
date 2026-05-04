# Site Documentation Structure

This directory contains the project marketing site plus generated monitor docs.

Canonical source for monitor documentation lives in:

- site/md/*.md

Generated HTML output lives in:

- site/docs/

Regenerate docs with:

```bash
make docs
```

The docs build uses scripts/build-docs and writes static HTML pages into site/docs.
Do not hand-edit files under site/docs unless you are changing the generator output format
and intentionally validating generated HTML changes.

What to edit for normal docs changes:

1. Update markdown under site/md/
2. Run make docs
3. Commit both source markdown and regenerated site/docs output

This keeps docs edits reviewable while ensuring published static output remains in sync.
