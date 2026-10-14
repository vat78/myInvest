import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
function harness(overrides = {}, props = {}) {
  const component = readFileSync(new URL('./QuoteManager.svelte', import.meta.url), 'utf8');
  const { instance } = parse(component);
  const script = component.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '');
  const sandbox = { $state: v => v, $effect() {}, $props: () => ({ onrate() {}, onaddasset() {}, ...props }), onMount() {}, localDate: () => '2026-10-14',
    ListQuotes: async () => [], ListCurrencyRates: async () => [], ListAssets: async () => [], ListCurrencies: async () => ['RUB','USD'], SaveQuote: async () => ({ errors: {}, message: '' }),
    document: { getElementById: () => ({ focus() {} }) }, ...overrides };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(`${script}
    globalThis.h = { refresh, edit, reset, save: () => save({preventDefault(){}}),
      get quotes() { return quotes; }, get price() { return price; }, get editing() { return editing; },
      get feedback() { return feedback; }, get error() { return loadError; }, get errors() { return errors; }, get busy() { return busy; }, get loading() { return loading; },
      destroy() { alive = false; ++request; } };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return sandbox.h;
}
const deferred = () => { let resolve; const promise = new Promise(r => resolve = r); return { promise, resolve }; };
const quote = { assetID: '1', date: '2026-10-14', currencyCode: 'USD', price: '0.1234567890123456789012345678900' };
test('quote edit passes exact strings and failed field validation retains input', async () => {
  let input;
  const h = harness({ SaveQuote: async q => { input=q; return { errors: { price: 'invalid' }, message: '' }; } });
  h.edit(quote); await h.save(); assert.equal(input.price, quote.price); assert.equal(input.assetID, '1');
  assert.equal(h.price, quote.price); assert.equal(h.errors.price, 'invalid'); assert.equal(h.feedback, '');
});
test('successful save remains successful if subsequent refresh fails', async () => {
  const h = harness({ ListQuotes: async () => { throw new Error('SQL private'); } });
  h.edit(quote); await h.save(); assert.equal(h.feedback, 'Котировка сохранена');
  assert.match(h.error, /устаревшими/); assert.doesNotMatch(h.error, /private/); assert.equal(h.price, quote.price);
});
test('repeat save and resetting while save is pending are blocked', async () => {
  const pending=deferred(); let calls=0;
  const h=harness({ SaveQuote: () => { calls++; return pending.promise; } });
  h.edit(quote); const saving=h.save(); await h.save(); h.reset();
  assert.equal(calls,1); assert.equal(h.busy,true); assert.equal(h.price,quote.price);
  pending.resolve({errors:{},message:''}); await saving; assert.equal(h.busy,false);
});
test('stale refresh cannot overwrite new data or finish newer loading state', async () => {
  const a=deferred(),b=deferred(); let calls=0;
  const h=harness({ ListQuotes: () => (++calls===1 ? a.promise : b.promise) });
  const first=h.refresh(),second=h.refresh(); a.resolve([{...quote,price:'old'}]); await first;
  assert.equal(h.quotes.length,0); assert.equal(h.loading,true);
  b.resolve([quote]); await second; assert.equal(h.quotes[0].price,quote.price); assert.equal(h.loading,false);
});
test('refresh preserves draft and destroyed component discards responses', async () => {
  const pending=deferred(); const h=harness({ ListQuotes: () => pending.promise });
  h.edit(quote); const loading=h.refresh(); h.destroy(); pending.resolve([quote]); await loading;
  assert.equal(h.quotes.length,0); assert.equal(h.price,quote.price);
});

test('a successful quote save refreshes valuation screens, validation failure does not', async () => {
 let changed=0;const h=harness({}, {onchanged(){changed++;}});h.edit(quote);await h.save();assert.equal(changed,1);
 const invalid=harness({SaveQuote:async()=>({errors:{price:'invalid'},message:''})},{onchanged(){changed++;}});invalid.edit(quote);await invalid.save();assert.equal(changed,1);
});
