'use strict';
// HTTPS production inspection. Only own login, explicit project open and logout mutate.
const fs = require('node:fs');
const path = require('node:path');
const assert = require('node:assert/strict');
const crypto = require('node:crypto');
const required = name => {assert(process.env[name], 'missing ' + name); return process.env[name];};
const {actionResponse, diagnostic, finish} = require('../task009/production-support.cjs');
let stage = 'INPUT', base, token, tokenFile, out, chromium;
const report = {schema: 2, started_at: new Date().toISOString(), result: 'FAIL', checks: [], screenshots: [],
  page_errors: [], severe_console_errors: [], failed_assets: [], response_status_counts: {}, prefetch_count: 0,
  browser_mutation_kinds: [], limitations: ['One ' + process.platform + ' host; mobile is a Chrome viewport; physical second PC NOT-RUN.', 'Read-only production data; no task/repository/ACL/upload/lease changes.']};
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
const pass = (id, data = {}) => {report.checks.push({id, result: 'PASS', ...data}); console.log(id + ' PASS');};
let browser, context, page, authCleanupNeeded = false, project;
const assets = new Map();
const allowedMutations = new Set(['/web/login', '/web/logout', '/web/catalog/switch']);
async function json(route, headers = {}) {
  const response = await context.request.get(base + route, {timeout: 15000, headers: {Accept: 'application/json', ...headers}});
  expectStatus(response, 200); return response.json();
}
async function app(route) {
  const response = await page.goto(base + route); expectStatus(response, 200);
  await page.locator('h1').first().waitFor(); await page.locator('.app-loading').waitFor({state: 'detached'});
  await page.waitForFunction(() => !document.querySelector('.refresh-indicator'));
  return response;
}
async function logout() {
  await page.setViewportSize({width: 1440, height: 900}); await app('/app/projects');
  // The global button is available in either the catalog or a redirected control shell.
  const response = await actionResponse(page, '/web/logout', () => page.getByRole('button', {name: 'Выйти', exact: true}).click());
  expectStatus(response, 200); authCleanupNeeded = false;
  await page.waitForURL(base + '/app/login');
}
async function cleanupLogout() {
  // Fresh catalog capability works even if the UI failed before its logout button rendered.
  const response = await context.request.get(base + '/web/projects', {timeout: 15000, headers: {Accept: 'application/json'}});
  if (response.status() === 401) {authCleanupNeeded = false; return;}
  expectStatus(response, 200);
  const registry = (await response.json()).page.registry;
  assert(registry.csrf && registry.actions.logout);
  const ended = await context.request.post(base + '/web/logout', {timeout: 15000, headers: {Accept: 'application/json', Origin: base},
    form: {csrf: registry.csrf, action_id: registry.actions.logout}});
  expectStatus(ended, 200); authCleanupNeeded = false;
}
function expectStatus(response, expected) {
  if (response.status() !== expected) {const error = new Error(); error.status = response.status(); throw error;}
}
async function rpc(method, params, id) {
  assert(['initialize', 'notifications/initialized', 'tools/list', 'resources/templates/list', 'resources/read'].includes(method));
  const payload = {jsonrpc: '2.0', method, params}; if (id) payload.id = id;
  const response = await context.request.post(base + '/mcp/p/' + project, {timeout: 15000,
    headers: {Authorization: 'Bearer ' + token, 'Content-Type': 'application/json', Accept: 'application/json, text/event-stream', 'MCP-Protocol-Version': '2025-11-25'}, data: payload});
  expectStatus(response, id ? 200 : 202);
  if (!id) return;
  const text = await response.text(); const body = response.headers()['content-type']?.includes('text/event-stream') ? JSON.parse(text.split('\n').find(line => line.startsWith('data:')).slice(5)) : JSON.parse(text);
  assert(!body.error, 'scoped MCP server error'); return body.result;
}
async function main() {
  const candidateOut = path.resolve(required('ACCORD_QA_DIR'));
  assert(candidateOut.includes(path.sep + '.local' + path.sep), 'private output directory required');
  for (let parent = candidateOut; ; parent = path.dirname(parent)) {
    try {const info = fs.lstatSync(parent); assert(info.isDirectory() && !info.isSymbolicLink() && !(info.mode & 0o022), 'unsafe output ancestry');} catch (error) {if (error.code !== 'ENOENT') throw error;}
    if (parent === path.dirname(parent)) break;
  }
  assert(!fs.existsSync(candidateOut), 'NEW private output directory required'); fs.mkdirSync(candidateOut, {recursive: true, mode: 0o700}); out = candidateOut;
  ({chromium} = require(required('ACCORD_QA_PLAYWRIGHT')));
  const configFile = required('ACCORD_QA_CLIENT_FILE');
  tokenFile = required('ACCORD_QA_TOKEN_FILE');
  for (const file of [configFile, tokenFile]) {
    const info = fs.lstatSync(file); assert(info.isFile() && !info.isSymbolicLink() && info.uid === process.getuid() && (info.mode & 0o777) === 0o600, 'private owned input file required');
  }
  const config = JSON.parse(fs.readFileSync(configFile, 'utf8'));
  const url = new URL(config.origin);
  assert(url.protocol === 'https:' && url.pathname === '/' && !url.username && !url.password && !url.search && !url.hash, 'canonical HTTPS origin required');
  base = url.origin;
  token = fs.readFileSync(tokenFile, 'utf8').trim(); assert(token.length && token.length <= 4096);
  report.revision = required('ACCORD_QA_REVISION'); assert.match(report.revision, /^[a-f0-9]{40}$/);
  stage = 'BROWSER_START';
  browser = await chromium.launch({executablePath: required('ACCORD_QA_CHROME'), headless: true});
  report.browser = browser.version(); report.node = process.version;
  context = await browser.newContext({viewport: {width: 1440, height: 900}, colorScheme: 'light'});
  await context.route('**/*', route => {
    const request = route.request(), target = new URL(request.url());
    if (target.origin === base && !['GET', 'HEAD'].includes(request.method()) && !allowedMutations.has(target.pathname)) {
      report.unexpected_mutation_blocked = true; return route.abort('blockedbyclient');
    }
    return route.continue();
  });
  page = await context.newPage(); page.setDefaultTimeout(15000); page.setDefaultNavigationTimeout(30000);
  page.on('pageerror', () => report.page_errors.push('PAGE_ERROR'));
  page.on('console', message => {if (message.type() === 'error' && /hydration|Content Security Policy|Refused to|Minified React|react\.dev/i.test(message.text())) report.severe_console_errors.push('CSP_OR_REACT_ERROR');});
  page.on('request', request => {
    const target = new URL(request.url()); if (target.origin !== base) return;
    if (!['GET', 'HEAD'].includes(request.method())) report.browser_mutation_kinds.push(allowedMutations.has(target.pathname) ? target.pathname : 'BLOCKED_UNEXPECTED');
    const headers = request.headers(); if (headers['next-router-prefetch'] === '1' || /prefetch/i.test(headers.purpose || headers['sec-purpose'] || '')) report.prefetch_count++;
  });
  page.on('requestfailed', request => {const target = new URL(request.url()); if (target.origin === base && /^(\/_next\/static\/|\/fonts\/)/.test(target.pathname)) report.failed_assets.push({kind: 'failed_asset'});});
  page.on('response', response => {
    const target = new URL(response.url()); if (target.origin !== base) return;
    const status = String(response.status()); report.response_status_counts[status] = (report.response_status_counts[status] || 0) + 1;
    if (/^(\/_next\/static\/|\/fonts\/)/.test(target.pathname)) {assets.set(target.pathname, response.status()); if (response.status() >= 400) report.failed_assets.push({kind: target.pathname.startsWith('/fonts/') ? 'font' : 'static', status: response.status()});}
  });
  stage = 'PUBLIC_ROOT';
  const root = await context.request.get(base + '/', {maxRedirects: 0, timeout: 15000});
  assert([301, 302, 307, 308].includes(root.status())); assert.equal(new URL(root.headers().location, base).pathname, '/app');
  assert.equal((await context.request.get(base + '/api/v1/tasks', {timeout: 15000})).status(), 401);
  assert.equal((await context.request.get(base + '/metrics', {timeout: 15000})).status(), 404);
  pass('HTTPS-root-native401-private-metrics404');
  stage = 'LOGIN_PAGE_CSP';
  const loginPage = await app('/app/login');
  const headers = loginPage.headers(), csp = headers['content-security-policy'] || '';
  const script = csp.split(';').find(value => value.trim().startsWith('script-src')) || '';
  const nonce = script.match(/'nonce-([^']+)'/); assert(nonce); assert(!/unsafe-inline|unsafe-eval/.test(script));
  assert((await loginPage.text()).includes('nonce="' + nonce[1] + '"')); assert.match(headers['cache-control'], /no-store/);
  await page.getByRole('button', {name: 'Файл токена', exact: true}).click();
  await page.locator('input[name=token_file]').setInputFiles(tokenFile);
  stage = 'LOGIN_SUBMIT'; authCleanupNeeded = true;
  const login = await actionResponse(page, '/web/login', () => page.getByRole('button', {name: 'Войти в Accord', exact: true}).click());
  expectStatus(login, 200);
  stage = 'LOGIN_REDIRECT';
  await page.waitForURL(/\/app\/(?:p\/p_[a-f0-9]{32}|projects)/);
  stage = 'AUTH_CATALOG';
  const catalog = (await json('/web/projects')).page; assert(catalog.registry.projects.length, 'own accessible project required');
  stage = 'AUTH_DEFAULT_METADATA';
  const legacy = (await json('/web/')).page; assert(legacy.project?.id && legacy.session?.session_id, 'this inspection requires authorized default access');
  project = legacy.project.id; assert(catalog.registry.projects.some(entry => entry.id === project));
  assert.equal(legacy.project.can_archive, false, 'default project offered archival');
  stage = 'AUTH_COOKIE';
  const cookie = (await context.cookies()).filter(value => value.name === '__Host-accord_session'); assert.equal(cookie.length, 1);
  assert(cookie[0].secure && cookie[0].httpOnly && cookie[0].sameSite === 'Strict' && cookie[0].path === '/');
  stage = 'AUTH_LOCAL_STORAGE';
  assert.equal(await page.evaluate(secret => Object.values(localStorage).some(value => value.includes(secret)), token), false);
  pass('HTTPS-existing-token-scoped-login-default-preserved-secure-cookie-CSP');
  stage = 'PROJECT_CATALOG_UI';
  await app('/app/projects');
  await page.locator('.catalog-workspace').waitFor();
  const card = page.locator('article.project-card').filter({has: page.locator(`a[href="/app/projects/${project}"]`)});
  stage = 'PROJECT_SWITCH';
  const switched = await actionResponse(page, '/web/catalog/switch', () => card.getByRole('button', {name: 'Открыть проект', exact: true}).click()); expectStatus(switched, 200);
  await page.waitForURL(new RegExp('/app/p/' + project)); await page.locator('.control-shell').waitFor();
  stage = 'NATIVE_READS';
  const before = (await json('/web/p/' + project + '/')).page;
  const nativeHeaders = {Authorization: 'Bearer ' + token, 'X-Session': before.session.session_id};
  const native = await json('/api/v1/p/' + project + '/tasks?limit=1', nativeHeaders); assert(Array.isArray(native.tasks));
  for (const origin of ['', 'null', base]) assert.equal((await context.request.get(base + '/api/v1/p/' + project + '/tasks', {timeout: 15000, headers: {...nativeHeaders, Origin: origin}})).status(), 403);
  stage = 'MCP_READS';
  await rpc('initialize', {protocolVersion: '2025-11-25', capabilities: {}, clientInfo: {name: 'accord-projects-readonly-inspection', version: '1'}}, 1);
  await rpc('notifications/initialized', {});
  assert.equal((await rpc('tools/list', {}, 2)).tools.length, 12);
  assert((await rpc('resources/templates/list', {}, 3)).resourceTemplates.length >= 3);
  const resource = await rpc('resources/read', {uri: 'accord://designs?session_id=' + before.session.session_id + '&limit=1'}, 4); assert(resource.contents.length);
  const designs = JSON.parse(resource.contents[0].text); assert(Array.isArray(designs.assets));
  for (const asset of designs.assets) assert(asset.download_url.startsWith('/api/v1/p/' + project + '/'));
  pass('HTTPS-explicit-project-open-native-reads-Origin-denied-scoped-MCP12-resource');
  const prefix = '/app/p/' + project;
  const screens = {profiles: prefix + '/profiles', ownSession: prefix + '/agents/' + before.session.session_id, overview: prefix, tasks: prefix + '/tasks', questions: prefix + '/questions', agents: prefix + '/agents', activity: prefix + '/activity', stats: prefix + '/stats', designs: prefix + '/designs', history: prefix + '/history', settings: prefix + '/settings', catalog: '/app/projects', repositories: '/app/projects/' + project};
  stage = 'PROFILE_BROWSER_READ';
  const profiles = (await json('/web/p/' + project + '/profiles')).page;
  // Browser DTO omits an empty collection; native ProfilesPage keeps its array.
  const profileList = profiles.profiles === undefined ? [] : profiles.profiles;
  assert(Array.isArray(profileList)); assert.equal(profiles.project.id, project);
  assert(profileList.every(profile => !Object.hasOwn(profile, 'instructions')));
  assert.equal(profiles.can_manage_profiles, profiles.project.can_write && !profiles.can_continue);
  stage = 'PROFILE_RESOURCE_READ';
  const profileResource = await rpc('resources/read', {uri: 'accord://profiles?session_id=' + before.session.session_id + '&state=all'}, 5);
  assert(Array.isArray(JSON.parse(profileResource.contents[0].text).profiles));
  stage = 'PROFILE_OWNER_CONTROLS';
  await app(prefix + '/profiles');
  if (profiles.can_manage_profiles) assert.equal(await page.getByRole('button', {name: 'Новый профиль', exact: true}).count(), 1);
  if (profileList.length) screens.profileDetail = prefix + '/profiles/' + profileList[0].id;
  pass('HTTPS-profile-storage-scoped-resource-ownercontrols-without-data-mutation');
  for (const theme of ['light', 'dark']) {
    await page.setViewportSize({width: 1440, height: 900}); await app(prefix);
    if ((await page.locator('html').getAttribute('data-theme') || 'light') !== theme) await page.getByRole('button', {name: theme === 'dark' ? 'Тёмная тема' : 'Светлая тема', exact: true}).click();
    for (const width of [1440, 390]) {
      await page.setViewportSize({width, height: width === 390 ? 844 : 900});
      for (const [name, route] of Object.entries(screens)) {
        stage = 'UI_' + name.toUpperCase() + '_' + width + '_' + theme.toUpperCase();
        await app(route); await page.evaluate(() => document.fonts.ready);
        const measured = await page.evaluate(() => ({width: innerWidth, document: document.documentElement.scrollWidth, theme: document.documentElement.dataset.theme, scale: visualViewport.scale}));
        assert(measured.document <= width + 1 && measured.scale === 1 && measured.theme === theme, 'responsive UI inspection failed');
        const file = name + '-' + width + '-' + theme + '.png'; await page.screenshot({path: path.join(out, file), fullPage: true, animations: 'disabled'});
        report.screenshots.push({file, sha256: sha(fs.readFileSync(path.join(out, file)))});
      }
    }
  }
  await page.setViewportSize({width: 1440, height: 900}); await app(prefix);
  stage = 'READONLY_NAVIGATION';
  const tasks = page.locator(`.desktop-sidebar a[href="${prefix}/tasks"]`); await tasks.hover(); await page.waitForTimeout(500);
  assert.equal(report.prefetch_count, 0);
  await tasks.click(); await page.waitForURL(base + prefix + '/tasks'); await page.locator('h1').first().waitFor();
  await page.keyboard.press('Meta+k'); await page.getByRole('dialog').waitFor(); await page.keyboard.press('Escape');
  const after = (await json('/web/p/' + project + '/')).page;
  assert.equal(after.session.session_id, before.session.session_id); assert.equal(after.session.expires_at, before.session.expires_at, 'reads renewed native session');
  assert.equal((await context.request.get(base + '/web/', {timeout: 15000})).status(), 200);
  assert.equal((await context.request.get(base + '/PAPERCLIP-LICENSE.txt', {timeout: 15000})).status(), 200);
  assert([...assets.keys()].some(name => name.endsWith('.js')) && [...assets.keys()].some(name => name.endsWith('.css')) && [...assets.keys()].some(name => name.startsWith('/fonts/')));
  assert.deepEqual(report.page_errors, []); assert.deepEqual(report.severe_console_errors, []); assert.deepEqual(report.failed_assets, []);
  assert(!report.unexpected_mutation_blocked, 'unexpected production mutation was blocked');
  assert(!Object.keys(report.response_status_counts).some(status => Number(status) >= 400), 'unexpected browser request failures');
  pass('HTTPS-profile-and-existing-screens-1440-390-themes-no-errors-overflow-prefetch-renew', {screenshots: report.screenshots.length, assets: assets.size});
  stage = 'LOGOUT_UI';
  await logout(); assert.equal((await context.request.get(base + '/web/projects', {timeout: 15000, headers: {Accept: 'application/json'}})).status(), 401);
  assert(report.browser_mutation_kinds.every(value => allowedMutations.has(value))); pass('HTTPS-own-browser-global-logout'); report.result = 'PASS';
}
main().catch(async error => {
  report.failure = diagnostic(error, stage); report.result = 'FAIL';
  if (page) await page.screenshot({path: path.join(out, 'failure.png'), fullPage: true, animations: 'disabled'}).catch(() => {});
}).finally(async () => {
  await finish(report, {logout: authCleanupNeeded ? cleanupLogout : null, close: () => browser?.close(), write: value => {
    value.script_sha256 = sha(fs.readFileSync(__filename));
    value.support_sha256 = sha(fs.readFileSync(path.join(__dirname, '../task009/production-support.cjs')));
    if (out) fs.writeFileSync(path.join(out, 'production-readonly-results.json'), JSON.stringify(value, null, 2), {mode: 0o600});
    else console.error('Production inspection failed before private output validation.');
  }});
  if (report.result !== 'PASS') process.exitCode = 1;
}).catch(() => {console.error('Production inspection report could not be persisted.'); process.exitCode = 1;});
