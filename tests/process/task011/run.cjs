'use strict';
const h = require('./harness.cjs');
const {fs,path,assert,root,binary,oldBinary,sha,save,admin,legacyToken,catalogToken,freePorts,startServer,stop,BrowserSession,Native,python} = h;
const catalog = require('./catalog.cjs');
function inventory() {
  const walk = folder => fs.readdirSync(folder,{withFileTypes:true}).flatMap(e => e.isDirectory() ? walk(path.join(folder,e.name)) : /\.(cjs|py|go|tsx?|css|conf|in)$/.test(e.name) ? [path.join(folder,e.name)] : []);
  return Object.fromEntries(['tests/process/task011','tests/process/task011-update','tests/browser/task011','internal','cmd','infra/accord','web/src'].flatMap(p=>walk(path.join(h.repo,p))).sort().map(p=>[path.relative(h.repo,p),sha(fs.readFileSync(p))]));
}
const source = inventory();
const baseline = process.env.ACCORD_TASK011_OLD_SHA256 ? {binary_sha256:process.env.ACCORD_TASK011_OLD_SHA256,source_revision:process.env.ACCORD_TASK011_OLD_REVISION} : JSON.parse(fs.readFileSync(path.join(h.repo,'.local/task011/legacy/proof.json')));
const trusted = {'343b93c71297c58c972f73fed4d1d4ca3a349740':'4e11d2569120dada3df15061bcfc5193aacb7931a048110f27242f0b9419e085','cbf202086431335d4657c722a0f75328a09dc4ea':'66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa'};
assert.equal(baseline.binary_sha256,trusted[baseline.source_revision],'untrusted schema3 provenance');
assert.equal(sha(fs.readFileSync(oldBinary)),baseline.binary_sha256,'actual baseline executable mismatch');
const report = {schema:1,started_at:new Date().toISOString(),node:process.version,platform:process.platform,source_revision:process.env.ACCORD_TASK011_REVISION||null,source_dirty:process.env.ACCORD_TASK011_SOURCE_DIRTY||'not-inspected-by-tester',binary_sha256:sha(fs.readFileSync(binary)),old_binary_sha256:baseline.binary_sha256,old_source_revision:baseline.source_revision,checks:[],scope:process.env.ACCORD_TASK011_BROWSER==='0'?'actual-process-only':'actual-process-and-Chrome',limitations:['one host; physical second PC NOT-RUN','loopback HTTP; production HTTPS separate','provider/native Claude-Codex/worker execution NOT-RUN']};
const pass=(id,extra={})=>{report.checks.push({id,result:'PASS',...extra});console.log(id+' PASS');};
let srv;
function preserved(before,after) {for(const [db,tables] of Object.entries(before.databases))for(const [table,rows] of Object.entries(tables))assert.deepEqual(after.databases[db][table],rows,'migration changed old rows '+db+'/'+table);assert.deepEqual(after.blobs,before.blobs);}
function restored(before,after) {assert.deepEqual(after.blobs,before.blobs);assert.deepEqual(after.repositories,before.repositories);for(const [id,b] of Object.entries(before.boards)){for(const key of ['task_history','notes','idempotency'])assert.deepEqual(after.boards[id][key],b[key]);for(const event of b.events)assert(after.boards[id].events.includes(event));}}
async function main(){
  if(process.env.ACCORD_TASK011_BROWSER!=='0'){report.next_build_id=fs.readFileSync(path.join(h.repo,'web/.local/next/BUILD_ID'),'utf8').trim();report.next_upstream='http://127.0.0.1:3101';}
  await freePorts(); const state=path.join(root,'state');fs.mkdirSync(state,{mode:0o700});
  const ownerToken=legacyToken(state,'QA-owner',oldBinary),workerToken=legacyToken(state,'QA-old-worker',oldBinary);
  srv=await startServer(state,{legacy:true,bin:oldBinary});await stop(srv);srv=null;
  admin(state,'bootstrap-owner',['--identity',ownerToken.id],oldBinary);
  const editor=catalogToken(state,'QA-editor',oldBinary),viewer=catalogToken(state,'QA-viewer',oldBinary);
  srv=await startServer(state,{bin:oldBinary});const owner=await new BrowserSession().login(ownerToken);
  const defaultId=(await owner.catalog()).registry.projects[0].id,a=await catalog.create(owner,'QA Alpha'),b=await catalog.create(owner,'QA Beta');
  await catalog.member(owner,a,editor.id,'editor');await catalog.member(owner,a,viewer.id,'viewer');
  const seed=new Native(ownerToken,a,oldBinary);seed.open('coordinator');
  const epic=seed.call('create',{type:'epic',title:'QA preserved epic',description:'Synthetic history'});
  const task=seed.call('create',{type:'task',epic_key:epic.key,role:'developer',title:'QA preserved task',description:'Synthetic history'});
  seed.call('note',{action:'create',scope_type:'epic',scope_id:epic.key,source_task_key:task.key,text:'QA old note'});
  const settings={name:'QA legacy snapshot',description:'Synthetic',role:'developer',instructions:'QA immutable legacy instructions',provider:'custom',model:'qa-only'};
  const profile=seed.call('profile_create',settings),oldWorker=new Native(editor,a,oldBinary);
  const opened=oldWorker.call('describe',{role:'developer',run_id:'QA-old-bound',machine:editor.machine,profile_id:profile.id,profile_version:1});oldWorker.session=opened.session_id;
  const claim=oldWorker.call('next',{task_key:task.key});assert(claim.lease);
  seed.call('profile_update',{...settings,instructions:'QA legacy instructions v2',profile_id:profile.id,version:1});
  const unbound=new Native(workerToken,'',oldBinary),unboundView=unbound.open('developer');assert(!unboundView.profile);
  const asset=await catalog.upload(owner,a,'QA preserved design');await owner.open(a);
  const oldPage=(await owner.read('/web/p/'+a+'/tasks/'+task.key)).data.page,oldCookie=owner.cookie;
  await stop(srv);srv=null;
  const before=python('capture',state),immutable3=python('immutable',state),snapshot3=python('snapshot',state,{name:'schema3-backup'});
  assert.equal(snapshot3.board_schema,3);assert.equal(Object.keys(before.databases).length,7);
  const migration=JSON.parse(admin(state,'migrate-storage').stdout);assert.deepEqual(migration,{storage_migrated:true,boards:3});
  const after=python('capture',state);preserved(before,after);assert(Object.entries(after.schemas).filter(([p])=>p.endsWith('/board')).every(([,n])=>n===4));
  assert.deepEqual(JSON.parse(admin(state,'migrate-storage').stdout),migration);preserved(after,python('capture',state));
  pass('actual-old3-default-plus2projects-sevenDB-profile-revision-rowbytes-blob-migrate4-repeat-preserved');
  const restored3=python('restore',state,{name:'schema3-restored',snapshot:snapshot3.snapshot,binary});restored(immutable3,restored3.immutable);
  assert(restored3.boot_guard&&restored3.reset_complete);assert.equal(restored3.inspect.live_catalog_credentials,0);assert(restored3.inspect.boards.every(x=>x.live_tokens===0&&x.browsers===0));
  assert.deepEqual(restored3.capture.profiles,before.profiles);pass('old3-full-format2-restore-migrate4-authreset-preserves-profilehistory');
  srv=await startServer(state);assert.equal(owner.cookie,oldCookie);
  const page=(await owner.read('/web/p/'+a+'/tasks/'+task.key)).data.page;assert.equal(page.session.session_id,oldPage.session.session_id);
  const hb=oldWorker.call('heartbeat',{task_key:task.key,generation:claim.lease.generation});assert.equal(hb.profile.version,1);assert.equal(hb.profile.instructions,settings.instructions);
  assert.equal(unbound.call('heartbeat',{}).session_id,unboundView.session_id);
  const download=await fetch(h.goOrigin+asset.download_url,{headers:{Cookie:owner.cookie}});assert.equal(download.status,200);assert.equal(sha(Buffer.from(await download.arrayBuffer())),asset.sha256);
  pass('postmigration-actual-restart-cookie-bound-unbound-session-livelease-snapshot-download-preserved');
  const f={state,ownerToken,owner,defaultId,a,b,editor,viewer,oldWorker,legacyProfile:profile,asset};
  await require('./routing.cjs').run(f,pass);await require('./workflow.cjs').run(f,pass);await require('./web.cjs').run(f,pass);
  if(process.env.ACCORD_TASK011_BROWSER!=='0')await require('../../browser/task011/browser.cjs').run(f,pass);
  await stop(srv);srv=null;
  const immutable4=python('immutable',state),snapshot4=python('snapshot',state,{name:'schema4-backup'});assert.equal(snapshot4.board_schema,4);
  const profiles4=python('capture',state).profiles,restore4=python('restore',state,{name:'schema4-restored',snapshot:snapshot4.snapshot,binary});restored(immutable4,restore4.immutable);assert.deepEqual(restore4.capture.profiles,profiles4);assert(restore4.boot_guard&&restore4.reset_complete);assert.equal(restore4.inspect.live_catalog_credentials,0);assert(restore4.inspect.boards.every(x=>x.live_tokens===0&&x.browsers===0));
  assert.deepEqual(python('guard',state,{old:oldBinary,new:binary}),{old:'denied',new:'allowed'});pass('new4-fullbackup-fullrestore-assignment-audit-profiles-history-authreset-oldbinary-denied');
  const damage=python('corrupt',state,{name:'damaged-schema4'}),failed=h.cli(['serve','--db',path.join(damage.state,'board.db'),'--web-data',path.join(damage.state,'web'),'--web-origin',h.origin,'--projects-data',path.join(damage.state,'projects-data'),'--listen','127.0.0.1:3101'],undefined,binary,true);
  assert.notEqual(failed.status,0);assert(!failed.stdout.includes('listening'));assert.deepEqual(python('blobs',damage.state),damage.blobs);assert.deepEqual(python('profile_tables',damage.state),[]);
  pass('damaged-published4-coordination-authority-failclosed-without-repair-or-blobcleanup');
  save(path.join(root,'fixture-private.json'),{state,a,b,defaultId,ownerToken,editor,viewer});report.result='PASS';
}
main().catch(e=>{report.result='FAIL';report.failure=String(e.message).slice(0,350);console.error(report.failure);process.exitCode=1;}).finally(async()=>{
  try{await stop(srv);}catch{report.result='FAIL';report.cleanup_failure=true;process.exitCode=1;}
  report.tests_and_source=source;report.source_unchanged=JSON.stringify(inventory())===JSON.stringify(source);if(!report.source_unchanged){report.result='FAIL';process.exitCode=1;}
  report.finished_at=new Date().toISOString();save(path.join(root,'results.json'),report);console.log('private evidence: '+path.relative(h.repo,root));
});
