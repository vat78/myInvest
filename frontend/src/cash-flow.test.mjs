import { formatDecimal } from './decimal-format.mjs';
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';

const portfolios = [{ id: '1', name: 'Первый', currencyCode: 'RUB' }, { id: '2', name: 'Второй', currencyCode: 'USD' }];
const snapshot = (balance = '0') => ({ balance, currencyCode: 'RUB', negative: false, operations: [] });
const ok = { errors: {}, message: '' };
const flush = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
function harness(overrides = {}) {
  const source = readFileSync(new URL('./CashFlowManager.svelte', import.meta.url), 'utf8');
  const { instance } = parse(source);
  const script = source.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '');
  const effects = [], writes = [], removals = [], focus = [];
  const sandbox = {
    formatDecimal,
    $props: () => ({ oncreate() {} }), $state: value => value,
    $derived: value => value, $effect: fn => effects.push(fn),
    tick: async () => {}, onDestroy() {}, localDate: () => '2026-10-13', setTimeout, clearTimeout,
    ListPortfolios: async () => portfolios, ListCurrencies: async () => ['RUB', 'USD'],
    ListCashFlows: async () => snapshot(), PreviewCashRate: async () => ({ rate: '94.5', errors: {} }),
    SaveCashFlow: async (pid, id, input) => { writes.push({ pid, id, ...input }); return ok; },
    DeleteCashFlow: async (pid, id) => { removals.push({ pid, id }); return ok; },
    ...overrides, focus, effects,
  };
  vm.createContext(sandbox);
  // These tests execute the component's asynchronous handlers. Derived values
  // are refreshed explicitly because this harness does not run the Svelte DOM.
  vm.runInContext(ts.transpileModule(`${script}
    const trigger = {isConnected: true, focus() {focus.push('trigger')}};
    portfolioSelect = {focus() {focus.push('portfolio')}};
    form = {showModal() {}, close() {}, querySelector() {return {focus() {if(!busy) focus.push('error')}}}};
    confirmation = {showModal() {}, close() {}};
    dateInput = {focus() {focus.push('date')}};
    globalThis.ui = {
      loadPortfolios, load,
      select(pid) {selected=pid;portfolio=portfolios.find(p=>p.id===pid);return load()},
      async open(op=null,k='deposit') {portfolio=portfolios.find(p=>p.id===selected);await open(op,k,trigger);different=currencyCode!==portfolio.currencyCode;},
      enter(a,b='',c='') {amount=a;portfolioAmount=b;comment=c;},
      currency(code) {currencyCode=code;different=code!==portfolio.currencyCode;currencyChanged();},
      save() {return save({preventDefault(){}})}, close,
      askDelete(op) {askDelete(op,trigger)}, closeDelete, remove,
      preview() {return effects[1]()},
      disconnectTrigger() {trigger.isConnected=false;root={querySelector(selector){focus.push(selector);return {focus(){focus.push('replacement')}}}}},
      get state() {return {selected,snapshot,loading,error,listError,listLoading,editing,deleting,busy,amount,portfolioAmount,currencyCode,date,rate,message,errors}},
    };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return { ui: sandbox.ui, writes, removals, focus };
}

test('cash form defaults, one/two amounts, currency clearing and cancel', async () => {
  const { ui, writes, focus } = harness(); await ui.loadPortfolios(); await ui.open(); await flush();
  assert.equal(ui.state.date, '2026-10-13'); assert.equal(ui.state.currencyCode, 'RUB'); assert.equal(focus.at(-1), 'date');
  ui.enter('0,1', '', 'text'); await ui.save();
  assert.equal(writes[0].amount, '0,1'); assert.equal(writes[0].portfolioAmount, '0,1'); assert.equal(ui.state.editing, false); assert.equal(focus.at(-1), 'trigger');
  await ui.open(); await flush(); ui.enter('old', 'old'); ui.currency('USD');
  assert.equal(ui.state.amount, ''); assert.equal(ui.state.portfolioAmount, '');
  ui.enter('1000', '94500'); await ui.save();
  assert.equal(writes[1].currencyCode, 'USD'); assert.equal(writes[1].portfolioAmount, '94500');
  await ui.open(); ui.close(); assert.equal(writes.length, 2);
});
test('stale cash response cannot replace selected portfolio or finish its loading', async () => {
  const pending = [];
  const { ui } = harness({ ListCashFlows: pid => new Promise(resolve => pending.push({ pid, resolve })) });
  const initial = ui.loadPortfolios(); await flush();
  const second = ui.select('2');
  pending[0].resolve(snapshot('111')); await initial;
  assert.equal(ui.state.selected, '2'); assert.equal(ui.state.snapshot, null); assert.equal(ui.state.loading, true);
  pending[1].resolve({ ...snapshot('222'), currencyCode: 'USD' }); await second;
  assert.equal(ui.state.snapshot.balance, '222');
});
test('cash read error is separate from zero balance and retries', async () => {
  let fails = true;
  const { ui } = harness({ ListCashFlows: async () => { if (fails) throw new Error('SQL private'); return snapshot(); } });
  await ui.loadPortfolios(); assert.equal(ui.state.snapshot, null); assert.ok(ui.state.error); assert.equal(ui.state.loading, false);
  fails = false; await ui.load(); assert.equal(ui.state.error, ''); assert.equal(ui.state.snapshot.balance, '0');
});
test('editing retains portfolio/id, validation keeps values and focuses enabled input', async () => {
  const writes = [];
  const { ui, focus } = harness({ SaveCashFlow: async (pid,id,input) => { writes.push({ pid,id,...input }); return { errors: { amount: 'invalid' }, message: '' }; } });
  await ui.loadPortfolios(); await ui.open({ id:'7',kind:'withdrawal',date:'2024-02-29',currencyCode:'USD',amount:'500',portfolioAmount:'48000',comment:'old' }); await flush();
  ui.enter('0','new','comment'); await ui.save();
  assert.equal(writes[0].id,'7'); assert.equal(writes[0].pid,'1'); assert.equal(ui.state.amount,'0'); assert.equal(ui.state.editing,true); assert.equal(focus.at(-1),'error');
});
test('pending save prevents duplicates/cancel; failure retains values', async () => {
  let reject, count = 0;
  const { ui } = harness({ SaveCashFlow: () => {count++;return new Promise((_,r)=>{reject=r})} });
  await ui.loadPortfolios(); await ui.open(); await flush(); ui.enter('1');
  const saving=ui.save(); await ui.save(); ui.close(); assert.equal(count,1); assert.equal(ui.state.editing,true); assert.equal(ui.state.busy,true);
  reject(new Error('SQL private')); await saving; assert.equal(ui.state.amount,'1'); assert.equal(ui.state.busy,false); assert.ok(!ui.state.message.includes('SQL'));
});
test('delete requires confirmation, blocks repeats and retains confirmation on failure', async () => {
  let resolve,count=0;
  const {ui}=harness({ DeleteCashFlow: ()=>{count++;return new Promise(r=>{resolve=r})} });
  await ui.loadPortfolios(); const op={id:'4',amount:'2',currencyCode:'USD',portfolioAmount:'200'};
  ui.askDelete(op); ui.closeDelete(); assert.equal(count,0);
  ui.askDelete(op); const deleting=ui.remove(); await ui.remove(); ui.closeDelete(); assert.equal(count,1); assert.equal(ui.state.deleting.id,'4');
  resolve({errors:{},message:'write failed'}); await deleting; assert.equal(ui.state.deleting.id,'4');assert.equal(ui.state.busy,false);
  const again=ui.remove();resolve(ok);await again;assert.equal(ui.state.deleting,null);
});
test('late completion of refresh cannot unlock a later save', async () => {
  let resolveRead,resolveSave,reads=0,writes=0;
  const {ui}=harness({
    ListCashFlows:()=>++reads===2?new Promise(r=>{resolveRead=r}):Promise.resolve(snapshot()),
    SaveCashFlow:()=>++writes===1?Promise.resolve(ok):new Promise(r=>{resolveSave=r}),
  });
  await ui.loadPortfolios(); await ui.open(); await flush(); ui.enter('1'); const first=ui.save(); await flush();
  const newerRead=ui.load(); await newerRead; await ui.open(); await flush(); ui.enter('2');const second=ui.save();
  resolveRead(snapshot());await first; assert.equal(ui.state.busy,true); await ui.save(); assert.equal(writes,2);
  resolveSave(ok);await second;assert.equal(ui.state.busy,false);
});
test('preview uses Go result and discards obsolete response', async () => {
  const pending=[];
  const {ui}=harness({ PreviewCashRate:(a,b)=>new Promise(resolve=>pending.push({a,b,resolve})) });
  await ui.loadPortfolios();await ui.open();await flush();ui.currency('USD');ui.enter('3','1');ui.preview();
  await new Promise(resolve=>setTimeout(resolve,220));assert.equal(pending[0].a,'3');
  ui.enter('6','1');ui.preview();await new Promise(resolve=>setTimeout(resolve,220));
  pending[1].resolve({rate:'0.166666666666666667'});await flush();pending[0].resolve({rate:'0.333333333333333333'});await flush();
  assert.equal(ui.state.rate,'0.166666666666666667');
});

test('successful edit returns focus to replacement row after refresh', async () => {
  const {ui,focus}=harness();await ui.loadPortfolios();await ui.open({id:'7',kind:'deposit',date:'2026-01-01',currencyCode:'RUB',amount:'1',portfolioAmount:'1',comment:''});await flush();
  ui.disconnectTrigger();await ui.save();assert.equal(focus.at(-1),'replacement');assert.equal(focus.at(-2),'[data-cash-id="7"]');
});
