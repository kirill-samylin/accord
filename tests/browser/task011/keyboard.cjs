'use strict';
const {assert}=require('../../process/task011/harness.cjs');
async function run(f,owner,pass){
  const p=owner.page;
  await p.getByRole('tab',{name:'Координация',exact:true}).click();
  const trigger=p.getByRole('button',{name:'Изменить назначение',exact:true});
  const before=f.aNative.call('get',{task_key:f.uiTask.key});
  const writes=[];
  const watch=request=>{if(request.method()==='POST'&&/\/tasks(?:\/|$)/.test(new URL(request.url()).pathname))writes.push(new URL(request.url()).pathname);};
  p.on('request',watch);
  try{
    await trigger.focus();assert(await trigger.evaluate(el=>el===document.activeElement));
    await p.keyboard.press('Enter');
    const dialog=p.getByRole('dialog');await dialog.waitFor();
    await dialog.getByRole('heading',{name:'Назначение задачи · '+f.uiTask.key,exact:true}).waitFor();
    assert(await dialog.evaluate(el=>el.contains(document.activeElement)),'autofocus escaped coordination dialog');
    const stops=await dialog.evaluate(el=>[...el.querySelectorAll('button,input,select,textarea,a[href],[tabindex]')].filter(n=>!n.disabled&&n.tabIndex>=0&&n.getClientRects().length).length);
    assert(stops>=3,'coordination dialog lacks expected keyboard controls');
    for(const key of ['Tab','Shift+Tab'])for(let i=0;i<=stops;i++){
      await p.keyboard.press(key);
      assert(await dialog.evaluate(el=>el.contains(document.activeElement)),key+' escaped coordination focus trap');
    }
    await p.keyboard.press('Escape');await dialog.waitFor({state:'hidden'});
    await p.waitForFunction(()=>document.activeElement?.textContent?.trim()==='Изменить назначение');
    assert(await trigger.evaluate(el=>el===document.activeElement),'Escape did not restore original trigger focus');
    assert.deepEqual(writes,[],'keyboard draft-only test submitted business mutation');
    const after=f.aNative.call('get',{task_key:f.uiTask.key});assert.equal(after.version,before.version);assert.deepEqual(after.assignment,before.assignment);
    pass('Chrome-coordination-keyboard-Enter-autofocus-Tab-ShiftTab-Escape-focusreturn-no-businesswrite');
  }finally{p.off('request',watch);}
}
module.exports={run};
