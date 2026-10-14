<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import { formatChartAmount } from './decimal-format.mjs';
  import type { valuation } from '../wailsjs/go/models';
  let { analysis }: { analysis: valuation.Analysis } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  let drawable = $derived(analysis.complete && !analysis.negative && analysis.assets.some(a => /[1-9]/.test(a.chartValue)) && analysis.assets.every(a => a.chartValue !== '' && coordinate(a.chartValue) !== null));
  const palette = ['#219773', '#5470c6', '#bf8b36', '#9b6fc3', '#73a0ac', '#ee8a43', '#d65f85', '#454e59'];
  function render() {
    if (!chart) return;
    chart.clear();
    if (!drawable) return;
    chart.setOption({ animation: false, color: palette,
      tooltip: { trigger: 'item', renderMode: 'richText', formatter: (p: unknown) => {
        const a = analysis.assets[(p as { dataIndex: number }).dataIndex];
        return `${a.ticker}\n${formatChartAmount(a.currentValue)} ${analysis.currency}\n${formatChartAmount(a.percent)}%`;
      } }, series: [{ type: 'pie', radius: ['35%', '75%'], label: { show: false }, data: analysis.assets.map(a => ({ name: a.ticker, value: coordinate(a.chartValue) })) }] });
  }
  $effect(() => { void analysis; void drawable; render(); });
  onMount(() => {
    chart = echarts.init(root, undefined, { renderer: 'svg' }); render();
    const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root);
    return () => { observer.disconnect(); chart?.dispose(); chart = undefined; };
  });
</script>
<div class="pie" class:empty={!drawable} bind:this={root} aria-hidden="true"></div>
{#if drawable}
  <ul aria-label="Доли активов">{#each analysis.assets as a, i (a.id)}<li><span class="dot" style:background={palette[i % palette.length]}></span><span>{a.ticker}</span><strong>{formatChartAmount(a.percent)}%</strong></li>{/each}</ul>
{:else if analysis.complete && !analysis.negative}
  <p>Нет данных в диапазоне диаграммы. Значения приведены в таблице.</p>
{:else}<p>Диаграмма недоступна. Проверьте данные портфеля.</p>{/if}
<style>
  .pie { height: 270px; width: 100%; } .empty { display: none; }
  ul { list-style: none; padding: 0; margin: 0; } li { display: flex; gap: 8px; align-items: center; margin: 10px 0; font-size: 13px; overflow-wrap: anywhere; }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; } strong { margin-left: auto; white-space: nowrap; }
  p { font-size: 13px; color: #506477; }
</style>
