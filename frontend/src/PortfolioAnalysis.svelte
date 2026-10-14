<script lang="ts">
  import { onMount } from 'svelte';
  import { ListPortfolios } from '../wailsjs/go/desktop/PortfolioAPI';
  import { GetPortfolioAnalysis } from '../wailsjs/go/desktop/OverviewAPI';
  import { localDate } from './calendar';
  import { formatDecimal, formatChartAmount } from './decimal-format.mjs';
  import AnalysisPie from './AnalysisPie.svelte';
  import TargetAllocation from './TargetAllocation.svelte';
  let targetAllocation = $state<TargetAllocation>();

  let { revision = 0, onquotes = () => {} }: { revision?: number; onquotes?: () => void } = $props();
  let portfolios = $state<Awaited<ReturnType<typeof ListPortfolios>>>([]);
  let selected = $state('');
  let data = $state<Awaited<ReturnType<typeof GetPortfolioAnalysis>> | null>(null);
  let loading = $state(false), error = $state('');
  let request = 0, alive = true;
  async function refresh() {
    const token = ++request, day = localDate();
    loading = true; error = ''; data = null;
    try {
      const list = await ListPortfolios();
      if (!alive || token !== request) return;
      portfolios = list;
      if (!list.some(p => p.id === selected)) selected = list[0]?.id ?? '';
      if (!selected) return;
      const pid = selected;
      const result = await GetPortfolioAnalysis(pid, day);
      if (alive && token === request && pid === selected && day === localDate()) data = result;
    } catch {
      if (alive && token === request) error = 'Не удалось загрузить анализ портфеля.';
    } finally { if (alive && token === request) loading = false; }
  }
  $effect(() => { void revision; void refresh(); });
  onMount(() => () => { alive = false; ++request; });
  function changePortfolio(event: Event) {
    const select = event.currentTarget as HTMLSelectElement;
    if (targetAllocation && !targetAllocation.canLeave()) { select.value = selected; return; }
    selected = select.value; void refresh();
  }
</script>
<section aria-label="Анализ портфеля">
  <label>Портфель <select value={selected} onchange={changePortfolio} disabled={!portfolios.length}>
    {#each portfolios as p (p.id)}<option value={p.id}>{p.name} · {p.currencyCode}</option>{/each}
  </select></label>
  {#if loading}<p role="status">Загрузка анализа…</p>{/if}
  {#if error}<p class="warning" role="alert">{error} <button onclick={() => void refresh()}>Повторить</button></p>
  {:else if !loading && !portfolios.length}<p>Портфелей пока нет. Добавьте портфель через меню.</p>{/if}
  {#if data}
    <p class="hint">На {data.date.split('-').reverse().join('.')} · {data.currency}. Последняя котировка или сделка; деньги и комиссии исключены. Будущие операции не учитываются.</p>
    {#if !data.assets.length}<p>В портфеле нет открытых позиций.</p>
    {:else}
      <div class="layout">
        <article><h2>Структура активов</h2><AnalysisPie analysis={data} /></article>
        <article class="details"><h2>Активы портфеля</h2>
          <p>{data.complete ? `Текущая стоимость: ${formatChartAmount(data.total)} ${data.currency}` : 'Оценка неполная'}</p>
          <div class="scroll"><table>
            <caption>Количество и стоимость активов · {data.currency}</caption>
            <thead><tr><th scope="col">Актив</th><th scope="col">Количество бумаг</th><th scope="col">Текущая стоимость</th><th scope="col">Источник цены</th><th scope="col">Доля</th><th scope="col">Средняя цена покупки</th><th scope="col">Стоимость по средней цене</th></tr></thead>
            <tbody>{#each data.assets as a (a.id)}<tr><th scope="row">{a.ticker}</th><td>{formatDecimal(a.quantity)}</td><td>{a.currentValue ? formatChartAmount(a.currentValue) : 'Нет цены'}</td><td class="source">{a.priceDate ? `${a.priceSource === 'quote' ? 'Котировка' : 'Сделка'} · ${a.priceDate.split('-').reverse().join('.')} · ${a.priceCurrency}` : '—'}
              {#if a.priceRate}<span>1 {a.priceCurrency} = {formatDecimal(a.priceRate)} {data.currency} · курс от {a.priceRateDate.split('-').reverse().join('.')}</span>{/if}
            </td><td>{a.percent ? `${formatChartAmount(a.percent)}%` : '—'}</td><td>{a.averagePrice ? formatChartAmount(a.averagePrice) : '—'}</td><td>{a.averageValue ? formatChartAmount(a.averageValue) : '—'}</td></tr>{/each}</tbody>
          </table></div>
          <p class="hint">Средняя цена: сумма всех покупок / купленное количество с учётом сплитов. Стоимости рассчитаны до округления цен.</p>
          <p class="hint">Суммы и проценты округлены до двух знаков; сумма долей может отличаться от 100%.</p>
        </article>
      </div>
    {/if}
    {#if data.negative}<p class="warning">Есть отрицательные позиции. Проверьте полноту истории; доли и диаграмма недоступны.</p>{/if}
    {#each data.issues as issue}<p class="warning">{issue}</p>{/each}
  {/if}
  {#if selected}<TargetAllocation bind:this={targetAllocation} pid={selected} {revision} {onquotes} />{/if}
</section>
<style>
  label { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 16px; }
  select, button { font: inherit; padding: 10px 12px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; color: inherit; max-width: 100%; }
  .layout { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(0, 2.2fr); gap: 20px; }
  article { min-width: 0; padding: 20px; background: white; border: 1px solid #dce3e9; border-radius: 14px; }
  h2 { font-size: 18px; margin: 0 0 12px; } .scroll { overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; font-size: 13px; } caption { text-align: left; margin-bottom: 12px; color: #506477; }
  th, td { padding: 12px 8px; border-bottom: 1px solid #dce3e9; text-align: right; font-variant-numeric: tabular-nums; }
  thead th { min-width: 110px; } th:first-child { text-align: left; overflow-wrap: anywhere; }
  td { white-space: nowrap; } .source { white-space: normal; min-width: 160px; font-size: 12px; } .source span { display: block; margin-top: 6px; overflow-wrap: anywhere; } .hint { font-size: 13px; color: #506477; } .warning { color: #a12e2e; }
  @media (max-width: 850px) { .layout { grid-template-columns: 1fr; } }
</style>
