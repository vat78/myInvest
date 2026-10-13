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
 const added=[],focus=[],writes=[],changes=[];
 const sandbox={
  $props:()=>({onadded:a=>added.push(a),onchanged:()=>changes.push(true)}),$state:v=>v,onDestroy(){},tick:async()=>{},
  ListAssets:async()=>[],ListAssetTypes:async()=>[{code:'stocks',name:'Stocks'}],AddAsset:async input=>{writes.push(input);return {asset:{id:'1',ticker:'SBER',typeCode:input.typeCode},errors:{},message:''}},UpdateAssetType:async input=>{writes.push(input);return {asset:{id:input.id,ticker:'OLD',typeCode:input.typeCode},errors:{},message:''}},
  HTMLElement:class {},document:{activeElement:null},focus,...overrides,
 };
 vm.createContext(sandbox);
 vm.runInContext(ts.transpileModule(`${script}
  dialog={showModal(){},close(){},querySelector(){return {focus(){focus.push('replacement')}}}};input={focus(){focus.push('input')}};typeSelect={focus(){focus.push('type')}};
  const trigger={focus(){focus.push('trigger')}};
  globalThis.ui={open:()=>openList(trigger),close,load,enter:(v,code='stocks')=>{ticker=v;typeCode=code},edit:a=>edit(a,trigger),cancelEdit,add:()=>add({preventDefault(){}}),
   get state(){return {ticker,opened,loading,busy,assets,types,typeCode,editing,error,fieldError,typeError,message}}};
 `,{compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,sandbox);
 return {ui:sandbox.ui,added,focus,writes,changes};
}
test('asset creation refreshes the directory, selects the normalized asset and restores focus',async()=>{
 const {ui,added,focus,writes}=harness();await ui.open();await flush();ui.enter(' sber ');await ui.add();
 assert.equal(writes[0].ticker,' sber ');assert.equal(writes[0].typeCode,'stocks');assert.equal(added[0].ticker,'SBER');assert.equal(ui.state.ticker,'');assert.equal(ui.state.message,'Добавлен SBER');
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

test('missing type preserves input and focuses its error',async()=>{
 const {ui,added,focus}=harness({AddAsset:async()=>({asset:{},errors:{typeCode:'Выберите тип актива'},message:''})});
 await ui.open();await flush();ui.enter('NEW','');await ui.add();
 assert.equal(ui.state.ticker,'NEW');assert.equal(ui.state.typeCode,'');assert.equal(ui.state.typeError,'Выберите тип актива');assert.equal(focus.at(-1),'type');assert.equal(added.length,0);
});
test('legacy type editing supports cancellation and assignment without selecting another trade asset',async()=>{
 const {ui,writes,changes,added,focus}=harness();await ui.open();await flush();
 await ui.edit({id:'42',ticker:'OLD',typeCode:''});assert.equal(ui.state.typeCode,'');assert.equal(focus.at(-1),'type');
 ui.enter('OLD','bonds');await ui.cancelEdit();assert.equal(writes.length,0);assert.equal(focus.at(-1),'trigger');
 await ui.edit({id:'42',ticker:'OLD',typeCode:'stocks'});assert.equal(ui.state.typeCode,'stocks');ui.enter('OLD','bonds');await ui.add();
 assert.equal(writes[0].id,'42');assert.equal(writes[0].typeCode,'bonds');assert.equal(changes.length,1);assert.equal(added.length,0);assert.equal(ui.state.editing,null);assert.equal(ui.state.typeCode,'');assert.equal(focus.at(-1),'replacement');
});
test('failed update retains selection and blocks repeat save and cancellation while pending',async()=>{
 let resolve,count=0;
 const {ui,changes}=harness({UpdateAssetType:()=>{count++;return new Promise(r=>resolve=r)}});
 await ui.open();await flush();await ui.edit({id:'42',ticker:'OLD',typeCode:'stocks'});ui.enter('OLD','bonds');
 const saving=ui.add();await ui.add();await ui.cancelEdit();ui.close();assert.equal(count,1);assert.equal(ui.state.opened,true);assert.ok(ui.state.editing);
 resolve({asset:{},errors:{typeCode:'Недопустимый тип актива'},message:''});await saving;
 assert.equal(ui.state.typeCode,'bonds');assert.equal(ui.state.editing.id,'42');assert.equal(changes.length,0);
});
test('catalog failure retries and ignores response after closing',async()=>{
 let failed=true,resolve;
 const {ui}=harness({ListAssetTypes:async()=>{if(failed)throw Error('catalog');return new Promise(r=>resolve=r)}});
 await ui.open();await flush();assert.ok(ui.state.error);failed=false;const loading=ui.load();ui.close();resolve([{code:'late',name:'late'}]);await loading;assert.equal(ui.state.types.length,0);
});
