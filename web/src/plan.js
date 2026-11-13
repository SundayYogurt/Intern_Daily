const DAY = 24 * 60 * 60 * 1000

function parseDate(value) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value || '')) return null
  const date = new Date(`${value}T00:00:00Z`)
  return Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== value ? null : date
}

export function calculatePlan(startDate, endDate, planType = 'internship', holidayDates = []) {
  const start = parseDate(startDate)
  const end = parseDate(endDate)
  if (!start || !end || end < start || (end - start) / DAY > 3660) return null

  const weekdays = new Set()
  for (let time = start.getTime(); time <= end.getTime(); time += DAY) {
    const date = new Date(time)
    if (date.getUTCDay() !== 0 && date.getUTCDay() !== 6) weekdays.add(date.toISOString().slice(0, 10))
  }
  const holidays = new Set(holidayDates.filter(date => weekdays.has(date)))
  const workdays = weekdays.size - holidays.size
  const hours = workdays * 8
  const cooperative = planType === 'cooperative'
  const requiredHours = cooperative ? 540 : 450
  const requiredWeekdays = cooperative ? 80 : 0
  return {
    weekdays: weekdays.size,
    holidays: holidays.size,
    workdays,
    hours,
    requiredHours,
    requiredWeekdays,
    meetsRequirement: hours >= requiredHours && weekdays.size >= requiredWeekdays,
  }
}

// Weekdays from the internship start up to today (or the end date) that have no
// daily log and are not marked as holidays. Newest first.
export function missingLogDates(startDate, endDate, holidayDates = [], loggedDates = [], today) {
  const start = parseDate(startDate)
  const end = parseDate(endDate)
  const now = parseDate(today)
  if (!start || !now) return []
  const last = end && end < now ? end : now
  if (last < start || (last - start) / DAY > 3660) return []
  const skip = new Set([...(holidayDates || []), ...(loggedDates || [])])
  const missing = []
  for (let time = last.getTime(); time >= start.getTime(); time -= DAY) {
    const date = new Date(time)
    const key = date.toISOString().slice(0, 10)
    if (date.getUTCDay() !== 0 && date.getUTCDay() !== 6 && !skip.has(key)) missing.push(key)
  }
  return missing
}
