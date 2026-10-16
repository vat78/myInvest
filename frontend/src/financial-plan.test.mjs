import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import {parse} from 'svelte/compiler';
import ts from 'typescript';
function harness(overrides={}){
 const text=readFileSync(new URL('./FinancialPlan.svelte',import.meta.url),'utf8');
 const {instance}=parse(text);const script=text.slice(instance.content.start,instance.content.end).replace(/^\s*import .*;$/gm,'');
 const sandbox={$state:v=>v,$derived:v=>v,$effect(){},$props:()=>({}),onMount(){},untrack:f=>f(),localDate:()=> '2026-10-14',ListCurrencies:async()=>['RUB','USD'],ListBenchmarks:async()=>[],GetCapitalHistory:async()=>[],GetFinancialPlan:async()=>({plan:{found:false},errors:{},message:''}),...overrides};
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}\nglobalThis.h={load,calculate,changed,refreshLists,select(id){currencyCode=id;draft=defaults(id)},edit(p){Object.assign(draft,p)},get draft(){return draft},get plan(){return plan},get preview(){return preview},get errors(){return errors},get message(){return message},get feedback(){return feedback},get actual(){return actual},get actualError(){return actualError},get actualLoading(){return actualLoading},refreshActual,get busy(){return busy},get benchmarks(){return benchmarks},destroy(){alive=false;++request;++listRequest;}}`,{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,sandbox);
 return sandbox.h;
}
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return{promise,resolve}};
const saved={found:true,parameters:{currencyCode:'RUB',revision:'1',initialCapital:'12.1234567890123456789'},current:{firstContribution:'100'},original:{firstContribution:'90'}};
test('save passes decimal strings unchanged and retains draft on validation failure',async()=>{
 let input;const h=harness({SaveFinancialPlan:async p=>{input=p;return{errors:{contribution:'invalid'},message:''}}});h.select('RUB');h.edit({contribution:'0.123456789012345678900'});await h.calculate(true);
 assert.equal(input.contribution,'0.123456789012345678900');assert.equal(h.draft.contribution,input.contribution);assert.equal(h.errors.contribution,'invalid');
});
test('preview does not replace saved original; refresh preserves unsaved parameters',async()=>{
 const h=harness({GetFinancialPlan:async()=>({plan:saved,errors:{},message:''}),PreviewFinancialPlan:async()=>({plan:{current:{firstContribution:'200',points:[{date:'2026-01-01'},{date:'2027-01-01'}]}},errors:{},message:''})});h.select('RUB');await h.load(true);h.edit({goal:'123456789'});await h.calculate(false);assert.equal(h.preview.firstContribution,'200');assert.equal(h.plan.original.firstContribution,'90');await h.load(false);assert.equal(h.draft.goal,'123456789');assert.equal(h.plan.original.firstContribution,'90');
});
test('pending save blocks duplicate submissions; success establishes new revision',async()=>{
 const p=deferred();let calls=0;const h=harness({SaveFinancialPlan:()=>{calls++;return p.promise}});h.select('RUB');const save=h.calculate(true);await h.calculate(true);assert.equal(calls,1);p.resolve({plan:saved,errors:{},message:''});await save;assert.equal(h.draft.revision,'1');assert.match(h.feedback,/сохранён/);assert.equal(h.busy,false);
});
test('old currency response cannot replace selected plan',async()=>{
 const a=deferred(),b=deferred();let calls=0;const h=harness({GetFinancialPlan:()=>++calls===1?a.promise:b.promise});h.select('RUB');const first=h.load(true);h.select('USD');const second=h.load(true);a.resolve({plan:saved,errors:{},message:''});await first;assert.equal(h.plan,null);b.resolve({plan:{found:false},errors:{},message:''});await second;assert.equal(h.draft.currencyCode,'USD');assert.equal(h.plan,null);
});
test('failed saved forecast refresh is explicit and retains original',async()=>{
 let fail=false;const h=harness({GetFinancialPlan:async()=>{if(fail)throw Error('private SQL');return{plan:saved,errors:{},message:''}}});h.select('RUB');await h.load(true);fail=true;await h.load(false);assert.equal(h.plan.original.firstContribution,'90');assert.match(h.message,/загрузить/);assert.doesNotMatch(h.message,/private/);
});
test('benchmarks with and without currency can be selected; destroyed component ignores results',async()=>{
 const h=harness({ListBenchmarks:async()=>[{id:'1',currencyCode:''},{id:'2',currencyCode:'RUB'}]});await h.refreshLists();assert.equal(h.benchmarks.length,2);assert.equal(h.benchmarks[0].id,'1');
 const d=deferred();const gone=harness({GetFinancialPlan:()=>d.promise});gone.select('RUB');const loading=gone.load(true);gone.destroy();d.resolve({plan:saved,errors:{},message:''});await loading;assert.equal(gone.plan,null);
});

const forecast={current:{firstContribution:'200',points:[{date:'2026-01-01'},{date:'2027-01-01'}]}};
const capital=[{date:'2026-01-01',value:'123.456789012345678',complete:true,issues:[]}];
test('preview loads all portfolios in plan currency and stops actual history today',async()=>{
 let args;const h=harness({PreviewFinancialPlan:async()=>({plan:forecast,errors:{},message:''}),GetCapitalHistory:async(...a)=>{args=a;return capital}});
 h.select('USD');await h.calculate(false);await new Promise(r=>setImmediate(r));
 assert.deepEqual(args,['USD','2026-01-01','2026-10-14']);assert.equal(h.actual[0].value,'123.456789012345678');
 assert.equal(h.preview.firstContribution,'200');assert.equal(h.actualLoading,false);
});
test('actual read failure retains forecast and can retry without saving',async()=>{
 let fail=true;const h=harness({PreviewFinancialPlan:async()=>({plan:forecast,errors:{},message:''}),GetCapitalHistory:async()=>{if(fail)throw Error('private SQL');return capital}});
 h.select('RUB');await h.calculate(false);await new Promise(r=>setImmediate(r));
 assert.equal(h.preview.firstContribution,'200');assert.match(h.actualError,/загрузить/);assert.doesNotMatch(h.actualError,/private/);
 fail=false;await h.refreshActual(forecast.current.points);assert.equal(h.actual.length,1);assert.equal(h.actualError,'');
});
test('editing parameters discards an outdated portfolio history response',async()=>{
 const pending=deferred();const h=harness({PreviewFinancialPlan:async()=>({plan:forecast,errors:{},message:''}),GetCapitalHistory:()=>pending.promise});
 h.select('RUB');await h.calculate(false);assert.equal(h.actualLoading,true);h.changed();pending.resolve(capital);await new Promise(r=>setImmediate(r));
 assert.equal(h.actual.length,0);assert.equal(h.actualLoading,false);assert.equal(h.preview,null);
});
test('actual history is clipped to terminal plan date and is not stored on save',async()=>{
 let args,calls=0;const h=harness({GetCapitalHistory:async(...a)=>{args=a;calls++;return capital},SaveFinancialPlan:async()=>({plan:saved,errors:{},message:''})});
 h.select('RUB');await h.refreshActual([{date:'2024-01-01'},{date:'2025-01-01'}]);assert.deepEqual(args,['RUB','2024-01-01','2025-01-01']);
 await h.calculate(true);assert.equal(calls,1);assert.equal(h.actual.length,0);
});
