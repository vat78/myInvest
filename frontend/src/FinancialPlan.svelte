<script lang="ts">
  import { onMount, untrack } from 'svelte';
  import { ListCurrencies } from '../wailsjs/go/desktop/API';
  import { ListBenchmarks } from '../wailsjs/go/desktop/BenchmarksAPI';
  import { GetFinancialPlan, PreviewFinancialPlan, SaveFinancialPlan } from '../wailsjs/go/desktop/PlanningAPI';
  import type { planning } from '../wailsjs/go/models';
  import { localDate } from './calendar';
  import { formatDecimal, formatChartAmount } from './decimal-format.mjs';
  import PlanChart from './PlanChart.svelte';
  let { revision = 0, active = false }: { revision?: number; active?: boolean } = $props();
  let currencies = $state<string[]>([]);
  let benchmarks = $state<Awaited<ReturnType<typeof ListBenchmarks>>>([]);
  let currencyCode = $state('');
  function defaults(id: string): planning.Parameters {
    return { currencyCode: id, revision: '', startYear: localDate().slice(0,4), years: '10', mode: 'goal', goal: '1000000', contribution: '100000', returnPercent: '10', inflationPercent: '5', benchmarkID: '', allowEstimated: false, initialCapital: '0' };
  }
  let draft = $state<planning.Parameters>(defaults(''));
  let plan = $state<planning.Plan | null>(null), preview = $state<planning.Scenario | null>(null);
  let errors = $state<Record<string,string>>({}), message = $state(''), feedback = $state('');
  let busy = $state(false), loading = $state(false), listError = $state('');
  let alive = true, request = 0, listRequest = 0;
  const currency = $derived(currencyCode);
  const scenario = $derived(preview ?? plan?.current);
  const sources: Record<string,string> = { historical: 'Факт: индекс на 31 декабря', expected: 'Ожидаемая', estimated: 'Оценка вместо отсутствующего факта', missing: 'Нет границ индекса' };
  const fieldLabels: Record<string,string> = { currencyCode: 'Валюта', revision: 'Сохранённый план', startYear: 'Начальный год', years: 'Число лет', mode: 'Режим', goal: 'Цель', contribution: 'Взнос', returnPercent: 'Доходность', inflationPercent: 'Инфляция', benchmarkID: 'Индекс', initialCapital: 'Начальный капитал', date: 'Дата расчёта' };
  async function refreshLists() {
    const id = ++listRequest; listError = '';
    try { const [codes, bs] = await Promise.all([ListCurrencies(), ListBenchmarks()]);
      if (!alive || id !== listRequest) return;
      currencies = codes; benchmarks = bs;
    } catch { if (alive && id === listRequest) listError = 'Не удалось загрузить валюты и бенчмарки.'; }
  }
  function changed() { preview = null; errors = {}; feedback = ''; ++request; }
  async function load(reset: boolean) {
    const id = ++request, pid = currencyCode; loading = true; message = ''; preview = null;
    if (reset) { plan = null; draft = defaults(pid); errors = {}; feedback = ''; }
    try {
      const result = await GetFinancialPlan(pid, localDate());
      if (!alive || id !== request || pid !== currencyCode) return;
      errors = result.errors; message = result.message;
      if (!message && !Object.keys(errors).length) { plan = result.plan.found ? result.plan : null; if (reset && plan) draft = { ...plan.parameters }; }
    } catch { if (alive && id === request) message = 'Не удалось загрузить финансовый план. Показанный прогноз может быть устаревшим. Повторите попытку.'; }
    finally { if (alive && id === request) loading = false; }
  }
  function selectCurrency() { if (busy) return; void load(true); }
  async function calculate(save: boolean) {
    if (busy || loading || !currencyCode) return;
    busy = true; message = ''; errors = {}; feedback = ''; preview = null;
    const id = ++request, input = { ...draft, currencyCode };
    try {
      const result = await (save ? SaveFinancialPlan(input, localDate()) : PreviewFinancialPlan(input, localDate()));
      if (!alive || id !== request) return;
      errors = result.errors; message = result.message;
      if (message || Object.keys(errors).length) return;
      if (save) { plan = result.plan; draft = { ...result.plan.parameters }; feedback = 'План сохранён. Исходный сценарий зафиксирован.'; }
      else preview = result.plan.current;
    } catch { if (alive && id === request) message = 'Не удалось обработать финансовый план. Повторите попытку.'; }
    finally { if (alive) busy = false; }
  }
  onMount(() => {
    void refreshLists();
    const refresh = () => { if (active && currencyCode && !busy && !loading) void load(false); };
    window.addEventListener('focus', refresh);
    return () => { alive = false; ++request; ++listRequest; window.removeEventListener('focus', refresh); };
  });
  $effect(() => { void revision; if (active) untrack(() => { void refreshLists(); if (currencyCode && !busy && !loading) void load(false); }); });
</script>
<section aria-label="Финансовый план">
  <h2>Финансовый план</h2>
  <p class="hint">Теоретический годовой план в выбранной валюте. Взносы в начале года; операции автоматически не создаются. Комиссии, налоги и снятия не моделируются.</p>
  {#if listError}<p class="error" role="alert">{listError} <button onclick={() => void refreshLists()}>Повторить</button></p>{/if}
  <label>Валюта <select bind:value={currencyCode} onchange={selectCurrency} disabled={busy || loading}><option value="">Выберите валюту</option>{#each currencies as code}<option value={code}>{code}</option>{/each}</select></label>
  {#if currencyCode}
    {#if loading}<p role="status">Загрузка…</p>{/if}
    <div class="actions"><button disabled={busy || loading} onclick={() => void load(true)}>Загрузить сохранённые параметры</button><button disabled={busy || loading || !plan} onclick={() => void load(false)}>Обновить прогноз по индексу</button></div>
    <form onsubmit={e => { e.preventDefault(); void calculate(true); }} oninput={changed} onchange={changed}>
      <fieldset disabled={busy || loading}>
        <div class="grid">
          <label>Начальный год (2–9998)<input type="text" inputmode="numeric" bind:value={draft.startYear} required /></label>
          <label>Число лет (1–200)<input type="text" inputmode="numeric" bind:value={draft.years} required /></label>
          <label>Режим<select bind:value={draft.mode}><option value="goal">Взнос для достижения цели</option><option value="contribution">Капитал при заданном взносе</option></select></label>
          {#if draft.mode === 'goal'}<label>Цель в деньгах 1 января начального года, {currency}<input type="text" inputmode="decimal" bind:value={draft.goal} required /></label>{:else}<label>Первый годовой взнос, {currency}<input type="text" inputmode="decimal" bind:value={draft.contribution} required /></label>{/if}
          <label>Номинальная годовая доходность, %<input type="text" inputmode="decimal" bind:value={draft.returnPercent} required /></label>
          <label>Ожидаемая годовая инфляция, %<input type="text" inputmode="decimal" bind:value={draft.inflationPercent} required /></label>
          <label>Начальный капитал, {currency}<input type="text" inputmode="decimal" bind:value={draft.initialCapital} required /></label>
          <label>Индекс покупательной способности {currency}<select bind:value={draft.benchmarkID} required><option value="">Выберите индекс</option>{#each benchmarks as b}<option value={b.id}>{b.name}{b.currencyCode ? ` · ${b.currencyCode}` : ''}</option>{/each}</select></label>
        </div>
        <p class="hint">Выберите индекс, соответствующий покупательной способности выбранной валюты. Для завершённых лет нужны значения строго на 31 декабря предыдущего и расчётного года. Конечный год — не позднее 9999.</p>
        <label class="check"><input type="checkbox" bind:checked={draft.allowEstimated} />Использовать ожидаемую инфляцию при отсутствии исторических границ (оценочные годы)</label>
        <p class="hint">Рекомендуемый первый взнос округляется вверх до 0,01. Последующие взносы индексируются без промежуточного округления.</p>
      </fieldset>
      {#each Object.entries(errors) as [field, error]}<p class="error" role="alert">{fieldLabels[field] ?? 'План'}: {error}</p>{/each}
      <div class="actions"><button type="button" disabled={busy || loading} onclick={() => void calculate(false)}>Рассчитать параметры</button><button type="submit" disabled={busy || loading}>{busy ? 'Обработка…' : 'Сохранить новый исходный сценарий'}</button></div>
    </form>
    {#if message}<p class="error" role="alert">{message}</p>{/if}
    {#if feedback}<p role="status">{feedback}</p>{/if}
    {#if scenario}
      <h3>{preview ? 'Предварительный расчёт введённых параметров' : 'Актуальный прогноз сохранённого плана'}</h3>
      {#if !preview && plan}<p>Сохранённый начальный капитал: {formatDecimal(plan.parameters.initialCapital)} {plan.currency}</p>{/if}
      <p class="hint">Пересчёт прошлых лет — теоретическая рекомендация, а не фактически внесённые суммы.</p>
      {#each scenario.issues as issue}<p class="error" role="alert">{issue}</p>{/each}
      {#if scenario.firstContribution}<p>Первый ежегодный взнос: <strong>{formatDecimal(scenario.firstContribution)} {currency}</strong></p>{/if}
      {#if scenario.complete}<p>Конечный капитал: {formatChartAmount(scenario.finalCapital)} {currency}; покупательная способность на базовую дату: {formatChartAmount(scenario.finalRealCapital)} {currency}.</p>
        {#if scenario.excess}<p>Цель {scenario.goalReached ? 'достигнута' : 'не достигнута'}. Превышение цели: {formatDecimal(scenario.excess)} {currency}.</p>{/if}
      {:else}<p class="error">Расчёт неполный. Конечный капитал и рекомендация для цели недоступны.</p>{/if}
      {#if scenario.points.length}<PlanChart points={scenario.points} {currency} />{/if}
      <div class="table"><table><thead><tr><th>Год</th><th>Капитал до взноса</th><th>Взнос 1 января</th><th>Инфляция, %</th><th>Источник</th><th>Инвестиционный прирост</th><th>Капитал на следующее 1 января</th><th>В деньгах базовой даты</th></tr></thead><tbody>{#each scenario.rows as row}<tr><td>{row.year}</td><td>{formatChartAmount(row.before)}</td><td>{formatChartAmount(row.contribution)}</td><td>{formatDecimal(row.inflation)}</td><td>{sources[row.source]}</td><td>{formatChartAmount(row.income)}</td><td>{formatChartAmount(row.capital)}</td><td>{formatChartAmount(row.realCapital)}</td></tr>{/each}</tbody></table></div>
    {/if}
    {#if plan}<details><summary>Исходный сценарий — сохранённый снимок</summary>
      <p>Первый взнос: {plan.original.firstContribution ? formatDecimal(plan.original.firstContribution) : 'недоступен'} {plan.currency}. Конечный капитал: {plan.original.complete ? formatChartAmount(plan.original.finalCapital) : 'неполный расчёт'} {plan.currency}; в деньгах базовой даты: {plan.original.complete ? formatChartAmount(plan.original.finalRealCapital) : 'недоступен'}.</p>
      {#each plan.original.issues as issue}<p class="error">{issue}</p>{/each}
      <div class="table"><table><thead><tr><th>Год</th><th>Взнос</th><th>Инфляция, %</th><th>Источник</th><th>Доход</th><th>Капитал</th><th>Реальный капитал</th></tr></thead><tbody>{#each plan.original.rows as row}<tr><td>{row.year}</td><td>{formatChartAmount(row.contribution)}</td><td>{formatDecimal(row.inflation)}</td><td>{sources[row.source]}</td><td>{formatChartAmount(row.income)}</td><td>{formatChartAmount(row.capital)}</td><td>{formatChartAmount(row.realCapital)}</td></tr>{/each}</tbody></table></div>
    </details>{/if}
  {/if}
</section>
<style>
  section { background: white; border: 1px solid #dce3e9; border-radius: 14px; padding: 24px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit,minmax(230px,1fr)); gap: 16px; }
  label { display: flex; flex-direction: column; gap: 8px; margin-bottom: 14px; font-size: 14px; }
  input,select,button { font: inherit; padding: 10px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; color: inherit; min-width: 0; }
  button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  fieldset { border: 0; padding: 0; margin-top: 20px; min-width: 0; }
  .check { flex-direction: row; align-items: center; }.check input { width: auto; }
  .hint { color: #506477; font-size: 13px; }.error { color: #a12e2e; overflow-wrap: anywhere; }
  .actions { display: flex; gap: 12px; flex-wrap: wrap; margin: 16px 0; }
  .table { overflow: auto; max-height: 480px; } table { width: 100%; border-collapse: collapse; font-size: 13px; text-align: left; }td,th { padding: 10px; border-bottom: 1px solid #dce3e9; min-width: 100px; }
  details { margin-top: 20px; }
</style>
