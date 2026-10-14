<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAllocation, SaveAllocation, DeleteAllocation, PreviewAllocation } from '../wailsjs/go/desktop/AllocationAPI';
  import { valuation } from '../wailsjs/go/models';
  import { localDate } from './calendar';
  import { formatDecimal, formatChartAmount } from './decimal-format.mjs';
  import TargetPie from './TargetPie.svelte';
  let { pid, revision = 0, onquotes = () => {} }: { pid: string; revision?: number; onquotes?: () => void } = $props();
  let data = $state<valuation.Allocation | null>(null);
  let loading = $state(false), error = $state(''), busy = $state(false), opened = $state(false);
  let draft = $state({ generation: '', revision: '', cashPercent: '100', targets: [] as {assetID: string; percent: string}[] });
  let errors = $state<Record<string, string>>({}), message = $state(''), conflict = $state(false);
  let preview = $state<valuation.AllocationPreview | null>(null);
  let dialog: HTMLDialogElement;
  let trigger: HTMLElement | null = null;
  let initial = '', editPid = '', request = 0, previewRequest = 0, alive = true;
  const input = () => new valuation.AllocationPlan(draft);
  function shown(value: string, money = false): string {
    if (!value) return '—';
    const approximate = value.startsWith('≈');
    const raw = approximate ? value.slice(1) : value;
    if (raw === '+0' || raw === '−0') return `≈${raw}`;
    return `${approximate ? '≈' : ''}${money ? formatChartAmount(raw) : formatDecimal(raw)}`;
  }
  function change(value: string, money = false): string {
    const formatted = shown(value, money);
    const raw = value.replace(/^≈/, '');
    if (/^[0-9]/.test(raw) && /[1-9]/.test(raw)) return formatted.startsWith('≈') ? `≈+${formatted.slice(1)}` : `+${formatted}`;
    return formatted;
  }
  async function refresh() {
    const token = ++request, selected = pid, day = localDate();
    data = null; error = ''; loading = true;
    try {
      if (!selected) return;
      const result = await GetAllocation(selected, day);
      if (alive && token === request && selected === pid && day === localDate()) data = result;
    } catch { if (alive && token === request) error = 'Не удалось загрузить целевое распределение.'; }
    finally { if (alive && token === request) loading = false; }
  }
  $effect(() => { void pid; void revision; void refresh(); });
  onMount(() => () => { alive = false; ++request; ++previewRequest; });
  function openEditor(event: MouseEvent) {
    if (!data || busy) return;
    trigger = event.currentTarget as HTMLElement;
    draft = { generation: data.plan.generation, revision: data.plan.revision, cashPercent: data.plan.generation ? data.plan.cashPercent : '100', targets: data.plan.targets.map(t => ({ assetID: t.assetID, percent: t.percent })) };
    initial = JSON.stringify(draft); editPid = pid; errors = {}; message = ''; conflict = false; opened = true;
    dialog.showModal(); void updatePreview();
  }
  export function canLeave() {
    if (busy) return false;
    if (opened && JSON.stringify(draft) !== initial && !window.confirm('Отбросить несохранённые изменения распределения?')) return false;
    if (opened) closeEditor();
    return true;
  }
  function closeEditor() { opened = false; ++previewRequest; dialog.close(); trigger?.focus(); }
  function cancel(event?: Event) { event?.preventDefault(); if (!busy) canLeave(); }
  async function updatePreview() {
    const token = ++previewRequest; preview = null;
    try { const result = await PreviewAllocation(input()); if (alive && opened && token === previewRequest) preview = result; }
    catch { if (alive && opened && token === previewRequest) message = 'Не удалось проверить сумму долей.'; }
  }
  function addRow() { draft.targets.push({ assetID: '', percent: '0' }); void updatePreview(); }
  function removeRow(index: number) { draft.targets.splice(index, 1); errors = {}; void updatePreview(); }
  async function save() {
    if (busy || editPid !== pid) return;
    busy = true; errors = {}; message = ''; conflict = false;
    try {
      const result = await SaveAllocation(editPid, input());
      if (!alive) return;
      errors = result.errors; message = result.message; conflict = result.conflict;
      if (!message && !Object.keys(errors).length) { closeEditor(); await refresh(); }
    } catch { if (alive) message = 'Не удалось сохранить распределение. Повторите попытку.'; }
    finally { if (alive) busy = false; }
  }
  async function loadVersion() {
    if (busy) return;
    busy = true; message = '';
    try {
      const result = await GetAllocation(editPid, localDate());
      if (!alive || editPid !== pid) return;
      data = result; draft.generation = result.plan.generation; draft.revision = result.plan.revision;
      conflict = false; message = 'Актуальная версия загружена. Ваши доли сохранены в форме; проверьте их перед сохранением.';
    } catch { if (alive) message = 'Не удалось загрузить актуальную версию.'; }
    finally { if (alive) busy = false; }
  }
  async function deletePlan() {
    if (!data || busy || !window.confirm('Удалить целевое распределение этого портфеля?')) return;
    const current = data; busy = true; error = '';
    try {
      const result = await DeleteAllocation(pid, current.plan.generation, current.plan.revision);
      if (!alive) return;
      if (result.message) error = result.message; else await refresh();
    } catch { if (alive) error = 'Не удалось удалить распределение.'; }
    finally { if (alive) busy = false; }
  }
</script>
<section aria-label="Целевое распределение">
  <header><h2>Целевое распределение</h2>{#if data}<button disabled={busy} onclick={openEditor}>{data.plan.generation ? 'Изменить распределение' : 'Задать распределение'}</button>{/if}</header>
  {#if loading}<p role="status">Загрузка целевого распределения…</p>{/if}
  {#if error}<p class="warning" role="alert">{error} <button disabled={busy} onclick={() => void refresh()}>Повторить</button></p>{/if}
  {#if data && !data.plan.generation}<p>Задайте целевые доли активов и денег для этого портфеля.</p>{/if}
  {#if data?.plan.generation}
    <p class="hint">На {data.date} · {data.currency}. Цена более поздней котировки или покупки. При равной дате — котировка.</p>
    <p>Стоимость с деньгами: <strong>{shown(data.total, true)} {data.currency}</strong> · Свободные деньги: {shown(data.cash, true)} · Цель денег: {shown(data.reserve, true)}</p>
    <div class="layout">
      <article><TargetPie rows={data.rows} /></article>
      <article class="details"><div class="scroll"><table>
        <caption>Текущие и целевые активы · {data.currency}</caption>
        <thead><tr><th scope="col">Актив</th><th scope="col">Сейчас, количество</th><th scope="col">Сейчас, стоимость</th><th scope="col">Целевая доля</th><th scope="col">Целевая стоимость</th><th scope="col">Целевое количество</th><th scope="col">Изменение без продаж</th><th scope="col">Изменение с продажами</th></tr></thead>
        <tbody>{#each data.rows as row (row.id)}<tr><th scope="row">{row.ticker}</th><td>{shown(row.quantity)}</td><td>{shown(row.currentValue, true)}</td><td>{shown(row.percent)}%</td><td>{shown(row.targetValue, true)}</td><td>{shown(row.targetQuantity)}</td><td>{#if !row.cash}<span>{change(row.buyQuantity)} шт.</span>{/if}{change(row.buyValue, true)} {data.currency}</td><td>{#if !row.cash}<span>{change(row.rebalanceQuantity)} шт.</span>{/if}{change(row.rebalanceValue, true)} {data.currency}</td></tr>
          {#if !row.cash && (row.priceDate || row.issue)}<tr class="source"><td colspan="8">{row.ticker}: {row.priceDate ? `${row.priceSource === 'quote' ? 'Котировка' : 'Покупка'} от ${row.priceDate} · ${row.priceCurrency}` : ''}{#if row.priceRate} · курс {formatDecimal(row.priceRate)} от {row.priceRateDate}{/if}{#if row.issue} · {row.issue}{/if}</td></tr>{/if}
        {/each}</tbody>
      </table></div></article>
    </div>
    {#each data.issues as issue}<p class="warning">{issue}</p>{/each}
    {#if !data.canCalculate}<button onclick={onquotes}>Открыть котировки</button>{/if}
    {#if data.canCalculate}<p>Без продаж: остаток денег {shown(data.cashAfterBuys, true)} {data.currency}; нехватка до резерва {shown(data.reserveShortfall, true)}. Целевые доли активов могут быть достигнуты лишь частично.</p><p>С продажами: целевые доли достигаются; остаток денег {shown(data.reserve, true)} {data.currency}.</p>{/if}
    <p class="hint">В изменениях: плюс — покупка, минус — продажа; в строке денег — изменение остатка. Только деньги сверх целевого резерва распределяются пропорционально дефицитам. Количества теоретические, без лотов, будущих комиссий и налогов. ≈ означает округление. План не создаёт операций.</p>
    <button disabled={busy} onclick={() => void deletePlan()}>Удалить план</button>
  {/if}
</section>
<dialog bind:this={dialog} oncancel={cancel} aria-labelledby="allocation-title">
  <form onsubmit={event => { event.preventDefault(); void save(); }}>
    <h2 id="allocation-title">Целевые доли портфеля</h2>
    <fieldset disabled={busy}><legend>Активы и деньги · всего 100%</legend>
      <label for="cash-target">Деньги, %</label><input id="cash-target" inputmode="decimal" bind:value={draft.cashPercent} oninput={() => void updatePreview()} aria-describedby="cash-error" aria-invalid={!!errors.cashPercent} />
      <p id="cash-error" class="warning">{errors.cashPercent || preview?.errors.cashPercent || ''}</p>
      {#each draft.targets as target, i}
        <div class="edit-row"><label for={`target-asset-${i}`}>Актив {i + 1}</label><select id={`target-asset-${i}`} bind:value={target.assetID} onchange={() => void updatePreview()} aria-describedby={`asset-error-${i}`} aria-invalid={!!errors[`targets.${i}.assetID`]}><option value="">Выберите актив</option>{#each data?.assets ?? [] as asset (asset.ID)}<option value={asset.ID}>{asset.Ticker}</option>{/each}</select>
        <label for={`target-percent-${i}`}>Доля, %</label><input id={`target-percent-${i}`} inputmode="decimal" bind:value={target.percent} oninput={() => void updatePreview()} aria-describedby={`percent-error-${i}`} aria-invalid={!!errors[`targets.${i}.percent`]} /><button type="button" onclick={() => removeRow(i)} aria-label={`Удалить строку ${i + 1}`}>Удалить</button></div>
        <p id={`asset-error-${i}`} class="warning">{errors[`targets.${i}.assetID`] || preview?.errors[`targets.${i}.assetID`] || ''}</p><p id={`percent-error-${i}`} class="warning">{errors[`targets.${i}.percent`] || preview?.errors[`targets.${i}.percent`] || ''}</p>
      {/each}
      <button type="button" onclick={addRow}>Добавить актив</button>
    </fieldset>
    <p aria-live="polite">{preview?.total ? `Всего: ${preview.total}%. До 100%: ${preview.difference} п. п.` : 'Проверьте введённые доли.'}</p>
    {#if errors.total}<p class="warning" role="alert">{errors.total}</p>{/if}
    {#if message}<p class="warning" role="alert">{message}</p>{/if}
    {#if conflict}<button type="button" disabled={busy} onclick={() => void loadVersion()}>Загрузить актуальную версию, сохранив введённые доли</button>{/if}
    <div class="actions"><button type="submit" disabled={busy || conflict}>Сохранить</button><button type="button" disabled={busy} onclick={() => cancel()}>Отмена</button></div>
  </form>
</dialog>
<style>
  section { margin-top: 28px; border-top: 1px solid #dce3e9; padding-top: 24px; } header { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 12px; align-items: center; } h2 { font-size: 18px; } button, select, input { font: inherit; padding: 10px 12px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; color: inherit; max-width: 100%; } button { cursor: pointer; } button:disabled { cursor: default; opacity: .6; } .layout { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(0, 3fr); gap: 20px; } article { min-width: 0; padding: 16px; background: white; border: 1px solid #dce3e9; border-radius: 14px; } .scroll { overflow-x: auto; } table { border-collapse: collapse; width: 100%; font-size: 13px; } caption { text-align: left; margin-bottom: 12px; } th, td { padding: 12px 8px; text-align: right; border-bottom: 1px solid #dce3e9; white-space: nowrap; font-variant-numeric: tabular-nums; } th:first-child { text-align: left; } td span { display: block; margin-bottom: 5px; } .source td { text-align: left; white-space: normal; color: #506477; font-size: 12px; } .hint { color: #506477; font-size: 13px; } .warning { color: #a12e2e; } dialog { border: 1px solid #c4cfd8; border-radius: 14px; padding: 24px; width: min(700px, calc(100vw - 48px)); max-height: calc(100vh - 48px); overflow: auto; } dialog::backdrop { background: #10203070; } fieldset { border: 0; padding: 0; margin: 0; } .edit-row { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; } input { width: 100px; } label { font-size: 13px; margin-right: 8px; } .actions { display: flex; gap: 12px; margin-top: 20px; } @media (max-width: 1000px) { .layout { grid-template-columns: 1fr; } }
</style>
