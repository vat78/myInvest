<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import type { valuation } from '../wailsjs/go/models';
  let { rows }: { rows: valuation.AllocationRow[] } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  const palette = ['#219773', '#5470c6', '#bf8b36', '#9b6fc3', '#73a0ac', '#ee8a43', '#8895a7'];
  let sectors = $derived(rows.filter(row => /[1-9]/.test(row.percent)));
  function render() {
    if (!chart) return;
    chart.setOption({ animation: false, color: palette,
      tooltip: { trigger: 'item', renderMode: 'richText', formatter: (p: unknown) => {
        const row = sectors[(p as { dataIndex: number }).dataIndex];
        return `${row.ticker}\n${row.percent}%`;
      } }, series: [{ type: 'pie', radius: ['35%', '75%'], label: { show: false }, data: sectors.map(row => ({ name: row.ticker, value: coordinate(row.percent) })) }] });
  }
  $effect(() => { void sectors; render(); });
  onMount(() => {
    chart = echarts.init(root, undefined, { renderer: 'svg' }); render();
    const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root);
    return () => { observer.disconnect(); chart?.dispose(); chart = undefined; };
  });
</script>
<div class="pie" bind:this={root} aria-hidden="true"></div>
<ul aria-label="Целевые доли">{#each sectors as row, i (row.id)}<li><span class="dot" style:background={palette[i % palette.length]}></span><span>{row.ticker}</span><strong>{row.percent}%</strong></li>{/each}</ul>
<style>
  .pie { height: 260px; width: 100%; } ul { list-style: none; padding: 0; } li { display: flex; gap: 8px; align-items: center; margin: 10px 0; font-size: 13px; overflow-wrap: anywhere; } .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; } strong { margin-left: auto; }
</style>
