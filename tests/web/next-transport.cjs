'use strict';
const h = require('./next-fixture.cjs');
module.exports = async (f, pass) => {
  const p = f.p; await p.clock.install({time:new Date(Date.now()-60000)});
  const question = (await h.call(f.worker, 'create', {type:'task', epic_key:f.epic, role:'human', title:'Synthetic network draft', description:'Preserve local input under real transport failure'})).key;
  await h.goto(f, '/app/tasks/' + question); await h.posted(p, () => p.getByRole('button', {name:'Взять и ответить'}).click()); await p.locator('[name=answer]').fill('Synthetic background read draft');
  let armed=true; let barrierEntered; const entered=new Promise(resolve=>{barrierEntered=resolve;});
  const pattern=f.origin+'/web/tasks/'+question;
  const barrier=async route=>{if(armed&&route.request().method()==='GET'){armed=false; f.go.child.kill('SIGSTOP'); barrierEntered();} await route.continue();};
  // Only a synchronization barrier: the actual GET is continued to the real Go server, never fulfilled or fabricated.
  await p.route(pattern,barrier);
  try {
    const failed=p.waitForEvent('requestfailed',r=>r.method()==='GET'&&new URL(r.url()).pathname==='/web/tasks/'+question);
    await h.posted(p,()=>p.getByRole('button',{name:'Продлить время',exact:true}).click()); await entered; await p.locator('[name=answer]').focus();
    await p.clock.fastForward(9000); await failed;
    h.assert.equal(await p.locator('[name=answer]').inputValue(),'Synthetic background read draft','failed read unmounted local answer'); h.assert(await p.locator('[name=answer]').evaluate(el=>el===document.activeElement),'background read stole active draft focus');
    await p.getByRole('button',{name:'Ответить и завершить'}).waitFor();
  } finally {f.go.child.kill('SIGCONT'); await p.unroute(pattern,barrier);}
  // Read recovery must not silently renew or acquire a different lease.
  const fresh = p.waitForResponse(r => r.request().method() === 'GET' && new URL(r.url()).pathname === '/web/tasks/' + question);
  await p.getByRole('button', {name:'Повторить', exact:true}).click(); h.assert.equal((await fresh).status(), 200);
  await p.clock.setSystemTime(new Date(Date.now()-60000));
  const bodies=[]; const capture=r => {if(r.method()==='POST' && new URL(r.url()).pathname.endsWith('/answer')) bodies.push(r.postData());}; p.on('request',capture);
  f.go.child.kill('SIGSTOP');
  try {
    const began=p.waitForRequest(r => r.method()==='POST' && new URL(r.url()).pathname.endsWith('/answer'));
    await p.getByRole('button',{name:'Ответить и завершить'}).click(); await began; await p.clock.fastForward(16000); await p.getByRole('button',{name:'Повторить ту же отправку'}).waitFor();
    h.assert(await p.locator('[name=answer]').isDisabled()); h.assert.equal(await p.locator('[name=answer]').inputValue(),'Synthetic background read draft');
  } finally {f.go.child.kill('SIGCONT');}
  await h.posted(p,()=>p.getByRole('button',{name:'Повторить ту же отправку'}).click()); p.off('request',capture); await p.clock.setSystemTime(new Date());
  h.assert.equal(bodies.length,2); h.assert.equal(bodies[0],bodies[1]);
  const events=await h.call(f.worker,'events',{since_cursor:0,limit:200,task_key:question}); h.assert.equal(events.events.filter(e=>e.type==='human_answered').length,1);
  pass('real-sigstop-backgroundGET8s-draft-mutation15s-sameID-retry', {answer_body_sha256:h.sha(bodies[0]), timers:'Only browser timers accelerated; Go state/clock unchanged'});
};
