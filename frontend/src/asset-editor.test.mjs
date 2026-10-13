import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import {parse} from 'svelte/compiler';
import ts from 'typescript';
const flush=async()=>{for(let i=0;i<8;i++)await Promise.resolve()};
function harness(overrides={}) {
 const source=readFileSync(new URL('./AssetManager.svelte',import.meta.url),'utf8');
 const {instance}=parse(source);
 const script=source.slice(instance.content.start,instance.content.end).replace(/^\s*import .*;$/gm,'').replace(/export /g,'');
 const added=[],focus=[],writes=[];
 const sandbox={
  $props:()=>({onadded:a=>added.push(a)}),$state:v=>v,onDestroy(){},tick:async()=>{},
  ListAssets:async()=>[],AddAsset:async ticker=>{writes.push(ticker);return {asset:{id:'1',ticker:'SBER'},errors:{},message:''}},
  HTMLElement:class {},document:{activeElement:null},focus,...overrides,
 };
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}
  dialog={showModal(){},close(){}};input={focus(){focus.push('input')}};
  const trigger={focus(){focus.push('trigger')}};
  globalThis.ui={open:()=>openList(trigger),close,load,enter:v=>ticker=v,add:()=>add({preventDefault(){}}),
   get state(){return {ticker,opened,loading,busy,assets,error,fieldError,message}}};
 `,{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,sandbox);
 return {ui:sandbox.ui,added,focus,writes};
}
test('asset creation refreshes the directory, selects the normalized asset and restores focus',async()=>{
 const {ui,added,focus,writes}=harness();await ui.open();await flush();ui.enter(' sber ');await ui.add();
 assert.equal(writes[0],' sber ');assert.equal(added[0].ticker,'SBER');assert.equal(ui.state.ticker,'');assert.equal(ui.state.message,'Добавлен SBER');
 ui.close();assert.equal(ui.state.opened,false);assert.equal(focus.at(-1),'trigger');
});
test('duplicate or failed save preserves ticker and prevents duplicate writes or cancellation',async()=>{
 let resolve,count=0;
 const {ui,added}=harness({AddAsset:()=>{count++;return new Promise(r=>{resolve=r})}});
 await ui.open();await flush();ui.enter('sber');const saving=ui.add();await ui.add();ui.close();
 assert.equal(count,1);assert.equal(ui.state.opened,true);assert.equal(ui.state.busy,true);
 resolve({asset:{},errors:{ticker:'duplicate'},message:''});await saving;
 assert.equal(ui.state.ticker,'sber');assert.equal(ui.state.fieldError,'duplicate');assert.equal(added.length,0);
});
test('asset read failure retries and ignores late answers from closed dialogs',async()=>{
 let fails=true,resolve;
 const {ui}=harness({ListAssets:async()=>{if(fails)throw new Error('SQL');return new Promise(r=>{resolve=r})}});
 await ui.open();await flush();assert.ok(ui.state.error);assert.equal(ui.state.assets.length,0);
 fails=false;const reading=ui.load();ui.close();resolve([{id:'1',ticker:'late'}]);await reading;assert.equal(ui.state.assets.length,0);
});
