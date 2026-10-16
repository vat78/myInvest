import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
function harness(overrides = {}) {
 const text = readFileSync(new URL('./BenchmarkDirectory.svelte', import.meta.url), 'utf8');
 const { instance } = parse(text);
 const script = text.slice(instance.content.start, instance.content.end).replace(/^\s*import .*;$/gm, '').replace('export async function', 'async function');
 const sandbox = { $state: v => v, tick: async () => {}, onDestroy() {}, ListBenchmarks: async () => [{id:'1',name:'Индекс',currencyCode:'',sources:'https://example.com'}], SaveBenchmarkSources: async () => ({errors:{},message:''}), ...overrides };
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}
 dialog={showModal(){},close(){}}; sourceInput={focus(){}};
 globalThis.h={open:()=>openList({focus(){}}),load,close,edit:async()=>edit(rows[0],{focus(){}}),save:()=>save({preventDefault(){}}),cancel,
 set sources(v){sources=v},get sources(){return sources},get rows(){return rows},get error(){return error},get fieldError(){return fieldError},get editing(){return editing},get busy(){return busy},get message(){return message}};`, {compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText, sandbox);
 return sandbox.h;
}
test('directory edits multiple source links and can clear them', async () => {
 let saved; const h = harness({SaveBenchmarkSources:async (...args)=>{saved=args;return {errors:{},message:''}}});
 await h.open(); await h.load(); await h.edit(); assert.equal(h.sources,'https://example.com');
 h.sources='https://one.example\nhttps://two.example'; await h.save();
 assert.deepEqual(saved,['1','https://one.example\nhttps://two.example']); assert.equal(h.rows[0].sources,h.sources); assert.match(h.message,/сохранены/);
 await h.edit(); h.sources=''; await h.save(); assert.equal(h.rows[0].sources,'');
});
test('failed source save retains draft and prevents duplicate submissions and cancellation', async () => {
 let resolve, calls=0;const pending = new Promise(r=>resolve=r);
 const h=harness({SaveBenchmarkSources:()=>{calls++;return pending}}); await h.open();await h.load();await h.edit(); h.sources='invalid';
 const saving=h.save();await h.save();await h.cancel();h.close();assert.equal(calls,1);assert.equal(h.editing,'1');
 resolve({errors:{sources:'invalid URL'},message:''});await saving;assert.equal(h.sources,'invalid');assert.equal(h.fieldError,'invalid URL');assert.equal(h.busy,false);
});
test('closed directory ignores late reads and supports retry after failure', async () => {
 let resolve; let fail=true;
 const h=harness({ListBenchmarks:()=> fail ? Promise.reject(Error('private')) : new Promise(r=>resolve=r)});
 await h.open();await h.load();assert.match(h.error,/загрузить/);fail=false;
 const loading=h.load();h.close();resolve([{id:'late'}]);await loading;assert.equal(h.rows.length,0);
});
