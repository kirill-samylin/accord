'use strict';
const http=require('node:http');
const h=require('./harness.cjs');
const {fs,path,cp,crypto,assert,root,repo,env,binary,cli,sleep}=h;
const provider=path.join(repo,'tests/process/task013/provider.py');
const configs=[];
function workerConfig(token,project,name,workspace,controllers=[],server=h.goOrigin){
 const folder=path.join(root,name);fs.mkdirSync(folder,{mode:0o700});fs.mkdirSync(workspace,{recursive:true,mode:0o700});
 const config={server,project,token_file:token.file,state_dir:path.join(folder,'worker-state'),name,machine:token.machine,controllers,executables:{codex:provider},codex_sandbox:'read-only',workspaces:{qa:{path:workspace,providers:['codex']}}};
 const file=path.join(folder,'config.json');h.save(file,config);const value={file,config,workspace,folder};configs.push(value);return value;
}
function plan(w,value){h.save(path.join(w.workspace,'qa-plan.json'),value);fs.rmSync(path.join(w.workspace,'qa-complete'),{force:true});}
async function worker(w,op='run',allowFailure=false){
 const child=cp.spawn(binary,['worker',op,'--config',w.file],{cwd:repo,env:{...env,...w.env},stdio:['ignore','pipe','pipe']});let stdout='',stderr='';child.stdout.on('data',c=>{stdout+=c;});child.stderr.on('data',c=>{stderr+=c;});let timer=setTimeout(()=>child.kill('SIGKILL'),30000);const status=await new Promise((resolve,reject)=>{child.once('error',reject);child.once('exit',resolve);});clearTimeout(timer);if(!allowFailure)assert.equal(status,0,'worker '+op+' rejected: '+stderr.trim());return{status,stdout,stderr};
}
function runWorker(w){const log=fs.openSync(path.join(w.folder,'worker.log'),'a',0o600);const child=cp.spawn(binary,['worker','run','--config',w.file],{cwd:repo,env:{...env,...w.env},stdio:['ignore',log,log]});fs.closeSync(log);const stopped=new Promise(resolve=>child.once('exit',(code,signal)=>resolve({code,signal})));return{child,stopped};}
async function stopWorker(proc,signal='SIGTERM'){if(!proc||proc.child.exitCode!==null||proc.child.signalCode!==null)return;proc.child.kill(signal);let timer;const result=await Promise.race([proc.stopped,new Promise(resolve=>{timer=setTimeout(()=>{proc.child.kill('SIGKILL');resolve({forced:true});},10000);})]);clearTimeout(timer);if(result.forced){await proc.stopped;throw new Error('worker shutdown deadline; owned process killed and reaped');}return result;}
async function until(fn,message,ms=20000){const start=Date.now();while(Date.now()-start<ms){const value=await fn();if(value)return value;await sleep(100);}throw new Error(message);}
function state(w){return JSON.parse(fs.readFileSync(path.join(w.config.state_dir,'state.json'),'utf8'));}
function calls(w){try{return fs.readFileSync(path.join(w.workspace,'qa-invocations.jsonl'),'utf8').split('\n').slice(0,-1).filter(Boolean).map(JSON.parse);}catch{return[];}}
function detail(n,id){return n.call('managed_run',{run_id:id});}
function command(n,id,kind,text){const r=detail(n,id).run;return n.call('managed_command',{run_id:id,version:r.version,kind,...(text?{text}:{})});}
function start(n,w,profile,prompt='QA start'){return n.call('managed_start',{host_id:state(w).host_id,workspace:'qa',provider:'codex',profile_id:profile.id,profile_version:profile.version,prompt});}
async function runState(n,id,wanted,timeout){try{return await until(()=>{const r=detail(n,id).run;return wanted.includes(r.state)&&r;},'run did not reach '+wanted,timeout);}catch(e){const diagnostics=[];for(const w of configs){try{const local=state(w).runs[id];if(local)diagnostics.push({host:w.config.name,state:local.state,local_diagnostic:local.local_diagnostic||null});}catch{}}h.save(path.join(root,'runtime-diagnostics.json'),{wanted,diagnostics});throw e;}}
async function beatGone(w,name='owned.beat'){const file=path.join(w.workspace,name);await until(()=>fs.existsSync(file),'fixture heartbeat absent');await sleep(250);const old=fs.readFileSync(file,'utf8');await sleep(600);assert.equal(fs.readFileSync(file,'utf8'),old,'owned group continues writing');}
function cleanDetached(w){fs.writeFileSync(path.join(w.workspace,'detached.stop'),'stop',{mode:0o600});}
function heartbeat(){const file=path.join(root,'unrelated.beat');const child=cp.spawn(provider,['heartbeat-child',file],{cwd:repo,env,stdio:'ignore'});const stopped=new Promise(resolve=>child.once('exit',resolve));return{file,child,stopped};}
async function stopHeartbeat(p){if(!p)return;fs.writeFileSync(p.file.replace(/\.beat$/,'.stop'),'stop',{mode:0o600});let timer;await Promise.race([p.stopped,new Promise(resolve=>{timer=setTimeout(()=>{p.child.kill('SIGTERM');resolve();},2000);})]);clearTimeout(timer);await p.stopped;}

async function proxy(){
 await h.freePorts([3102]);const drops={accepted:0,started:0},observed=[],connections=new Set(),control={offline:false};
 const server=http.createServer(async(req,res)=>{try{
  if(control.offline){res.destroy();return;}const chunks=[];for await(const c of req)chunks.push(c);const body=Buffer.concat(chunks);let value;try{value=JSON.parse(body);}catch{}
  const upstream=await fetch(h.goOrigin+req.url,{method:req.method,headers:{...req.headers,host:'127.0.0.1:3101'},body:['GET','HEAD'].includes(req.method)?undefined:body,redirect:'manual',signal:AbortSignal.timeout(5000)});const data=Buffer.from(await upstream.arrayBuffer());
  if(req.url.endsWith('/managed/report')){observed.push({phase:value.phase,request_id:value.request_id,status:upstream.status});if(drops[value.phase]){drops[value.phase]--;res.destroy();return;}}
  res.writeHead(upstream.status,Object.fromEntries(upstream.headers));res.end(data);
 }catch{res.destroy();}});
 server.on('connection',socket=>{connections.add(socket);socket.once('close',()=>connections.delete(socket));});
 await new Promise(resolve=>server.listen(3102,'127.0.0.1',resolve));return{drops,observed,control,close:()=>new Promise(resolve=>{server.close(resolve);for(const socket of connections)socket.destroy();})};
}
module.exports={workerConfig,plan,worker,runWorker,stopWorker,until,state,calls,detail,command,start,runState,beatGone,cleanDetached,proxy,heartbeat,stopHeartbeat};
