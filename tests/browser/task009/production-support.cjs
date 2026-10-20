'use strict';
// A response observation must never reject independently of the action that owns it.
async function actionResponse(page, route, action, timeout = 15000) {
  let settle, timer;
  const observed = new Promise(resolve => {settle = resolve;});
  const listener = response => {
    if (response.request().method() === 'POST' && new URL(response.url()).pathname === route) settle({response});
  };
  page.on('response', listener);
  timer = setTimeout(() => settle({timeout: true}), timeout);
  try {
    await action();
    const result = await observed;
    if (result.timeout) {const error = new Error(); error.name = 'TimeoutError'; throw error;}
    return result.response;
  } finally {
    clearTimeout(timer); page.off('response', listener); settle({cancelled: true});
  }
}
function diagnostic(error, stage) {
  const names = new Set(['AssertionError', 'TimeoutError', 'SyntaxError', 'TypeError', 'Error']);
  const code = error?.code === 'ERR_ASSERTION' ? 'ASSERTION' : error?.name === 'TimeoutError' ? 'TIMEOUT' : error?.name === 'SyntaxError' ? 'JSON_PARSE' : 'INSPECTION_ERROR';
  const result = {stage, code, name: names.has(error?.name) ? error.name : 'Error'};
  if (Number.isInteger(error?.status) && error.status >= 100 && error.status <= 599) result.status = error.status;
  if (code === 'ASSERTION') result.assertion_label = stage;
  return result;
}
// Cleanup errors are secondary. Always attempt close and persist the original failure.
async function finish(report, {logout, close, write}) {
  try {
    if (logout) try {await logout(); report.cleanup_logout = 'PASS';} catch (error) {report.cleanup_logout = 'FAIL'; report.cleanup_failure = diagnostic(error, 'CLEANUP_LOGOUT'); report.result = 'FAIL';}
  } finally {
    try {if (close) await close(); report.browser_closed = true;} catch (error) {report.browser_closed = false; report.close_failure = diagnostic(error, 'CLEANUP_BROWSER'); report.result = 'FAIL';}
    finally {report.finished_at = new Date().toISOString(); write(report);}
  }
}
module.exports = {actionResponse, diagnostic, finish};
