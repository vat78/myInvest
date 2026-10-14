import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
function harness(getOverview) {
  const component = readFileSync(new URL('./App.svelte', import.meta.url), 'utf8');
  const { instance } = parse(component);
  const script = component.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '');
  let date = '2026-10-14';
  const sandbox = { $state: v => v, $derived: v => v, onMount() {}, formatDecimal: v => v, localDate: () => date, GetOverview: getOverview,
    document: { getElementById: id => ({ focus() { sandbox.focused = id; } }) } };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(`${script}
    globalThis.overviewTest = { refresh: refreshOverview, keys: tabKeys, get data() { return overview; }, get loading() { return overviewLoading; }, get error() { return overviewError; }, get tab() { return tab; }, destroy() { alive=false; ++overviewRequest; } };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return { h: sandbox.overviewTest, setDate: v => date = v, sandbox };
}
const deferred = () => { let resolve; const promise = new Promise(r => resolve = r); return { promise, resolve }; };
test('overview ignores older responses and cannot finish a newer loading state', async () => {
  const first = deferred(), second = deferred(); let calls = 0;
  const { h } = harness(() => ++calls === 1 ? first.promise : second.promise);
  const a = h.refresh(), b = h.refresh(); first.resolve({date: 'old'}); await a;
  assert.equal(h.loading, true); assert.equal(h.data, null);
  second.resolve({date: 'new'}); await b; assert.equal(h.data.date, 'new'); assert.equal(h.loading, false);
});
test('failed overview retains prior data with an error, and retries successfully', async () => {
  let fail = false; const { h } = harness(async () => { if (fail) throw new Error('SQL private'); return { date: '2026-10-14' }; });
  await h.refresh(); fail = true; await h.refresh(); assert.equal(h.data.date, '2026-10-14'); assert.match(h.error, /устарели/); assert.doesNotMatch(h.error, /private/);
  fail = false; await h.refresh(); assert.equal(h.error, '');
});
test('response for a previous local date or destroyed screen is discarded', async () => {
  const pending = deferred(); const { h, setDate } = harness(() => pending.promise);
  const request = h.refresh(); setDate('2026-10-15'); pending.resolve({date:'2026-10-14'}); await request; assert.equal(h.data, null);
  const next = deferred(); const other = harness(() => next.promise); const closed = other.h.refresh(); other.h.destroy(); next.resolve({}); await closed; assert.equal(other.h.data,null);
});
test('keyboard tab switching chooses the matching panel and restores tab focus', () => {
  const {h,sandbox} = harness(async () => ({})); assert.equal(h.tab, 'overview');
  h.keys({key:'ArrowRight',preventDefault(){}}); assert.equal(h.tab,'operations'); assert.equal(sandbox.focused,'tab-operations');
  h.keys({key:'Home',preventDefault(){}}); assert.equal(h.tab,'overview'); assert.equal(sandbox.focused,'tab-overview');
});
