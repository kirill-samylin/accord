'use strict';
// Synthetic real-process harness; credentials stay in private files and memory.
const fs = require('node:fs');
const path = require('node:path');
const cp = require('node:child_process');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const readline = require('node:readline');
const net = require('node:net');
const repo = path.resolve(__dirname, '../../..');
const root = path.resolve(process.env.ACCORD_TASK013_QA_DIR || path.join(repo, '.local/task013-qa', 'run-' + Date.now()));
assert(root.startsWith(path.join(repo, '.local/task013-qa') + path.sep), 'private QA directory required');
for (let parent = root; ; parent = path.dirname(parent)) {
  if (fs.existsSync(parent) || (() => {try {fs.lstatSync(parent); return true;} catch {return false;}})()) {
    const info = fs.lstatSync(parent);
    assert(info.isDirectory() && !info.isSymbolicLink() && !(info.mode & 0o022), 'unsafe fixture directory ancestry');
    if (parent.startsWith(path.join(repo, '.local'))) assert.equal(info.uid, process.getuid(), 'fixture directory owner differs');
  }
  if (parent === path.dirname(parent)) break;
}
assert(!fs.existsSync(root), 'choose a NEW private QA run directory');
fs.mkdirSync(root, {recursive: true, mode: 0o700});
assert.equal(fs.statSync(root).mode & 0o777, 0o700);
const candidateBinary = path.resolve(process.env.ACCORD_TASK013_BINARY || path.join(repo, 'bin/agent-board'));
const binary = path.join(root, 'agent-board');
fs.copyFileSync(candidateBinary, binary, fs.constants.COPYFILE_EXCL);
fs.chmodSync(binary, 0o700);
const oldBinary = path.resolve(process.env.ACCORD_TASK013_OLD_BINARY || path.join(repo, '.local/task013/legacy/agent-board'));
const env = {...process.env, PYTHONDONTWRITEBYTECODE: '1', NEXT_TELEMETRY_DISABLED: '1'};
for (const key of Object.keys(env)) if (key.startsWith('AGENT_BOARD_') || key.startsWith('ACCORD_') && !key.startsWith('ACCORD_TASK013_') && !key.startsWith('ACCORD_QA_')) delete env[key];
env.HOME = path.join(root, 'home');
fs.mkdirSync(env.HOME, {mode: 0o700});
const origin = 'http://127.0.0.1:3100';
const goOrigin = 'http://127.0.0.1:3101';
const sha = data => crypto.createHash('sha256').update(data).digest('hex');
const save = (file, value) => fs.writeFileSync(file, JSON.stringify(value, null, 2), {mode: 0o600});
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
function cli(args, input, bin = binary, allowFailure = false) {
  const result = cp.spawnSync(bin, args, {input: input && JSON.stringify(input), env, encoding: 'utf8', timeout: 30000});
  if (!allowFailure) assert.equal(result.status, 0, `CLI ${args[0]} rejected (output suppressed)`);
  return result;
}
function admin(state, op, args = [], bin = binary) {
  return cli(['projects', op, '--projects-data', path.join(state, 'projects-data'), '--db', path.join(state, 'board.db'), '--web-data', path.join(state, 'web'), ...args], undefined, bin);
}
function legacyToken(state, name, bin = binary) {
  const file = path.join(root, name + '.token');
  const result = cli(['token', 'create', '--db', path.join(state, 'board.db'), '--owner', name, '--machine', name, '--output', file], undefined, bin);
  const match = result.stdout.match(/^Token (\S+) saved/); assert(match, 'legacy token ID missing');
  assert.equal(fs.statSync(file).mode & 0o777, 0o600);
  return {file, id: match[1], machine: name};
}
function catalogToken(state, name, bin = binary) {
  const file = path.join(root, name + '.token');
  const output = JSON.parse(admin(state, 'token-create', ['--owner', name, '--machine', name, '--output', file], bin).stdout);
  return {file, id: output.identity.id, machine: name};
}
async function startServer(state, {legacy = false, bin = binary, mode = 'warn'} = {}) {
  const args = ['serve', '--db', path.join(state, 'board.db'), '--listen', '127.0.0.1:3101', '--web-origin', origin,
    '--web-data', path.join(state, 'web'), '--duplicate-mode', mode, '--task-ttl', '30m', '--session-ttl', '30m'];
  if (!legacy) args.push('--projects-data', path.join(state, 'projects-data'));
  return start(bin, args, '/readyz', goOrigin);
}
async function startNext() {
  return start(process.execPath, [path.join(repo, 'web/node_modules/next/dist/bin/next'), 'start', '--hostname', '127.0.0.1', '--port', '3100'], '/app/login', origin, path.join(repo, 'web'));
}
async function freePorts(ports = [3100, 3101]) {
  for (const number of ports) {
    const probe = net.createServer();
    await new Promise((resolve, reject) => {probe.once('error', () => reject(new Error('fixture port already owned; no HTTP sent'))); probe.listen(number, '127.0.0.1', resolve);});
    await new Promise(resolve => probe.close(resolve));
  }
}
async function start(bin, args, readyPath, base, cwd = repo) {
  await freePorts([Number(new URL(base).port)]);
  const child = cp.spawn(bin, args, {cwd, env, stdio: ['ignore', 'pipe', 'ignore']});
  const stopped = new Promise(resolve => child.once('exit', (code, signal) => resolve({code, signal})));
  let spawnedError = false, ownReady = false, output = '';
  child.once('error', () => {spawnedError = true;});
  child.stdout.on('data', chunk => {output = (output + chunk.toString()).slice(-8192); ownReady ||= readyPath === '/readyz' ? output.includes('agent-board listening on ' + base) : /Ready in/.test(output);});
  for (let i = 0; i < 200; i++) {
    assert(!spawnedError && child.exitCode === null && child.signalCode === null, 'owned process exited before readiness');
    try {if (ownReady && (await fetch(base + readyPath, {signal: AbortSignal.timeout(3000)})).status === 200) return {child, stopped, kind: readyPath === '/readyz' ? 'go' : 'next', port: Number(new URL(base).port)};} catch {}
    await sleep(50);
  }
  child.kill('SIGKILL'); await stopped; throw new Error('owned process readiness timeout');
}
async function stop(proc) {
  if (!proc) return;
  if (proc.child.exitCode !== null || proc.child.signalCode !== null) return;
  proc.child.kill('SIGTERM');
  let timeout;
  const result = await Promise.race([proc.stopped, new Promise(resolve => {timeout = setTimeout(() => {proc.child.kill('SIGKILL'); resolve({forced: true});}, 7000);})]);
  clearTimeout(timeout); if (result.forced) await proc.stopped;
  await freePorts([proc.port]);
  assert(result.code === 0 || proc.kind === 'next' && result.code === 143, 'owned process failed graceful shutdown');
}
async function response(url, options = {}) {
  const result = await fetch(url, {redirect: 'manual', signal: AbortSignal.timeout(20000), ...options});
  const raw = await result.text(); let data; try {data = JSON.parse(raw);} catch {data = null;}
  return {status: result.status, data, raw, headers: result.headers};
}
class BrowserSession {
  constructor(cookie = '') {this.cookie = cookie;}
  capture(headers) {
    const cookies = new Map(this.cookie.split('; ').filter(Boolean).map(value => value.split('=')));
    for (const value of headers.getSetCookie()) {const [name, secret] = value.split(';')[0].split('='); if (secret) cookies.set(name, secret); else cookies.delete(name);}
    this.cookie = [...cookies].map(([name, value]) => name + '=' + value).join('; ');
  }
  async read(route) {return response(goOrigin + route, {headers: {Cookie: this.cookie, Accept: 'application/json'}});}
  async post(route, fields, extras = {}) {
    return response(goOrigin + route, {method: 'POST', headers: {Cookie: this.cookie, Accept: 'application/json', Origin: origin, ...extras}, body: new URLSearchParams(fields)});
  }
  async login(token) {
    const first = await this.read('/web/login'); assert.equal(first.status, 200); this.capture(first.headers);
    const csrf = first.data?.page?.csrf || first.raw.match(/name="csrf" value="([^"]+)"/)?.[1]; assert(csrf, 'preauth CSRF missing');
    const result = await this.post('/web/login', {csrf, token: fs.readFileSync(token.file, 'utf8').trim()});
    assert([200, 303].includes(result.status), 'synthetic login failed'); this.capture(result.headers); return this;
  }
  async catalog() {const result = await this.read('/web/projects'); assert.equal(result.status, 200, 'catalog rejected'); return result.data.page;}
  async detail(id) {const result = await this.read('/web/catalog/projects/' + id); assert.equal(result.status, 200, 'project details rejected'); return result.data.page;}
  async action(page, key, route, fields) {
    const id = page.registry.actions[key]; assert(id, 'catalog capability missing: ' + key);
    return this.post(route, {csrf: page.registry.csrf, action_id: id, ...fields});
  }
  async open(id, target = '') {
    const page = await this.catalog();
    const result = await this.action(page, 'switch:' + id, '/web/catalog/switch', {project_id: id, ...(target ? {target} : {})});
    assert.equal(result.status, 200, 'explicit project open rejected'); return result;
  }
}
class Native {
  constructor(token, project = '', bin = binary) {this.token = token; this.project = project; this.bin = bin; this.session = '';}
  raw(op, input = {}) {
    const args = ['call', op, '--server', goOrigin, '--token-file', this.token.file];
    if (this.project) args.push('--project', this.project);
    const body = {...input}; if (op !== 'describe' && this.session && !body.session_id) body.session_id = this.session;
    if (!['get', 'tasks', 'epics', 'sessions', 'notes', 'stats', 'events', 'profiles', 'profile', 'coordination', 'similar', 'team', 'managed_list', 'managed_run'].includes(op) && !body.request_id) body.request_id = crypto.randomUUID();
    const result = cli(args, body, this.bin, true);
    let data; try {data = JSON.parse(result.stdout);} catch {data = null;}
    return {status: result.status, data};
  }
  call(op, input = {}) {const result = this.raw(op, input); assert.equal(result.status, 0, `native ${op} rejected (body suppressed)`); return result.data;}
  reject(op, input = {}, codes = ['FORBIDDEN_ROLE', 'UNAUTHORIZED', 'NOT_FOUND', 'NO_SESSION', 'SESSION_MISMATCH', 'VALIDATION', 'IDEMPOTENCY_CONFLICT']) {
    const result = this.raw(op, input);
    assert(Number.isInteger(result.status) && result.status !== 0 && codes.includes(result.data?.error?.code), `native ${op} lacked explicit expected domain denial`);
    return result.data.error.code;
  }
  open(role = 'coordinator') {const result = this.call('describe', {role, run_id: crypto.randomUUID(), machine: this.token.machine}); this.session = result.session_id; return result;}
  async http(route, options = {}) {
    const prefix = '/api/v1/' + (this.project ? 'p/' + this.project + '/' : '');
    return response(goOrigin + prefix + route, {...options, headers: {Authorization: 'Bearer ' + fs.readFileSync(this.token.file, 'utf8').trim(), 'X-Session': this.session, ...(options.headers || {})}});
  }
}
module.exports = {fs, path, cp, crypto, assert, repo, root, binary, oldBinary, origin, goOrigin, env, sha, save, sleep, cli, admin, legacyToken, catalogToken, freePorts, startServer, startNext, stop, response, BrowserSession, Native};
