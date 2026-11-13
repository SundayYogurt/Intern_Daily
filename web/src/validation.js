const bytes = value => new TextEncoder().encode(String(value ?? '')).length

export function validDate(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value || '')) return false
  const date = new Date(`${value}T00:00:00Z`)
  return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === value
}

export function validateProfile(user, student, provinces, today) {
  if (!user.name?.trim() || bytes(user.name.trim()) > 150) return 'ชื่อ–นามสกุลต้องไม่ว่างและไม่เกิน 150 ไบต์'
  for (const [key, label, limit] of [
    ['phone', 'เบอร์โทรศัพท์', 40], ['position', 'ตำแหน่ง', 120],
    ['home_address', 'ที่อยู่', 300], ['subdistrict', 'ตำบล/แขวง', 100],
    ['district', 'อำเภอ/เขต', 100], ['country', 'ประเทศ', 100],
    ['skills', 'ความรู้ความสามารถพิเศษ', 1000], ['interests', 'ความสนใจพิเศษ', 1000],
  ]) {
    if (bytes(user[key]) > limit) return `${label}ยาวเกินไป`
  }
  if (user.birth_date && (!validDate(user.birth_date) || user.birth_date > today)) return 'วันเกิดต้องเป็นวันที่ผ่านมาแล้ว'
  if (student && user.province === 'other' && !user.country?.trim()) return 'กรอกชื่อประเทศ'
  if (student && user.province && user.province !== 'other' && !provinces.includes(user.province)) return 'เลือกจังหวัดจากรายการ'
  return ''
}

export function validateInternship(internship, logs) {
  if (!/^6642590\d{2}$/.test(internship.student_number || '')) return 'รหัสนักศึกษาต้องเป็น 6642590xx'
  if (bytes(internship.class_group) > 40) return 'หมู่เรียนยาวเกินไป'
  if (bytes(internship.company) > 200) return 'ชื่อบริษัท/หน่วยงานยาวเกินไป'
  if (bytes(internship.company_address) > 400) return 'ที่อยู่บริษัทยาวเกินไป'
  if (!['internship', 'cooperative'].includes(internship.plan_type)) return 'เลือกแผนการฝึก'
  if (internship.mentor_email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(internship.mentor_email.trim())) return 'อีเมลพี่เลี้ยงไม่ถูกต้อง'
  const start = internship.start_date || ''
  const end = internship.end_date || ''
  if (start && !validDate(start) || end && !validDate(end)) return 'วันที่ฝึกงานไม่ถูกต้อง'
  if ((start && !end) || (!start && end)) return 'กรอกวันเริ่มและวันสิ้นสุดให้ครบ'
  if (start && end && start > end) return 'วันสิ้นสุดต้องไม่อยู่ก่อนวันเริ่ม'
  if (logs.length && (!start || !end)) return 'ต้องระบุช่วงวันที่ให้ครอบคลุมบันทึกที่มีอยู่'
  if (logs.some(log => log.date < start || log.date > end)) return 'ช่วงวันที่ใหม่ต้องครอบคลุมบันทึกรายวันที่มีอยู่'
  const holidays = internship.holiday_dates || []
  if (holidays.length > 100 || new Set(holidays).size !== holidays.length) return 'วันหยุดซ้ำกันหรือมากเกิน 100 วัน'
  if (holidays.some(date => !validDate(date) || start && date < start || end && date > end)) return 'วันหยุดต้องอยู่ในช่วงฝึกงาน'
  return ''
}

export function validateLog(log, internship, existingLogs, editingId, today) {
  if (!validDate(log.date)) return 'เลือกวันที่ให้ถูกต้อง'
  if (log.date > today) return 'ยังบันทึกวันในอนาคตไม่ได้'
  if (internship?.start_date && log.date < internship.start_date || internship?.end_date && log.date > internship.end_date) return 'วันที่อยู่นอกช่วงฝึกงาน'
  if (existingLogs.some(item => item.date === log.date && item.id !== editingId)) return 'วันที่นี้มีบันทึกแล้ว'
  if (!['work', 'sick', 'personal', 'holiday'].includes(log.kind)) return 'ประเภทบันทึกไม่ถูกต้อง'
  if (log.kind === 'work') {
    if (!/^\d{2}:\d{2}$/.test(log.start_time || '') || !/^\d{2}:\d{2}$/.test(log.end_time || '') || log.start_time >= log.end_time) return 'เวลาเริ่มและเวลาสิ้นสุดไม่ถูกต้อง'
    const start = Number(log.start_time.slice(0, 2)) * 60 + Number(log.start_time.slice(3))
    const end = Number(log.end_time.slice(0, 2)) * 60 + Number(log.end_time.slice(3))
    if (Number(log.start_time.slice(0, 2)) > 23 || Number(log.end_time.slice(0, 2)) > 23 || Number(log.start_time.slice(3)) > 59 || Number(log.end_time.slice(3)) > 59) return 'เวลาเริ่มและเวลาสิ้นสุดไม่ถูกต้อง'
    const breakMinutes = Number(log.break_minutes)
    if (!Number.isInteger(breakMinutes) || breakMinutes < 0 || breakMinutes > 600 || end - start <= breakMinutes) return 'เวลาพักต้องเป็น 0–600 นาทีและเหลือเวลาทำงานจริง'
    if (!log.work?.trim()) return 'กรอกงานที่ปฏิบัติ'
  }
  if (bytes(log.work) > 4000 || bytes(log.blockers) > 2000 || bytes(log.learning) > 2000) return 'ข้อความบันทึกยาวเกินไป'
  return ''
}

export function validateApproval(range, typedName, mentorName, pendingCount, today) {
  if (!validDate(range.from) || !validDate(range.to) || range.from > range.to || range.to > today) return 'ช่วงวันที่รับรองไม่ถูกต้อง'
  const days = (Date.parse(`${range.to}T00:00:00Z`) - Date.parse(`${range.from}T00:00:00Z`)) / 86400000
  if (days > 31) return 'รับรองได้ครั้งละไม่เกิน 31 วัน'
  if (pendingCount === 0) return 'ไม่มีบันทึกที่รอรับรองในช่วงนี้'
  if (typedName.trim() !== mentorName) return 'พิมพ์ชื่อให้ตรงกับโปรไฟล์พี่เลี้ยง'
  return ''
}
