'use strict';
const h=require('./harness.cjs'),m=require('./managed.cjs');
// Two native-like process groups under one engine exercise parallel stop grace.
async function run(f,pass){
 const first=m.workerConfig(f.aliceToken,f.project,'qa-parallel',h.path.join(h.root,'workspace-parallel-a'),[],'http://127.0.0.1:3102');
 const secondPath=h.path.join(h.root,'workspace-parallel-b');h.fs.mkdirSync(secondPath,{mode:0o700});first.config.workspaces.other={path:secondPath,providers:['codex']};h.save(first.file,first.config);
 m.plan(first,{delay:90,ignore_interrupt:true});h.save(h.path.join(secondPath,'qa-plan.json'),{delay:90,ignore_interrupt:true});let proc;
 try{
  await m.worker(first,'enroll');proc=m.runWorker(first);await m.until(()=>f.alice.call('managed_list').hosts.find(x=>x.id===m.state(first).host_id)?.worker_epoch>0,'parallel worker did not poll');
  const one=m.start(f.alice,first,f.profile,'QA parallel owned group one');const two=f.alice.call('managed_start',{host_id:m.state(first).host_id,workspace:'other',provider:'codex',profile_id:f.profile.id,profile_version:f.profile.version,prompt:'QA parallel owned group two'});
  await m.runState(f.alice,one.run.id,['running']);await m.runState(f.alice,two.run.id,['running']);
  const beats=[h.path.join(first.workspace,'owned.beat'),h.path.join(secondPath,'owned.beat')];await m.until(()=>beats.every(file=>h.fs.existsSync(file)&&Date.now()-h.fs.statSync(file).mtimeMs<500),'two fresh owned heartbeats absent');
  const cut=Date.now();f.proxy.control.offline=true;await m.until(()=>beats.every(file=>Date.now()-h.fs.statSync(file).mtimeMs>600),'parallel deadman did not stop both groups',32000);h.assert(Date.now()-cut<30000,'sequential stop graces exceeded ownership lease');
  for(const run of[one,two])h.assert(['running','stopping'].includes(m.detail(f.alice,run.run.id).run.state),'offline was declared stopped before receipt');
  f.proxy.control.offline=false;await m.runState(f.alice,one.run.id,['stopped'],15000);await m.runState(f.alice,two.run.id,['stopped'],15000);
  for(const run of[one,two])h.assert(m.detail(f.alice,run.run.id).commands.some(c=>c.kind==='start'&&c.forced),'forced exit was not retained on terminal receipt');
  pass('two-owned-runs-SAME-worker-parallel-forced-deadman-both-proven-before30s');
 }catch(e){try{const journal=m.state(first);h.save(h.path.join(first.folder,'diagnostic-summary.json'),Object.values(journal.runs).map(r=>({state:r.state,local_diagnostic:r.local_diagnostic||null,error:r.error||null})));}catch{}throw e;
 }finally{f.proxy.control.offline=false;await m.stopWorker(proc);}
}
module.exports={run};
