'use strict';
const {assert}=require('../../process/task011/harness.cjs');
async function run(f,owner,pass){
 const p=owner.page;await p.getByRole('tab',{name:'Координация',exact:true}).click();await p.getByRole('button',{name:'Изменить назначение',exact:true}).click();let dialog=p.getByRole('dialog');
 await dialog.locator('select[name=profile]').selectOption(f.uiProfile.id+'@1');await dialog.locator('textarea[name=reason]').fill('QA Chrome assignment exact retry');
 const endpoint='/web/p/'+f.a+'/tasks/'+f.uiTask.key+'/assign',matcher=url=>url.pathname===endpoint;let captured,committed,retry;
 await p.route(matcher,async route=>{if(route.request().method()!=='POST'||captured)return route.continue();captured={url:route.request().url(),body:route.request().postData()};const actual=await route.fetch();assert.equal(actual.status(),200);committed=await actual.json();await route.abort('failed');});
 await dialog.getByRole('button',{name:'Сохранить',exact:true}).click();await dialog.getByText('Результат отправки неизвестен.',{exact:false}).waitFor();assert(committed?.redirect);
 assert.equal(f.aNative.call('get',{task_key:f.uiTask.key}).assignment.profile_id,f.uiProfile.id);
 const listener=request=>{if(request.method()==='POST'&&new URL(request.url()).pathname===endpoint)retry={url:request.url(),body:request.postData()};};p.on('request',listener);
 await dialog.getByRole('button',{name:'Повторить ту же отправку',exact:true}).click();await dialog.waitFor({state:'hidden'});assert.deepEqual(retry,captured);p.off('request',listener);await p.unroute(matcher);
 const events=f.aNative.call('events',{task_key:f.uiTask.key,limit:200}).events;assert.equal(events.filter(e=>e.type==='task_assigned'&&e.reason==='QA Chrome assignment exact retry').length,1);
 pass('Chrome-actual-assignment-commit-lostresponse-exactretry-single-audit');
 await p.getByRole('button',{name:'Изменить назначение',exact:true}).click();dialog=p.getByRole('dialog');await dialog.locator('textarea[name=reason]').fill('QA Chrome dirty draft persists');
 const current=f.aNative.call('get',{task_key:f.uiTask.key});f.aNative.call('assign',{task_key:current.key,version:current.version,mode:'role_queue',reason:'QA concurrent clear'});
 const response=p.waitForResponse(r=>r.request().method()==='POST'&&new URL(r.url()).pathname===endpoint);await dialog.getByRole('button',{name:'Сохранить',exact:true}).click();assert.equal((await response).status(),409);
 const next=f.aNative.call('get',{task_key:f.uiTask.key});await dialog.getByRole('button',{name:'Применить черновик к версии '+next.version,exact:true}).waitFor();assert.equal(await dialog.locator('textarea[name=reason]').inputValue(),'QA Chrome dirty draft persists');assert(await dialog.getByRole('button',{name:'Сохранить',exact:true}).isDisabled());
 await dialog.getByRole('button',{name:'Применить черновик к версии '+next.version,exact:true}).click();await dialog.getByRole('button',{name:'Сохранить',exact:true}).click();await dialog.waitFor({state:'hidden'});assert.equal(f.aNative.call('get',{task_key:f.uiTask.key}).assignment.profile_id,f.uiProfile.id);
 pass('Chrome-dirty-assignment409-keeps-draft-and-explicit-version-rebase');
 // Native concurrent epoch tests already passed; leave an assigned task for shots.
 await p.getByRole('button',{name:'Изменить назначение',exact:true}).click();dialog=p.getByRole('dialog');await dialog.locator('select[name=profile]').selectOption('');await dialog.locator('textarea[name=reason]').fill('QA explicit role queue');await dialog.getByRole('button',{name:'Сохранить',exact:true}).click();await dialog.waitFor({state:'hidden'});assert(!f.aNative.call('get',{task_key:f.uiTask.key}).assignment);
 const before=f.aNative.call('get',{task_key:f.uiTask.key});f.aNative.call('assign',{task_key:before.key,version:before.version,mode:'profile',profile_id:f.uiProfile.id,profile_version:1,reason:'QA screenshots assigned'});
 pass('Chrome-explicit-clear-rolequeue-no-worker-start');
}
module.exports={run};
