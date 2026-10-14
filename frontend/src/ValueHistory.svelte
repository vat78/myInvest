<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import { formatChartAmount } from './decimal-format.mjs';
  import type { valuation } from '../wailsjs/go/models';
  let { points }: { points: valuation.Point[] } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  // Axis ticks describe approximate chart coordinates, not exact portfolio values.
  const axisAmount = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  function render() {
    if (!chart) return;
    chart.setOption({ animation: false, grid: { left: 24, right: 24, top: 45, bottom: 85, containLabel: true },
      legend: { top: 0, data: ['Стоимость портфелей', 'Пополнения накопительным итогом'] },
      tooltip: { trigger: 'axis', renderMode: 'richText', formatter: (params: unknown) => {
        const p = (params as { dataIndex: number }[])[0];
        const point = points[p.dataIndex];
        const value = point.complete ? `${formatChartAmount(point.value)} RUB` : point.issues.join('; ');
        const deposits = point.depositsComplete ? `${formatChartAmount(point.deposits)} RUB` : point.depositIssues.join('; ');
        return `${point.date}\nСтоимость портфелей: ${value}\nПополнения накопительным итогом: ${deposits}`;
      } },
      xAxis: { type: 'category', data: points.map(p => p.date) }, yAxis: { type: 'value', name: 'RUB', axisLabel: { formatter: (value: number) => axisAmount.format(value) } },
      dataZoom: [{ type: 'slider', bottom: 8 }, { type: 'inside' }],
      series: [
        { name: 'Стоимость портфелей', type: 'line', connectNulls: false, showSymbol: points.length < 40, itemStyle: { color: '#219773' }, data: points.map(p => p.complete ? coordinate(p.value) : null) },
        { name: 'Пополнения накопительным итогом', type: 'line', step: 'end', connectNulls: false, showSymbol: points.length < 40, z: 1, lineStyle: { type: 'solid', width: 1 }, areaStyle: { color: '#f4a64b', opacity: 0.2 }, itemStyle: { color: '#d99036' }, data: points.map(p => p.depositsComplete ? coordinate(p.deposits) : null) },
      ] }, true);
  }
  $effect(() => { void points; render(); });
  onMount(() => { chart = echarts.init(root, undefined, { renderer: 'svg' }); render(); const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root); return () => { observer.disconnect(); chart?.dispose(); chart = undefined; }; });
</script>
<div class="history" bind:this={root} aria-hidden="true"></div>
{#if points.some(p => !p.complete)}<p class="warning">В истории есть неполные оценки. Они показаны разрывами графика.</p>{/if}
{#if points.some(p => p.complete && coordinate(p.value) === null)}<p class="warning">Некоторые суммы выходят за диапазон графика.</p>{/if}
{#if points.some(p => !p.depositsComplete)}<p class="warning">У некоторых пополнений нет сохранённой рублёвой суммы. Накопительный итог показан с разрывами; пересчёт по курсу не выполняется.</p>{/if}
{#if points.some(p => p.depositsComplete && coordinate(p.deposits) === null)}<p class="warning">Некоторые суммы пополнений выходят за диапазон графика.</p>{/if}
<details><summary>Дневные значения и отсутствующие данные</summary><div class="table"><table><thead><tr><th>Дата</th><th>Совокупная стоимость, RUB</th><th>Пополнения накопительным итогом, RUB</th></tr></thead><tbody>{#each points as p}<tr><td>{p.date}</td><td>{p.complete ? formatChartAmount(p.value) : p.issues.join('; ')}</td><td>{p.depositsComplete ? formatChartAmount(p.deposits) : p.depositIssues.join('; ')}</td></tr>{/each}</tbody></table></div></details>
<style>.history { width: 100%; height: 330px; } .warning { color: #a12e2e; font-size: 13px; } details { font-size: 13px; } .table { max-height: 300px; overflow: auto; } table { width: 100%; text-align: left; border-collapse: collapse; } td,th { padding: 8px; border-bottom: 1px solid #dce3e9; }</style>
