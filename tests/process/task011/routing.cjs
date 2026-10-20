'use strict';
const h=require('./harness.cjs');
const {assert,Native,MCP,crypto}=h;
const settings=(name='QA Same Name',role='developer',instructions='QA immutable v1')=>({name,role,instructions,description:'Synthetic coordination',provider:'custom',model:'qa-no-worker'});
function bound(token,project,p){const n=new Native(token,project);const v=n.call('describe',{role:p.role,run_id:crypto.randomUUID(),machine:token.machine,profile_id:p.id,profile_version:p.version});n.session=v.session_id;return n;}
function create(n,epic,role,title,extra={}){return n.call('create',{type:'task',epic_key:epic.key,role,title,description:'Synthetic coordination task',...extra});}
function done(n,t,evidence,outcome){const c=n.call('next',{task_key:t.key});assert(c.task,'expected actual claim '+t.key);return n.call('transition',{task_key:t.key,generation:c.lease.generation,state:'done',evidence,...(outcome?{outcome}:{})});}
async function run(f,pass){
 const a=f.aNative=new Native(f.editor,f.a);a.open('coordinator');const b=new Native(f.ownerToken,f.b);b.open('coordinator');f.bNative=b;
 const epic=f.routingEpic=a.call('create',{type:'epic',title:'QA routing',description:'Synthetic'});
 const p=a.call('profile_create',settings()),q=a.call('profile_create',settings()),foreign=b.call('profile_create',settings());f.profile=p;f.otherProfile=q;
 const w1=bound(f.editor,f.a,p),w2=bound(f.editor,f.a,p),wrong=bound(f.editor,f.a,q),unbound=new Native(f.editor,f.a);unbound.open('developer');
 const assigned=create(a,epic,'developer','QA pinned routing',{profile_id:p.id,profile_version:1});
 assert.equal(assigned.assignment.profile_id,p.id);assert.equal(wrong.call('next',{task_key:assigned.key}).task,null);assert.equal(unbound.call('next',{task_key:assigned.key}).task,null);
 const race=await Promise.all([w1,w2].map(w=>w.http('tasks/next',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({request_id:crypto.randomUUID(),task_key:assigned.key})})));
 assert(race.every(x=>x.status===200));assert.equal(race.filter(x=>x.data.task).length,1,'concurrent matching sessions must have one owner');
 const win=race[0].data.task?w1:w2,claim=race.find(x=>x.data.task).data;f.bound=win;
 a.reject('assign',{task_key:assigned.key,version:claim.task.version,mode:'role_queue',reason:'Cannot steal'},['CONFLICT']);
 const free=create(a,epic,'developer','QA generic role pool');assert(unbound.call('next',{task_key:free.key}).task);unbound.call('release',{task_key:free.key,generation:1,reason:'QA release'});
 const waiting=create(a,epic,'developer','QA v1 waiting',{profile_id:p.id,profile_version:1});
 const p2=a.call('profile_update',{...settings(p.name,'developer','QA immutable v2'),profile_id:p.id,version:1});
 assert.equal(w2===win?w1.call('next',{task_key:waiting.key}).task:w2.call('next',{task_key:waiting.key}).task,null);
 assert(a.call('coordination',{task_key:waiting.key}).routing_reasons.some(r=>r.code==='assignment_changed'));
 const hb=win.call('heartbeat',{task_key:assigned.key,generation:claim.lease.generation});assert.equal(hb.profile.version,1);assert.equal(hb.profile.instructions,'QA immutable v1');
 win.call('transition',{task_key:assigned.key,generation:claim.lease.generation,state:'done',evidence:{commit:'a'.repeat(40),run_summary:'Actual synthetic completion'}});
 let t=a.call('get',{task_key:waiting.key});const req={request_id:crypto.randomUUID(),task_key:t.key,version:t.version,mode:'profile',profile_id:p2.id,profile_version:2,reason:'QA explicit refresh'};
 const refreshed=a.call('assign',req);assert.equal(refreshed.assignment.profile_version,2);assert.deepEqual(a.call('assign',req),refreshed);a.reject('assign',{...req,reason:'different'},['IDEMPOTENCY_CONFLICT']);
 a.reject('assign',{...req,request_id:crypto.randomUUID()},['CONFLICT']);
 const w3=bound(f.editor,f.a,p2),opened=w3.call('heartbeat',{});assert(opened.queue.ready>=1);const c=w3.call('next',{task_key:t.key});assert(c.task);w3.call('release',{task_key:t.key,generation:c.lease.generation,reason:'QA returned'});
 a.call('profile_archive',{profile_id:p2.id,version:2,state:'archived'});assert.equal(w3.call('next',{task_key:t.key}).task,null);assert(a.call('coordination',{task_key:t.key}).routing_reasons.some(r=>r.code==='assignment_archived'));
 const clear={request_id:crypto.randomUUID(),task_key:t.key,version:a.call('get',{task_key:t.key}).version,mode:'role_queue',reason:'QA clear pinned route'};assert(!a.call('assign',clear).assignment);assert(unbound.call('next',{task_key:t.key}).task);unbound.call('release',{task_key:t.key,generation:2,reason:'QA released'});
 const changes=a.call('events',{task_key:t.key,limit:200}).events.filter(e=>e.type==='task_assigned');assert.equal(changes.length,2);assert.equal(changes[0].data.assignment_change.previous.profile_version,1);assert.equal(changes[1].data.assignment_change.next,null);
 pass('profile-pinned-routing-concurrent-two-session-oneclaim-unbound-pool-edit-archive-liveholder-snapshot-reassign-replay-audit');
 const invalid=create(a,epic,'developer','QA invalid assignment');
 for(const pair of [{profile_id:q.id},{profile_version:1},{profile_id:foreign.id,profile_version:1},{profile_id:q.id,profile_version:0}])a.reject('assign',{task_key:invalid.key,version:invalid.version,mode:'profile',reason:'QA invalid',...pair},['VALIDATION','NOT_FOUND']);
 a.reject('create',{type:'task',epic_key:epic.key,role:'human',title:'QA invalid human',description:'Synthetic',profile_id:q.id,profile_version:1},['VALIDATION']);
 a.reject('create',{type:'task',epic_key:epic.key,role:'architect',title:'QA invalid role',description:'Synthetic',profile_id:q.id,profile_version:1},['VALIDATION']);
 const route='tasks/'+invalid.key+'/assign',base={request_id:crypto.randomUUID(),version:invalid.version,mode:'role_queue',reason:'QA strict'};
 for(const body of [JSON.stringify({...base,reason:null}),JSON.stringify({...base,unknown:'x'}),JSON.stringify(base).replace('"reason":"QA strict"','"reason":"QA strict","reason":"duplicate"')])assert.equal((await a.http(route,{method:'POST',headers:{'Content-Type':'application/json'},body})).status,400);
 const detail=await a.http('tasks/'+invalid.key+'/coordination');assert.equal(detail.status,200);assert.equal(detail.data.task_key,invalid.key);
 pass('strict-scoped-native-REST-assignment-partial-null-duplicate-unknown-role-foreignproject-denials');
 const mcp=new MCP(f.editor,f.a);try{await mcp.init();assert.equal((await mcp.rpc('tools/list',{})).tools.length,12);const opened=await mcp.call('describe',{request_id:crypto.randomUUID(),role:'developer',run_id:'QA-mcp',machine:f.editor.machine,profile_id:q.id,profile_version:1});assert(!opened.isError);const session=opened.data.session_id;
 const res=await mcp.rpc('resources/read',{uri:'accord://coordination?session_id='+session+'&task_key='+invalid.key});assert.equal(JSON.parse(res.contents[0].text).task_key,invalid.key);
 const created=await mcp.call('create',{request_id:crypto.randomUUID(),session_id:session,type:'task',epic_key:epic.key,role:'developer',title:'QA MCP assigned',description:'Actual MCP assigned create',profile_id:q.id,profile_version:1});assert(!created.isError);const mcptask=created.data;assert.equal(mcptask.assignment.profile_id,q.id);const mc=await mcp.call('next',{request_id:crypto.randomUUID(),session_id:session,task_key:mcptask.key});assert(!mc.isError);assert.equal(mc.data.task.key,mcptask.key);assert(!(await mcp.call('release',{request_id:crypto.randomUUID(),session_id:session,task_key:mcptask.key,generation:mc.data.lease.generation,reason:'QA release'})).isError);
 let denied=false;try{await mcp.rpc('resources/read',{uri:'accord://coordination?session_id='+b.session+'&task_key='+invalid.key});}catch{denied=true;}assert(denied);
 }finally{await mcp.close();}
 pass('actual-stdio-MCP12tools-bound-create-next-routing-coordination-resource-projectsession-denial');
 f.uiTask=create(a,epic,'developer','QA Browser coordination');f.uiBlocker=create(a,epic,'human','QA Browser blocker');f.uiProfile=q;
}
module.exports={run,settings,bound,create,done};
