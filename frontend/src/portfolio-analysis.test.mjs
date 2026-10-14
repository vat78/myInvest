import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
function harness(list, analysis) {
  const component = readFileSync(new URL('./PortfolioAnalysis.svelte', import.meta.url), 'utf8');
  const { instance } = parse(component);
  const script = component.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '');
  let date = '2026-10-14';
  const sandbox = { $state: v => v, $props: () => ({}), $effect() {}, onMount() {}, localDate: () => date, ListPortfolios: list, GetPortfolioAnalysis: analysis };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(`${script}
    globalThis.h = { refresh, select(v) { selected=v; }, get selected() { return selected; }, get data() { return data; }, get loading() { return loading; }, get error() { return error; }, get portfolios() { return portfolios; }, destroy() { alive=false; ++request; } };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return { h: sandbox.h, setDate: v => date = v };
}
const list = async () => [{ id:'1', name:'Первый', currencyCode:'RUB' }, { id:'2', name:'Второй', currencyCode:'USD' }];
const deferred = () => { let resolve; const promise = new Promise(r => resolve=r); return {promise,resolve}; };
test('analysis selects the first portfolio and preserves selection on refresh', async () => {
  const {h}=harness(list, async (pid,date) => ({portfolioID:pid,date}));
  await h.refresh(); assert.equal(h.selected,'1'); assert.equal(h.data.portfolioID,'1');
  h.select('2'); await h.refresh(); await h.refresh(); assert.equal(h.selected,'2'); assert.equal(h.data.portfolioID,'2');
});
test('late portfolio response does not overwrite a new selection or finish loading', async () => {
  const a=deferred(),b=deferred(), firstStarted=deferred(), secondStarted=deferred(); let calls=0;
  const {h}=harness(list, () => { if (++calls===1) { firstStarted.resolve(); return a.promise; } secondStarted.resolve(); return b.promise; });
  const first=h.refresh(); await firstStarted.promise; h.select('2'); const second=h.refresh(); await secondStarted.promise;
  a.resolve({portfolioID:'1'}); await first; assert.equal(h.data,null); assert.equal(h.loading,true);
  b.resolve({portfolioID:'2'}); await second; assert.equal(h.data.portfolioID,'2'); assert.equal(h.loading,false);
});
test('analysis read errors clear previous data and retry; empty portfolios do not call analysis', async () => {
  let fail=false;
  const {h}=harness(list, async pid => { if(fail) throw new Error('private SQL'); return {portfolioID:pid}; });
  await h.refresh(); fail=true; await h.refresh(); assert.equal(h.data,null); assert.match(h.error,/Не удалось/); assert.doesNotMatch(h.error,/private/);
  fail=false; await h.refresh(); assert.equal(h.error,''); assert.equal(h.data.portfolioID,'1');
  const empty=harness(async()=>[],()=>{throw new Error('unexpected');}).h; await empty.refresh(); assert.equal(empty.selected,''); assert.equal(empty.error,''); assert.equal(empty.loading,false);
});
test('analysis discards responses after date change and destruction', async () => {
  const a=deferred(); const {h,setDate}=harness(list,()=>a.promise);
  const first=h.refresh(); await Promise.resolve(); setDate('2026-10-15'); a.resolve({}); await first; assert.equal(h.data,null);
  const b=deferred();const second=harness(list,()=>b.promise).h;
  const pending=second.refresh();await Promise.resolve();second.destroy();b.resolve({});await pending;assert.equal(second.data,null);
});
