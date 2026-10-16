<script lang="ts">
  import { onMount } from 'svelte';
  import { echarts, coordinate } from './charts';
  import { formatChartAmount } from './decimal-format.mjs';
  import type { planning, valuation } from '../wailsjs/go/models';
  let { points, currency, actual = [] }: { points: planning.Point[]; currency: string; actual?: valuation.CapitalPoint[] } = $props();
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  const labels = ['Номинальный капитал', 'В деньгах базовой даты', 'Фактическая стоимость всех портфелей'];
  const axis = new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 2 });
  // Dates and amounts become numbers exclusively for chart geometry.
  const timestamp = (date: string) => Date.parse(`${date}T00:00:00Z`);
  function render() {
    if (!chart) return;
    chart.setOption({ animation: false, useUTC: true, legend: { data: labels, top: 0, type: 'scroll' }, grid: { left: 20, right: 20, top: 65, bottom: 70, containLabel: true },
      tooltip: { trigger: 'axis', confine: true, renderMode: 'richText', formatter: (params: unknown) => {
        return (params as { seriesIndex: number; dataIndex: number }[]).map(({ seriesIndex, dataIndex }) => {
          if (seriesIndex === 2) {
            const p = actual[dataIndex];
            return p ? `${p.date}\n${labels[2]}: ${p.complete ? `${formatChartAmount(p.value)} ${currency}` : p.issues.join('; ')}` : '';
          }
          const p = points[dataIndex];
          return p ? `${p.date}\n${labels[seriesIndex]}: ${formatChartAmount(seriesIndex === 0 ? p.nominal : p.real)} ${currency}` : '';
        }).filter(Boolean).join('\n');
      } }, xAxis: { type: 'time', axisLabel: { formatter: (value: number) => {
        const date = new Date(value);
        return date.getUTCMonth() === 0 && date.getUTCDate() === 1 ? String(date.getUTCFullYear()).padStart(4, '0') : new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'short', timeZone: 'UTC' }).format(date);
      } } }, yAxis: { type: 'value', name: currency, axisLabel: { formatter: (value: number) => axis.format(value) } },
      dataZoom: [{ type: 'slider', bottom: 5 }, { type: 'inside' }], series: [
        { name: labels[0], type: 'line', connectNulls: false, data: points.map(p => [timestamp(p.date), coordinate(p.nominal)]), itemStyle: { color: '#219773' } },
        { name: labels[1], type: 'line', connectNulls: false, data: points.map(p => [timestamp(p.date), coordinate(p.real)]), itemStyle: { color: '#5470c6' } },
        { name: labels[2], type: 'line', connectNulls: false, showSymbol: actual.length < 40, data: actual.map(p => [timestamp(p.date), p.complete ? coordinate(p.value) : null]), itemStyle: { color: '#d99036' } },
      ] }, true);
  }
  $effect(() => { void points; void currency; void actual; render(); });
  onMount(() => { chart = echarts.init(root, undefined, { renderer: 'svg' }); render(); const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root); return () => { observer.disconnect(); chart?.dispose(); chart = undefined; }; });
</script>
<div bind:this={root} class="chart" aria-hidden="true"></div>
{#if points.some(p => coordinate(p.nominal) === null || coordinate(p.real) === null) || actual.some(p => p.complete && coordinate(p.value) === null)}<p>Некоторые суммы выходят за диапазон графика; значения доступны в таблице.</p>{/if}
{#if actual.length}
  <p class="hint">Фактическая стоимость денег и активов всех портфелей в {currency} на конец дня; до сегодняшнего дня, в пределах периода расчёта. Пропуски цен или прямых курсов показаны разрывами.</p>
  <details><summary>Фактическая стоимость портфелей по дням</summary><div class="table"><table><thead><tr><th>Дата</th><th>Стоимость, {currency}</th></tr></thead><tbody>{#each actual as p}<tr><td>{p.date}</td><td>{p.complete ? formatChartAmount(p.value) : p.issues.join('; ')}</td></tr>{/each}</tbody></table></div></details>
{/if}
<style>.chart { width: 100%; height: 380px; }.hint { color: #506477; font-size: 13px; }details { font-size: 13px; }.table { max-height: 300px; overflow: auto; }table { width: 100%; text-align: left; border-collapse: collapse; }td,th { padding: 8px; border-bottom: 1px solid #dce3e9; }</style>
