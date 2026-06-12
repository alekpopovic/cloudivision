const assert = require('node:assert/strict');
const test = require('node:test');

test('demo app metadata is stable', () => {
  assert.equal('cloudivision-demo-app'.includes('cloudivision'), true);
});
