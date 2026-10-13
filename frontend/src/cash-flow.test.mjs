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
  const script = source.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '').replace(/export /g, '');
  const effects = [], writes = [], removals = [], focus = [];
  const sandbox = {
    formatDecimal,
    $props: () => ({ oncreate() {} }), $state: value => value,
    $derived: value => value, $effect: fn => effects.push(fn),
    tick: async () => {}, onDestroy() {}, localDate: () => '2026-10-13', setTimeout, clearTimeout,
    ListPortfolios: async () => portfolios, ListCurrencies: async () => ['RUB', 'USD'],
    ListAssets: async () => [{id:'1',ticker:'SBER'}],
    SaveSplit: async (pid,id,input) => { writes.push({pid,id,...input}); return ok; },
    PreviewSplit: async () => ({quantityBefore:'12',quantityAfter:'120',errors:{},message:''}),
    SaveTrade: async (pid,id,input) => { writes.push({pid,id,...input}); return ok; },
    PreviewTrade: async () => ({averagePrice:'100',rounded:false,moneyChange:'-255',errors:{}}),
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
      tradeEnter(asset,q,a,e='0') {assetID=asset;quantity=q;amount=a;expenses=e;},
      assetAdded, refreshAssets, loadAssets, tradePreview() {return effects[2]()},
      splitEnter(asset,old,next,d=date) {assetID=asset;oldUnits=old;newUnits=next;date=d;},
      splitPreview() {return effects[3]()},
      askDelete(op) {askDelete(op,trigger)}, closeDelete, remove,
      preview() {return effects[1]()},
      disconnectTrigger() {trigger.isConnected=false;root={querySelector(selector){focus.push(selector);return {focus(){focus.push('replacement')}}}}},
      get state() {return {selected,snapshot,loading,error,listError,listLoading,editing,deleting,busy,amount,portfolioAmount,currencyCode,date,rate,message,errors,assetID,quantity,expenses,assetError,price,priceRounded,moneyChange,previewError,oldUnits,newUnits,quantityBefore,quantityAfter,splitPreviewError}},
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

test('trade form defaults, exact strings, errors, selection of a new asset and cancel', async () => {
  const {ui,writes} = harness(); await ui.loadPortfolios(); await ui.open(null,'buy'); await flush();
  assert.equal(ui.state.assetID,''); assert.equal(ui.state.expenses,'0');
  await ui.assetAdded({id:'1',ticker:'SBER'}); assert.equal(ui.state.assetID,'1');
  ui.tradeEnter('1','2,5','250','5'); await ui.save();
  assert.equal(writes[0].quantity,'2,5'); assert.equal(writes[0].expenses,'5');
  assert.equal(writes[0].currencyCode,undefined); assert.equal(writes[0].portfolioAmount,undefined);
  assert.equal(ui.state.editing,false);
  await ui.open(null,'sell'); ui.close(); assert.equal(writes.length,1);
});
test('failed trade retains values and prohibits repeated writes', async () => {
  let resolve,count=0;
  const {ui,focus}=harness({SaveTrade:()=>{count++;return new Promise(r=>{resolve=r})}});
  await ui.loadPortfolios();await ui.open(null,'sell');await flush();ui.tradeEnter('1','0','120','2');
  const pending=ui.save();await ui.save();ui.close();assert.equal(count,1);assert.equal(ui.state.editing,true);
  resolve({errors:{quantity:'invalid'},message:''});await pending;
  assert.equal(ui.state.quantity,'0');assert.equal(ui.state.expenses,'2');assert.equal(focus.at(-1),'error');
});
test('trade preview discards outdated and closed-form results', async () => {
  const pending=[];
  const {ui}=harness({PreviewTrade:(k,q,a,e)=>new Promise(resolve=>pending.push({k,q,a,e,resolve}))});
  await ui.loadPortfolios();await ui.open(null,'buy');await flush();
  ui.tradeEnter('1','3','1');ui.tradePreview();await new Promise(r=>setTimeout(r,220));
  ui.tradeEnter('1','6','1');ui.tradePreview();await new Promise(r=>setTimeout(r,220));
  pending[1].resolve({averagePrice:'0.166666666666666667',rounded:true,moneyChange:'-1',errors:{}});await flush();
  pending[0].resolve({averagePrice:'old',rounded:false,moneyChange:'old',errors:{}});await flush();
  assert.equal(ui.state.price,'0.166666666666666667');assert.equal(ui.state.priceRounded,true);
  ui.tradeEnter('1','2','1');ui.tradePreview();await new Promise(r=>setTimeout(r,220));ui.close();
  pending[2].resolve({averagePrice:'closed',moneyChange:'closed',errors:{}});await flush();assert.notEqual(ui.state.price,'closed');
});
test('failed asset read blocks saving and supports retry', async () => {
  let fails=true;
  const {ui,writes}=harness({ListAssets:async()=>{if(fails)throw new Error('SQL');return [{id:'1',ticker:'SBER'}]}});
  await ui.loadPortfolios();await ui.open(null,'buy');await flush();ui.tradeEnter('1','1','1');await ui.save();
  assert.equal(writes.length,0);assert.ok(ui.state.assetError);fails=false;await ui.loadAssets();await ui.save();assert.equal(writes.length,1);
});

test('type refresh preserves the selected trade asset',async()=>{
 const {ui}=harness();await ui.loadPortfolios();await ui.open(null,'buy');await flush();
 await ui.assetAdded({id:'42',ticker:'OLD',typeCode:''});assert.equal(ui.state.assetID,'42');
 await ui.refreshAssets();assert.equal(ui.state.assetID,'42');
});

test('split form sends exact integer strings, defaults, edit values and Enter handler', async () => {
  const {ui,writes,focus}=harness();await ui.loadPortfolios();await ui.open(null,'split');await flush();
  assert.equal(ui.state.date,'2026-10-13');assert.equal(ui.state.oldUnits,'1');assert.equal(ui.state.newUnits,'10');
  await ui.assetAdded({id:'1',ticker:'SBER'});assert.equal(ui.state.assetID,'1');
  ui.splitEnter('1','10000000000000000000000000','1');await ui.save();
  assert.equal(writes[0].oldUnits,'10000000000000000000000000');assert.equal(writes[0].newUnits,'1');
  assert.equal(writes[0].amount,undefined);assert.equal(ui.state.editing,false);assert.equal(focus.at(-1),'trigger');
  await ui.open({id:'9',kind:'split',assetID:'1',date:'2024-02-29',oldUnits:'10',newUnits:'1',comment:'saved'});await flush();
  assert.equal(ui.state.oldUnits,'10');assert.equal(ui.state.newUnits,'1');assert.equal(ui.state.date,'2024-02-29');
  ui.close();assert.equal(writes.length,1);assert.equal(focus.at(-1),'trigger');
});
test('split save errors retain form and values, block duplicates and cancel while busy', async () => {
  let resolve,count=0;
  const {ui,focus}=harness({SaveSplit:()=>{count++;return new Promise(r=>{resolve=r})}});
  await ui.loadPortfolios();await ui.open(null,'split');await flush();ui.splitEnter('1','3','1');
  const pending=ui.save();await ui.save();ui.close();assert.equal(count,1);assert.equal(ui.state.editing,true);
  resolve({errors:{newUnits:'invalid'},message:''});await pending;
  assert.equal(ui.state.oldUnits,'3');assert.equal(ui.state.newUnits,'1');assert.equal(focus.at(-1),'error');
  const again=ui.save();resolve({errors:{},message:'split SBER periodic'});await again;
  assert.equal(ui.state.message,'split SBER periodic');assert.equal(ui.state.editing,true);
});
test('split preview sends portfolio/id/date, discards obsolete, closed and other-portfolio responses', async () => {
  const pending=[];
  const {ui}=harness({PreviewSplit:(pid,id,input)=>new Promise(resolve=>pending.push({pid,id,...input,resolve}))});
  await ui.loadPortfolios();await ui.open({id:'9',kind:'split',assetID:'1',date:'2026-01-02',oldUnits:'1',newUnits:'10'});await flush();
  ui.splitPreview();await new Promise(r=>setTimeout(r,220));assert.equal(pending[0].pid,'1');assert.equal(pending[0].id,'9');assert.equal(pending[0].date,'2026-01-02');
  ui.splitEnter('1','10','1','2026-01-03');ui.splitPreview();await new Promise(r=>setTimeout(r,220));
  pending[1].resolve({quantityBefore:'12',quantityAfter:'1.2',errors:{},message:''});await flush();
  pending[0].resolve({quantityBefore:'old',quantityAfter:'old',errors:{},message:''});await flush();assert.equal(ui.state.quantityAfter,'1.2');
  ui.splitEnter('1','3','1');ui.splitPreview();await new Promise(r=>setTimeout(r,220));
  pending[2].resolve({quantityBefore:'',quantityAfter:'',errors:{},message:'split SBER periodic'});await flush();assert.equal(ui.state.quantityAfter,'');assert.equal(ui.state.splitPreviewError,'split SBER periodic');
  ui.splitEnter('1','1','10');ui.splitPreview();await new Promise(r=>setTimeout(r,220));ui.close();await ui.select('2');
  pending[3].resolve({quantityBefore:'closed',quantityAfter:'closed',errors:{},message:''});await flush();assert.notEqual(ui.state.quantityAfter,'closed');
});
test('split deletion preserves actionable error and confirmation', async () => {
  const {ui}=harness({DeleteCashFlow:async()=>({errors:{},message:'split SBER от 13.10.2026 (3 → 1)'})});
  await ui.loadPortfolios();ui.askDelete({id:'9',kind:'split'});await ui.remove();
  assert.equal(ui.state.deleting.id,'9');assert.match(ui.state.message,/SBER/);ui.closeDelete();assert.equal(ui.state.deleting,null);
});
