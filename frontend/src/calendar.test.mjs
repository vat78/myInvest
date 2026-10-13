import { test } from 'node:test';
import assert from 'node:assert/strict';
import { localDate } from './calendar.js';

test('local calendar date changes at local midnight, without UTC truncation', () => {
  const previous = process.env.TZ;
  try {
    process.env.TZ = 'Europe/Moscow';
    assert.equal(localDate(new Date('2026-10-12T20:59:59Z')), '2026-10-12');
    assert.equal(localDate(new Date('2026-10-12T21:00:00Z')), '2026-10-13');
    process.env.TZ = 'America/New_York';
    assert.equal(localDate(new Date('2026-01-01T04:59:59Z')), '2025-12-31');
    assert.equal(localDate(new Date('2026-01-01T05:00:00Z')), '2026-01-01');
    assert.equal(localDate(new Date('2024-02-29T17:00:00Z')), '2024-02-29');
  } finally {
    if (previous === undefined) delete process.env.TZ;
    else process.env.TZ = previous;
  }
});
