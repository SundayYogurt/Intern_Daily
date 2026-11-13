import test from 'node:test'
import assert from 'node:assert/strict'
import { validateProfile, validateInternship, validateLog, validateApproval } from './validation.js'

test('profile validates required name, date, and foreign country', () => {
  assert.match(validateProfile({ name: '  ' }, true, [], '2026-11-13'), /ชื่อ/)
  assert.match(validateProfile({ name: 'Student', birth_date: '2027-01-01' }, true, [], '2026-11-13'), /วันเกิด/)
  assert.match(validateProfile({ name: 'Student', province: 'other', country: '' }, true, [], '2026-11-13'), /ประเทศ/)
  assert.equal(validateProfile({ name: 'Student', province: 'กรุงเทพมหานคร' }, true, ['กรุงเทพมหานคร'], '2026-11-13'), '')
})

test('internship rejects invalid number and date range that excludes logs', () => {
  const internship = { student_number: '664259024', plan_type: 'internship', start_date: '2026-10-01', end_date: '2026-11-30', holiday_dates: [] }
  assert.equal(validateInternship(internship, [{ date: '2026-10-20' }]), '')
  assert.match(validateInternship({ ...internship, student_number: '123' }, []), /รหัสนักศึกษา/)
  assert.match(validateInternship({ ...internship, start_date: '2026-11-01' }, [{ date: '2026-10-20' }]), /ครอบคลุม/)
  assert.match(validateInternship({ ...internship, holiday_dates: ['2026-12-01'] }, []), /วันหยุด/)
})

test('daily log rejects duplicates, bad times, and breaks longer than the shift', () => {
  const internship = { start_date: '2026-10-01', end_date: '2026-11-30' }
  const log = { date: '2026-10-20', kind: 'work', start_time: '09:00', end_time: '17:00', break_minutes: 60, work: 'เขียน API' }
  assert.equal(validateLog(log, internship, [], null, '2026-11-13'), '')
  assert.match(validateLog(log, internship, [{ id: 1, date: log.date }], null, '2026-11-13'), /บันทึกแล้ว/)
  assert.equal(validateLog(log, internship, [{ id: 1, date: log.date }], 1, '2026-11-13'), '')
  assert.match(validateLog({ ...log, end_time: '08:00' }, internship, [], null, '2026-11-13'), /เวลา/)
  assert.match(validateLog({ ...log, break_minutes: 600 }, internship, [], null, '2026-11-13'), /เวลาพัก/)
})

test('approval requires a valid period and matching typed name', () => {
  const range = { from: '2026-10-20', to: '2026-10-25' }
  assert.equal(validateApproval(range, 'Mentor', 'Mentor', 2, '2026-11-13'), '')
  assert.match(validateApproval(range, 'Other', 'Mentor', 2, '2026-11-13'), /ชื่อ/)
  assert.match(validateApproval({ from: '2026-10-01', to: '2026-11-05' }, 'Mentor', 'Mentor', 2, '2026-11-13'), /31 วัน/)
})
