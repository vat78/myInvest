<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import { formatDecimal } from './decimal-format.mjs';
  import type { valuation } from '../wailsjs/go/models';
  let { points }: { points: valuation.Point[] } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  function render() {
    if (!chart) return;
    chart.setOption({ animation: false, grid: { left: 95, right: 24, top: 20, bottom: 85 },
      tooltip: { trigger: 'axis', renderMode: 'richText', formatter: (params: unknown) => { const p = (params as { dataIndex: number }[])[0]; const point = points[p.dataIndex]; return `${point.date}\n${point.complete ? `${formatDecimal(point.value)} RUB` : point.issues.join('\n')}`; } },
      xAxis: { type: 'category', data: points.map(p => p.date) }, yAxis: { type: 'value', name: 'RUB' },
      dataZoom: [{ type: 'slider', bottom: 8 }, { type: 'inside' }],
      series: [{ type: 'line', connectNulls: false, showSymbol: points.length < 40, itemStyle: { color: '#219773' }, data: points.map(p => p.complete ? coordinate(p.value) : null) }] }, true);
  }
  $effect(() => { void points; render(); });
  onMount(() => { chart = echarts.init(root, undefined, { renderer: 'svg' }); render(); const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root); return () => { observer.disconnect(); chart?.dispose(); chart = undefined; }; });
</script>
<div class="history" bind:this={root} aria-hidden="true"></div>
{#if points.some(p => !p.complete)}<p class="warning">В истории есть неполные оценки. Они показаны разрывами графика.</p>{/if}
{#if points.some(p => p.complete && coordinate(p.value) === null)}<p class="warning">Некоторые суммы выходят за диапазон графика.</p>{/if}
<details><summary>Дневные значения и отсутствующие данные</summary><div class="table"><table><thead><tr><th>Дата</th><th>Совокупная стоимость, RUB</th></tr></thead><tbody>{#each points as p}<tr><td>{p.date}</td><td>{p.complete ? formatDecimal(p.value) : p.issues.join('; ')}</td></tr>{/each}</tbody></table></div></details>
<style>.history { width: 100%; height: 330px; } .warning { color: #a12e2e; font-size: 13px; } details { font-size: 13px; } .table { max-height: 300px; overflow: auto; } table { width: 100%; text-align: left; border-collapse: collapse; } td,th { padding: 8px; border-bottom: 1px solid #dce3e9; }</style>
