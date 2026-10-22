'use strict';
const h=require('../../process/task012/harness.cjs');
const {assert,Native}=h;
async function run(f,{page:p,context},pass){
 const {open}=require('./browser.cjs');
 const task=f.create('QA polling isolated temporal routing work');
 const dev=new Native(f.ownerToken,f.a);dev.open('developer');
 await open(p,f.a);const currentURL=p.url(),endpoint='/web/p/'+f.a+'/data/team';
 const pageData=async()=>{const r=await context.request.get(h.origin+'/web/p/'+f.a+'/team',{headers:{Accept:'application/json'}});assert.equal(r.status(),200);return(await r.json()).page;};
 const before=(await pageData()).session;let reads=0;const listener=r=>{if(r.method()==='GET'&&new URL(r.url()).pathname===endpoint)reads++;};p.on('request',listener);
 const claim=dev.call('next',{task_key:task.key});
 const busy=p.locator('.team-agent-busy').filter({has:p.locator('a.team-agent-task',{hasText:task.key})});
 await busy.waitFor({timeout:15000});assert.equal(p.url(),currentURL);assert(reads>0);
 const releaseRead=reads;dev.call('release',{task_key:task.key,generation:claim.lease.generation,reason:'QA polling release'});
 await busy.waitFor({state:'hidden',timeout:15000});assert(reads>releaseRead);assert.equal(p.url(),currentURL);
 pass('Chrome-team-real10sec-poll-observes-native-claim-release-without-navigation');
 // setOffline is a real browser network/offline event, not an intercepted response.
 await context.setOffline(true);await p.waitForFunction(()=>!navigator.onLine);await p.waitForTimeout(250);let paused=reads;await p.waitForTimeout(11000);assert.equal(reads,paused,'team GET continued while offline');
 await context.setOffline(false);await p.waitForFunction(()=>navigator.onLine);await p.waitForResponse(r=>new URL(r.url()).pathname===endpoint&&r.status()===200,{timeout:15000});
 pass('Chrome-team-browser-offline-pauses-and-online-resumes-GET');
 // Headless Chrome does not expose an OS tab switch reliably. Inject only the
 // visibility input; the hook, cancellation, timers and network remain real.
 await p.evaluate(()=>{Object.defineProperty(document,'visibilityState',{configurable:true,get:()=> 'hidden'});document.dispatchEvent(new Event('visibilitychange'));});await p.waitForTimeout(250);paused=reads;await p.waitForTimeout(11000);assert.equal(reads,paused,'team GET continued while visibility input hidden');
 await p.evaluate(()=>{delete document.visibilityState;document.dispatchEvent(new Event('visibilitychange'));});await p.waitForResponse(r=>new URL(r.url()).pathname===endpoint&&r.status()===200,{timeout:15000});
 const after=(await pageData()).session;assert.equal(after.session_id,before.session_id);assert.equal(after.expires_at,before.expires_at);p.off('request',listener);
 pass('Chrome-team-visibility-input-pauses-resumes-real-GET-and-reads-never-renew',{visibility_input:'DOM visibilityState/event injected; OS tab lifecycle NOT-RUN'});
 await open(p,f.b);assert.equal(await p.locator('.person-card').filter({hasText:'QA-Bob'}).count(),0);assert.equal(await p.locator('.team-page').getByText(task.title,{exact:true}).count(),0);
 await open(p,f.a);assert(await p.locator('.person-card').filter({hasText:'QA-Bob'}).count());pass('Chrome-team-project-switch-newscope-B-excludes-A-people-work');
}
module.exports={run};
