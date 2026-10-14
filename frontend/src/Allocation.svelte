<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, colors, coordinate } from './charts';
  import { formatDecimal } from './decimal-format.mjs';
  import type { valuation } from '../wailsjs/go/models';
  let { summary, currency }: { summary: valuation.Summary; currency: string } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  let drawable = $derived(summary.complete && !summary.negative && summary.groups.length > 0 && summary.groups.every(g => coordinate(g.percent) !== null));
  function render() {
    if (!chart) return;
    chart.clear();
    if (!drawable) return;
    chart.setOption({ animation: false, tooltip: { trigger: 'item', renderMode: 'richText', formatter: (p: unknown) => {
      const index = (p as { dataIndex: number }).dataIndex;
      const g = summary.groups[index]; return `${g.name}\n${formatDecimal(g.value)} ${currency}\n${formatDecimal(g.percent)}%`;
    } }, series: [{ type: 'pie', radius: ['38%', '76%'], label: { show: false }, data: summary.groups.map(g => ({ name: g.name, value: coordinate(g.percent), itemStyle: { color: colors[g.code] } })) }] });
  }
  $effect(() => { void summary; void drawable; void currency; render(); });
  onMount(() => {
    chart = echarts.init(root, undefined, { renderer: 'svg' }); render();
    const observer = new ResizeObserver(() => { chart?.resize(); }); observer.observe(root);
    return () => { observer.disconnect(); chart?.dispose(); chart = undefined; };
  });
</script>
<div class="allocation">
  <div class="pie" class:empty={!drawable} bind:this={root} aria-hidden="true"></div>
  {#if !summary.complete}<p class="warning">Оценка неполная</p>
  {:else if summary.negative}<p class="warning">Есть отрицательные остатки. Проверьте полноту истории; доли не показаны.</p>
  {:else if !summary.groups.length}<p class="hint">Нет данных для распределения</p>
  {:else if !drawable}<p class="warning">Значения выходят за диапазон графика. Точные суммы приведены ниже.</p>{/if}
  <ul aria-label="Стоимость по типам активов">
    {#each summary.groups as g (g.code)}
      <li><span class="dot" style:background={colors[g.code]}></span><span>{g.name}</span><strong>{formatDecimal(g.value)} {currency}{g.percent ? ` · ${formatDecimal(g.percent)}%` : ''}</strong></li>
    {/each}
  </ul>
  {#if drawable}<p class="hint">Проценты округлены; их сумма может отличаться от 100%.</p>{/if}
  {#each summary.issues as issue}<p class="warning">{issue}</p>{/each}
</div>
<style>
  .allocation { min-width: 0; } .pie { width: 100%; height: 210px; } .pie.empty { display: none; }
  ul { list-style: none; padding: 0; margin: 0; } li { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; font-size: 13px; margin: 8px 0; overflow-wrap: anywhere; }
  strong { margin-left: auto; font-weight: 550; } .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
  .hint { font-size: 12px; color: #506477; } .warning { font-size: 13px; color: #a12e2e; }
</style>
