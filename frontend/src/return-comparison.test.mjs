import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
import { formatDecimal } from './decimal-format.mjs';

function harness(api = async () => fixture) {
  const source = fs.readFileSync(new URL('./ReturnComparison.svelte', import.meta.url), 'utf8')
    .match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
  let options;
  const sandbox = {
    $props: () => ({ benchmarks: [] }), $state: v => v, $derived: v => v, $effect: () => {},
    onMount: fn => fn(), ResizeObserver: class { observe() {} disconnect() {} },
    echarts: { init: () => ({ setOption: value => { options = value; }, clear(){ options = null; }, resize() {}, dispose() {} }) },
    coordinate: value => Number(value), formatDecimal, localDate: () => '2026-10-16', GetReturnComparison: api,
  };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(`${source}\nglobalThis.h = {select(id){inflationID=id;return refresh()},annual(){metric="annual";render()},get result(){return result},get loading(){return loading},get error(){return error},render,destroy(){alive=false;++request;}};`, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None } }).outputText, sandbox);
  return { h: sandbox.h, options: () => options };
}
const fixture = { baseDate: '2024-12-31', dates: ['2024-12-31', '2025-12-31', '2026-10-16'], series: [
  { id: 'portfolio:1', name: 'Портфель: <Example>', startDate: '2025-01-01', points: [
    { date: '2024-12-31', annual: '', cumulative: '', issue: 'Ещё не сформирован' },
    { date: '2025-12-31', annual: '10.000000000000000001', cumulative: '32.000000000000000001', issue: '' },
    { date: '2026-10-16', annual: '', cumulative: '', issue: 'Нет курса USD → RUB' },
  ] },
] };
test('annual comparison plots cumulative percentages and uses exact strings in the tooltip', async () => {
 const x = harness();await x.h.select('1');x.h.render();const o = x.options();
 assert.equal(o.series[0].data[0], null);assert.equal(o.series[0].data[1],32);assert.equal(o.series[0].data[2],null);
 assert.equal(o.series[0].connectNulls,false);assert.equal(o.tooltip.renderMode,'richText');
 assert.match(o.tooltip.formatter([{seriesIndex:0,dataIndex:1}]),/32,000000000000000001%/);
 assert.match(o.tooltip.formatter([{seriesIndex:0,dataIndex:1}]),/10,000000000000000001%/);
 assert.match(o.tooltip.formatter([{seriesIndex:0,dataIndex:2}]),/Нет курса USD/);
 assert.equal(o.xAxis.data[2],'2026*');assert.equal(o.xAxis.axisLabel.interval,0);
 assert.match(o.tooltip.formatter([{seriesIndex:0,dataIndex:2}]),/2026 \(по 16.10.2026\)/);
 x.h.annual();assert.equal(x.options().series[0].data[1],10);
});
test('late comparison answers cannot replace a new inflation selection', async () => {
 let a,b;let calls=0;const x=harness(()=>new Promise(resolve=>{if(++calls===1)a=resolve;else b=resolve}));
 const first=x.h.select('1'),second=x.h.select('2');a(fixture);await first;
 assert.equal(x.h.result,null);assert.equal(x.h.loading,true);b({...fixture,baseDate:'2025-01-01'});await second;
 assert.equal(x.h.result.baseDate,'2025-01-01');assert.equal(x.h.loading,false);
});
test('comparison clears stale data on failure, can retry, and ignores answers after destruction', async () => {
 let fail=true;const x=harness(async()=>{if(fail)throw Error('Не удалось рассчитать сравнение');return fixture});
 await x.h.select('1');assert.match(x.h.error,/Не удалось/);assert.equal(x.h.result,null);
 fail=false;await x.h.select('1');assert.equal(x.h.result.baseDate,fixture.baseDate);
 let resolve;const y=harness(()=>new Promise(r=>{resolve=r}));const pending=y.h.select('1');y.h.destroy();resolve(fixture);await pending;assert.equal(y.h.result,null);
});
