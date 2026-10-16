import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
import { formatChartAmount } from './decimal-format.mjs';

test('plan chart places annual plan and daily actuals on one proportional time axis', () => {
  const source = fs.readFileSync(new URL('./PlanChart.svelte', import.meta.url), 'utf8')
    .match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
  let options;
  const sandbox = {
    $props: () => ({ currency: 'USD', points: [{date:'2026-01-01',nominal:'100',real:'90'}, {date:'2027-01-01',nominal:'200',real:'180'}], actual: [
      {date:'2026-01-01',value:'123.456789012345678',complete:true,issues:[]},
      {date:'2026-01-02',value:'',complete:false,issues:['Нет курса RUB → USD']},
      {date:'2026-01-03',value:'-1.25',complete:true,issues:[]},
    ] }),
    $effect: () => {}, onMount: fn => fn(), ResizeObserver: class { observe(){} disconnect(){} },
    echarts: { init: () => ({ setOption: o => {options=o}, resize(){},dispose(){} }) },
    coordinate: value => Number(value), formatChartAmount,
  };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(source, {compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText, sandbox);
  assert.equal(options.xAxis.type,'time');assert.equal(options.useUTC,true);
  assert.equal(options.series.length,3);assert.equal(options.series[2].connectNulls,false);
  assert.equal(options.series[0].data[1][0]-options.series[0].data[0][0],365*86400000);
  assert.equal(options.series[2].data[1][0]-options.series[2].data[0][0],86400000);
  assert.equal(options.series[2].data[1][1],null);assert.equal(options.series[2].data[2][1],-1.25);
  assert.match(options.tooltip.formatter([{seriesIndex:2,dataIndex:0}]),/123,46 USD/);
  assert.match(options.tooltip.formatter([{seriesIndex:2,dataIndex:1}]),/Нет курса RUB → USD/);
  assert.match(options.tooltip.formatter([{seriesIndex:1,dataIndex:1}]),/2027-01-01/);
});
