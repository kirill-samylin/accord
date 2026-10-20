'use strict';
const {assert,BrowserSession,Native}=require('./harness.cjs');const catalog=require('./catalog.cjs');
async function action(web,project,key,op,fields){const base='/web/p/'+project+'/tasks/'+key,page=(await web.read(base)).data.page;return web.post(base+'/'+op,{csrf:page.csrf,action_id:page.actions['coord:'+op],...fields});}
async function run(f,pass){
 await f.owner.open(f.a);const base='/web/p/'+f.a+'/tasks/'+f.uiTask.key,page=(await f.owner.read(base)).data.page;assert(page.can_coordinate);assert(page.assignment_profiles.some(p=>p.id===f.uiProfile.id));assert(!page.assignment_profiles.some(p=>p.instructions));
 const fields={csrf:page.csrf,action_id:page.actions['coord:assign'],version:String(page.task.version),mode:'profile',profile_id:f.uiProfile.id,profile_version:'1',reason:'QA browser JSON assignment'};
 const bad=await f.owner.post(base+'/assign',{...fields,reason:''});assert.equal(bad.status,400);assert.notEqual(bad.data.page.actions['coord:assign'],fields.action_id);
 fields.action_id=bad.data.page.actions['coord:assign'];const assigned=await f.owner.post(base+'/assign',fields);assert.equal(assigned.status,200);const replay=await f.owner.post(base+'/assign',fields);assert.equal(replay.status,200);assert.equal(replay.data.redirect,assigned.data.redirect);
 const concurrent=(await f.owner.read(base)).data.page;const conflict=await f.owner.post(base+'/assign',{...fields,action_id:concurrent.actions['coord:assign']});assert.equal(conflict.status,409);assert.equal(conflict.data.draft.reason,fields.reason);
 assert.equal((await action(f.owner,f.a,f.uiTask.key,'assign',{version:String(concurrent.task.version),mode:'role_queue',reason:'QA clear before Chrome'})).status,200);
 assert.equal((await action(f.owner,f.a,f.uiTask.key,'link',{from:f.uiBlocker.key,type:'blocks',condition:'',generation:'0',unlink:'false'})).status,200);
 const blocked=(await f.owner.read(base)).data.page;assert(blocked.coordination.blockers.length>0);assert(blocked.coordination.dependencies.some(d=>d.link.from===f.uiBlocker.key&&!d.satisfied));
 assert.equal((await action(f.owner,f.a,f.uiTask.key,'link',{from:f.uiBlocker.key,type:'blocks',condition:'',generation:'0',unlink:'true'})).status,200);
 assert.equal((await action(f.owner,f.a,f.uiTask.key,'transition',{state:'cancelled',reason:'QA administrative cancel',generation:'0'})).status,200);
 assert.equal((await action(f.owner,f.a,f.uiTask.key,'transition',{state:'ready',reason:'QA administrative restore',generation:'0'})).status,200);
 pass('actual-browser-JSON-assignment-validation-replay-409draft-dependency-link-unlink-cancel-restore');
 const viewer=await new BrowserSession().login(f.viewer);await viewer.open(f.a);const vp=(await viewer.read(base)).data.page;assert.equal(vp.can_coordinate,false);assert(!vp.actions['coord:assign']);assert(vp.coordination);
 assert.equal((await viewer.post(base+'/assign',{...fields,csrf:vp.csrf})).status,409);
 const nativeViewer=new Native(f.viewer,f.a);nativeViewer.open('human');nativeViewer.reject('assign',{task_key:f.uiTask.key,version:vp.task.version,mode:'role_queue',reason:'Denied'},['FORBIDDEN_ROLE']);nativeViewer.reject('check_create',{target_key:f.uiDone.key,target_version:f.uiDone.version,role:'reviewer'},['FORBIDDEN_ROLE']);
 const editor=await new BrowserSession().login(f.editor);await editor.open(f.a);const ep=(await editor.read(base)).data.page,cached={csrf:ep.csrf,action_id:ep.actions['coord:assign'],version:String(ep.task.version),mode:'role_queue',reason:'QA cached epoch action'};assert.equal((await editor.post(base+'/assign',cached)).status,200);
 await catalog.member(f.owner,f.a,f.editor.id,'viewer','set');assert.equal((await editor.post(base+'/assign',cached)).status,403);await catalog.member(f.owner,f.a,f.editor.id,'editor','set');await catalog.member(f.owner,f.a,f.editor.id,'','revoke');await catalog.member(f.owner,f.a,f.editor.id,'editor','grant');assert([401,409].includes((await editor.post(base+'/assign',cached)).status));
 f.aNative.open('coordinator');f.reviewer.open('reviewer');f.tester.open('tester');
 await catalog.state(f.owner,f.b,'archived');f.bNative.reject('assign',{task_key:'T-1',version:1,mode:'role_queue',reason:'QA archived'},['FORBIDDEN_ROLE']);await catalog.state(f.owner,f.b,'active');
 pass('actual-browser-native-viewer-denials-downgrade-replay-revoke-regrant-epoch-archivedproject');
}
module.exports={run,action};
