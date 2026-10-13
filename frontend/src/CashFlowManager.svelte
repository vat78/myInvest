<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { ListPortfolios } from '../wailsjs/go/desktop/PortfolioAPI';
  import { ListCurrencies } from '../wailsjs/go/desktop/API';
  import { ListCashFlows, SaveCashFlow, DeleteCashFlow, PreviewCashRate } from '../wailsjs/go/desktop/CashFlowAPI';
  import ActionIcon from './ActionIcon.svelte';
  import { formatDecimal } from './decimal-format.mjs';
  import { localDate } from './calendar';
  let { revision = 0, oncreate }: { revision?: number; oncreate: () => void } = $props();
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
  const kindLabel = (k: string) => k === 'deposit' ? 'Пополнение' : 'Вывод';
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
    } catch { if (alive && request === readRequest && selected === pid) error = 'Не удалось загрузить денежные операции'; }
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
    errors = {}; message = ''; rate = ''; editing = true;
    await tick(); form.showModal(); dateInput.focus(); void loadCurrencies();
  }
  function close() {
    if (busy) return;
    editing = false; ++currencyRequest; ++rateRequest;
    form.close(); returnTo?.focus();
  }
  function currencyChanged() { amount = ''; portfolioAmount = ''; errors = {}; }
  $effect(() => {
    const a = amount, b = portfolioAmount, active = editing && different;
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
    if (busy || currencyLoading || currencyError) return;
    const pid = selected, savedID = id;
    busy = true; message = ''; errors = {};
    let result: Awaited<ReturnType<typeof SaveCashFlow>>;
    try {
      result = await SaveCashFlow(pid, id, { kind, date, currencyCode, amount, portfolioAmount: different ? portfolioAmount : amount, comment });
    } catch {
      if (alive && pid === selected) message = 'Не удалось сохранить операцию. Повторите попытку';
      return;
    } finally { busy = false; }
    if (!alive || pid !== selected) return;
    errors = result.errors; message = result.message;
    if (!message && !Object.keys(errors).length) {
      close(); await load();
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
      closeDelete(); await load(); await tick();
      if (alive && !editing && !deleting) portfolioSelect?.focus();
    }
  }

  onDestroy(() => { alive = false; ++readRequest; ++listRequest; ++rateRequest; ++currencyRequest; });
</script>

<section bind:this={root} aria-labelledby="cash-title">
  <h2 id="cash-title">Денежные средства</h2>
  {#if listLoading}<p role="status">Загрузка портфелей…</p>
  {:else if listError}<p class="error" role="alert">{listError}</p><button onclick={() => void loadPortfolios()}>Повторить загрузку</button>
  {:else if !portfolios.length}<p>Создайте портфель, чтобы записывать денежные операции.</p><button onclick={oncreate}>Создать портфель</button>
  {:else}
    <label for="cash-portfolio">Портфель</label>
    <select id="cash-portfolio" bind:this={portfolioSelect} bind:value={selected} onchange={() => void load()} disabled={editing || !!deleting || busy}>
      {#each portfolios as p (p.id)}<option value={p.id}>{p.name} ({p.currencyCode})</option>{/each}
    </select>
    {#if loading}<p role="status">Загрузка денежных операций…</p>
    {:else if error}<p class="error" role="alert">{error}</p><button onclick={() => void load()}>Повторить загрузку</button>
    {:else if snapshot}
      <p class="balance">{display(snapshot.balance)} {snapshot.currencyCode}</p>
      {#if snapshot.negative}<p class="warning" role="status">Остаток отрицательный. Проверьте полноту истории денежных операций.</p>{/if}
      <div class="actions"><button class="primary" onclick={event => void open(null, 'deposit', event.currentTarget)}>Пополнить</button><button onclick={event => void open(null, 'withdrawal', event.currentTarget)}>Вывести</button></div>
      <h3>Журнал денежных операций</h3>
      {#if !snapshot.operations.length}<p>Денежных операций пока нет</p>
      {:else}<div class="table-scroll"><table>
        <thead><tr><th>Дата</th><th>Тип</th><th>Исходная сумма</th><th>Сумма в валюте портфеля</th><th>Курс ≈</th><th>Комментарий</th><th>Действия</th></tr></thead>
        <tbody>{#each snapshot.operations as op (op.id)}<tr>
          <td><time datetime={op.date}>{op.date.split('-').reverse().join('.')}</time></td><td>{kindLabel(op.kind)}</td>
          <td>{display(op.amount)} {op.currencyCode}</td><td>{display(op.portfolioAmount)} {snapshot.currencyCode}</td>
          <td>≈ {display(op.rate)} {snapshot.currencyCode} за 1 {op.currencyCode}</td><td class="comment">{op.comment}</td>
          <td><div class="row-actions"><button class="icon-button" title="Редактировать" data-cash-id={op.id} aria-label={`Редактировать ${kindLabel(op.kind)} ${op.date} ${op.amount} ${op.currencyCode}`} onclick={event => void open(op, op.kind, event.currentTarget)}><ActionIcon kind="edit" /></button><button class="icon-button delete-button" title="Удалить" aria-label={`Удалить ${kindLabel(op.kind)} ${op.date} ${op.amount} ${op.currencyCode}`} onclick={event => askDelete(op, event.currentTarget)}><ActionIcon kind="delete" /></button></div></td>
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
      <label for="cash-kind">Тип</label><select id="cash-kind" bind:value={kind} aria-invalid={!!errors.kind}><option value="deposit">Пополнение</option><option value="withdrawal">Вывод</option></select>
      {#if errors.kind}<p class="error" role="alert">{errors.kind}</p>{/if}
      <label for="cash-date">Дата</label><input id="cash-date" bind:this={dateInput} type="date" bind:value={date} aria-invalid={!!errors.date} aria-describedby={errors.date ? 'cash-date-error' : undefined} />
      {#if errors.date}<p id="cash-date-error" class="error" role="alert">{errors.date}</p>{/if}
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
      <label for="cash-comment">Комментарий</label><textarea id="cash-comment" bind:value={comment}></textarea>
    </fieldset>
    {#if message}<p class="error" role="alert">{message}</p>{/if}
    <div class="actions"><button type="button" disabled={busy} onclick={close}>Отмена</button><button class="primary" type="submit" disabled={busy || currencyLoading || !!currencyError}>{busy ? 'Сохранение…' : 'Сохранить'}</button></div>
  </form>
</dialog>
<dialog bind:this={confirmation} aria-labelledby="cash-delete-title" oncancel={event => { event.preventDefault(); closeDelete(); }}>
  <h2 id="cash-delete-title">Удалить операцию?</h2>
  {#if deleting}<p>{portfolio?.name}: {kindLabel(deleting.kind)}, {deleting.date}, {display(deleting.amount)} {deleting.currencyCode} / {display(deleting.portfolioAmount)} {portfolio?.currencyCode}</p><p class="comment">{deleting.comment}</p>{/if}
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
  .row-actions { display: flex; gap: 8px; }
  .icon-button { display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0; flex: 0 0 40px; }
  .icon-button:hover { background: #edf3f7; }
  .delete-button { color: #a12e2e; }
  .delete-button:hover { background: #fff0f0; }
  .icon-button:focus-visible { outline: 2px solid #207653; outline-offset: 2px; }
  .comment { white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
