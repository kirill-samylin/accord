'use strict';
// HTTPS production read-only smoke. Only own browser login/logout may mutate.
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const required = name => {assert(process.env[name], 'missing ' + name); return process.env[name];};
const {chromium} = require(required('ACCORD_QA_PLAYWRIGHT'));
const out = path.resolve(required('ACCORD_QA_DIR'));
assert(out.includes(path.sep + '.local' + path.sep), 'private output directory required');
fs.mkdirSync(out, {recursive:true, mode:0o700}); fs.chmodSync(out, 0o700);
const config = JSON.parse(fs.readFileSync(required('ACCORD_QA_CLIENT_FILE'), 'utf8'));
const origin = new URL(config.origin);
assert.equal(origin.protocol, 'https:'); assert.equal(origin.pathname, '/');
assert(!origin.username && !origin.password && !origin.search && !origin.hash);
const base = origin.origin;
const tokenFile = required('ACCORD_QA_TOKEN_FILE');
assert.equal(fs.statSync(tokenFile).mode & 0o777, 0o600, 'token permissions');
const token = fs.readFileSync(tokenFile, 'utf8').trim(); assert(token.length && token.length <= 4096);
const revision = required('ACCORD_QA_REVISION'); assert.match(revision, /^[a-f0-9]{40}$/);
const report = {schema:1, revision, started_at:new Date().toISOString(), result:'FAIL', checks:[], measurements:[], screenshots:[], page_errors:[], severe_console_errors:[], failed_assets:[], mutations:[], limitations:['Production HTTPS from one physical macOS host; mobile is a Chrome viewport.', 'Read-only production data; task/upload/lease mutations are outside this smoke.']};
const sha = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const pass = (id, data={}) => {report.checks.push({id,result:'PASS',...data}); console.log(id+' PASS');};
let browser, ctx, page, loggedIn=false;
const assets = new Map();
async function app(route) {
  const response = await page.goto(base+route);
  assert.equal(response.status(), 200, 'app HTML status '+route);
  await page.locator('h1').first().waitFor();
  await page.locator('.app-loading').waitFor({state:'detached'});
  await page.waitForFunction(() => !document.querySelector('.refresh-indicator'));
  return response;
}
async function json(route, headers={}) {
  const response = await ctx.request.get(base+route,{headers:{Accept:'application/json',...headers}});
  assert.equal(response.status(),200,'JSON status '+route);
  return response.json();
}
async function logout() {
  await page.setViewportSize({width:1440,height:900});
  await app('/app');
  await page.getByRole('button',{name:'Справка и сессия',exact:true}).click();
  const pending = page.waitForResponse(r => new URL(r.url()).pathname==='/web/logout' && r.request().method()==='POST');
  await page.getByRole('button',{name:'Выйти',exact:true}).click();
  assert.equal((await pending).status(),200,'own logout status');
  await page.waitForURL(base+'/app/login'); loggedIn=false;
}
async function main() {
  browser=await chromium.launch({executablePath:required('ACCORD_QA_CHROME'),headless:true});
  report.browser=browser.version(); report.node=process.version;
  ctx=await browser.newContext({viewport:{width:1440,height:900},colorScheme:'light'});
  page=await ctx.newPage(); page.setDefaultTimeout(15000); page.setDefaultNavigationTimeout(30000);
  page.on('pageerror',error=>report.page_errors.push(String(error.message).slice(0,500)));
  page.on('console',message=>{if(message.type()==='error' && /hydration|Content Security Policy|Refused to|Minified React|react\.dev/i.test(message.text())) report.severe_console_errors.push(message.text().slice(0,500));});
  page.on('request',request=>{if(new URL(request.url()).origin===base && !['GET','HEAD'].includes(request.method())) report.mutations.push({method:request.method(),path:new URL(request.url()).pathname});});
  page.on('response',response=>{const url=new URL(response.url()); if(url.origin===base && /^(\/_next\/static\/|\/fonts\/)/.test(url.pathname)){assets.set(url.pathname,response.status()); if(response.status()>=400)report.failed_assets.push({path:url.pathname,status:response.status()});}});
  const root=await ctx.request.get(base+'/',{maxRedirects:0});
  assert([301,302,307,308].includes(root.status()),'root redirect status');
  assert.equal(new URL(root.headers().location,base).pathname,'/app');
  assert.equal((await ctx.request.get(base+'/api/v1/tasks')).status(),401);
  assert.equal((await ctx.request.get(base+'/metrics')).status(),404);
  pass('https-root-app-native401-metrics404');
  const response=await app('/app/login'); const headers=response.headers();
  const csp=headers['content-security-policy']||'';
  const script=csp.split(';').find(value=>value.trim().startsWith('script-src'))||'';
  const nonce=script.match(/'nonce-([^']+)'/); assert(nonce,'script nonce missing');
  assert(!/unsafe-inline|unsafe-eval/.test(script),'unsafe script policy');
  assert((await response.text()).includes('nonce="'+nonce[1]+'"'),'HTML nonce mismatch');
  assert.match(headers['cache-control'],/no-store/); assert.equal(headers['x-content-type-options'],'nosniff');
  pass('https-next-csp-nonce-no-store');
  await page.getByRole('button',{name:'Файл токена',exact:true}).click();
  await page.locator('[name=token_file]').setInputFiles(tokenFile);
  const login=page.waitForResponse(r=>new URL(r.url()).pathname==='/web/login' && r.request().method()==='POST');
  await page.getByRole('button',{name:'Войти в Accord',exact:true}).click();
  assert.equal((await login).status(),200,'real browser login'); loggedIn=true;
  await page.waitForURL(base+'/app'); await page.locator('.sidebar').first().waitFor();
  const cookies=(await ctx.cookies()).filter(cookie=>cookie.name.includes('session'));
  assert.equal(cookies.length,1,'one own browser session cookie');
  for(const cookie of cookies){assert(cookie.httpOnly && cookie.secure);assert.equal(cookie.sameSite,'Strict');assert.equal(cookie.path,'/');}
  assert.equal(await page.evaluate(secret=>Object.values(localStorage).some(value=>value.includes(secret)),token),false,'token persisted in localStorage');
  pass('https-existing-token-browser-login-secure-cookie');
  const before=(await json('/web/')).page; assert(before?.session?.session_id);
  const native=await json('/api/v1/tasks?limit=1',{Authorization:'Bearer '+token,'X-Session':before.session.session_id});
  assert(Array.isArray(native.tasks));
  async function rpc(method,params,id) {
    const payload={jsonrpc:'2.0',method,params};if(id)payload.id=id;
    const result=await ctx.request.post(base+'/mcp',{headers:{Authorization:'Bearer '+token,'Content-Type':'application/json',Accept:'application/json, text/event-stream','MCP-Protocol-Version':'2025-11-25'},data:payload});
    assert.equal(result.status(),id?200:202,'MCP '+method+' status');
    if(id){const body=await result.json();assert(!body.error,'MCP protocol rejected');return body.result;}
  }
  const initialized=await rpc('initialize',{protocolVersion:'2025-11-25',capabilities:{},clientInfo:{name:'accord-vps-readonly-qa132',version:'1'}},1);
  assert.equal(initialized.protocolVersion,'2025-11-25'); await rpc('notifications/initialized',{});
  const tools=await rpc('tools/list',{},2);assert.equal(tools.tools.length,12);
  pass('https-native-own-session-read-mcp12-no-describe',{native_rows:native.tasks.length,tools:12});
  const routes={overview:'/app',tasks:'/app/tasks',questions:'/app/questions',projects:'/app/projects',agents:'/app/agents',activity:'/app/activity',stats:'/app/stats',designs:'/app/designs',history:'/app/history',settings:'/app/settings'};
  for(const theme of ['light','dark']) {
    await page.setViewportSize({width:1440,height:900});await app('/app');
    if((await page.locator('html').getAttribute('data-theme')||'light')!==theme) await page.getByRole('button',{name:theme==='dark'?'Тёмная тема':'Светлая тема',exact:true}).click();
    for(const width of [1440,390]) {
      await page.setViewportSize({width,height:width===390?844:900});
      for(const [name,route] of Object.entries(routes)) {
        await app(route);await page.evaluate(()=>document.fonts.ready);
        assert(await page.evaluate(()=>document.fonts.check('14px Inter')),'Inter font unavailable');
        const measured=await page.evaluate(()=>({document:document.documentElement.scrollWidth,viewport:innerWidth,scale:visualViewport.scale,theme:document.documentElement.dataset.theme}));
        assert(measured.document<=width,'horizontal page overflow '+name);assert.equal(measured.scale,1);assert.equal(measured.theme,theme);
        if(width===390){assert(await page.locator('.mobile-nav').isVisible());assert(await page.getByRole('button',{name:'Открыть навигацию',exact:true}).isVisible());}
        else assert(await page.locator('.desktop-sidebar').isVisible());
        const file=name+'-'+width+'-'+theme+'.png'; await page.screenshot({path:path.join(out,file),fullPage:true,animations:'disabled'});
        report.screenshots.push({file,sha256:sha(fs.readFileSync(path.join(out,file)))});report.measurements.push({name,width,theme,...measured});
      }
      if(width===390){await page.getByRole('button',{name:'Открыть навигацию',exact:true}).click();await page.getByRole('dialog').waitFor();await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'detached'});}
      await page.keyboard.press('Meta+k');await page.getByRole('dialog').waitFor();await page.getByPlaceholder('Найти задачу или перейти к разделу…').waitFor();await page.keyboard.press('Escape');await page.getByRole('dialog').waitFor({state:'detached'});
    }
  }
  pass('https-10screens-light-dark-desktop-mobile-navigation-font-overflow',{screenshots:report.screenshots.length,screens:10});
  const after=(await json('/web/')).page;
  assert.equal(after.session.session_id,before.session.session_id);assert.equal(after.session.expires_at,before.session.expires_at,'reads renewed own board session');
  assert.equal((await ctx.request.get(base+'/web/')).status(),200,'legacy SSR');
  const license=await ctx.request.get(base+'/PAPERCLIP-LICENSE.txt');assert.equal(license.status(),200);assert.match(await license.text(),/MIT License/);
  assert([...assets.keys()].some(name=>name.endsWith('.js')));assert([...assets.keys()].some(name=>name.endsWith('.css')));assert([...assets.keys()].some(name=>name.startsWith('/fonts/')));
  assert.deepEqual(report.page_errors,[]);assert.deepEqual(report.severe_console_errors,[]);assert.deepEqual(report.failed_assets,[]);
  assert(report.mutations.every(row=>['/web/login','/web/logout'].includes(row.path)),'unexpected board mutation');
  pass('https-direct-go-legacy-assets-no-renew-no-runtime-errors',{assets:assets.size});
  await logout();assert.equal((await ctx.request.get(base+'/web/tasks',{headers:{Accept:'application/json'}})).status(),401);
  assert.deepEqual(report.mutations.map(row=>row.path),['/web/login','/web/logout'],'unexpected browser mutation');
  pass('https-own-browser-logout-cookie-invalid');report.result='PASS';
}
main().catch(error=>{report.failure=String(error.stack).slice(0,1600);console.error('production smoke FAIL; private report contains diagnostics');process.exitCode=1;}).finally(async()=>{
  if(loggedIn){try{await logout();report.cleanup_logout='PASS';}catch{report.cleanup_logout='FAIL';report.result='FAIL';process.exitCode=1;}}
  await browser?.close();report.finished_at=new Date().toISOString();report.script_sha256=sha(fs.readFileSync(__filename));report.asset_statuses=Object.fromEntries(assets);
  fs.writeFileSync(path.join(out,'next-vps-results.json'),JSON.stringify(report,null,2),{mode:0o600});
});
