import test from 'node:test'
import assert from 'node:assert/strict'
import { calculatePlan, missingLogDates } from './plan.js'

test('cooperative plan counts 16 Monday-Friday weeks and excludes entered holidays', () => {
  const result = calculatePlan('2026-01-05', '2026-04-24', 'cooperative', ['2026-01-06', '2026-01-06', '2026-01-10'])
  assert.equal(result.weekdays, 80)
  assert.equal(result.holidays, 1)
  assert.equal(result.hours, 632)
  assert.equal(result.meetsRequirement, true)
})

test('internship plan requires at least 450 planned hours', () => {
  assert.equal(calculatePlan('2026-01-05', '2026-03-23').hours, 448)
  assert.equal(calculatePlan('2026-01-05', '2026-03-23').meetsRequirement, false)
  assert.equal(calculatePlan('2026-01-05', '2026-03-24').meetsRequirement, true)
})

test('rejects invalid dates', () => {
  assert.equal(calculatePlan('2026-02-30', '2026-03-01'), null)
  assert.equal(calculatePlan('2026-04-01', '2026-03-01'), null)
})

test('missingLogDates lists unlogged weekdays newest first, skipping holidays and future days', () => {
  // 2026-11-02 is a Monday
  const missing = missingLogDates('2026-11-02', '2026-11-30', ['2026-11-04'], ['2026-11-02', '2026-11-06'], '2026-11-10')
  assert.deepEqual(missing, ['2026-11-10', '2026-11-09', '2026-11-05', '2026-11-03'])
  assert.deepEqual(missingLogDates('', '', [], [], '2026-11-10'), [])
  assert.deepEqual(missingLogDates('2026-11-20', '2026-11-30', [], [], '2026-11-10'), [])
  assert.deepEqual(missingLogDates('2026-11-02', '2026-11-03', [], [], '2026-11-10'), ['2026-11-03', '2026-11-02'])
})
