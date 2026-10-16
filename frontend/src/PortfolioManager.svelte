<script lang="ts">
  import { tick, onDestroy } from 'svelte';
  import { ListCurrencies } from '../wailsjs/go/desktop/API';
  import { ListPortfolios, CreatePortfolio, UpdatePortfolio } from '../wailsjs/go/desktop/PortfolioAPI';
  import ActionIcon from './ActionIcon.svelte';
  import { localDate } from './calendar';
  let { onchanged = () => {}, onassets, onbenchmarks }: { onchanged?: () => void; onbenchmarks?: (trigger: HTMLButtonElement) => void; onassets?: (trigger: HTMLButtonElement) => void } = $props();
  type Portfolio = { id: string; name: string; openedOn: string; currencyCode: string };
  let menu = $state(false);
  let menuRoot: HTMLDivElement;
  let menuButton: HTMLButtonElement;
  let portfolioItem = $state<HTMLButtonElement>();
  let listDialog: HTMLDialogElement;
  let formDialog: HTMLDialogElement;
  let nameInput: HTMLInputElement;
  let addButton: HTMLButtonElement;
  let returnButton: HTMLButtonElement;
  let opened = $state(false);
  let editing = $state(false);
  let portfolios = $state<Portfolio[]>([]);
  let loading = $state(false);
  let listError = $state('');
  let currencies = $state<string[]>([]);
  let currencyLoading = $state(false);
  let currencyError = $state('');
  let id = $state('');
  let name = $state('');
  let openedOn = $state('');
  let currencyCode = $state('');
  let errors = $state<Record<string, string>>({});
  let message = $state('');
  let busy = $state(false);
  let listRequest = 0;
  let currencyRequest = 0;
  let alive = true;

  async function toggleMenu() {
    menu = !menu;
    if (menu) { await tick(); if (menu) portfolioItem?.focus(); }
  }
  function openAssets() { menu = false; onassets?.(menuButton); }
  function openBenchmarks() { menu = false; onbenchmarks?.(menuButton); }
  function dismissMenu(event: MouseEvent) {
    if (menu && !menuRoot.contains(event.target as Node)) menu = false;
  }
  function menuKey(event: KeyboardEvent) {
    if (menu && event.key === 'Escape') { event.preventDefault(); menu = false; menuButton.focus(); }
  }
  async function loadList() {
    const request = ++listRequest;
    loading = true; listError = '';
    try {
      const rows = await ListPortfolios();
      if (alive && opened && request === listRequest) portfolios = rows;
    } catch {
      if (alive && opened && request === listRequest) listError = 'Не удалось загрузить портфели. Повторите попытку';
    } finally { if (request === listRequest) loading = false; }
  }
  export function openList() {
    menu = false; opened = true;
    listDialog.showModal();
    void loadList();
  }
  function closeList() {
    if (editing) return;
    opened = false; ++listRequest; loading = false;
    listDialog.close(); menuButton.focus();
  }
  async function loadCurrencies() {
    const request = ++currencyRequest;
    currencyLoading = true; currencyError = ''; currencies = [];
    try {
      const codes = await ListCurrencies();
      if (alive && editing && request === currencyRequest) currencies = codes;
    } catch {
      if (alive && editing && request === currencyRequest) currencyError = 'Не удалось загрузить валюты. Повторите попытку';
    } finally { if (request === currencyRequest) currencyLoading = false; }
  }
  async function openForm(p: Portfolio | null, trigger: HTMLButtonElement) {
    returnButton = trigger;
    id = p?.id ?? ''; name = p?.name ?? ''; openedOn = p?.openedOn ?? localDate(); currencyCode = p?.currencyCode ?? '';
    errors = {}; message = ''; currencyError = ''; editing = true;
    await tick();
    formDialog.showModal(); nameInput.focus();
    if (!p) void loadCurrencies();
  }
  function closeForm() {
    if (busy) return;
    editing = false; ++currencyRequest; currencyLoading = false;
    formDialog.close(); returnButton?.focus();
  }
  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (busy || (!id && (currencyLoading || currencyError))) return;
    busy = true; errors = {}; message = '';
    let result: Awaited<ReturnType<typeof CreatePortfolio>>;
    try {
      const input = { name, openedOn, currencyCode };
      result = id ? await UpdatePortfolio(id, input) : await CreatePortfolio(input);
    } catch {
      if (alive) message = 'Не удалось сохранить портфель. Повторите попытку';
      return;
    } finally { busy = false; }
    if (!alive) return;
    errors = result.errors; message = result.message;
    if (message || Object.keys(errors).length) {
      await tick();
      formDialog.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
      return;
    }
    onchanged();
    const savedID = id;
    closeForm();
    // Refreshing the list must not change the busy state of a later save.
    await loadList();
    await tick();
    if (alive && opened && !editing) {
      const trigger = savedID ? listDialog.querySelector<HTMLButtonElement>(`[data-portfolio-id="${savedID}"]`) : addButton;
      (trigger ?? addButton).focus();
    }
  }
  onDestroy(() => { alive = false; ++listRequest; ++currencyRequest; });
</script>

<svelte:window onclick={dismissMenu} onkeydown={menuKey} />
<div class="menu-root" bind:this={menuRoot}>
  <button bind:this={menuButton} class="menu-button" aria-label="Меню" aria-expanded={menu} aria-controls="app-menu" onclick={() => void toggleMenu()}>⋯</button>
  {#if menu}
    <nav id="app-menu" aria-label="Меню приложения">
      <button bind:this={portfolioItem} onclick={openList}>Портфели</button>
      <button onclick={openAssets}>Активы</button>
      <button onclick={openBenchmarks}>Бенчмарки</button>
    </nav>
  {/if}
</div>

<dialog class="list" bind:this={listDialog} aria-labelledby="portfolios-title" oncancel={(event) => { event.preventDefault(); closeList(); }}>
  <div class="title"><h2 id="portfolios-title">Портфели</h2><button onclick={closeList} aria-label="Закрыть портфели">×</button></div>
  <button bind:this={addButton} class="primary" onclick={() => void openForm(null, addButton)}>Добавить портфель</button>
  {#if loading}
    <p role="status">Загрузка портфелей…</p>
  {:else if listError}
    <p class="error" role="alert">{listError}</p><button onclick={() => void loadList()}>Повторить загрузку</button>
  {:else if portfolios.length === 0}
    <p>Портфелей пока нет</p>
  {:else}
    <div class="table-scroll"><table>
      <thead><tr><th>Название</th><th>Дата открытия</th><th>Валюта</th><th>Действия</th></tr></thead>
      <tbody>{#each portfolios as p (p.id)}
        <tr><td>{p.name}</td><td><time datetime={p.openedOn}>{p.openedOn.split('-').reverse().join('.')}</time></td><td>{p.currencyCode}</td>
          <td><button class="icon-button" title="Редактировать" data-portfolio-id={p.id} aria-label={`Редактировать портфель ${p.name}`} onclick={(event) => void openForm(p, event.currentTarget)}><ActionIcon kind="edit" /></button></td></tr>
      {/each}</tbody>
    </table></div>
  {/if}
</dialog>

<dialog bind:this={formDialog} aria-labelledby="portfolio-form-title" oncancel={(event) => { event.preventDefault(); closeForm(); }}>
  <form onsubmit={save} novalidate>
    <h2 id="portfolio-form-title">{id ? 'Редактировать портфель' : 'Добавить портфель'}</h2>
    <fieldset disabled={busy}>
      <label for="portfolio-name">Название</label>
      <input id="portfolio-name" bind:this={nameInput} bind:value={name} required aria-invalid={!!errors.name} aria-describedby={errors.name ? 'name-error' : undefined} />
      {#if errors.name}<p id="name-error" class="error" role="alert">{errors.name}</p>{/if}
      <label for="portfolio-date">Дата открытия</label>
      <input id="portfolio-date" type="date" bind:value={openedOn} required aria-invalid={!!errors.openedOn} aria-describedby={errors.openedOn ? 'date-error' : undefined} />
      {#if errors.openedOn}<p id="date-error" class="error" role="alert">{errors.openedOn}</p>{/if}
      <label for="portfolio-currency">Валюта</label>
      {#if id}
        <input id="portfolio-currency" value={currencyCode} readonly aria-describedby={errors.currencyCode ? 'currency-error' : undefined} />
      {:else}
        <select id="portfolio-currency" bind:value={currencyCode} required disabled={currencyLoading || !!currencyError} aria-invalid={!!errors.currencyCode} aria-describedby={errors.currencyCode ? 'currency-error' : undefined}>
          <option value="">Выберите валюту</option>
          {#each currencies as code}<option value={code}>{code}</option>{/each}
        </select>
        {#if currencyLoading}<p role="status">Загрузка валют…</p>{/if}
        {#if currencyError}<p class="error" role="alert">{currencyError}</p><button type="button" onclick={() => void loadCurrencies()}>Повторить загрузку валют</button>{/if}
        {#if !currencyLoading && !currencyError && currencies.length === 0}<p>Справочник валют пуст. Добавьте валюту в справочнике.</p>{/if}
      {/if}
      {#if errors.currencyCode}<p id="currency-error" class="error" role="alert">{errors.currencyCode}</p>{/if}
    </fieldset>
    {#if message}<p class="error" role="alert">{message}</p>{/if}
    <div class="actions">
      <button type="button" disabled={busy} onclick={closeForm}>Отмена</button>
      <button type="submit" class="primary" disabled={busy || (!id && (currencyLoading || !!currencyError))}>{busy ? 'Сохранение…' : 'Сохранить'}</button>
    </div>
  </form>
</dialog>

<style>
  button, input, select { font: inherit; color: inherit; }
  button { cursor: pointer; padding: 10px 16px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; }
  button:disabled { cursor: default; opacity: .6; }
  .menu-root { position: relative; }
  .menu-button { font-size: 24px; line-height: 1; }
  nav { position: absolute; right: 0; top: calc(100% + 8px); z-index: 1; padding: 6px; background: white; border: 1px solid #dce3e9; border-radius: 10px; box-shadow: 0 4px 16px #172a3a22; }
  nav button { border: 0; width: 100%; text-align: left; }
  dialog { width: min(520px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 28px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog.list { width: min(850px, calc(100% - 32px)); }
  dialog::backdrop { background: #172a3a66; }
  h2 { margin: 0 0 12px; }
  .title { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 16px; }
  fieldset { padding: 0; margin: 24px 0; border: 0; min-width: 0; }
  label { display: block; margin: 18px 0 8px; font-size: 14px; }
  input, select { width: 100%; min-width: 0; border: 1px solid #c4cfd8; border-radius: 8px; padding: 12px; background: white; }
  input[readonly] { background: #f5f7fa; }
  [aria-invalid="true"] { border-color: #a12e2e; }
  .actions { display: flex; justify-content: flex-end; gap: 12px; }
  .icon-button { display: inline-flex; align-items: center; justify-content: center; width: 40px; height: 40px; padding: 0; }
  .icon-button:hover { background: #edf3f7; }
  .icon-button:focus-visible { outline: 2px solid #207653; outline-offset: 2px; }
  .primary { background: #207653; color: white; border-color: #207653; }
  .error { color: #a12e2e; overflow-wrap: anywhere; font-size: 14px; }
  .table-scroll { overflow-x: auto; margin-top: 20px; }
  table { border-collapse: collapse; width: 100%; text-align: left; }
  th, td { padding: 12px; border-bottom: 1px solid #dce3e9; }
  td:first-child { overflow-wrap: anywhere; max-width: 280px; }
  th { font-size: 14px; color: #506477; }
</style>
