<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { ListPortfolios } from '../wailsjs/go/desktop/PortfolioAPI';
  import { ListCurrencies } from '../wailsjs/go/desktop/API';
  import { ListCashFlows, SaveCashFlow, DeleteCashFlow, PreviewCashRate, SaveTrade, PreviewTrade, ListAssets, SaveSplit, PreviewSplit } from '../wailsjs/go/desktop/CashFlowAPI';
  import ActionIcon from './ActionIcon.svelte';
  import { formatDecimal } from './decimal-format.mjs';
  import { localDate } from './calendar';
  let { revision = 0, oncreate, onaddasset, onchanged = () => {} }: { revision?: number; onchanged?: () => void; oncreate: () => void; onaddasset: (trigger: HTMLButtonElement) => void } = $props();
  type Portfolio = Awaited<ReturnType<typeof ListPortfolios>>[number];
  type Snapshot = Awaited<ReturnType<typeof ListCashFlows>>;
  type Operation = Snapshot['operations'][number];
  let portfolios = $state<Portfolio[]>([]);
  let selected = $state('');
  let snapshot = $state<Snapshot | null>(null);
  let loading = $state(true);
  let error = $state('');
  let listError = $state('');
  let listLoading = $state(true);
  let currencies = $state<string[]>([]);
  let currencyLoading = $state(false);
  let currencyError = $state('');
  let root = $state<HTMLElement>();
  let form: HTMLDialogElement;
  let confirmation: HTMLDialogElement;
  let dateInput: HTMLInputElement;
  let portfolioSelect = $state<HTMLSelectElement>();
  let returnTo: HTMLButtonElement | undefined;
  let editing = $state(false);
  let deleting = $state<Operation | null>(null);
  let id = $state('');
  let kind = $state('deposit');
  let date = $state('');
  let currencyCode = $state('');
  let amount = $state('');
  let portfolioAmount = $state('');
  let comment = $state('');
  let assetID = $state(''), quantity = $state(''), expenses = $state('0');
  let assets = $state<Awaited<ReturnType<typeof ListAssets>>>([]);
  let assetLoading = $state(false), assetError = $state('');
  let price = $state(''), priceRounded = $state(false), moneyChange = $state(''), previewError = $state('');
  let assetRequest = 0, previewRequest = 0;
  const isTrade = (k: string) => k === 'buy' || k === 'sell';
  let trade = $derived(isTrade(kind));
  let split = $derived(kind === 'split');
  let oldUnits = $state('1'), newUnits = $state('10');
  let quantityBefore = $state(''), quantityAfter = $state(''), splitPreviewError = $state('');
  let splitRequest = 0;
  export async function assetAdded(asset: Awaited<ReturnType<typeof ListAssets>>[number]) {
    if (!editing || (!isTrade(kind) && kind !== 'split')) return;
    assetID = asset.id; await loadAssets();
  }
  export async function refreshAssets() {
    if (editing && isTrade(kind)) await loadAssets();
  }
  async function loadAssets() {
    const request = ++assetRequest; assetLoading = true; assetError = '';
    try { const rows = await ListAssets(); if (alive && editing && request === assetRequest) assets = rows; }
    catch { if (alive && editing && request === assetRequest) assetError = 'Не удалось загрузить активы'; }
    finally { if (alive && request === assetRequest) assetLoading = false; }
  }
  let errors = $state<Record<string, string>>({});
  let message = $state('');
  let busy = $state(false);
  let rate = $state('');
  let rateError = $state('');
  let readRequest = 0, listRequest = 0, rateRequest = 0, currencyRequest = 0;
  let alive = true;
  let portfolio = $derived(portfolios.find(p => p.id === selected));
  let different = $derived(currencyCode !== portfolio?.currencyCode);
  const display = formatDecimal;
  const kindLabel = (k: string) => ({ deposit: 'Пополнение', withdrawal: 'Вывод', buy: 'Покупка', sell: 'Продажа', split: 'Сплит' }[k] ?? k);
  const operationLabel = (op: Operation) => op.kind === 'split' ? (op.reverseSplit ? 'Обратный сплит' : 'Сплит') : kindLabel(op.kind);
  const signed = (value: string) => value.startsWith('-') || value === '0' ? display(value) : `+${display(value)}`;
  async function loadPortfolios() {
    const request = ++listRequest;
    listLoading = true; listError = '';
    try {
      const rows = await ListPortfolios();
      if (!alive || request !== listRequest) return;
      portfolios = rows;
      if (!rows.some(p => p.id === selected)) selected = rows[0]?.id ?? '';
      await load();
    } catch { if (alive && request === listRequest) listError = 'Не удалось загрузить портфели'; }
    finally { if (alive && request === listRequest) listLoading = false; }
  }
  async function load() {
    const request = ++readRequest, pid = selected;
    snapshot = null; error = ''; loading = true;
    if (!pid) { loading = false; return; }
    try {
      const data = await ListCashFlows(pid);
      if (alive && request === readRequest && selected === pid) snapshot = data;
    } catch { if (alive && request === readRequest && selected === pid) error = 'Не удалось загрузить операции'; }
    finally { if (alive && request === readRequest) loading = false; }
  }
  $effect(() => { void revision; void loadPortfolios(); });
  async function loadCurrencies() {
    const request = ++currencyRequest;
    currencyLoading = true; currencyError = '';
    try {
      const codes = await ListCurrencies();
      if (alive && editing && request === currencyRequest) currencies = codes;
    } catch { if (alive && editing && request === currencyRequest) currencyError = 'Не удалось загрузить валюты'; }
    finally { if (alive && request === currencyRequest) currencyLoading = false; }
  }
  async function open(op: Operation | null, k: string, trigger: HTMLButtonElement) {
    if (!portfolio || !snapshot) return;
    returnTo = trigger; id = op?.id ?? ''; kind = op?.kind ?? k;
    date = op?.date ?? localDate(); currencyCode = op?.currencyCode ?? portfolio.currencyCode;
    amount = op?.amount ?? ''; portfolioAmount = op?.portfolioAmount ?? ''; comment = op?.comment ?? '';
    assetID = op?.assetID ?? ''; quantity = op?.quantity ?? ''; expenses = op?.expenses ?? '0';
    oldUnits = op?.oldUnits || '1'; newUnits = op?.newUnits || '10';
    quantityBefore = ''; quantityAfter = ''; splitPreviewError = '';
    price = ''; moneyChange = ''; previewError = '';
    errors = {}; message = ''; rate = ''; editing = true;
    await tick(); form.showModal(); dateInput.focus(); if (isTrade(kind) || kind === 'split') void loadAssets(); else void loadCurrencies();
  }
  function close() {
    if (busy) return;
    editing = false; ++currencyRequest; ++rateRequest; ++assetRequest; ++previewRequest; ++splitRequest;
    form.close(); returnTo?.focus();
  }
  function currencyChanged() { amount = ''; portfolioAmount = ''; errors = {}; }
  $effect(() => {
    const a = amount, b = portfolioAmount, active = editing && different && !isTrade(kind) && kind !== 'split';
    const request = ++rateRequest;
    rate = ''; rateError = '';
    if (!active || !a || !b) return;
    const timer = setTimeout(async () => {
      try {
        const result = await PreviewCashRate(a, b);
        if (alive && editing && request === rateRequest) rate = result.rate;
      } catch { if (alive && editing && request === rateRequest) rateError = 'Не удалось рассчитать справочный курс'; }
    }, 200);
    return () => clearTimeout(timer);
  });
  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy || ((isTrade(kind) || kind === 'split') ? assetLoading || assetError : currencyLoading || currencyError)) return;
    const pid = selected, savedID = id;
    busy = true; message = ''; errors = {};
    let result: Awaited<ReturnType<typeof SaveCashFlow>>;
    try {
      result = kind === 'split'
        ? await SaveSplit(pid, id, { date, assetID, oldUnits, newUnits, comment })
        : isTrade(kind)
        ? await SaveTrade(pid, id, { kind, date, assetID, quantity, amount, expenses, comment })
        : await SaveCashFlow(pid, id, { kind, date, currencyCode, amount, portfolioAmount: different ? portfolioAmount : amount, comment });
    } catch {
      if (alive && pid === selected) message = 'Не удалось сохранить операцию. Повторите попытку';
      return;
    } finally { busy = false; }
    if (!alive || pid !== selected) return;
    errors = result.errors; message = result.message;
    if (!message && !Object.keys(errors).length) {
      onchanged(); close(); await load();
      await tick(); if (alive && !editing && !deleting) (returnTo?.isConnected ? returnTo : root?.querySelector<HTMLButtonElement>(`[data-cash-id="${savedID}"]`) ?? portfolioSelect)?.focus();
    } else {
      await tick(); form.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
    }
  }

  function askDelete(op: Operation, trigger: HTMLButtonElement) {
    deleting = op; returnTo = trigger; message = ''; confirmation.showModal();
  }
  function closeDelete() { if (busy) return; deleting = null; confirmation.close(); returnTo?.focus(); }
  async function remove() {
    if (busy || !deleting) return;
    const pid = selected, operationID = deleting.id;
    busy = true; message = '';
    let result: Awaited<ReturnType<typeof DeleteCashFlow>>;
    try { result = await DeleteCashFlow(pid, operationID); }
    catch {
      if (alive && pid === selected) message = 'Не удалось удалить операцию. Повторите попытку';
      return;
    } finally { busy = false; }
    if (!alive || pid !== selected) return;
    message = result.message;
    if (!message) {
      onchanged(); closeDelete(); await load(); await tick();
      if (alive && !editing && !deleting) portfolioSelect?.focus();
    }
  }

  $effect(() => {
    const q = quantity, a = amount, e = expenses, k = kind, active = editing && isTrade(kind);
    const request = ++previewRequest;
    price = ''; moneyChange = ''; previewError = '';
    if (!active || !q || !a) return;
    const timer = setTimeout(async () => {
      try {
        const result = await PreviewTrade(k, q, a, e);
        if (alive && editing && request === previewRequest) {
          price = result.averagePrice; priceRounded = result.rounded; moneyChange = result.moneyChange;
          previewError = Object.values(result.errors).join('. ');
        }
      } catch { if (alive && editing && request === previewRequest) previewError = 'Не удалось рассчитать среднюю цену'; }
    }, 200);
    return () => clearTimeout(timer);
  });
  $effect(() => {
    const pid = selected, operationID = id, d = date, asset = assetID, old = oldUnits, next = newUnits;
    const active = editing && kind === 'split';
    const request = ++splitRequest;
    quantityBefore = ''; quantityAfter = ''; splitPreviewError = '';
    if (!active || !d || !asset || !old || !next) return;
    const timer = setTimeout(async () => {
      try {
        const result = await PreviewSplit(pid, operationID, { date: d, assetID: asset, oldUnits: old, newUnits: next, comment: '' });
        if (alive && editing && selected === pid && request === splitRequest) {
          quantityBefore = result.quantityBefore; quantityAfter = result.quantityAfter;
          splitPreviewError = result.message || Object.values(result.errors).join('. ');
        }
      } catch { if (alive && editing && request === splitRequest) splitPreviewError = 'Не удалось рассчитать количество после сплита'; }
    }, 200);
    return () => clearTimeout(timer);
  });
  onDestroy(() => { ++assetRequest; ++previewRequest; ++splitRequest; alive = false; ++readRequest; ++listRequest; ++rateRequest; ++currencyRequest; });
</script>

<section bind:this={root} aria-labelledby="cash-title">
  <h2 id="cash-title">Денежные средства</h2>
  {#if listLoading}<p role="status">Загрузка портфелей…</p>
  {:else if listError}<p class="error" role="alert">{listError}</p><button onclick={() => void loadPortfolios()}>Повторить загрузку</button>
  {:else if !portfolios.length}<p>Создайте портфель, чтобы записывать операции.</p><button onclick={oncreate}>Создать портфель</button>
  {:else}
    <label for="cash-portfolio">Портфель</label>
    <select id="cash-portfolio" bind:this={portfolioSelect} bind:value={selected} onchange={() => void load()} disabled={editing || !!deleting || busy}>
      {#each portfolios as p (p.id)}<option value={p.id}>{p.name} ({p.currencyCode})</option>{/each}
    </select>
    {#if loading}<p role="status">Загрузка операций…</p>
    {:else if error}<p class="error" role="alert">{error}</p><button onclick={() => void load()}>Повторить загрузку</button>
    {:else if snapshot}
      <p class="balance">{display(snapshot.balance)} {snapshot.currencyCode}</p>
      {#if snapshot.negative}<p class="warning" role="status">Остаток отрицательный. Проверьте полноту истории денежных операций.</p>{/if}
      <div class="actions"><button class="primary" onclick={event => void open(null, 'deposit', event.currentTarget)}>Пополнить</button><button onclick={event => void open(null, 'withdrawal', event.currentTarget)}>Вывести</button><button onclick={event => void open(null, 'buy', event.currentTarget)}>Купить</button><button onclick={event => void open(null, 'sell', event.currentTarget)}>Продать</button><button onclick={event => void open(null, 'split', event.currentTarget)}>Сплит</button></div>
      {#each snapshot.warnings ?? [] as warning (warning.operationID)}<p class="warning" role="status">{warning.date}: продажа {warning.ticker} сверх остатка — доступно {display(warning.available)}, продано {display(warning.quantity)}. Проверьте полноту истории.</p>{/each}
      {#each snapshot.cashWarnings ?? [] as warning (warning.date)}<p class="warning" role="status">{warning.date}: отрицательный денежный остаток на конец дня — {display(warning.balance)} {snapshot.currencyCode}. Проверьте полноту истории.</p>{/each}
      <h3>Журнал операций</h3>
      {#if !snapshot.operations.length}<p>Операций пока нет</p>
      {:else}<div class="table-scroll"><table>
        <thead><tr><th>Дата</th><th>Операция</th><th>Тикер</th><th>Детали</th><th>Изменение денег</th><th>Действия</th></tr></thead>
        <tbody>{#each snapshot.operations as op (op.id)}<tr class:cash-row={!isTrade(op.kind) && op.kind !== 'split'}>
          <td><time datetime={op.date}>{op.date.split('-').reverse().join('.')}</time></td><td>{!isTrade(op.kind) && op.kind !== 'split' ? '💵 ' : ''}{operationLabel(op)}</td>
          <td>{op.ticker || '—'}</td>
          <td>{#if op.kind === 'split'}
            <div>{display(op.oldUnits)} → {display(op.newUnits)}; количество {display(op.quantityBefore)} → {display(op.quantityAfter)}</div>
          {:else if isTrade(op.kind)}
            <div>Количество: {display(op.quantity)}</div><div>Сумма: {display(op.amount)} {snapshot.currencyCode}</div>
            <div>Средняя цена: {op.rounded ? '≈ ' : ''}{display(op.averagePrice)} {snapshot.currencyCode}{op.rounded ? ' (до 18 знаков)' : ''}</div>
            <div>Расходы: {display(op.expenses)} {snapshot.currencyCode}</div>
          {:else}<div>{display(op.amount)} {op.currencyCode} / {display(op.portfolioAmount)} {snapshot.currencyCode}</div>
            <div>Курс ≈ {display(op.rate)} {snapshot.currencyCode} за 1 {op.currencyCode}</div>{/if}
            <div class="comment">{op.comment}</div>
          </td>
          <td>{signed(op.moneyChange)} {snapshot.currencyCode}</td>
          <td><div class="row-actions"><button class="icon-button" title="Редактировать" data-cash-id={op.id} aria-label={`Редактировать ${operationLabel(op)} ${op.date} ${op.amount} ${op.currencyCode}`} onclick={event => void open(op, op.kind, event.currentTarget)}><ActionIcon kind="edit" /></button><button class="icon-button delete-button" title="Удалить" aria-label={`Удалить ${operationLabel(op)} ${op.date} ${op.amount} ${op.currencyCode}`} onclick={event => askDelete(op, event.currentTarget)}><ActionIcon kind="delete" /></button></div></td>
        </tr>{/each}</tbody>
      </table></div>{/if}
      <p class="hint">Справочный курс округлён до 4 знаков. Остаток рассчитан по точным суммам.</p>
    {/if}
  {/if}
</section>

<dialog bind:this={form} aria-labelledby="cash-form-title" oncancel={event => { event.preventDefault(); close(); }}>
  <form onsubmit={save} novalidate>
    <h2 id="cash-form-title">{id ? 'Редактировать операцию' : kindLabel(kind)}</h2>
    <p>{portfolio?.name} · {portfolio?.currencyCode}</p>
    <fieldset disabled={busy}>
      <label for="cash-kind">Тип</label><select id="cash-kind" bind:value={kind} aria-invalid={!!errors.kind}>{#if split}<option value="split">Сплит</option>{:else if trade}<option value="buy">Покупка</option><option value="sell">Продажа</option>{:else}<option value="deposit">Пополнение</option><option value="withdrawal">Вывод</option>{/if}</select>
      {#if errors.kind}<p class="error" role="alert">{errors.kind}</p>{/if}
      <label for="cash-date">Дата</label><input id="cash-date" bind:this={dateInput} type="date" bind:value={date} aria-invalid={!!errors.date} aria-describedby={errors.date ? 'cash-date-error' : undefined} />
      {#if errors.date}<p id="cash-date-error" class="error" role="alert">{errors.date}</p>{/if}
      {#if trade || split}
        <label for="trade-asset">Актив</label><select id="trade-asset" bind:value={assetID} disabled={assetLoading || !!assetError} aria-invalid={!!errors.assetID} aria-describedby={errors.assetID ? 'trade-asset-error' : undefined}>
          <option value="">Выберите актив</option>{#each assets as asset (asset.id)}<option value={asset.id}>{asset.ticker}</option>{/each}
        </select>
        {#if assetLoading}<p role="status">Загрузка активов…</p>{/if}
        {#if assetError}<p class="error" role="alert">{assetError}</p><button type="button" onclick={() => void loadAssets()}>Повторить загрузку активов</button>{/if}
        {#if errors.assetID}<p id="trade-asset-error" class="error" role="alert">{errors.assetID}</p>{/if}
        <button type="button" onclick={event => onaddasset(event.currentTarget)}>Добавить актив</button>
        {#if split}
          <p class="hint">Сплит применяется в начале дня к остатку предыдущих дат. Сделки этой даты вводятся в новых единицах.</p>
          <label for="split-old">Старых акций</label><input id="split-old" inputmode="numeric" bind:value={oldUnits} aria-invalid={!!errors.oldUnits} aria-describedby={errors.oldUnits ? 'split-old-error' : undefined} />
          {#if errors.oldUnits}<p id="split-old-error" class="error" role="alert">{errors.oldUnits}</p>{/if}
          <label for="split-new">Новых акций</label><input id="split-new" inputmode="numeric" bind:value={newUnits} aria-invalid={!!errors.newUnits} aria-describedby={errors.newUnits ? 'split-new-error' : undefined} />
          {#if errors.newUnits}<p id="split-new-error" class="error" role="alert">{errors.newUnits}</p>{/if}
          {#if quantityBefore !== ''}<p role="status">Количество: {display(quantityBefore)} → {display(quantityAfter)}. Изменение денег: 0 {portfolio?.currencyCode}</p>{/if}
          {#if splitPreviewError}<p class="error" role="alert">{splitPreviewError}</p>{/if}
        {:else}
        <label for="trade-quantity">Количество</label><input id="trade-quantity" inputmode="decimal" bind:value={quantity} aria-invalid={!!errors.quantity} aria-describedby={errors.quantity ? 'trade-quantity-error' : undefined} />
        {#if errors.quantity}<p id="trade-quantity-error" class="error" role="alert">{errors.quantity}</p>{/if}
        <label for="trade-amount">Сумма сделки ({portfolio?.currencyCode})</label><input id="trade-amount" inputmode="decimal" bind:value={amount} aria-invalid={!!errors.amount} aria-describedby={errors.amount ? 'trade-amount-error' : undefined} />
        {#if errors.amount}<p id="trade-amount-error" class="error" role="alert">{errors.amount}</p>{/if}
        <label for="trade-expenses">Дополнительные расходы ({portfolio?.currencyCode})</label><input id="trade-expenses" inputmode="decimal" bind:value={expenses} aria-invalid={!!errors.expenses} aria-describedby={errors.expenses ? 'trade-expenses-error' : undefined} />
        {#if errors.expenses}<p id="trade-expenses-error" class="error" role="alert">{errors.expenses}</p>{/if}
        {#if price}<p role="status">Средняя цена: {priceRounded ? '≈ ' : ''}{display(price)} {portfolio?.currencyCode}{priceRounded ? ' (округлено до 18 знаков)' : ''}. Изменение денег: {signed(moneyChange)} {portfolio?.currencyCode}</p>{/if}
        {#if previewError}<p class="error" role="alert">{previewError}</p>{/if}
        {/if}
      {:else}
      <label for="cash-currency">Валюта операции</label><select id="cash-currency" bind:value={currencyCode} onchange={currencyChanged} disabled={currencyLoading || !!currencyError} aria-invalid={!!errors.currencyCode} aria-describedby={errors.currencyCode ? 'cash-currency-error' : undefined}>
        {#each currencies as code}<option value={code}>{code}</option>{/each}
      </select>
      {#if currencyLoading}<p role="status">Загрузка валют…</p>{/if}
      {#if currencyError}<p class="error" role="alert">{currencyError}</p><button type="button" onclick={() => void loadCurrencies()}>Повторить загрузку валют</button>{/if}
      {#if errors.currencyCode}<p id="cash-currency-error" class="error" role="alert">{errors.currencyCode}</p>{/if}
      <label for="cash-amount">{different ? 'Сумма в валюте операции' : 'Сумма'} ({currencyCode})</label><input id="cash-amount" inputmode="decimal" bind:value={amount} aria-invalid={!!errors.amount} aria-describedby={errors.amount ? 'cash-amount-error' : undefined} />
      {#if errors.amount}<p id="cash-amount-error" class="error" role="alert">{errors.amount}</p>{/if}
      {#if different}
        <label for="cash-portfolio-amount">Сумма в валюте портфеля ({portfolio?.currencyCode})</label><input id="cash-portfolio-amount" inputmode="decimal" bind:value={portfolioAmount} aria-invalid={!!errors.portfolioAmount} aria-describedby={errors.portfolioAmount ? 'cash-portfolio-amount-error' : undefined} />
      {/if}
      {#if errors.portfolioAmount}<p id="cash-portfolio-amount-error" class="error" role="alert">{errors.portfolioAmount}</p>{/if}
      {#if different && rate}<p role="status">Справочный курс ≈ {display(rate)} {portfolio?.currencyCode} за 1 {currencyCode} (до 4 знаков)</p>{/if}
      {#if rateError}<p class="error" role="alert">{rateError}</p>{/if}
      {/if}
      <label for="cash-comment">Комментарий</label><textarea id="cash-comment" bind:value={comment}></textarea>
    </fieldset>
    {#if message}<p class="error" role="alert">{message}</p>{/if}
    <div class="actions"><button type="button" disabled={busy} onclick={close}>Отмена</button><button class="primary" type="submit" disabled={busy || (trade || split ? assetLoading || !!assetError : currencyLoading || !!currencyError)}>{busy ? 'Сохранение…' : 'Сохранить'}</button></div>
  </form>
</dialog>
<dialog bind:this={confirmation} aria-labelledby="cash-delete-title" oncancel={event => { event.preventDefault(); closeDelete(); }}>
  <h2 id="cash-delete-title">Удалить операцию?</h2>
  {#if deleting}<p>{portfolio?.name}: {operationLabel(deleting)}, {deleting.date}, {#if deleting.kind === 'split'}{deleting.ticker}, {display(deleting.oldUnits)} → {display(deleting.newUnits)}; количество {display(deleting.quantityBefore)} → {display(deleting.quantityAfter)}{:else if isTrade(deleting.kind)}{deleting.ticker}, {display(deleting.quantity)} ед., {display(deleting.amount)} {portfolio?.currencyCode}, расходы {display(deleting.expenses)} {portfolio?.currencyCode}{:else}{display(deleting.amount)} {deleting.currencyCode} / {display(deleting.portfolioAmount)} {portfolio?.currencyCode}{/if}</p><p class="comment">{deleting.comment}</p>{/if}
  {#if message}<p class="error" role="alert">{message}</p>{/if}
  <div class="actions"><button disabled={busy} onclick={closeDelete}>Отмена</button><button disabled={busy} onclick={() => void remove()}>{busy ? 'Удаление…' : 'Удалить'}</button></div>
</dialog>
<style>
  section { margin-top: 32px; }
  button, input, select, textarea { font: inherit; color: inherit; }
  button { cursor: pointer; padding: 10px 16px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; }
  button:disabled { cursor: default; opacity: .6; }
  input, select, textarea { width: 100%; border: 1px solid #c4cfd8; border-radius: 8px; padding: 12px; background: white; }
  label { display: block; margin: 16px 0 8px; }
  textarea { min-height: 80px; resize: vertical; }
  dialog { width: min(560px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 28px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog::backdrop { background: #172a3a66; }
  fieldset { border: 0; padding: 0; margin: 20px 0; min-width: 0; }
  .actions { display: flex; gap: 12px; justify-content: flex-end; }
  .primary { background: #207653; color: white; border-color: #207653; }
  .balance { font-size: 28px; overflow-wrap: anywhere; }
  .error { color: #a12e2e; }
  .warning { color: #825800; }
  .hint { color: #506477; font-size: 14px; }
  [aria-invalid="true"] { border-color: #a12e2e; }
  .table-scroll { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; text-align: left; }
  th, td { padding: 10px; border-bottom: 1px solid #dce3e9; vertical-align: top; overflow-wrap: anywhere; }
  th { font-size: 13px; }
  td:first-child, td:nth-child(2) { white-space: nowrap; }
  .cash-row { background: #edf5f0; }
  .row-actions { display: flex; gap: 8px; }
  .icon-button { display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0; flex: 0 0 40px; }
  .icon-button:hover { background: #edf3f7; }
  .delete-button { color: #a12e2e; }
  .delete-button:hover { background: #fff0f0; }
  .icon-button:focus-visible { outline: 2px solid #207653; outline-offset: 2px; }
  .comment { white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
