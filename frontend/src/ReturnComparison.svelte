<script lang="ts">
  import { onMount } from 'svelte';
  import { GetReturnComparison } from '../wailsjs/go/desktop/ComparisonAPI';
  import { echarts, coordinate } from './charts';
  import { formatDecimal } from './decimal-format.mjs';
  import { localDate } from './calendar';
  import type { benchmarks } from '../wailsjs/go/models';
  let { benchmarks: rows, revision = 0 }: { benchmarks: benchmarks.Benchmark[]; revision?: number } = $props();
  let inflationID = $state('');
  let metric = $state('cumulative');
  let result = $state<Awaited<ReturnType<typeof GetReturnComparison>> | null>(null);
  let loading = $state(false), error = $state('');
  let root: HTMLDivElement;
  let chart: ReturnType<typeof echarts.init> | undefined;
  let alive = true, request = 0;
  const eligible = $derived(rows.filter(b => !b.currencyCode || b.currencyCode === 'RUB'));
  const label = (date: string) => date.endsWith('-12-31') ? date.slice(0, 4) : `${date.slice(0, 4)} (по ${date.split('-').reverse().join('.')})`;
  async function refresh() {
    const id = ++request, selected = inflationID;
    result = null; error = ''; loading = !!selected;
    if (!selected) return;
    try {
      const value = await GetReturnComparison(selected, localDate());
      if (alive && id === request && selected === inflationID) result = value;
    } catch (reason) { if (alive && id === request) error = String(reason); }
    finally { if (alive && id === request) loading = false; }
  }
  $effect(() => { revision; rows; inflationID; void refresh(); });
  function render() {
    if (!chart) return;
    if (!result) { chart.clear(); return; }
    const data = result;
    const field = metric === 'annual' ? 'annual' : 'cumulative';
    chart.setOption({ animation: false,
      grid: { left: 12, right: 20, top: 60, bottom: 75, containLabel: true },
      legend: { type: 'scroll', top: 0, data: data.series.map(s => s.name) },
      tooltip: { trigger: 'axis', renderMode: 'richText', confine: true, formatter: (params: unknown) => {
        const entries = params as { seriesIndex: number; dataIndex: number }[];
        if (!entries.length) return '';
        return [label(data.dates[entries[0].dataIndex]), ...entries.map(e => {
          const s = data.series[e.seriesIndex], p = s.points.find(p => p.date === data.dates[e.dataIndex]);
          if (!p || p.cumulative === '') return `${s.name}: ${p?.issue || 'Нет данных'}`;
          return `${s.name}: ${formatDecimal(p.cumulative)}%\nЗа период года: ${p.annual === '' ? '—' : `${formatDecimal(p.annual)}%`}${p.issue ? `\n${p.issue}` : ''}`;
        })].join('\n');
      } },
      xAxis: { type: 'category', data: data.dates.map(d => d.slice(0, 4) + (d.endsWith('-12-31') ? '' : '*')), axisLabel: { interval: data.dates.length <= 8 ? 0 : 'auto' } },
      yAxis: { type: 'value', name: '%', axisLabel: { formatter: (v: number) => `${v.toLocaleString('ru-RU')}%` } },
      dataZoom: [{ type: 'slider', bottom: 4 }, { type: 'inside' }],
      series: data.series.map(s => ({ name: s.name, type: 'line', connectNulls: false, showSymbol: true,
        lineStyle: { type: s.id.startsWith('portfolio:') ? 'solid' : 'dashed' },
        data: data.dates.map(d => { const p = s.points.find(p => p.date === d); return p && p[field] !== '' ? coordinate(p[field]) : null; }) })),
    }, true);
  }
  $effect(() => { result; metric; render(); });
  onMount(() => {
    chart = echarts.init(root, undefined, { renderer: 'svg' }); render();
    const observer = new ResizeObserver(() => chart?.resize()); observer.observe(root);
    return () => { alive = false; ++request; observer.disconnect(); chart?.dispose(); chart = undefined; };
  });
</script>
<section aria-labelledby="return-comparison-title">
  <h2 id="return-comparison-title">Доходность портфелей и бенчмарков</h2>
  <p class="hint">Доходность в RUB, %. Годовые точки; текущий год — неполный, по {localDate().split('-').reverse().join('.')} (*). Портфели: TWR с пополнениями и снятиями на конец дня, включая комиссии и дивиденды.</p>
  <label>Бенчмарк рублёвой инфляции
    <select bind:value={inflationID}>
      <option value="">Выберите инфляцию для общей базы сравнения</option>
      {#each eligible as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
    </select>
  </label>
  <label class="metric">Показатель графика
    <select bind:value={metric}><option value="cumulative">Накопленная доходность с привязкой к инфляции</option><option value="annual">Доходность за каждый год</option></select>
  </label>
  {#if !eligible.length}<p>Добавьте бенчмарк инфляции с единицей «Индекс» или RUB и его историю.</p>{/if}
  {#if loading}<p role="status">Расчёт доходности…</p>{/if}
  {#if error}<p class="error" role="alert">{error} <button onclick={() => void refresh()}>Повторить</button></p>{/if}
  <div class="chart" bind:this={root} aria-hidden="true" hidden={!result}></div>
  {#if result}
    <p class="hint">Общая база: {result.baseDate.split('-').reverse().join('.')} = 0%. В накопленном режиме поздние ряды начинают с инфляции на дату формирования. История и прямые курсы — последние доступные на дату расчёта, без интерполяции. Нажмите на название в легенде, чтобы скрыть ряд.</p>
    {#if result.series.some(s => s.points.some(p => p.issue && p.date >= s.startDate))}<p class="warning">Не для всех периодов доступна доходность. Причины указаны в таблице; отсутствующие значения показаны разрывами.</p>{/if}
    {#if result.series.some(s => s.points.some(p => p.cumulative !== '' && coordinate(p.cumulative) === null))}<p class="warning">Некоторые значения выходят за диапазон графика. Точные проценты доступны в таблице.</p>{/if}
    <details><summary>Годовая доходность и значения графика</summary>
      <div class="table"><table><thead><tr><th>Ряд / начало</th><th>Период по</th><th>За период года, %</th><th>Накопленная, %</th><th>Примечание</th></tr></thead><tbody>
        {#each result.series as s (s.id)}
          {#if !s.points.length}<tr><td>{s.name}</td><td colspan="4">Нет истории до даты расчёта</td></tr>{/if}
          {#each s.points as p (p.date)}<tr><td>{s.name}<small>{s.startDate}</small></td><td>{p.date}</td><td>{p.annual === '' ? '—' : formatDecimal(p.annual)}</td><td>{p.cumulative === '' ? '—' : formatDecimal(p.cumulative)}</td><td>{p.issue || (p.date.slice(0,4) === s.startDate.slice(0,4) ? 'С даты начала ряда; без экстраполяции' : '')}</td></tr>{/each}
        {/each}
      </tbody></table></div>
    </details>
  {/if}
</section>
<style>
  section { background: white; border: 1px solid #dce3e9; border-radius: 12px; padding: 20px; margin-bottom: 20px; min-width: 0; }
  h2 { margin-top: 0; font-size: 20px; } .hint { color: #506477; font-size: 14px; }
  .metric { margin-top: 12px; }
  label { display: flex; flex-direction: column; gap: 8px; font-size: 14px; }
  select,button { font: inherit; padding: 10px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; color: inherit; min-width: 0; }
  button { cursor: pointer; } .chart { height: 400px; width: 100%; } .chart[hidden] { display: none; }
  .error,.warning { color: #a12e2e; overflow-wrap: anywhere; font-size: 14px; }
  details { font-size: 13px; } summary { cursor: pointer; } .table { overflow: auto; max-height: 340px; }
  table { width: 100%; border-collapse: collapse; text-align: left; } td,th { padding: 10px; border-bottom: 1px solid #dce3e9; min-width: 110px; }
  small { display: block; color: #506477; } td { font-variant-numeric: tabular-nums; overflow-wrap: anywhere; }
</style>
