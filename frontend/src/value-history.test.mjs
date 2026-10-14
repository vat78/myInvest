import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';
import { formatChartAmount } from './decimal-format.mjs';

function chartOptions(points) {
  const source = fs.readFileSync(new URL('./ValueHistory.svelte', import.meta.url), 'utf8')
    .match(/<script lang="ts">([\s\S]*?)<\/script>/)[1].replace(/^\s*import .*;$/gm, '');
  let options;
  const sandbox = {
    $props: () => ({ points }), $effect: () => {},
    onMount: fn => fn(), ResizeObserver: class { observe() {} disconnect() {} },
    echarts: { init: () => ({ setOption: value => { options = value; }, resize() {}, dispose() {} }) },
    coordinate: value => Number(value), formatChartAmount, Intl,
  };
  vm.runInNewContext(ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.None } }).outputText, sandbox);
  return options;
}

test('deposit series stays visible when portfolio valuation is incomplete, with exact labels', () => {
  const options = chartOptions([
    { date: '2026-01-01', value: '', complete: false, issues: ['Нет курса'], deposits: '94500.125', depositsComplete: true, depositIssues: [] },
    { date: '2026-01-02', value: '200000', complete: true, issues: [], deposits: '', depositsComplete: false, depositIssues: ['Нет суммы RUB'] },
  ]);
  assert.equal(options.series.length, 2);
  assert.equal(options.series[0].data[0], null);
  assert.equal(options.series[1].data[0], 94500.125);
  assert.equal(options.series[1].data[1], null);
  assert.equal(options.series[1].connectNulls, false);
  assert.equal(options.series[1].step, 'end');
  assert.match(options.tooltip.formatter([{ dataIndex: 0 }]), /Пополнения накопительным итогом: 94 500,13 RUB/);
  assert.match(options.tooltip.formatter([{ dataIndex: 1 }]), /Нет суммы RUB/);
});
