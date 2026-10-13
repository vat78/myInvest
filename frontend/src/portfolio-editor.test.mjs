import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';

function harness(overrides = {}) {
  const source = readFileSync(new URL('./PortfolioManager.svelte', import.meta.url), 'utf8');
  const { instance } = parse(source);
  const script = source.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '').replace(/export function /g, 'function ');
  const writes = [];
  const calls = { focus: [], lists: 0 };
  const ok = { portfolio: { id: '1' }, errors: {}, message: '' };
  const sandbox = {
    $state: v => v, $props: () => ({}), tick: async () => {}, onDestroy: () => {}, localDate: () => '2026-10-13',
    ListCurrencies: async () => ['RUB', 'USD'],
    ListPortfolios: async () => { calls.lists++; return []; },
    CreatePortfolio: async input => { writes.push(input); return ok; },
    UpdatePortfolio: async (id, input) => { writes.push({ id, ...input }); return ok; },
    ...overrides,
  };
  vm.createContext(sandbox);
  sandbox.calls = calls;
  vm.runInContext(ts.transpileModule(`${script}
    menuButton = {focus() {calls.focus.push('menu')}};
    portfolioItem = {focus() {calls.focus.push('item')}};
    menuRoot = {contains(node) {return node === 'inside'}};
    listDialog = {showModal() {}, close() {}, querySelector() { return addButton; }};
    formDialog = {showModal() {}, close() {}, querySelector() {return {focus() {if (!busy) calls.focus.push('error')}}}};
    nameInput = {focus() {calls.focus.push('name')}};
    addButton = {focus() {calls.focus.push('add')}};
    globalThis.ui = {
      openList, closeList, toggleMenu, dismissMenu, menuKey,
      open(p = null) { return openForm(p, addButton); },
      close: closeForm,
      enter(n, d, c) { name = n; openedOn = d; currencyCode = c; },
      save() { return save({preventDefault() {}}); }, loadList, loadCurrencies,
      get state() { return {menu, opened, editing, name, openedOn, currencyCode, busy, errors, message, listError, loading, portfolios}; },
    };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return { ui: sandbox.ui, writes, calls };
}
const flush = async () => { await Promise.resolve(); await Promise.resolve(); };

test('menu toggles, closes outside or on Escape, and opens the list', async () => {
  const { ui, calls } = harness();
  await ui.toggleMenu(); assert.equal(ui.state.menu, true); assert.equal(calls.focus.at(-1), 'item');
  ui.dismissMenu({ target: 'inside' }); assert.equal(ui.state.menu, true);
  ui.dismissMenu({ target: 'outside' }); assert.equal(ui.state.menu, false);
  await ui.toggleMenu(); ui.menuKey({ key: 'Escape', preventDefault() {} }); assert.equal(ui.state.menu, false); assert.equal(calls.focus.at(-1), 'menu');
  await ui.toggleMenu(); await ui.toggleMenu(); assert.equal(ui.state.menu, false);
  ui.openList(); await flush(); assert.equal(ui.state.opened, true); assert.equal(ui.state.portfolios.length, 0);
  ui.closeList(); assert.equal(calls.focus.at(-1), 'menu');
});

test('creation defaults, cancellation, and successful save', async () => {
  const { ui, writes, calls } = harness();
  ui.openList(); await ui.open(); await flush();
  assert.equal(ui.state.name, ''); assert.equal(ui.state.openedOn, '2026-10-13'); assert.equal(ui.state.currencyCode, ''); assert.equal(calls.focus.at(-1), 'name');
  ui.enter('Основной', '2024-02-29', 'RUB'); ui.close(); assert.equal(writes.length, 0); assert.equal(calls.focus.at(-1), 'add');
  await ui.open(); await flush(); ui.enter('Основной', '2024-02-29', 'RUB'); await ui.save();
  assert.equal(writes.length, 1); assert.equal(writes[0].openedOn, '2024-02-29'); assert.equal(ui.state.editing, false);
});

test('editing keeps identity and currency, failed save keeps entered values', async () => {
  const { ui, writes } = harness({ UpdatePortfolio: async (id, input) => { writes.push({ id, ...input }); return { errors: { name: 'duplicate' }, message: '' }; } });
  ui.openList(); await ui.open({ id: '42', name: 'Original', openedOn: '2024-02-29', currencyCode: 'USD' });
  ui.enter('Updated', '2099-12-31', 'USD'); await ui.save();
  assert.equal(writes[0].id, '42'); assert.equal(writes[0].currencyCode, 'USD'); assert.equal(ui.state.editing, true); assert.equal(ui.state.name, 'Updated'); assert.equal(ui.state.openedOn, '2099-12-31'); assert.equal(ui.state.errors.name, 'duplicate');
});

test('storage failure preserves form and blocks repeat saves or cancellation in flight', async () => {
  let reject;
  const { ui, writes } = harness({ CreatePortfolio: input => { writes.push(input); return new Promise((_, r) => { reject = r; }); } });
  ui.openList(); await ui.open(); await flush(); ui.enter('P', '2026-01-01', 'USD');
  const saving = ui.save(); await ui.save(); ui.close();
  assert.equal(writes.length, 1); assert.equal(ui.state.editing, true); assert.equal(ui.state.busy, true);
  reject(new Error('SQL secret')); await saving;
  assert.equal(ui.state.editing, true); assert.equal(ui.state.name, 'P'); assert.ok(!ui.state.message.includes('SQL')); assert.equal(ui.state.busy, false);
});

test('read error has a retry and does not masquerade as an empty result', async () => {
  let fails = true;
  const { ui } = harness({ ListPortfolios: async () => { if (fails) throw new Error('read'); return [{ id: '1' }]; } });
  ui.openList(); await flush(); assert.ok(ui.state.listError); assert.equal(ui.state.loading, false);
  fails = false; await ui.loadList(); assert.equal(ui.state.listError, ''); assert.equal(ui.state.portfolios.length, 1);
});

test('successful save closes the form even when refreshing the list fails', async () => {
  const { ui, writes } = harness({ ListPortfolios: async () => { throw new Error('read'); } });
  ui.openList(); await ui.open(); await flush(); ui.enter('P', '2026-01-01', 'USD'); await ui.save();
  assert.equal(writes.length, 1); assert.equal(ui.state.editing, false); assert.ok(ui.state.listError);
});

test('validation failure enables the field before focusing it', async () => {
  const { ui, calls } = harness({ CreatePortfolio: async () => ({ errors: { name: 'duplicate' }, message: '' }) });
  ui.openList(); await ui.open(); await flush();
  ui.enter('Duplicate', '2026-01-01', 'USD'); await ui.save();
  assert.equal(ui.state.editing, true);
  assert.equal(ui.state.busy, false);
  assert.equal(calls.focus.at(-1), 'error');
});

test('finishing an earlier list refresh keeps a new save locked', async () => {
  let finishRefresh, finishSecondSave;
  let reads = 0, writes = 0;
  const ok = { errors: {}, message: '' };
  const { ui } = harness({
    ListPortfolios: () => ++reads === 2 ? new Promise(resolve => { finishRefresh = resolve; }) : Promise.resolve([]),
    CreatePortfolio: () => ++writes === 1 ? Promise.resolve(ok) : new Promise(resolve => { finishSecondSave = resolve; }),
  });
  ui.openList(); await ui.open(); await flush();
  ui.enter('First', '2026-01-01', 'USD');
  const firstSave = ui.save(); await flush();
  assert.equal(ui.state.editing, false);
  await ui.open(); await flush();
  ui.enter('Second', '2026-01-01', 'USD');
  const secondSave = ui.save();
  finishRefresh([]); await firstSave;
  assert.equal(ui.state.busy, true);
  await ui.save(); ui.close();
  assert.equal(writes, 2);
  assert.equal(ui.state.editing, true);
  assert.equal(ui.state.name, 'Second');
  finishSecondSave(ok); await secondSave;
  assert.equal(ui.state.editing, false);
  assert.equal(ui.state.busy, false);
});
