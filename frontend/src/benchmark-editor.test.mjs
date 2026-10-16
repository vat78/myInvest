import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
function harness(overrides = {}) {
 const text = readFileSync(new URL('./BenchmarkManager.svelte', import.meta.url), 'utf8');
 const {instance} = parse(text);
 const script = text.slice(instance.content.start,instance.content.end).replace(/^\s*import .*;$/gm,'');
 const sandbox = {$props:()=>({}),$state:v=>v,$derived:v=>v,$effect(){},onMount(){},localDate:()=> '2026-10-14',
  ListBenchmarks:async()=>[],ListCurrencies:async()=>['RUB'],BenchmarkHistory:async()=>[],SaveBenchmarkPoint:async()=>({errors:{},message:''}),
  CreateBenchmark:async()=>({benchmark:{id:'1',name:'Inflation',currencyCode:''},errors:{},message:''}),document:{getElementById:()=>({focus(){}})},...overrides};
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}\nglobalThis.h = {edit,refreshHistory,refreshList, reset, save:()=>save({preventDefault(){}}),create:()=>create({preventDefault(){}}),
 select(id){benchmarkID=id;selectBenchmark();}, get input(){return input},get id(){return benchmarkID},get points(){return points},get errors(){return errors},get feedback(){return feedback},get error(){return loadError},get busy(){return busy},get loading(){return loading},get listError(){return listError},get listLoading(){return listLoading},get benchmarks(){return benchmarks},destroy(){alive=false;++request;++listRequest;++previewRequest;}};`,{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,sandbox);
 return sandbox.h;
}
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return {promise,resolve}};
const point={date:'2026-01-01',mode:'growth',input:'-0.123456789012345678900'};
test('edit preserves exact entered mode and value, field errors retain draft', async()=>{
 let saved;const h=harness({SaveBenchmarkPoint:async p=>{saved=p;return {errors:{input:'invalid'},message:''}}});h.select('1');h.edit(point);await h.save();assert.equal(saved.input,point.input);assert.equal(saved.mode,'growth');assert.equal(saved.benchmarkID,'1');assert.equal(h.input,point.input);assert.equal(h.errors.input,'invalid');
});
test('failed refresh does not hide successful save',async()=>{const h=harness({BenchmarkHistory:async()=>{throw Error('private')}});h.select('1');h.edit(point);await h.save();assert.match(h.feedback,/сохранены/);assert.match(h.error,/обновить/);assert.doesNotMatch(h.error,/private/)});
test('pending save blocks duplicates and draft reset',async()=>{const p=deferred();let calls=0;const h=harness({SaveBenchmarkPoint:()=>{calls++;return p.promise}});h.select('1');h.edit(point);const saving=h.save();await h.save();h.reset();assert.equal(calls,1);assert.equal(h.input,point.input);p.resolve({errors:{},message:''});await saving;assert.equal(h.busy,false)});
test('old benchmark response cannot replace selected history',async()=>{const a=deferred(),b=deferred();let calls=0;const h=harness({BenchmarkHistory:()=>++calls===1?a.promise:b.promise});h.select('1');const first=h.refreshHistory();h.select('2');const second=h.refreshHistory();a.resolve([point]);await first;assert.equal(h.points.length,0);assert.equal(h.loading,true);b.resolve([{...point,input:'2'}]);await second;assert.equal(h.points[0].input,'2')});
test('new benchmark remains selected when directory refresh fails',async()=>{const h=harness({ListBenchmarks:async()=>{throw Error('private')}});await h.create();assert.equal(h.id,'1');assert.equal(h.benchmarks[0].name,'Inflation');assert.match(h.listError,/загрузить/);assert.equal(h.listLoading,false)});
test('destroyed editor ignores history and directory responses',async()=>{const p=deferred();const h=harness({ListBenchmarks:()=>p.promise});const loading=h.refreshList();h.destroy();p.resolve([{id:'1'}]);await loading;assert.equal(h.benchmarks.length,0)});
