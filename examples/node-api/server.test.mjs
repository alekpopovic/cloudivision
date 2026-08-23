import assert from 'node:assert/strict';
import test from 'node:test';
import { handler } from './server.mjs';

test('health response', () => {
  let status;
  let body = '';
  handler({ url: '/healthz' }, { writeHead(value) { status = value; return this; }, end(value = '') { body += value; } });
  assert.equal(status, 200);
  assert.deepEqual(JSON.parse(body), { status: 'ok' });
});
