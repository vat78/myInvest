import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';

// Exercise the component's handlers with a mocked Wails bridge, without a DOM.
function editorHarness(addCurrency = async code => code) {
  const component = readFileSync(new URL('./App.svelte', import.meta.url), 'utf8');
  const { instance } = parse(component);
  const script = component.slice(instance.content.start, instance.content.end)
    .replace(/^\s*import .*;$/gm, '');
  const saved = [];
  const sandbox = {
    $state: value => value,
    $derived: value => value,
    onMount: () => {},
    localDate: () => '2026-10-13',
    AddCurrency: addCurrency,
    GetCurrencyRate: async () => ({ found: false, value: '' }),
    SaveCurrencyRate: async input => { saved.push(input); },
  };
  vm.createContext(sandbox);
  vm.runInContext(ts.transpileModule(`${script}
    dialog = { showModal() {}, close() {} };
    currencies = ['RUB', 'USD'];
    globalThis.editor = {
      open: openEditor,
      async selectPair(first, second) { base = first; quote = second; await loadRate(); },
      enterValue(input) { value = input; },
      add: addCurrency,
      save() { return save({ preventDefault() {} }); },
      get value() { return value; },
      get error() { return error; },
      get currencies() { return currencies; },
    };
  `, { compilerOptions: { target: ts.ScriptTarget.ES2022 } }).outputText, sandbox);
  return { editor: sandbox.editor, saved };
}

test('adding both currencies preserves the exact entered rate through saving', async () => {
  const { editor, saved } = editorHarness();
  editor.open();
  await editor.selectPair('EUR', 'GBP');
  const input = '104,1234567890123456789000';
  editor.enterValue(input);
  await editor.add('EUR');
  assert.equal(editor.value, input);
  await editor.add('GBP');
  assert.equal(editor.value, input);
  assert.ok(editor.currencies.includes('EUR') && editor.currencies.includes('GBP'));
  await editor.save();
  assert.equal(saved.length, 1);
  assert.equal(saved[0].base, 'EUR');
  assert.equal(saved[0].quote, 'GBP');
  assert.equal(saved[0].value, input);
});

test('failed currency addition preserves the entered rate and reports the error', async () => {
  const { editor } = editorHarness(async () => { throw new Error('storage unavailable'); });
  editor.open();
  await editor.selectPair('EUR', 'RUB');
  editor.enterValue('104,50');
  await editor.add('EUR');
  assert.equal(editor.value, '104,50');
  assert.match(editor.error, /storage unavailable/);
  assert.ok(!editor.currencies.includes('EUR'));
});
