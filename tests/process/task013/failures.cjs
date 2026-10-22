'use strict';
const h=require('./harness.cjs'),m=require('./managed.cjs');
const{fs,path,assert,crypto,root}=h;
async function run(f,pass){
 const {alice,bob,profile,a,b,proxy}=f;let wa=f.wa,wb=f.wb,other;
 try{
  const proof=m.state(a);const forged=await alice.http('managed/poll',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({request_id:crypto.randomUUID(),host_id:proof.host_id,host_secret:'0'.repeat(64),worker_id:'worker_forged',worker_epoch:proof.epoch})});assert.equal(forged.status,401);assert.equal(forged.data.error.code,'UNAUTHORIZED');pass('spoofed-host-proof-denied-no-dispatch');
  m.plan(a,{delay:90});const offline=m.start(bob,a,profile);await m.runState(alice,offline.run.id,['running']);const cut=Date.now();proxy.control.offline=true;
  await m.until(()=>{const beat=path.join(a.workspace,'owned.beat');return Date.now()-fs.statSync(beat).mtimeMs>700;},'offline deadman did not stop owned group',32000);assert(Date.now()-cut<30000,'local deadman exceeded server lease');const before=m.detail(alice,offline.run.id).run;assert(['running','stopping'].includes(before.state),'offline was prematurely reported stopped');proxy.control.offline=false;await m.runState(alice,offline.run.id,['stopped'],15000);pass('offline-monotonic-deadman-owned-PGID-before-lease-confirmed-after-network');
  await m.stopWorker(wb);wb=null;const payload={request_id:crypto.randomUUID(),host_id:m.state(b).host_id,workspace:'qa',provider:'codex',profile_id:profile.id,profile_version:1,prompt:'QA revoked creator queued'};const revoked=alice.call('managed_start',payload);b.config.controllers=[];h.save(b.file,b.config);await m.worker(b,'consent');alice.reject('managed_start',payload,['FORBIDDEN','FORBIDDEN_ROLE']);wb=m.runWorker(b);await m.runState(bob,revoked.run.id,['stopped'],45000);assert.equal(m.calls(b).length,2);pass('local-consent-revoke-denies-exact-replay-cancels-undispatched-start');
  // A different enrollment in the same checkout cannot bypass live local authority.
  const c=m.workerConfig(f.alice.token,alice.project,'qa-cross-state',a.workspace,[]);await m.worker(c,'enroll');other=m.runWorker(c);
  m.plan(a,{delay:90});const crash=m.start(bob,a,profile);await m.runState(alice,crash.run.id,['running']);const count=m.calls(a).length;const busy=m.start(alice,c,profile);await m.runState(alice,busy.run.id,['failed']);assert.equal(m.calls(a).length,count);pass('cross-state-host-same-checkout-live-lock-no-second-launch');
  await m.stopWorker(wa,'SIGKILL');wa=null;await m.beatGone(a);const journalBefore=m.state(a);assert.equal(journalBefore.runs[crash.run.id].state,'running');
  const restarted=m.runWorker(a);wa=restarted;await m.until(()=>m.state(a).runs[crash.run.id].state==='needs_recovery','restart did not quarantine journal');await m.runState(alice,crash.run.id,['needs_recovery'],35000);await m.until(()=>restarted.child.exitCode!==null||restarted.child.signalCode!==null,'recovery worker did not exit',15000);wa=null;assert.equal(m.calls(a).length,count);
  const again=m.start(alice,c,profile);await m.runState(alice,again.run.id,['failed']);assert.equal(m.calls(a).length,count);pass('SIGKILL-lifeline-owned-PGID-exit-restart-needs-recovery-global-intent-retained');
  await quota(f,pass);
 }finally{proxy.control.offline=false;const cleanup=await Promise.allSettled([m.stopWorker(wa),m.stopWorker(wb),m.stopWorker(other)]);assert(cleanup.every(x=>x.status==='fulfilled'),'owned workers cleanup failed');}
}
async function quota(f,pass){
 const n=f.alice,secret=crypto.randomBytes(32).toString('hex');const host=n.call('managed_host_enroll',{name:'qa-quota',host_secret:secret,controllers:[],workspaces:['quota'],providers:['codex']});
 const auth={host_id:host.id,host_secret:secret,worker_id:'worker_quota'};let poll=n.call('managed_poll',auth);auth.worker_epoch=poll.worker_epoch;
 const started=n.call('managed_start',{host_id:host.id,workspace:'quota',provider:'codex',profile_id:f.profile.id,profile_version:1,prompt:'QA quota protocol only'});poll=n.call('managed_poll',auth);
 const report={...auth,command_id:poll.command.id,dispatch_id:poll.command.dispatch_id,run_id:poll.run.id,generation:poll.run.generation};n.call('managed_report',{...report,phase:'accepted'});n.call('managed_report',{...report,phase:'started',provider_session_id:crypto.randomUUID()});
 for(let i=0;i<16;i++)m.command(n,started.run.id,'clarify','QA pending '+i);
 n.reject('managed_command',{run_id:started.run.id,version:m.detail(n,started.run.id).run.version,kind:'clarify',text:'overflow'},['VALIDATION','CONFLICT','QUOTA_EXCEEDED','MANAGED_LIMIT']);
 const stop=m.command(n,started.run.id,'stop');assert.equal(stop.command.kind,'stop');poll=n.call('managed_poll',auth);assert.equal(poll.command.id,stop.command.id);pass('sixteen-mailbox-quota-reserved-stop-dispatch-priority-protocol-only');
 const stopReport={...auth,command_id:poll.command.id,dispatch_id:poll.command.dispatch_id,run_id:poll.run.id,generation:poll.run.generation};n.call('managed_report',{...stopReport,phase:'accepted'});n.call('managed_report',{...stopReport,phase:'stopped'});
 let current=n.call('managed_start',{host_id:host.id,workspace:'quota',provider:'codex',profile_id:f.profile.id,profile_version:1,prompt:'QA 256 history protocol only'}).run;const session=crypto.randomUUID();
 function turn(complete){const issued=n.call('managed_poll',auth),receipt={...auth,command_id:issued.command.id,dispatch_id:issued.command.dispatch_id,run_id:issued.run.id,generation:issued.run.generation};n.call('managed_report',{...receipt,phase:'accepted'});current=n.call('managed_report',{...receipt,phase:'started',provider_session_id:session}).run;if(complete)current=n.call('managed_report',{...receipt,phase:'completed',provider_session_id:session}).run;}
 turn(true);for(let i=0;i<255;i++){current=n.call('managed_command',{run_id:current.id,version:current.version,kind:'resume',text:'QA history '+i}).run;turn(i<254);}
 n.reject('managed_command',{run_id:current.id,version:current.version,kind:'clarify',text:'history overflow'},['CONFLICT']);const reserve=n.call('managed_command',{run_id:current.id,version:current.version,kind:'stop'});assert.equal(reserve.command.sequence,257);assert(m.detail(n,current.id).commands_truncated);assert.equal(n.call('managed_poll',auth).command.id,reserve.command.id);pass('256-history-quota-separate-257-stop-reserve-bounded-detail-protocol-only');
}
module.exports={run};
