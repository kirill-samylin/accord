'use strict';
const h = require('./next-fixture.cjs');
async function focused(p, target) {
  for(let i=0;i<40;i++){if(await target.evaluate(el=>el===document.activeElement))return; await h.sleep(30);}
  const tag=await p.evaluate(()=>document.activeElement?.tagName); throw new Error('focus not returned to expected control; active '+tag);
}
async function close(p, target) {await p.keyboard.press('Escape'); await p.getByRole('dialog').waitFor({state:'detached'}); await focused(p,target);}
module.exports=async(f,pass)=>{
  const p=f.p, sizes=[]; await p.emulateMedia({reducedMotion:'reduce'});
  for(const [width,height] of [[1440,900],[1440,500],[768,500]]){
    await p.setViewportSize({width,height}); await h.goto(f,'/app');
    const measured=await p.evaluate(()=>({height:document.querySelector('.desktop-sidebar .sidebar').getBoundingClientRect().height, viewport:innerHeight, footer:document.querySelector('.desktop-sidebar .sidebar-bottom').getBoundingClientRect().bottom, scroll:document.querySelector('.desktop-sidebar .sidebar-nav').scrollHeight, client:document.querySelector('.desktop-sidebar .sidebar-nav').clientHeight}));
    h.assert(Math.abs(measured.height-height)<1);h.assert(measured.footer<=height+1);if(height===500){h.assert(measured.scroll>measured.client);await p.locator('.desktop-sidebar .sidebar-nav').evaluate(el=>{el.scrollTop=el.scrollHeight;});}
    await p.screenshot({animations:'disabled',path:h.path.join(h.root,`sidebar-${width}-${height}.png`)}); sizes.push({width,height,...measured});
  }
  await p.setViewportSize({width:1440,height:900}); await h.goto(f,'/app/tasks');
  const field=p.getByRole('textbox',{name:'Поиск задач',exact:true}); await field.fill('Сохраняемый текст'); await field.focus(); await p.keyboard.press('Meta+k'); await p.getByRole('dialog').waitFor(); await close(p,field);h.assert.equal(await field.inputValue(),'Сохраняемый текст');
  const create=p.getByRole('button',{name:'Новая задача',exact:true}).first();await create.click();await p.getByRole('dialog').waitFor();await close(p,create);
  const peek=p.getByRole('button',{name:'Быстрый просмотр '+f.task,exact:true});await peek.click();await p.locator('.task-peek h1').waitFor();await close(p,peek);
  await h.goto(f,'/app/tasks/'+f.task);const edit=p.getByRole('button',{name:'Изменить',exact:true});await edit.click();await p.getByRole('dialog').waitFor();await close(p,edit);
  await h.goto(f,'/app/designs');const upload=p.getByRole('button',{name:'Загрузить',exact:true});await upload.click();await p.getByRole('dialog').waitFor();await close(p,upload);
  const preview=p.getByRole('button',{name:'Просмотреть Synthetic synthetic.png, версия 1',exact:true});await preview.click();await p.getByRole('dialog').waitFor();await close(p,preview);
  for(const width of [767,390,320]){
    await p.setViewportSize({width,height:844}); await h.goto(f,'/app/tasks'); const burger=p.getByRole('button',{name:'Открыть навигацию',exact:true});const header=p.locator('.breadcrumb-actions button[data-command-trigger]');h.assert(await burger.isVisible());h.assert(await header.isVisible());
    await burger.click();await p.getByRole('dialog').waitFor();await close(p,burger);
    await burger.click();const drawer=p.getByRole('dialog');await drawer.getByRole('button',{name:/^Поиск/}).click();const command=p.getByRole('dialog',{name:'Поиск и навигация',exact:true});await command.waitFor();const input=p.getByPlaceholder('Найти задачу или перейти к разделу…');await focused(p,input);await close(p,header);
    await burger.click();await p.getByRole('dialog').getByRole('button',{name:/^Поиск/}).click();await command.waitFor();await input.fill('ёжик');await p.getByRole('option').filter({hasText:f.tasks[111].key}).waitFor();await p.keyboard.press('ArrowDown');await p.keyboard.press('Enter');await p.waitForURL(f.origin+'/app/tasks/'+f.tasks[111].key);await command.waitFor({state:'detached'});await p.getByRole('heading',{name:f.tasks[111].title,exact:true}).waitFor();
    let routeFocus;for(let i=0;i<40;i++){routeFocus=await p.evaluate(()=>{const el=document.activeElement;return {usable:!!(el&&el!==document.body&&el.isConnected&&el.getClientRects().length>0&&!el.closest('[inert]')),tag:el?.tagName,aria:el?.getAttribute('aria-label'),text:el?.textContent?.slice(0,80)};});if(routeFocus.usable)break;await h.sleep(30);}h.fs.writeFileSync(h.path.join(h.root,`route-focus-${width}.json`),JSON.stringify(routeFocus,null,2));if(!routeFocus.usable)f.findings.push({id:'route-selection-focus-'+width,result:'FAIL',...routeFocus}); await p.screenshot({animations:'disabled',path:h.path.join(h.root,`drawer-command-route-${width}.png`)});
  }
  await p.setViewportSize({width:1440,height:900});await h.goto(f,'/app/activity');await p.getByRole('button',{name:/^Фильтры/}).click();const options=await p.locator('.filter-popover select[name=type] option').evaluateAll(els=>els.map(el=>el.value).filter(Boolean));h.assert.equal(options.length,27);h.assert.equal(new Set(options).size,27);h.assert(options.includes('blocked_changed'));h.assert(options.includes('tokens_revoked_all'));
  await p.locator('.filter-popover select[name=type]').selectOption('blocked_changed');await p.getByRole('button',{name:'Применить',exact:true}).click();await p.waitForURL(/type=blocked_changed/);await p.locator('.event-title').filter({hasText:'Блокировка изменена'}).first().waitFor();h.assert(!/blocked_changed/.test(await p.locator('main').innerText()));
  const actual=await h.json(f,'/web/activity?type=blocked_changed');h.assert(actual.activity.length>0&&actual.activity.every(e=>e.type==='blocked_changed'));await p.screenshot({animations:'disabled',path:h.path.join(h.root,'activity-blocked-localized.png')});
  h.fs.writeFileSync(h.path.join(h.root,'next-focus-sidebar-measurements.json'),JSON.stringify({sizes,event_types:options},null,2),{mode:0o600});
  pass('controlled-dialogs-input-create-edit-peek-upload-preview-focus-restored');if(!f.findings.some(item=>item.id.startsWith('route-selection-focus-')))pass('mobile-drawer-command-handoff-route-selection-focus-767-390-320');pass('sidebar-fullheight-lowheight-scroll-and-27-native-event-filters-localized', {sizes});
};
