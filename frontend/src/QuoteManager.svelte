<script lang="ts">
  import { onMount } from 'svelte';
  import { ListQuotes, SaveQuote } from '../wailsjs/go/desktop/QuotesAPI';
  import { ListAssets } from '../wailsjs/go/desktop/CashFlowAPI';
  import { ListCurrencies, ListCurrencyRates } from '../wailsjs/go/desktop/API';
  import { localDate } from './calendar';
  import { formatDecimal } from './decimal-format.mjs';
  let { revision = 0, onrate, onaddasset, onchanged = () => {} }: {
    revision?: number;
    onchanged?: () => void;
    onrate: (rate?: {date: string; base: string; quote: string; value: string}) => void;
    onaddasset: (trigger: HTMLElement) => void;
  } = $props();
  let quotes = $state<Awaited<ReturnType<typeof ListQuotes>>>([]);
  let rates = $state<Awaited<ReturnType<typeof ListCurrencyRates>>>([]);
  let assets = $state<Awaited<ReturnType<typeof ListAssets>>>([]);
  let currencies = $state<string[]>([]);
  let loading = $state(true), loadError = $state(''), busy = $state(false);
  let editing = $state(false), assetID = $state(''), date = $state(localDate());
  let currencyCode = $state('RUB'), price = $state('');
  let errors = $state<Record<string,string>>({}), message = $state(''), feedback = $state('');
  let request = 0, alive = true;
  async function refresh() {
    const id = ++request; loading = true; loadError = '';
    try {
      const result = await Promise.all([ListQuotes(), ListCurrencyRates(), ListAssets(), ListCurrencies()]);
      if (!alive || id !== request) return;
      [quotes, rates, assets, currencies] = result;
    } catch {
      if (alive && id === request) loadError = 'Не удалось обновить котировки и курсы. Ранее показанные данные могут быть устаревшими.';
    } finally { if (alive && id === request) loading = false; }
  }
  $effect(() => { revision; void refresh(); });
  onMount(() => () => { alive = false; ++request; });
  function reset() {
    if (busy) return;
    editing = false; assetID = ''; date = localDate(); currencyCode = 'RUB'; price = '';
    errors = {}; message = ''; feedback = '';
  }
  function edit(q: {assetID: string; date: string; currencyCode: string; price: string}) {
    if (busy) return;
    editing = true; assetID = q.assetID; date = q.date; currencyCode = q.currencyCode; price = q.price;
    errors = {}; message = ''; feedback = '';
    document.getElementById('quote-price')?.focus();
  }
  async function save(event: SubmitEvent) {
    event.preventDefault(); if (busy) return;
    busy = true; errors = {}; message = ''; feedback = '';
    try {
      const result = await SaveQuote({assetID, date, currencyCode, price, ticker: ''});
      if (!alive) return;
      errors = result.errors; message = result.message;
      if (!message && !Object.keys(errors).length) {
        feedback = 'Котировка сохранена';
        onchanged();
        // A failed refresh cannot turn a successful write into a failed save.
        await refresh();
      }
    } catch { if (alive) message = 'Не удалось сохранить котировку. Повторите попытку.'; }
    finally { if (alive) busy = false; }
  }
</script>

<section aria-labelledby="quotes-title">
  <h2 id="quotes-title">Котировки активов</h2>
  <p class="hint">Цена одной единицы актива. Для оценки используется котировка или сделка с наиболее поздней датой.</p>
  <form onsubmit={save}>
    <h3>{editing ? 'Изменить котировку' : 'Новая котировка'}</h3>
    <fieldset disabled={busy}>
      <div class="fields">
        <label>Код актива
          <select bind:value={assetID} disabled={editing} required aria-invalid={!!errors.assetID} aria-describedby="quote-asset-error">
            <option value="">Выберите актив</option>
            {#each assets as a}<option value={a.id}>{a.ticker}</option>{/each}
          </select>
          <span id="quote-asset-error" class="error">{errors.assetID ?? ''}</span>
        </label>
        <label>Дата <input type="date" bind:value={date} disabled={editing} required aria-invalid={!!errors.date} aria-describedby="quote-date-error" />
          <span id="quote-date-error" class="error">{errors.date ?? ''}</span>
        </label>
        <label>Валюта
          <select bind:value={currencyCode} disabled={editing} required aria-invalid={!!errors.currencyCode} aria-describedby="quote-currency-error">
            {#each currencies as c}<option value={c}>{c}</option>{/each}
          </select>
          <span id="quote-currency-error" class="error">{errors.currencyCode ?? ''}</span>
        </label>
        <label>Цена <input id="quote-price" type="text" inputmode="decimal" bind:value={price} placeholder="Например, 125,50" required aria-invalid={!!errors.price} aria-describedby="quote-price-error" />
          <span id="quote-price-error" class="error">{errors.price ?? ''}</span>
        </label>
      </div>
      <div class="actions">
        <button class="primary" type="submit" disabled={loading}>{busy ? 'Сохранение…' : 'Сохранить'}</button>
        <button type="button" onclick={reset}>{editing ? 'Отмена изменения' : 'Очистить'}</button>
        <button type="button" onclick={event => onaddasset(event.currentTarget)}>Добавить актив</button>
      </div>
    </fieldset>
    {#if message}<p class="error" role="alert">{message}</p>{/if}
    {#if feedback}<p role="status">{feedback}</p>{/if}
    <p class="hint">Повторное сохранение актива на ту же дату в той же валюте заменяет цену.</p>
  </form>
  {#if loading}<p role="status">Загрузка котировок и курсов…</p>{/if}
  {#if loadError}<p class="error" role="alert">{loadError} <button disabled={busy} onclick={() => void refresh()}>Повторить</button></p>{/if}
  {#if !loading && !loadError && !quotes.length}<p>Котировок пока нет.</p>{/if}
  {#if quotes.length}
    <div class="table-wrap"><table><caption>Сохранённые котировки</caption><thead><tr><th>Код актива</th><th>Дата</th><th>Валюта</th><th>Цена</th><th>Действия</th></tr></thead>
      <tbody>{#each quotes as q (`${q.assetID}:${q.date}:${q.currencyCode}`)}<tr>
        <td>{q.ticker}</td><td>{q.date.split('-').reverse().join('.')}</td><td>{q.currencyCode}</td><td class="decimal">{formatDecimal(q.price)}</td>
        <td><button disabled={busy} onclick={() => edit(q)} aria-label={`Изменить ${q.ticker} от ${q.date} в ${q.currencyCode}`}>Изменить</button></td>
      </tr>{/each}</tbody></table></div>
  {/if}
</section>
<section aria-labelledby="rates-title">
  <div class="heading"><h2 id="rates-title">Курсы валют</h2><button disabled={busy} onclick={() => onrate()}>Ввести курс</button></div>
  <p class="hint">1 единица первой валюты равна указанному количеству второй. Обратные пары сохраняются автоматически.</p>
  {#if !loading && !loadError && !rates.length}<p>Курсов пока нет.</p>{/if}
  {#if rates.length}<div class="table-wrap"><table><caption>Сохранённые курсы</caption><thead><tr><th>Дата</th><th>Пара</th><th>Курс</th><th>Действия</th></tr></thead>
    <tbody>{#each rates as r (`${r.date}:${r.base}:${r.quote}`)}<tr><td>{r.date.split('-').reverse().join('.')}</td><td>{r.base}/{r.quote}</td><td class="decimal">{formatDecimal(r.value)}</td><td><button disabled={busy} onclick={() => onrate(r)} aria-label={`Изменить ${r.base}/${r.quote} от ${r.date}`}>Изменить</button></td></tr>{/each}</tbody>
  </table></div>{/if}
</section>
<style>
  section, form { background: white; border: 1px solid #dce3e9; border-radius: 12px; padding: 20px; margin-bottom: 20px; min-width: 0; }
  form { background: #f5f7fa; } h2,h3 { margin-top: 0; }
  .hint { font-size: 14px; color: #506477; } .error { color: #a12e2e; overflow-wrap: anywhere; font-size: 14px; }
  fieldset { border: 0; padding: 0; margin: 0; min-width: 0; }
  .fields { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(100%,180px),1fr)); gap: 16px; }
  label { display: flex; flex-direction: column; gap: 8px; min-width: 0; font-size: 14px; }
  button,input,select { font: inherit; border: 1px solid #c4cfd8; border-radius: 8px; padding: 10px; color: inherit; background: white; min-width: 0; }
  button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  .primary { background: #207653; color: white; border-color: #207653; }
  .actions,.heading { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-top: 16px; }
  .heading { justify-content: space-between; margin-top: 0; } .heading h2 { margin-bottom: 0; }
  .table-wrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; }
  caption { text-align: left; font-size: 14px; color: #506477; padding: 12px 0; }
  th,td { text-align: left; padding: 12px 10px; border-bottom: 1px solid #dce3e9; }
  .decimal { overflow-wrap: anywhere; min-width: 140px; max-width: 320px; font-variant-numeric: tabular-nums; }
</style>
