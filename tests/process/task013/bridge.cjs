'use strict';
const readline=require('node:readline');const h=require('./harness.cjs');
class Bridge{
 constructor(args){this.pending=new Map();this.sequence=0;this.child=h.cp.spawn(h.binary,args,{env:h.env,stdio:['pipe','pipe','ignore']});this.stopped=new Promise(resolve=>this.child.once('exit',(code,signal)=>resolve({code,signal})));const fail=()=>{for(const p of this.pending.values()){clearTimeout(p.timer);p.reject(new Error('owned MCP process failed'));}this.pending.clear();};this.child.once('error',fail);this.child.once('exit',fail);this.child.stdin.on('error',fail);readline.createInterface({input:this.child.stdout}).on('line',line=>{let value;try{value=JSON.parse(line);}catch{fail();return;}const p=this.pending.get(value.id);if(!p)return;clearTimeout(p.timer);this.pending.delete(value.id);if(value.error)p.reject(new Error('MCP protocol rejected'));else p.resolve(value.result);});}
 rpc(method,params){const id=++this.sequence;return new Promise((resolve,reject)=>{const timer=setTimeout(()=>{this.pending.delete(id);reject(new Error('MCP response deadline'));},12000);this.pending.set(id,{resolve,reject,timer});this.child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');});}
 async init(){await this.rpc('initialize',{protocolVersion:'2025-03-26',capabilities:{},clientInfo:{name:'task013-independent-qa',version:'1'}});this.child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');return this;}
 async call(name,args){const value=await this.rpc('tools/call',{name,arguments:args});return{isError:value.isError===true,data:value.structuredContent||JSON.parse(value.content.find(c=>c.type==='text').text)};}
 async close(){this.child.stdin.end();let timer;const result=await Promise.race([this.stopped,new Promise(resolve=>{timer=setTimeout(()=>{this.child.kill('SIGKILL');resolve({forced:true});},7000);})]);clearTimeout(timer);if(result.forced)await this.stopped;h.assert.equal(result.code,0,'owned bridge shutdown required force');}
}
async function run(f,pass){
 const m=require('./managed.cjs'),{alice,a}=f;
 const base={name:'QA immutable native bridge',description:'Synthetic profile snapshot test',role:'developer',instructions:'QA PINNED VERSION ONE',provider:'unconfigured',model:''};
 const profile=alice.call('profile_create',base),epic=alice.call('create',{type:'epic',title:'QA managed handoff'}),task=alice.call('create',{type:'task',epic_key:epic.key,title:'QA exact managed lease handoff',role:'developer',description:'Synthetic only'});
 m.plan(a,{delay:90});const result=m.start(alice,a,profile,'QA pinned managed bridge');await m.runState(alice,result.run.id,['running']);
 alice.call('profile_update',{...base,profile_id:profile.id,version:1,instructions:'QA UPDATED VERSION TWO'});alice.call('profile_archive',{profile_id:profile.id,version:2,state:'archived'});
 const describe={run_id:result.run.id,machine:a.config.machine,role:'developer',profile_id:profile.id,profile_version:1};
 const binding=generation=>h.path.join(a.config.state_dir,'runs',result.run.id,'attempt-'+generation,'bridge.json');
 async function claim(generation){let bridge;try{
  bridge=await new Bridge(['worker','bridge','--config',binding(generation)]).init();h.assert.equal((await bridge.rpc('tools/list',{})).tools.length,12);
  const opened=await bridge.call('describe',{...describe,request_id:h.crypto.randomUUID()});h.assert(!opened.isError,'pinned managed bridge Describe rejected after archive');h.assert.equal(opened.data.profile.instructions,'QA PINNED VERSION ONE');h.assert.equal(opened.data.profile.version,1);
  const next=await bridge.call('next',{request_id:h.crypto.randomUUID(),task_key:task.key});h.assert(!next.isError,'fresh managed bridge could not acquire same task');h.assert.equal(next.data.lease.session_id,opened.data.session_id);
  return{session:opened.data.session_id,generation:next.data.lease.generation};
 }finally{await bridge?.close();}}
 try{
  const first=await claim(1);h.assert.equal(alice.call('get',{task_key:task.key}).lease.session_id,first.session,'ordinary stdio EOF implicitly released ownership');
  const ordinary=new h.Native(alice.token,alice.project);ordinary.reject('describe',{role:'developer',machine:a.config.machine,run_id:h.crypto.randomUUID(),profile_id:profile.id,profile_version:1},['VALIDATION','CONFLICT','PROFILE_ARCHIVED','VERSION_CONFLICT','PROFILE_VERSION_CONFLICT']);
  pass('actual-stdio-managed-bridge-pinned-snapshot-after-edit-archive-ordinary-oldprofile-denied');
  m.command(alice,result.run.id,'stop');const stopped=await m.runState(alice,result.run.id,['stopped']);const released=alice.call('get',{task_key:task.key});h.assert(!released.lease&&released.state==='ready');h.assert(released.attempts.at(-1).reason.startsWith('managed_stopped:'));
  const old=alice.call('sessions',{limit:100,include_closed:true}).sessions.find(s=>s.session_id===first.session);h.assert(old,'closed managed session missing from explicit session history');h.assert(new Date(old.expires_at)<=new Date(stopped.updated_at),'old managed session remains live after proven terminal report');
  m.plan(a,{delay:90});m.command(alice,result.run.id,'resume','QA same task handoff');await m.runState(alice,result.run.id,['running']);const second=await claim(2);h.assert.notEqual(second.session,first.session);h.assert(second.generation>first.generation,'task fencing generation did not advance');
  m.command(alice,result.run.id,'stop');await m.runState(alice,result.run.id,['stopped']);h.assert(!alice.call('get',{task_key:task.key}).lease);pass('actual-stdio-terminal-handoff-same-task-new-board-session-greater-fence-ordinary-EOF-retains-lease');
 }finally{const current=m.detail(alice,result.run.id).run;if(['queued','starting','running','stopping'].includes(current.state)){if(current.state!=='stopping')m.command(alice,result.run.id,'stop');await m.runState(alice,result.run.id,['stopped']);}}
}
module.exports={Bridge,run};
