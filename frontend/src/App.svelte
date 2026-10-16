<script lang="ts">
  import { onMount } from 'svelte';
  import Allocation from './Allocation.svelte';
  import PortfolioAnalysis from './PortfolioAnalysis.svelte';
  let analysisRevision = $state(0);
  import ValueHistory from './ValueHistory.svelte';
  import { GetOverview } from '../wailsjs/go/desktop/OverviewAPI';
  let overview = $state<Awaited<ReturnType<typeof GetOverview>> | null>(null);
  let overviewLoading = $state(true), overviewError = $state('');
  let overviewRequest = 0;
  let tab = $state('overview');
  async function refreshOverview() {
    analysisRevision++;
    const request = ++overviewRequest, day = localDate(); overviewLoading = true; overviewError = '';
    try { const result = await GetOverview(day); if (alive && request === overviewRequest && day === localDate()) overview = result; }
    catch { if (alive && request === overviewRequest) overviewError = 'Не удалось обновить обзор. Показанные ранее данные устарели.'; }
    finally { if (alive && request === overviewRequest) overviewLoading = false; }
  }
  function tabKeys(event: KeyboardEvent) {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    const tabs = ['overview', 'operations', 'analysis', 'quotes', 'benchmarks'];
    event.preventDefault();
    const step = event.key === 'ArrowLeft' ? -1 : 1;
    tab = event.key === 'Home' ? tabs[0] : event.key === 'End' ? tabs[tabs.length - 1] : tabs[(tabs.indexOf(tab) + step + tabs.length) % tabs.length];
    document.getElementById(`tab-${tab}`)?.focus();
  }
  import { formatDecimal, formatChartAmount } from './decimal-format.mjs';
  import PortfolioManager from './PortfolioManager.svelte';
  import CashFlowManager from './CashFlowManager.svelte';
  import AssetManager from './AssetManager.svelte';
  import QuoteManager from './QuoteManager.svelte';
  import BenchmarkManager from './BenchmarkManager.svelte';
  import BenchmarkDirectory from './BenchmarkDirectory.svelte';
  let benchmarkDirectory: BenchmarkDirectory;
  let quoteRevision = $state(0);
  let assetManager: AssetManager;
  let cashManager: CashFlowManager;
  let portfolioManager: PortfolioManager;
  let portfolioRevision = $state(0);
  import { AddCurrency, GetCurrencyRate, ListCurrencies, SaveCurrencyRate } from '../wailsjs/go/desktop/API';
  import { localDate } from './calendar';
  const normalize = (code: string) => code.trim().toUpperCase();
  let today = $state(localDate());
  let todayRate = $state('');
  let mainError = $state('');
  let currencies = $state<string[]>([]);
  let dialog: HTMLDialogElement;
  let opened = $state(false);
  let date = $state('');
  let base = $state('USD');
  let quote = $state('RUB');
  let value = $state('');
  let error = $state('');
  let loading = $state(false);
  let busy = $state(false);
  let feedback = $state('');
  let mainRequest = 0;
  let editRequest = 0;
  let alive = true;
  const unknown = $derived([...new Set([normalize(base), normalize(quote)])]
    .filter(code => /^[A-Z]{3}$/.test(code) && !currencies.includes(code)));

  async function refreshToday() {
    const id = ++mainRequest;
    const day = localDate();
    if (day !== today) todayRate = '';
    today = day;
    try {
      const result = await GetCurrencyRate({ date: day, base: 'USD', quote: 'RUB' });
      if (!alive || id !== mainRequest || day !== localDate()) return;
      todayRate = result.found ? result.displayValue : '';
      mainError = '';
    } catch (reason) {
      if (alive && id === mainRequest) mainError = `Не удалось загрузить курс: ${String(reason)}`;
    }
  }

  async function loadRate() {
    const id = ++editRequest;
    const pair = { date, base: normalize(base), quote: normalize(quote) };
    value = '';
    error = '';
    feedback = '';
    loading = false;
    if (!date || !currencies.includes(pair.base) || !currencies.includes(pair.quote) || pair.base === pair.quote) return;
    loading = true;
    try {
      const result = await GetCurrencyRate(pair);
      if (alive && opened && id === editRequest) value = result.found ? result.value : '';
    } catch (reason) {
      if (alive && opened && id === editRequest) error = String(reason);
    } finally {
      if (id === editRequest) loading = false;
    }
  }

  function openEditor(rate?: {date: string; base: string; quote: string; value: string}) {
    date = rate?.date ?? localDate(); base = rate?.base ?? 'USD'; quote = rate?.quote ?? 'RUB'; value = rate?.value ?? ''; error = ''; feedback = '';
    opened = true;
    dialog.showModal();
    if (!rate) void loadRate();
    else { ++editRequest; loading = false; }
  }
  function closeEditor() {
    if (busy) return;
    opened = false;
    ++editRequest;
    loading = false;
    dialog.close();
  }
  async function addCurrency(code: string) {
    busy = true; error = '';
    try {
      const added = await AddCurrency(code);
      currencies = [...new Set([...currencies, added])].sort();
      quoteRevision++;
    } catch (reason) { error = String(reason); }
    finally { busy = false; }
  }
  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy || loading) return;
    busy = true; error = ''; feedback = '';
    try {
      await SaveCurrencyRate({ date, base, quote, value });
      void refreshOverview();
      // Loading the header is separate from saving: a read failure must not imply a failed write.
      feedback = 'Курс сохранён';
      quoteRevision++;
      await refreshToday();
      busy = false;
      closeEditor();
    } catch (reason) { error = String(reason); }
    finally { busy = false; }
  }

  onMount(() => {
    alive = true;
    void refreshToday();
    void refreshOverview();
    void ListCurrencies().then(codes => {
      if (!alive) return;
      currencies = codes;
      if (opened) void loadRate();
    }).catch(reason => { if (alive) mainError = `Не удалось загрузить валюты: ${String(reason)}`; });
    const checkDay = () => { if (localDate() !== today) { void refreshToday(); void refreshOverview(); } };
    const focus = () => { void refreshToday(); void refreshOverview(); };
    const visibility = () => { if (!document.hidden) focus(); };
    const timer = window.setInterval(checkDay, 1000);
    window.addEventListener('focus', focus);
    document.addEventListener('visibilitychange', visibility);
    return () => {
      alive = false; ++mainRequest; ++editRequest; ++overviewRequest;
      window.clearInterval(timer);
      window.removeEventListener('focus', focus);
      document.removeEventListener('visibilitychange', visibility);
    };
  });
</script>

<header>
  <time datetime={today}>{today.split('-').reverse().join('.')}</time>
  <button class:missing={!todayRate} class="rate" onclick={() => openEditor()} aria-label="Ввести или изменить курс валют">
    {todayRate ? `1 USD = ${formatDecimal(todayRate)} RUB` : '???'}
  </button>
  <BenchmarkDirectory bind:this={benchmarkDirectory} />
  <div class="menu-slot"><AssetManager bind:this={assetManager} onadded={asset => { void cashManager.assetAdded(asset); quoteRevision++; void refreshOverview(); }} onchanged={() => { void cashManager.refreshAssets(); quoteRevision++; void refreshOverview(); }} /><PortfolioManager bind:this={portfolioManager} onchanged={() => { portfolioRevision++; void refreshOverview(); }} onassets={trigger => void assetManager.openList(trigger)} onbenchmarks={trigger => void benchmarkDirectory.openList(trigger)} /></div>
</header>
<main>
  <section class="hero" aria-label="Сводка инвестиций">
    <div class="brand"><span class="eyebrow">Личный учёт инвестиций</span><h1>myInvest</h1><p>Ваши инвестиции. Ваши данные.</p><p class="hint">Оценка по котировкам и сделкам на {overview?.date ?? today}. Совокупные итоги в RUB.</p></div>
    <div class="total"><h2>Структура всех портфелей</h2>
      {#if overview}<strong class="total-value">{overview.total.complete ? `${formatChartAmount(overview.total.value)} RUB` : 'Оценка неполная'}</strong><Allocation summary={overview.total} currency="RUB" />{/if}
    </div>
  </section>
  {#if mainError}<p class="error" role="alert">{mainError}</p>{/if}
  {#if feedback}<p role="status">{feedback}</p>{/if}
  {#if overviewLoading}<p role="status">Загрузка обзора…</p>{/if}
  {#if overviewError}<p class="error" role="alert">{overviewError} <button onclick={() => void refreshOverview()}>Повторить</button></p>{/if}
  <div class="tabs" role="tablist" aria-label="Основные разделы">
    <button id="tab-overview" role="tab" aria-selected={tab === 'overview'} aria-controls="panel-overview" tabindex={tab === 'overview' ? 0 : -1} onclick={() => tab = 'overview'} onkeydown={tabKeys}>Обзор</button>
    <button id="tab-operations" role="tab" aria-selected={tab === 'operations'} aria-controls="panel-operations" tabindex={tab === 'operations' ? 0 : -1} onclick={() => tab = 'operations'} onkeydown={tabKeys}>Операции</button>
    <button id="tab-analysis" role="tab" aria-selected={tab === 'analysis'} aria-controls="panel-analysis" tabindex={tab === 'analysis' ? 0 : -1} onclick={() => tab = 'analysis'} onkeydown={tabKeys}>Анализ</button>
    <button id="tab-quotes" role="tab" aria-selected={tab === 'quotes'} aria-controls="panel-quotes" tabindex={tab === 'quotes' ? 0 : -1} onclick={() => tab = 'quotes'} onkeydown={tabKeys}>Котировки</button>
    <button id="tab-benchmarks" role="tab" aria-selected={tab === 'benchmarks'} aria-controls="panel-benchmarks" tabindex={tab === 'benchmarks' ? 0 : -1} onclick={() => tab = 'benchmarks'} onkeydown={tabKeys}>Бенчмарки</button>
  </div>
  <div class="panel" id="panel-overview" role="tabpanel" aria-labelledby="tab-overview" hidden={tab !== 'overview'} tabindex="0">
    {#if overview}
      {#if !overview.portfolios.length}<p>Портфелей пока нет. Добавьте портфель через меню.</p>{/if}
      <article class="history-card"><h2>Совокупная стоимость всех портфелей</h2><p class="hint">На конец каждого дня · RUB · оценка по котировкам и сделкам</p>
        {#if overview.points.length}<ValueHistory points={overview.points} />{:else}<p>Нет истории операций до сегодняшней даты.</p>{/if}
      </article>
      <div class="portfolios">{#each overview.portfolios as p (p.id)}<article class="portfolio-card"><h2>{p.name}</h2><strong class="total-value">{p.summary.complete ? `${formatChartAmount(p.summary.value)} ${p.currency}` : 'Оценка неполная'}</strong><Allocation summary={p.summary} currency={p.currency} />
        {#if p.currency !== 'RUB'}<p class="hint">{p.rate ? `Для совокупной оценки: 1 ${p.currency} = ${formatDecimal(p.rate)} RUB · курс от ${p.rateDate}` : `Курс ${p.currency} → RUB отсутствует`}</p>{/if}
      </article>{/each}</div>
    {/if}
  </div>
  <div class="panel" id="panel-operations" role="tabpanel" aria-labelledby="tab-operations" hidden={tab !== 'operations'} tabindex="0">
    <p class="hint">Баланс журнала учитывает всю введённую историю, включая будущие операции.</p>
    <CashFlowManager bind:this={cashManager} revision={portfolioRevision} onchanged={() => void refreshOverview()} oncreate={() => portfolioManager.openList()} onaddasset={trigger => void assetManager.openList(trigger)} />
  </div>
  <div class="panel" id="panel-analysis" role="tabpanel" aria-labelledby="tab-analysis" hidden={tab !== 'analysis'} tabindex="0">
    <PortfolioAnalysis revision={analysisRevision} onquotes={() => { tab = 'quotes'; document.getElementById('tab-quotes')?.focus(); }} />
  </div>
  <div class="panel" id="panel-quotes" role="tabpanel" aria-labelledby="tab-quotes" hidden={tab !== 'quotes'} tabindex="0">
    <QuoteManager revision={quoteRevision} onchanged={() => void refreshOverview()} onrate={openEditor} onaddasset={trigger => void assetManager.openList(trigger)} />
  </div>
  <div class="panel" id="panel-benchmarks" role="tabpanel" aria-labelledby="tab-benchmarks" hidden={tab !== 'benchmarks'} tabindex="0">
    <BenchmarkManager />
  </div>
</main>

<dialog bind:this={dialog} oncancel={(event) => { event.preventDefault(); closeEditor(); }}>
  <form onsubmit={save}>
    <h2>Ручной курс валют</h2>
    <p class="hint">1 единица первой валюты = введённый курс во второй валюте</p>
    <p class="hint">Обратная пара обновится автоматически. Обратный курс округляется до 18 знаков после запятой.</p>
    <fieldset disabled={busy}>
      <label>Дата <input type="date" bind:value={date} onchange={(event) => { date = event.currentTarget.value; void loadRate(); }} required /></label>
      <div class="pair">
        <label>Первая валюта <input list="currency-codes" bind:value={base} oninput={(event) => { base = event.currentTarget.value; void loadRate(); }} placeholder="USD" autocomplete="off" required /></label>
        <label>Вторая валюта <input list="currency-codes" bind:value={quote} oninput={(event) => { quote = event.currentTarget.value; void loadRate(); }} placeholder="RUB" autocomplete="off" required /></label>
      </div>
      <datalist id="currency-codes">
        {#each currencies as code}<option value={code}></option>{/each}
      </datalist>
      {#each unknown as code}
        <div class="add"><span>{code} отсутствует в справочнике</span><button type="button" onclick={() => void addCurrency(code)}>Добавить {code}</button></div>
      {/each}
      <label>Курс <input type="text" inputmode="decimal" bind:value={value} disabled={loading} placeholder={loading ? 'Загрузка…' : 'Например, 95,125'} required /></label>
    </fieldset>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <div class="actions">
      <button type="button" onclick={closeEditor} disabled={busy}>Отмена</button>
      <button class="primary" type="submit" disabled={busy || loading || unknown.length > 0}>{busy ? 'Сохранение…' : 'Сохранить'}</button>
    </div>
  </form>
</dialog>

<style>
  :global(body) { margin: 0; background: #f5f7fa; color: #172a3a; font-family: system-ui, sans-serif; }
  :global(*) { box-sizing: border-box; }
  header { display: flex; align-items: center; flex-wrap: wrap; gap: 18px; padding: 24px 32px; background: white; border-bottom: 1px solid #dce3e9; }
  button, input { font: inherit; }
  button { cursor: pointer; padding: 10px 16px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; color: inherit; }
  button:disabled { cursor: default; opacity: .6; }
  .menu-slot { margin-left: auto; }
  .rate { font-weight: 600; overflow-wrap: anywhere; text-align: left; max-width: 100%; }
  .missing, .error { color: #a12e2e; }
  main { max-width: 1200px; margin: 0 auto; padding: 32px; }
  .hero { display: grid; grid-template-columns: 1fr 1fr; gap: 40px; align-items: center; margin-bottom: 28px; }
  .brand > p { color: #506477; font-size: 20px; } .brand > .hint { font-size: 14px; }
  .total, .history-card, .portfolio-card { background: white; padding: 24px; border-radius: 14px; border: 1px solid #dce3e9; min-width: 0; }
  .total h2, .portfolio-card h2, .history-card h2 { font-size: 18px; overflow-wrap: anywhere; }
  .total-value { display: block; font-size: 24px; overflow-wrap: anywhere; }
  .tabs { display: flex; flex-wrap: wrap; gap: 8px; border-bottom: 1px solid #dce3e9; padding-bottom: 12px; }
  .tabs button[aria-selected="true"] { background: #207653; color: white; border-color: #207653; }
  .panel { padding-top: 24px; } .panel[hidden] { display: none; }
  .portfolios { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(100%,300px),1fr)); gap: 20px; margin-top: 20px; }
  @media (max-width: 760px) { .hero { grid-template-columns: 1fr; gap: 20px; } main { padding: 24px 16px; } }
  .eyebrow, .hint { color: #506477; font-size: 14px; }
  h1 { margin: 16px 0; font-size: 56px; font-weight: 650; letter-spacing: -.04em; }
  main > p { color: #506477; font-size: 20px; }
  main > .error { color: #a12e2e; }
  dialog { width: min(520px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 28px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog::backdrop { background: #172a3a66; }
  h2 { margin: 0 0 12px; }
  fieldset { padding: 0; margin: 24px 0; border: 0; min-width: 0; }
  label { display: flex; flex-direction: column; gap: 8px; margin-bottom: 18px; font-size: 14px; }
  input { width: 100%; min-width: 0; border: 1px solid #c4cfd8; border-radius: 8px; padding: 12px; background: white; color: inherit; }
  .pair { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
  .add { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; font-size: 14px; margin-bottom: 18px; }
  .actions { display: flex; justify-content: flex-end; gap: 12px; }
  .primary { background: #207653; color: white; border-color: #207653; }
  .error { overflow-wrap: anywhere; font-size: 14px; }
</style>
