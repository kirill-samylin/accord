'use strict';
const {assert}=require('../../process/task011/harness.cjs');
async function run(f,owner,pass){
 const {open}=require('./browser.cjs'),p=owner.page;await open(p,f.a,'/tasks');
 const second=await owner.context.newPage();second.setDefaultTimeout(15000);await open(second,f.b,'/tasks');assert.doesNotMatch(await second.locator('main').innerText(),/QA Browser coordination/);assert(new URL(p.url()).pathname.includes(f.a));
 let signal,release,finish,failure,request;const observed=new Promise(r=>signal=r),gate=new Promise(r=>release=r),delivery=new Promise(r=>finish=r);const matcher=url=>url.pathname==='/web/p/'+f.a+'/tasks/'+f.uiTask.key;
 await p.route(matcher,async route=>{if(request||route.request().method()!=='GET')return route.continue();request=route.request();try{const actual=await route.fetch();assert.equal(actual.status(),200);signal();await gate;try{await route.fulfill({response:actual});finish();}catch{finish();}}catch(e){failure=e;signal();finish();await route.abort().catch(()=>{});}});
 try{
   // A real refresh is captured before switching; it is never fulfilled with invented data.
   await p.getByRole('button',{name:'Быстрый просмотр '+f.uiTask.key,exact:true}).click();let timer;await Promise.race([observed,new Promise((_,rej)=>{timer=setTimeout(()=>rej(new Error('actual task read not observed')),12000);})]);clearTimeout(timer);if(failure)throw failure;
   await p.getByRole('button',{name:'Закрыть просмотр',exact:true}).click();const catalogResponse=p.waitForResponse(r=>new URL(r.url()).pathname==='/web/projects');await p.getByRole('button',{name:'Переключить проект',exact:true}).click();const cr=await catalogResponse;assert.equal(cr.status(),200,'actual project catalog HTTP');const cb=await cr.json();assert(cb.ok&&cb.page?.registry,'actual project catalog JSON');await p.getByRole('menuitem',{name:/QA Beta/}).click();await p.waitForURL(new RegExp('/app/p/'+f.b));await p.locator('.control-shell').waitFor();release();await delivery;assert.equal(await p.locator('.project-switch strong').innerText(),'QA Beta');assert.doesNotMatch(await p.locator('main').innerText(),/QA Browser coordination/);await second.reload();await second.locator('.control-shell').waitFor();assert.doesNotMatch(await second.locator('main').innerText(),/QA Browser coordination/);
   pass('Chrome-two-tabs-and-actual-late-projectA-task-read-after-switchB');
 }finally{release();await p.unroute(matcher);await second.close();}
}
module.exports={run};
