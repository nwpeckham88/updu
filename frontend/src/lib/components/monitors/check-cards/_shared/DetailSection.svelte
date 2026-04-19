<script lang="ts">
    import type { Snippet } from "svelte";

    let {
        title,
        children,
    }: {
        title: string;
        children: Snippet;
    } = $props();
</script>

<section class="space-y-3">
    <h3
        class="text-[11px] font-semibold uppercase tracking-[0.16em] text-text-subtle"
    >
        {title}
    </h3>
    <div class="grid gap-3 sm:grid-cols-2">
        {@render children()}
    </div>
</section>
