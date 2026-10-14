<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import { formatChartAmount } from './decimal-format.mjs';
  import type { planning } from '../wailsjs/go/models';
  let { points, currency }: { points: planning.Point[]; currency: string } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  const labels = ['Номинальный капитал', 'В деньгах базовой даты'];
  const axis = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 });
  function render() {
    if (!chart) return;
    chart.setOption({ animation: false, legend: { data: labels }, grid: { left: 20, right: 20, top: 50, bottom: 70, containLabel: true },
      tooltip: { trigger: 'axis', renderMode: 'richText', formatter: (params: unknown) => {
        const index = (params as { dataIndex: number }[])[0]?.dataIndex; const p = points[index];
        return p ? `${p.date}\n${labels[0]}: ${formatChartAmount(p.nominal)} ${currency}\n${labels[1]}: ${formatChartAmount(p.real)} ${currency}` : '';
      } }, xAxis: { type: 'category', data: points.map(p => p.date) }, yAxis: { type: 'value', name: currency, axisLabel: { formatter: (value: number) => axis.format(value) } },
      dataZoom: [{ type: 'slider', bottom: 5 }, { type: 'inside' }], series: [
        { name: labels[0], type: 'line', connectNulls: false, data: points.map(p => coordinate(p.nominal)), itemStyle: { color: '#219773' } },
        { name: labels[1], type: 'line', connectNulls: false, data: points.map(p => coordinate(p.real)), itemStyle: { color: '#5470c6' } },
      ] }, true);
  }
  $effect(() => { void points; void currency; render(); });
  onMount(() => { chart = echarts.init(root, undefined, { renderer: 'svg' }); render(); const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root); return () => { observer.disconnect(); chart?.dispose(); chart = undefined; }; });
</script>
<div bind:this={root} class="chart" aria-hidden="true"></div>
{#if points.some(p => coordinate(p.nominal) === null || coordinate(p.real) === null)}<p>Некоторые суммы выходят за диапазон графика; значения доступны в таблице.</p>{/if}
<style>.chart { width: 100%; height: 360px; }</style>
