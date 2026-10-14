import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { parse, compile } from 'svelte/compiler';
import ts from 'typescript';
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return{promise,resolve};};
const allocation=(pid='1')=>({portfolioID:pid,plan:{generation:'g1',revision:'1',cashPercent:'20',targets:[{assetID:'1',percent:'80'}]},assets:[{ID:'1',Ticker:'A'}],rows:[]});
function harness(api={}) {
 const component=readFileSync(new URL('./TargetAllocation.svelte',import.meta.url),'utf8');
 const {instance}=parse(component);
 const script=component.slice(instance.content.start,instance.content.end).replace(/^\s*import .*;$/gm,'').replace('export function canLeave','function canLeave');
 let confirms=true,date='2026-10-14';const calls=[];
 class Plan {constructor(v){Object.assign(this,v);}}
 const sandbox={$state:v=>v,$props:()=>({pid:'1'}),$effect(){},onMount(){},valuation:{AllocationPlan:Plan},localDate:()=>date,formatDecimal:v=>v,formatChartAmount:v=>v,window:{confirm:()=>confirms},
  GetAllocation:async pid=>allocation(pid),SaveAllocation:async(...args)=>{calls.push(args);return{errors:{},message:'',conflict:false};},DeleteAllocation:async()=>({message:''}),PreviewAllocation:async()=>({total:'100',difference:'0',errors:{}}),...api};
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}
 globalThis.h={refresh,save,deletePlan,loadVersion,updatePreview,canLeave,open(){openEditor({currentTarget:{focus(){}}})},select(v){pid=v},set(field,value){draft[field]=value},get draft(){return draft},get data(){return data},get busy(){return busy},get opened(){return opened},get error(){return error},get message(){return message},get conflict(){return conflict},get preview(){return preview},destroy(){alive=false;++request;++previewRequest;}};
 dialog={showModal(){},close(){}};
 `,{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,sandbox);
 return {h:sandbox.h,calls,confirm:v=>confirms=v,date:v=>date=v};
}
test('target components compile without diagnostics',()=>{
 for(const file of ['TargetAllocation.svelte','TargetPie.svelte']){
  const output=compile(readFileSync(new URL(`./${file}`,import.meta.url),'utf8'),{filename:file,generate:'client'});
  assert.deepEqual(output.warnings,[]);
 }
});
test('target read discards stale portfolio and calendar responses',async()=>{
 const first=deferred(),second=deferred();let count=0;
 const {h}=harness({GetAllocation:()=>++count===1?first.promise:second.promise});
 const a=h.refresh();h.select('2');const b=h.refresh();first.resolve(allocation('1'));await a;assert.equal(h.data,null);
 second.resolve(allocation('2'));await b;assert.equal(h.data.portfolioID,'2');
 const late=deferred();const day=harness({GetAllocation:()=>late.promise});const pending=day.h.refresh();day.date('2026-10-15');late.resolve(allocation());await pending;assert.equal(day.h.data,null);
});
test('read failure clears previous target data and supports retry',async()=>{
 let fail=false;const {h}=harness({GetAllocation:async()=>{if(fail)throw Error('private SQL');return allocation();}});
 await h.refresh();fail=true;await h.refresh();assert.equal(h.data,null);assert.match(h.error,/Не удалось/);assert.doesNotMatch(h.error,/private/);
 fail=false;await h.refresh();assert.equal(h.error,'');assert.equal(h.data.plan.generation,'g1');
});
test('saving is single-flight; failure preserves entered shares',async()=>{
 const pending=deferred();let count=0;const {h}=harness({SaveAllocation:()=>{count++;return pending.promise;}});
 await h.refresh();h.open();h.set('cashPercent','25');const save=h.save();await h.save();assert.equal(count,1);assert.equal(h.busy,true);assert.equal(h.canLeave(),false);
 pending.resolve({errors:{total:'Сумма должна быть 100%'},message:'',conflict:false});await save;assert.equal(h.opened,true);assert.equal(h.draft.cashPercent,'25');assert.equal(h.busy,false);
});
test('conflict refresh preserves user shares and obtains new version explicitly',async()=>{
 const {h}=harness({SaveAllocation:async()=>({errors:{},message:'Конфликт',conflict:true})});
 await h.refresh();h.open();h.set('cashPercent','25');await h.save();assert.equal(h.conflict,true);assert.equal(h.draft.cashPercent,'25');await h.loadVersion();assert.equal(h.conflict,false);assert.equal(h.draft.cashPercent,'25');assert.equal(h.draft.generation,'g1');
});
test('cancel and switching portfolios require choosing to discard dirty input',async()=>{
 const {h,confirm}=harness();await h.refresh();h.open();h.set('cashPercent','25');confirm(false);assert.equal(h.canLeave(),false);assert.equal(h.opened,true);confirm(true);assert.equal(h.canLeave(),true);assert.equal(h.opened,false);
});
test('saved plan survives a failed refresh and reports read failure separately',async()=>{
 let reads=0;const {h,calls}=harness({GetAllocation:async()=>{if(++reads>1)throw Error('read failed');return allocation();}});
 await h.refresh();h.open();await h.save();assert.equal(calls.length,1);assert.equal(h.opened,false);assert.match(h.error,/Не удалось загрузить/);assert.equal(h.message,'');
});
