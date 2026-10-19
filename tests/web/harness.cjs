'use strict';
// Real processes and OS pipes only. Credentials never enter reports or stdout.
const fs = require('node:fs');
const path = require('node:path');
const cp = require('node:child_process');
const net = require('node:net');
const readline = require('node:readline');
const crypto = require('node:crypto');
const assert = require('node:assert/strict');
const required = (key) => { assert(process.env[key], `missing ${key}`); return process.env[key]; };
const binary = required('ACCORD_QA_BINARY');
const root = required('ACCORD_QA_DIR');
const env = {...process.env, PYTHONDONTWRITEBYTECODE: '1'}; delete env.AGENT_BOARD_TOKEN;
const sha = (data) => crypto.createHash('sha256').update(data).digest('hex');
const sleep = (ms) => new Promise(resolve => setTimeout(resolve, ms));
function cli(args, input) {
  const r = cp.spawnSync(binary, args, {input: input && JSON.stringify(input), env, encoding: 'utf8', timeout: 15000});
  assert.equal(r.status, 0, `CLI ${args[0]} failed (output suppressed)`);
  return r.stdout;
}
function token(state, name) {
  const file = path.join(root, name + '.token');
  cli(['token', 'create', '--db', path.join(state, 'board.db'), '--owner', name, '--machine', name, '--output', file]);
  assert.equal(fs.statSync(file).mode & 0o777, 0o600);
  return file;
}
async function port() {
  const s = net.createServer(); await new Promise(r => s.listen(0, '127.0.0.1', r));
  const n = s.address().port; await new Promise(r => s.close(r)); return n;
}
async function server(bin, state, n, web = true) {
  const origin = `http://127.0.0.1:${n}`;
  const args = ['serve', '--db', path.join(state, 'board.db'), '--listen', `127.0.0.1:${n}`, '--task-ttl', '20s', '--session-ttl', '10m'];
  if (web) args.push('--web-origin', origin, '--web-data', path.join(state, 'web'));
  const child = cp.spawn(bin, args, {env, stdio: ['ignore', 'ignore', 'ignore']});
  let exited = false; child.once('exit', () => {exited = true;});
  for (let i = 0; i < 150; i++) {
    assert(!exited, 'server exited before readiness');
    try { if ((await fetch(origin + '/readyz')).status === 200) return {child, origin}; } catch {}
    await sleep(50);
  }
  child.kill(); throw new Error('server readiness timeout');
}
async function stop(proc) {
  if (!proc || proc.child.exitCode !== null) return;
  let timer;
  const done = new Promise(resolve => proc.child.once('exit', resolve));
  proc.child.kill('SIGTERM');
  await Promise.race([done, new Promise(resolve => {timer = setTimeout(() => {proc.child.kill('SIGKILL'); resolve();}, 7000);})]);
  clearTimeout(timer); assert(proc.child.exitCode === 0, 'server did not shut down cleanly');
}
class Worker {
  constructor(origin, file) {
    this.pending = new Map(); this.seq = 0;
    this.proc = cp.spawn(binary, ['mcp', '--server', origin, '--token-file', file], {env, stdio: ['pipe', 'pipe', 'ignore']});
    readline.createInterface({input: this.proc.stdout}).on('line', line => {
      let message; try { message = JSON.parse(line); } catch {return;}
      const waiting = this.pending.get(message.id); if (!waiting) return;
      this.pending.delete(message.id); clearTimeout(waiting.timer);
      if (message.error) waiting.reject(new Error('MCP protocol rejected (body suppressed)')); else waiting.resolve(message.result);
    });
  }
  rpc(method, params) {
    const id = ++this.seq;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {this.pending.delete(id); reject(new Error(`MCP ${method} timeout`));}, 15000);
      this.pending.set(id, {resolve, reject, timer});
      this.proc.stdin.write(JSON.stringify({jsonrpc: '2.0', id, method, params}) + '\n');
    });
  }
  async start() {
    await this.rpc('initialize', {protocolVersion: '2025-03-26', capabilities: {}, clientInfo: {name: 'synthetic-web-qa', version: '1'}});
    this.proc.stdin.write(JSON.stringify({jsonrpc: '2.0', method: 'notifications/initialized'}) + '\n');
    return this;
  }
  async call(name, args) {
    const r = await this.rpc('tools/call', {name, arguments: args});
    assert(!r.isError, `native ${name} rejected (body suppressed)`);
    if (r.structuredContent) return r.structuredContent;
    return JSON.parse(r.content.find(c => c.type === 'text').text);
  }
  async resource(uri) {
    const r = await this.rpc('resources/read', {uri});
    return JSON.parse(r.contents[0].text);
  }
  async close() {
    if (this.proc.exitCode !== null) return;
    let timer;
    const ended = new Promise(resolve => this.proc.once('exit', resolve));
    this.proc.stdin.end();
    await Promise.race([ended, new Promise(resolve => {timer = setTimeout(() => {this.proc.kill(); resolve();}, 7000);})]);
    clearTimeout(timer); assert.equal(this.proc.exitCode, 0, 'native worker did not stop cleanly after EOF');
  }
}
const manifest = () => Object.fromEntries(fs.readdirSync(__dirname).filter(name => /\.(cjs|py)$/.test(name)).sort().map(name => [name, sha(fs.readFileSync(path.join(__dirname, name)))]));
module.exports = {manifest, fs, path, cp, assert, required, binary, root, env, sha, sleep, cli, token, port, server, stop, Worker};
