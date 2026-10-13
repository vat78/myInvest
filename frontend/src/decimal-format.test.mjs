import test from 'node:test';
import assert from 'node:assert/strict';
import { formatDecimal } from './decimal-format.mjs';

test('grouping preserves exact digits, signs and long fractional parts', () => {
  for (const [value, expected] of [
    ['0', '0'], ['999', '999'], ['1000', '1 000'], ['94500', '94 500'],
    ['-1234567.8901234567890123456789', '-1 234 567,8901234567890123456789'],
    ['123456789012345678901234567890.000001', '123 456 789 012 345 678 901 234 567 890,000001'],
    ['0.3333', '0,3333'], ['1000.0000', '1 000,0000'],
  ]) assert.equal(formatDecimal(value), expected);
});
