<script lang="ts">
  import { onMount } from 'svelte';
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
      todayRate = result.found ? result.value : '';
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

  function openEditor() {
    date = localDate(); base = 'USD'; quote = 'RUB'; value = ''; error = ''; feedback = '';
    opened = true;
    dialog.showModal();
    void loadRate();
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
    } catch (reason) { error = String(reason); }
    finally { busy = false; }
  }
  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy || loading) return;
    busy = true; error = ''; feedback = '';
    try {
      await SaveCurrencyRate({ date, base, quote, value });
      // Loading the header is separate from saving: a read failure must not imply a failed write.
      feedback = 'Курс сохранён';
      await refreshToday();
      busy = false;
      closeEditor();
    } catch (reason) { error = String(reason); }
    finally { busy = false; }
  }

  onMount(() => {
    alive = true;
    void refreshToday();
    void ListCurrencies().then(codes => {
      if (!alive) return;
      currencies = codes;
      if (opened) void loadRate();
    }).catch(reason => { if (alive) mainError = `Не удалось загрузить валюты: ${String(reason)}`; });
    const checkDay = () => { if (localDate() !== today) void refreshToday(); };
    const focus = () => { void refreshToday(); };
    const visibility = () => { if (!document.hidden) focus(); };
    const timer = window.setInterval(checkDay, 1000);
    window.addEventListener('focus', focus);
    document.addEventListener('visibilitychange', visibility);
    return () => {
      alive = false; ++mainRequest; ++editRequest;
      window.clearInterval(timer);
      window.removeEventListener('focus', focus);
      document.removeEventListener('visibilitychange', visibility);
    };
  });
</script>

<header>
  <time datetime={today}>{today.split('-').reverse().join('.')}</time>
  <button class:missing={!todayRate} class="rate" onclick={openEditor} aria-label="Ввести или изменить курс валют">
    {todayRate ? `1 USD = ${todayRate.replace('.', ',')} RUB` : '???'}
  </button>
</header>
<main>
  <span class="eyebrow">Личный учёт инвестиций</span>
  <h1>myInvest</h1>
  <p>Ваши инвестиции. Ваши данные.</p>
  {#if mainError}<p class="error" role="alert">{mainError}</p>{/if}
  {#if feedback}<p role="status">{feedback}</p>{/if}
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
        {#each currencies as code}<option value={code}>{code}</option><option value={code.toLowerCase()}>{code}</option>{/each}
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
  .rate { font-weight: 600; overflow-wrap: anywhere; text-align: left; max-width: 100%; }
  .missing, .error { color: #a12e2e; }
  main { max-width: 760px; margin: 0 auto; padding: 80px 32px; }
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
