import React, { useEffect, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster, toast } from 'react-hot-toast'
import { LayoutDashboard, UserRound, CalendarDays, BadgeCheck, ClipboardCheck, BookOpenText, Files, LogOut, ChevronRight, Plus, ArrowUpRight, Download, Clock3, CheckCircle2, FileText, FileUp, ImagePlus, Copy, ListChecks, Building2, Eye, X, CircleHelp, Sparkles, BellRing, TriangleAlert, ChevronLeft, Users, Search, RefreshCw, KeyRound } from 'lucide-react'
import { calculatePlan, missingLogDates } from './plan.js'
import { provinces } from './provinces.js'
import { validateProfile, validateInternship, validateLog, validateApproval } from './validation.js'
import '@fontsource/sarabun/400.css'
import '@fontsource/sarabun/500.css'
import '@fontsource/sarabun/600.css'
import '@fontsource/sarabun/700.css'
import './tailwind.css'

const criteria = [
  ['1. ระเบียบวินัย', [['1.1', 'การแต่งกายสุภาพเรียบร้อย และถูกระเบียบ'], ['1.2', 'การตรงต่อเวลา และมาปฏิบัติงานอย่างสม่ำเสมอ'], ['1.3', 'ปฏิบัติตามกฎระเบียบ ข้อบังคับ ที่กำหนดไว้อย่างเคร่งครัด'], ['1.4', 'เชื่อฟังและปฏิบัติตามคำแนะนำของหัวหน้างาน']]],
  ['2. พฤติกรรมในการทำงาน', [['2.1', 'ความซื่อตรงต่องาน'], ['2.2', 'ความขยันขันแข็งในการทำงาน'], ['2.3', 'การรู้จักควบคุมอารมณ์'], ['2.4', 'การรู้จักสิทธิ หน้าที่ และความรับผิดชอบ'], ['2.5', 'ความไว้วางใจสามารถมอบหมายงานให้ได้'], ['2.6', 'ความสามารถในการสื่อสาร ได้ถูกต้องชัดเจนกับผู้ร่วมงานและหัวหน้างาน']]],
  ['3. คุณภาพของงาน', [['3.1', 'ความสามารถปฏิบัติงานที่ได้รับมอบหมาย'], ['3.2', 'ความสามารถในการแก้ปัญหาเฉพาะหน้าในการทำงาน'], ['3.3', 'ปฏิบัติงานถูกต้องตามลักษณะงาน'], ['3.4', 'ผลงานเป็นที่ยอมรับของหน่วยงาน']]],
  ['4. วิธีการปฏิบัติงาน', [['4.1', 'ปฏิบัติงานด้วยความรอบคอบและคำนึงถึงความปลอดภัย'], ['4.2', 'เข้าใจขั้นตอนและขอบเขตของการปฏิบัติงาน'], ['4.3', 'การเห็นคุณค่าของสิ่งของ เครื่องใช้'], ['4.4', 'มีความคิดริเริ่มสร้างสรรค์และไหวพริบในการทำงาน']]],
  ['5. มนุษย์สัมพันธ์', [['5.1', 'ความมีน้ำใจ ให้ความร่วมมือ และประสานงานร่วมกับผู้อื่น'], ['5.2', 'ความสามารถปรับตัวเข้ากับสภาพแวดล้อม'], ['5.3', 'ความสุภาพอ่อนน้อมรู้จักกาละเทศะ'], ['5.4', 'ความสามารถในการแสดงความคิดเห็นและรับฟังผู้อื่น']]],
]

const reportFields = [
  ['บทที่ 1 บทนำ', [['background', '1.1 ความเป็นมาและความสำคัญ'], ['objectives', '1.2 วัตถุประสงค์'], ['scope', '1.3 ขอบเขต'], ['company', '1.4 ข้อมูลบริษัท'], ['benefits', '1.5 ประโยชน์ที่คาดว่าจะได้รับ'], ['role', '1.6 ตำแหน่งที่ปฏิบัติ']]],
  ['บทที่ 2 รายละเอียดโครงงาน', [['project', '2.1 วิเคราะห์และออกแบบการทำงานของระบบ'], ['methods', '2.2 วิธีดำเนินงาน']]],
  ['บทที่ 3 ผลการดำเนินงาน', [['tools', '3.1 เครื่องมือที่ใช้'], ['process', '3.2 ขั้นตอนการดำเนินงาน'], ['results', '3.3 ผลการดำเนินงาน']]],
  ['บทที่ 4 สรุปและข้อเสนอแนะ', [['conclusion', '4.1 สรุปผลการดำเนินงาน'], ['received_benefits', '4.2 ประโยชน์ที่ได้รับ'], ['constraints', '4.3 ปัญหาและข้อจำกัด'], ['suggestions', '4.4 ข้อเสนอแนะ']]],
]

const blankLog = () => ({ date: new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Bangkok' }).format(new Date()), start_time: '09:00', end_time: '17:00', break_minutes: 60, kind: 'work', late: false, work: '', blockers: '', learning: '' })

async function request(path, options = {}) {
  const response = await fetch('/api' + path, {
    credentials: 'same-origin',
    ...options,
    headers: options.body instanceof FormData ? options.headers : { 'Content-Type': 'application/json', ...options.headers },
  })
  const type = response.headers.get('content-type') || ''
  const body = type.includes('application/json') ? await response.json() : null
  if (!response.ok) throw new Error(body?.error || 'ดำเนินการไม่สำเร็จ')
  return body
}
const post = (path, value, method = 'POST') => request(path, { method, body: JSON.stringify(value) })

async function downloadDocument(kind, format) {
  const toastId = toast.loading('กำลังสร้างเอกสาร…')
  try {
    const response = await fetch(`/api/export/${kind}.${format}`, { credentials: 'same-origin' })
    if (!response.ok) {
      const body = await response.json().catch(() => null)
      throw new Error(body?.error || 'สร้างเอกสารไม่สำเร็จ')
    }
    const content = response.headers.get('content-disposition') || ''
    const encoded = content.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
    const name = encoded ? decodeURIComponent(encoded) : `internship-${kind}.${format}`
    const url = URL.createObjectURL(await response.blob())
    const link = document.createElement('a')
    link.href = url
    link.download = name
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    toast.success('ดาวน์โหลดเอกสารแล้ว', { id: toastId })
  } catch (error) {
    toast.error(error.message, { id: toastId })
  }
}

function Field({ label, value, onChange, type = 'text', placeholder = '', required = false, maxLength, note, readOnly = false, disabled = false, ...inputProps }) {
  return <label className="field"><span>{label}</span><input type={type} value={value ?? ''} onChange={e => onChange?.(e.target.value)} placeholder={placeholder} required={required} maxLength={maxLength} readOnly={readOnly} disabled={disabled} aria-readonly={readOnly || undefined} {...inputProps} />{note && <small>{note}</small>}</label>
}

function TextArea({ label, value, onChange, rows = 4, placeholder = '', readOnly = false, disabled = false, maxLength, required = false }) {
  return <label className="field"><span>{label}</span><textarea rows={rows} value={value ?? ''} onChange={e => onChange?.(e.target.value)} placeholder={placeholder} readOnly={readOnly} disabled={disabled} maxLength={maxLength} required={required} /></label>
}

function Button({ children, busy, ...props }) { return <button {...props} disabled={busy || props.disabled}>{busy ? 'กำลังบันทึก…' : children}</button> }

// Debounced autosave: saves `value` after the user stops typing, skips invalid
// drafts, never overlaps requests, and flushes pending edits on unmount.
function useAutosave(value, { save, validate, delay = 1000 }) {
  const serialized = JSON.stringify(value)
  const savedRef = useRef(serialized)
  const latest = useRef({ value, save, validate })
  latest.current = { value, save, validate }
  const timer = useRef(null)
  const running = useRef(false)
  const [state, setState] = useState({ status: 'saved', message: '' })
  const flushRef = useRef(null)
  flushRef.current = async () => {
    clearTimeout(timer.current)
    timer.current = null
    const current = latest.current
    const snapshot = JSON.stringify(current.value)
    if (snapshot === savedRef.current) { setState({ status: 'saved', message: '' }); return }
    const error = current.validate?.(current.value)
    if (error) { setState({ status: 'invalid', message: error }); return }
    if (running.current) { timer.current = setTimeout(() => flushRef.current(), 300); return }
    running.current = true
    setState({ status: 'saving', message: '' })
    try {
      await current.save(JSON.parse(snapshot))
      savedRef.current = snapshot
      const changedSince = JSON.stringify(latest.current.value) !== snapshot
      setState({ status: changedSince ? 'pending' : 'saved', message: '' })
      if (changedSince) timer.current = setTimeout(() => flushRef.current(), delay)
    } catch (e) {
      setState({ status: 'error', message: e.message })
    } finally {
      running.current = false
    }
  }
  useEffect(() => {
    if (serialized === savedRef.current) return
    setState({ status: 'pending', message: '' })
    clearTimeout(timer.current)
    timer.current = setTimeout(() => flushRef.current(), delay)
  }, [serialized])
  useEffect(() => {
    const warn = event => { if (JSON.stringify(latest.current.value) !== savedRef.current) { event.preventDefault(); event.returnValue = '' } }
    window.addEventListener('beforeunload', warn)
    return () => {
      window.removeEventListener('beforeunload', warn)
      if (timer.current) flushRef.current()
    }
  }, [])
  return { ...state, flush: () => flushRef.current() }
}

function SaveStatus({ state }) {
  const view = {
    saved: ['text-slate-500', 'บันทึกอัตโนมัติแล้ว'],
    pending: ['text-slate-500', 'มีการแก้ไข…'],
    saving: ['text-blue-700', 'กำลังบันทึก…'],
    invalid: ['text-amber-700', 'ยังไม่บันทึก: ' + state.message],
    error: ['text-red-700', 'บันทึกไม่สำเร็จ: ' + state.message],
  }[state.status]
  return <div className={`flex items-center gap-2 text-xs font-medium ${view[0]}`} role="status" aria-live="polite">
    {state.status === 'saved' && <CheckCircle2 size={14} aria-hidden="true" />}
    <span>{view[1]}</span>
    {state.status === 'error' && <button type="button" className="underline" onClick={state.flush}>ลองอีกครั้ง</button>}
  </div>
}

// AI suggestion button + editable preview. Nothing changes until the user applies it.
function AiAssist({ label = 'เกลาด้วย AI', run, onApply, modes = [['replace', 'ใช้ข้อความนี้']], disabled = false, hint = '' }) {
  const [busy, setBusy] = useState(false)
  const [text, setText] = useState('')
  async function generate() {
    setBusy(true)
    try { const result = await run(); setText(result.text) }
    catch (e) { toast.error(e.message) }
    finally { setBusy(false) }
  }
  return <div className="-mt-1 mb-4">
    <button type="button" title={disabled ? hint : ''} className="inline-flex items-center gap-1.5 rounded-md border border-violet-200 bg-violet-50 px-2.5 py-1 text-xs font-semibold text-violet-800 hover:bg-violet-100 disabled:cursor-not-allowed disabled:opacity-50" onClick={generate} disabled={busy || disabled}><Sparkles size={13} aria-hidden="true" />{busy ? 'AI กำลังเขียน…' : label}</button>
    {disabled && hint && <span className="ml-2 text-[11px] text-slate-400">{hint}</span>}
    {text && <div className="mt-2 rounded-lg border border-violet-200 bg-violet-50/60 p-3">
      <p className="mb-1.5 text-[11px] font-semibold text-violet-700">ข้อเสนอจาก AI · แก้ไขได้ก่อนใช้ ตรวจข้อเท็จจริงทุกครั้ง</p>
      <textarea className="w-full rounded-md border border-violet-200 bg-white p-2 text-sm leading-relaxed" value={text} onChange={e => setText(e.target.value)} rows={Math.min(14, text.split('\n').length + 1)} aria-label="ข้อความจาก AI" />
      <div className="mt-2 flex flex-wrap gap-2">{modes.map(([mode, modeLabel]) => <button key={mode} type="button" className="rounded-md bg-violet-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-violet-800" onClick={() => { onApply(text, mode); setText('') }}>{modeLabel}</button>)}<button type="button" className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600" onClick={() => setText('')}>ไม่ใช้</button></div>
    </div>}
  </div>
}

const todayISO = () => blankLog().date
const bangkokClock = () => new Intl.DateTimeFormat('en-GB', { timeZone: 'Asia/Bangkok', hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date())
const thaiShortDate = iso => new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', weekday: 'short', timeZone: 'UTC' }).format(new Date(iso + 'T00:00:00Z'))
function missedDays(data) {
  const x = data.internship || {}
  const today = todayISO()
  const all = missingLogDates(x.start_date, x.end_date, x.holiday_dates || [], (data.logs || []).map(l => l.date), today)
  return { missed: all.filter(d => d !== today), todayMissing: all.includes(today) }
}

// Profile picture with the first letter of the name as a fallback.
function Avatar({ user, version = 0, className = 'avatar', title }) {
  const [broken, setBroken] = useState(false)
  useEffect(() => { setBroken(false) }, [user?.id, user?.has_photo, version])
  const showPhoto = user?.has_photo && !broken
  return <div className={`${className} overflow-hidden`} title={title}>{showPhoto ? <img src={`/api/profile/photo/${user.id}?v=${version}`} alt="" className="h-full w-full object-cover" onError={() => setBroken(true)} /> : (user?.name?.slice(0, 1) || '?')}</div>
}

function ChangePassword() {
  const [form, setForm] = useState({ current: '', next: '', confirm: '' })
  const [busy, setBusy] = useState(false)
  const set = key => value => setForm(f => ({ ...f, [key]: value }))
  async function submit(event) {
    event.preventDefault()
    if ([...form.next].length < 8) { toast.error('รหัสผ่านใหม่ต้องมีอย่างน้อย 8 ตัวอักษร'); return }
    if (form.next !== form.confirm) { toast.error('ยืนยันรหัสผ่านใหม่ไม่ตรงกัน'); return }
    setBusy(true)
    try {
      await post('/profile/password', { current: form.current, new: form.next }, 'PUT')
      setForm({ current: '', next: '', confirm: '' })
      toast.success('เปลี่ยนรหัสผ่านแล้ว อุปกรณ์อื่นจะถูกออกจากระบบ')
    } catch (e) { toast.error(e.message) }
    finally { setBusy(false) }
  }
  return <section className="card" aria-label="เปลี่ยนรหัสผ่าน">
    <div className="section-head"><div><p className="eyebrow">ความปลอดภัย</p><h2>เปลี่ยนรหัสผ่าน</h2></div><KeyRound size={19} className="text-slate-500" aria-hidden="true" /></div>
    <form onSubmit={submit}>
      <div className="form-grid"><Field label="รหัสผ่านปัจจุบัน" type="password" value={form.current} onChange={set('current')} autoComplete="current-password" required /><Field label="รหัสผ่านใหม่" type="password" value={form.next} onChange={set('next')} autoComplete="new-password" minLength={8} required note="อย่างน้อย 8 ตัวอักษร" /><Field label="ยืนยันรหัสผ่านใหม่" type="password" value={form.confirm} onChange={set('confirm')} autoComplete="new-password" minLength={8} required /></div>
      <Button className="primary" type="submit" busy={busy}>เปลี่ยนรหัสผ่าน</Button>
    </form>
  </section>
}

function Auth({ onAuth }) {
  const invite = location.pathname.startsWith('/invite/') ? location.pathname.split('/').pop() : ''
  const [mode, setMode] = useState(invite ? 'register' : 'login')
  const [info, setInfo] = useState(null)
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => { if (invite) request('/invite/' + invite).then(v => { setInfo(v); setEmail(v.email) }).catch(e => { setError(e.message); toast.error(e.message) }) }, [invite])
  async function submit(e) {
    e.preventDefault(); setError(''); setBusy(true)
    try { await post('/auth/' + (mode === 'login' ? 'login' : 'register'), { email, password, name, invite }); history.replaceState({}, '', '/'); await onAuth(); toast.success('เข้าสู่ระบบแล้ว') }
    catch (e) { setError(e.message); toast.error(e.message) } finally { setBusy(false) }
  }
  return <main className="auth-layout">
    <div className="auth-panel">
      <div className="auth-brand"><img className="brand-logo" src="/brand-mark.svg" alt="" /><strong>สมุดฝึกงาน</strong></div>
      <div className="auth-editorial"><span className="kicker">บันทึก / รับรอง / ส่งรายงาน</span><h1>ทุกวันที่ฝึกงาน<br />อยู่ในที่เดียว</h1><p>จดงานที่ทำ เก็บเวลาให้ครบ และเตรียมเอกสารจากข้อมูลจริงของคุณ</p></div>
      <div className="auth-steps"><span>01&nbsp; บันทึกประจำวัน</span><span>02&nbsp; พี่เลี้ยงรับรอง</span><span>03&nbsp; ส่งออกเอกสาร</span></div>
    </div>
    <section className="auth-card"><p className="eyebrow">เริ่มต้นใช้งาน</p><h2>{invite ? 'ตอบรับคำเชิญ' : mode === 'login' ? 'เข้าสู่ระบบ' : 'สร้างบัญชีนักศึกษา'}</h2>{info && <p className="muted">ดูแลการฝึกงานของ {info.student} ที่ {info.company || 'สถานประกอบการ'}</p>}
      <form onSubmit={submit}>{mode === 'register' && <Field label="ชื่อ–นามสกุล" value={name} onChange={setName} maxLength={150} required />}<Field label="อีเมล" type="email" value={email} onChange={setEmail} disabled={Boolean(invite && info)} required note={invite && info ? 'อีเมลนี้ผูกกับคำเชิญ' : ''} /><Field label="รหัสผ่าน" type="password" value={password} onChange={setPassword} minLength={mode === 'register' ? 8 : undefined} required note={mode === 'register' ? 'อย่างน้อย 8 ตัวอักษร' : ''} />{error && <p className="error">{error}</p>}<Button className="primary full" type="submit" busy={busy} disabled={Boolean(invite && !info)}>{mode === 'login' ? 'เข้าสู่ระบบ' : 'สร้างบัญชี'}</Button></form>
      {!invite && <button className="text-button" onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError('') }}>{mode === 'login' ? 'ยังไม่มีบัญชี? สมัครเป็นนักศึกษา' : 'มีบัญชีแล้ว? เข้าสู่ระบบ'}</button>}
    </section>
  </main>
}

function SignedInInvite({ user, onSignout, onBack }) {
  const token = location.pathname.split('/').pop()
  const [info, setInfo] = useState(null)
  const [error, setError] = useState('')
  useEffect(() => { request('/invite/' + token).then(setInfo).catch(e => setError(e.message)) }, [token])
  return <main className="auth-layout"><div className="auth-panel"><div className="auth-brand"><img className="brand-logo" src="/brand-mark.svg" alt="" /><strong>สมุดฝึกงาน</strong></div><div className="auth-editorial"><span className="kicker">คำเชิญพี่เลี้ยง</span><h1>ลิงก์เชิญพี่เลี้ยง</h1><p>ลิงก์นี้ใช้สมัครบัญชีพี่เลี้ยงเพื่อดูแลการฝึกงาน</p></div></div><section className="auth-card"><p className="eyebrow">ตรวจคำเชิญ</p><h2>{error ? 'ลิงก์ใช้ไม่ได้' : info ? 'กำลังเข้าสู่ระบบอยู่' : 'กำลังตรวจลิงก์…'}</h2>{error ? <p className="error">{error}</p> : info && <><p className="muted">คำเชิญสำหรับ {info.email} เพื่อดูแล {info.student}</p><p className="mt-4 text-sm text-slate-700">บัญชีปัจจุบัน: {user.name} ({user.role === 'student' ? 'นักศึกษา' : 'พี่เลี้ยง'}) หากพี่เลี้ยงจะรับคำเชิญ ให้เปิดลิงก์ในหน้าต่างส่วนตัว หรือออกจากบัญชีนี้ก่อน ระบบจะเก็บลิงก์ไว้บนหน้านี้</p><button className="primary full mt-5" onClick={onSignout}>ออกจากระบบเพื่อรับคำเชิญ</button></>}<button className="text-button mt-4" onClick={onBack}>กลับไปสมุดฝึกงาน</button></section></main>
}

function App() {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState('overview')
  const [touring, setTouring] = useState(false)
  const [avatarVersion, setAvatarVersion] = useState(() => Date.now())
  const [collapsed, setCollapsed] = useState(() => { try { return localStorage.getItem('sidebar-collapsed') === '1' } catch { return false } })
  function toggleSidebar() {
    setCollapsed(v => { try { localStorage.setItem('sidebar-collapsed', v ? '0' : '1') } catch {}; return !v })
  }
  const [ai, setAi] = useState(false)
  const [logDate, setLogDate] = useState('')
  const [notify, setNotify] = useState(() => { try { return localStorage.getItem('log-reminder') === 'on' } catch { return false } })
  const dataRef = useRef(null)
  dataRef.current = data
  useEffect(() => {
    if (data?.user?.role !== 'student') return
    request('/ai/status').then(v => setAi(Boolean(v?.enabled))).catch(() => setAi(false))
    const { missed } = missedDays(data)
    try {
      const key = `missed-toast-${data.user.id}-${todayISO()}`
      if (missed.length && !sessionStorage.getItem(key)) {
        sessionStorage.setItem(key, '1')
        toast(`ยังไม่ได้บันทึกงาน ${missed.length} วันทำงาน ดูได้ที่หน้าภาพรวม`, { icon: '⏰', duration: 6000 })
      }
    } catch {}
  }, [data?.user?.id])
  // Browser reminder at 17:30 on workdays without a log (works while the site is open).
  useEffect(() => {
    if (!notify || data?.user?.role !== 'student') return
    const check = () => {
      const current = dataRef.current
      if (!current || typeof Notification === 'undefined' || Notification.permission !== 'granted') return
      if (bangkokClock() < '17:30' || !missedDays(current).todayMissing) return
      const key = `log-reminder-sent-${current.user.id}`
      try { if (localStorage.getItem(key) === todayISO()) return; localStorage.setItem(key, todayISO()) } catch {}
      const n = new Notification('สมุดฝึกงาน', { body: 'วันนี้ยังไม่ได้บันทึกงาน อย่าลืมบันทึกก่อนเลิกงานนะ', tag: 'log-reminder' })
      n.onclick = () => { window.focus(); setTab('logs'); n.close() }
    }
    check()
    const timer = setInterval(check, 60 * 1000)
    return () => clearInterval(timer)
  }, [notify, data?.user?.id])
  async function toggleNotify() {
    if (notify) { setNotify(false); try { localStorage.setItem('log-reminder', 'off') } catch {}; toast.success('ปิดการแจ้งเตือนแล้ว'); return }
    if (typeof Notification === 'undefined') { toast.error('เบราว์เซอร์นี้ไม่รองรับการแจ้งเตือน'); return }
    const permission = Notification.permission === 'granted' ? 'granted' : await Notification.requestPermission()
    if (permission !== 'granted') { toast.error('ต้องอนุญาตการแจ้งเตือนในเบราว์เซอร์ก่อน'); return }
    setNotify(true)
    try { localStorage.setItem('log-reminder', 'on') } catch {}
    toast.success('เปิดแจ้งเตือนแล้ว จะเตือนตอน 17:30 ถ้าวันนั้นยังไม่บันทึก')
  }
  function startLog(date) { setLogDate(date); setTab('logs') }
  const tourKey = data?.user ? `tour-done-v1-${data.user.id}` : ''
  useEffect(() => {
    if (data?.user?.role !== 'student' || location.pathname.startsWith('/invite/')) return
    try { if (!localStorage.getItem(tourKey)) setTouring(true) } catch {}
  }, [tourKey])
  function closeTour() {
    setTouring(false)
    try { localStorage.setItem(tourKey, '1') } catch {}
  }
  async function load() {
    try { const v = await request('/data'); setData(v) } catch { setData(null) } finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  async function action(fn, success) {
    try { await fn(); await load(); toast.success(success || 'บันทึกแล้ว'); return true }
    catch (e) { toast.error(e.message); return false }
  }
  async function signout() {
    try { await request('/auth/logout', { method: 'POST' }); setData(null); setTab('overview'); toast.success('ออกจากระบบแล้ว') }
    catch (e) { toast.error(e.message) }
  }
  if (loading) return <div className="loading">กำลังเปิดสมุดฝึกงาน…</div>
  if (!data) return <Auth onAuth={load} />
  if (location.pathname.startsWith('/invite/')) return <SignedInInvite user={data.user} onSignout={signout} onBack={() => { history.replaceState({}, '', '/'); setTab('overview') }} />
  const student = data.user.role === 'student'
  const tabs = [
    ['overview', 'ภาพรวม', LayoutDashboard],
    ['profile', 'ข้อมูลส่วนตัว', UserRound],
    ['logs', student ? 'บันทึกรายวัน' : 'บันทึกของนักศึกษา', CalendarDays],
    ['approvals', 'การรับรอง', BadgeCheck],
    ['evaluation', 'แบบประเมิน', ClipboardCheck],
    ['report', 'รายงาน', BookOpenText],
    ['documents', 'เอกสาร', Files],
  ].filter(([key]) => student ? key !== 'evaluation' : key !== 'report' && key !== 'documents')
  if (data.user.is_admin) tabs.push(['admin', 'ผู้ใช้งานระบบ', Users])
  const headings = Object.fromEntries(tabs.map(([key, label]) => [key, label]))
  const groupKeys = student
    ? [['ภาพรวม', ['overview']], ['การฝึกงาน', ['profile', 'logs', 'approvals']], ['รายงานและเอกสาร', ['report', 'documents']], ['ผู้ดูแลระบบ', ['admin']]]
    : [['ภาพรวม', ['overview']], ['นักศึกษาที่ดูแล', ['logs', 'approvals', 'evaluation']], ['บัญชี', ['profile']], ['ผู้ดูแลระบบ', ['admin']]]
  const navGroups = groupKeys.map(([title, keys]) => [title, tabs.filter(([key]) => keys.includes(key))]).filter(([, items]) => items.length)
  return <div className="app-shell">
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
      <div className="brand"><img className="brand-logo" src="/brand-mark.svg" alt="" /><strong>สมุดฝึกงาน</strong><button type="button" className="collapse-button" aria-label={collapsed ? 'ขยายเมนู' : 'ย่อเมนู'} title={collapsed ? 'ขยายเมนู' : 'ย่อเมนู'} onClick={toggleSidebar}><ChevronLeft size={16} aria-hidden="true" /></button></div>
      <nav aria-label="เมนูหลัก" data-tour="nav">{navGroups.map(([title, items]) => <div className="nav-group" key={title}><p className="nav-group-title">{title}</p>{items.map(([key, label, Icon]) => <button key={key} className={tab === key ? 'active' : ''} aria-current={tab === key ? 'page' : undefined} title={collapsed ? label : undefined} onClick={() => setTab(key)}><Icon size={18} strokeWidth={1.9} aria-hidden="true" /><span>{label}</span></button>)}</div>)}</nav>
      <div className="sidebar-footer"><Avatar user={data.user} version={avatarVersion} /><div className="sidebar-user"><strong>{data.user.name}</strong><small>{student ? 'นักศึกษา' : 'พี่เลี้ยง'}</small></div><button aria-label="ออกจากระบบ" title="ออกจากระบบ" onClick={signout}><LogOut size={17} aria-hidden="true" /></button></div>
      <p className="sidebar-copy">© 2026 สมุดฝึกงาน · มรภ.นครปฐม</p>
    </aside>
    <div className="main-shell">
      <header className="topbar">
        <div className="topbar-title"><nav className="breadcrumb" aria-label="เส้นทางการนำทาง"><button onClick={() => setTab('overview')}>สมุดฝึกงาน</button><ChevronRight size={13} aria-hidden="true" /><span aria-current="page">{headings[tab]}</span></nav><h1>{headings[tab]}</h1></div>
        <div className="topbar-right">{student && <button type="button" data-tour="tour-button" className="inline-flex items-center gap-1.5 rounded-full border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-blue-900 hover:bg-slate-50" onClick={() => setTouring(true)}><CircleHelp size={15} aria-hidden="true" />คู่มือ</button>}<span className="pill"><Building2 size={14} aria-hidden="true" />{data.internship?.company || 'ยังไม่ระบุบริษัท'}</span><Avatar user={data.user} version={avatarVersion} className="avatar small" title={data.user.name} /></div>
      </header>
      <main className="content">{tab === 'overview' && <Overview data={data} student={student} go={setTab} startLog={startLog} notify={notify} toggleNotify={toggleNotify} />}{tab === 'profile' && <Profile data={data} student={student} action={action} refresh={load} avatarVersion={avatarVersion} onPhotoChanged={() => setAvatarVersion(Date.now())} />}{tab === 'logs' && <Logs data={data} student={student} action={action} ai={ai} initialDate={logDate} onDateUsed={() => setLogDate('')} />}{tab === 'approvals' && <Approvals data={data} student={student} action={action} />}{tab === 'evaluation' && !student && <Evaluation data={data} student={student} action={action} />}{tab === 'report' && student && <Report data={data} student={student} action={action} ai={ai} />}{tab === 'documents' && student && <Documents data={data} student={student} go={setTab} />}{tab === 'admin' && data.user.is_admin && <AdminUsers me={data.user} />}</main>
    </div>
    {touring && student && <Tour steps={studentTour} go={setTab} onClose={closeTour} />}
  </div>
}

const studentTour = [
  { tab: 'overview', title: 'ยินดีต้อนรับสู่สมุดฝึกงาน 👋', body: 'คู่มือนี้จะพาไปทีละหน้า ตั้งแต่กรอกข้อมูล บันทึกงานรายวัน ให้พี่เลี้ยงรับรอง จนถึงดาวน์โหลดเอกสารส่งอาจารย์ ใช้เวลาประมาณ 2 นาที' },
  { tab: 'overview', target: 'nav', title: 'เมนูหลัก', body: 'ทุกหน้าที่ต้องใช้อยู่ตรงนี้ เรียงตามลำดับการทำงาน: ข้อมูลส่วนตัว → บันทึกรายวัน → การรับรอง → รายงาน → เอกสาร' },
  { tab: 'profile', target: 'photo', title: '1. อัปโหลดรูปโปรไฟล์', body: 'ใช้รูปหน้าตรง JPG หรือ PNG ไม่เกิน 2 MB รูปนี้จะไปอยู่ในหน้าประวัติของแบบบันทึก' },
  { tab: 'profile', target: 'profile-form', title: '2. กรอกข้อมูลส่วนตัว', body: 'ชื่อ เบอร์โทร วันเกิด ที่อยู่ ความสามารถ และความสนใจพิเศษ ไม่ต้องกดปุ่มเซฟ ระบบบันทึกอัตโนมัติ ดูสถานะได้ที่มุมขวาบนของกล่อง' },
  { tab: 'profile', target: 'internship-form', title: '3. กรอกข้อมูลการฝึกงาน', body: 'ใส่รหัสนักศึกษา (6642590xx) บริษัท แผนก ตำแหน่ง และอีเมลพี่เลี้ยง ถ้าข้อมูลยังไม่ถูกต้อง สถานะจะบอกว่าต้องแก้ตรงไหนก่อนบันทึก' },
  { tab: 'profile', target: 'plan', title: '4. ตั้งแผนและช่วงวันฝึก', body: 'เลือกฝึกงานทั่วไปหรือสหกิจศึกษา กรอกวันเริ่ม–วันสิ้นสุด และวันหยุดที่ต้องหัก ระบบจะคำนวณชั่วโมงตามแผนให้' },
  { tab: 'profile', target: 'invite', title: '5. เชิญพี่เลี้ยง', body: 'พอบันทึกอีเมลพี่เลี้ยงแล้ว กดสร้างลิงก์คำเชิญแล้วส่งให้พี่เลี้ยงเอง (LINE/อีเมล) ลิงก์ใช้ได้ 7 วัน พี่เลี้ยงต้องสมัครด้วยอีเมลเดียวกับที่ระบุไว้' },
  { tab: 'logs', target: 'log-form', title: '6. บันทึกงานทุกวัน', body: 'เลือกวันที่ เวลาเข้า–ออก เวลาพัก แล้วเขียนงานที่ทำ ปัญหา และสิ่งที่ได้เรียนรู้ ถ้าลา/หยุด ให้เปลี่ยนประเภทบันทึก' },
  { tab: 'logs', target: 'csv-import', title: 'นำเข้าทีละหลายวัน', body: 'ถ้าจดไว้ที่อื่นแล้ว ดาวน์โหลดแม่แบบ CSV กรอกแล้วอัปโหลดได้เลย ระบบจะตรวจวันซ้ำและข้อมูลผิดให้ก่อนนำเข้า' },
  { tab: 'logs', target: 'log-history', title: 'แก้ไขบันทึกย้อนหลัง', body: 'รายการที่บันทึกแล้วอยู่ตรงนี้ แก้ไขหรือลบได้จนกว่าพี่เลี้ยงจะรับรอง หลังรับรองแล้วจะถูกล็อก' },
  { tab: 'logs', target: 'weekly-summary', title: 'ผู้ช่วย AI ✨', body: 'จดโน้ตสั้นๆ แล้วกด “เกลาด้วย AI” ใต้ช่องงานที่ทำ/ปัญหา/สิ่งที่เรียนรู้ ให้เป็นภาษาทางการ และใช้กล่องนี้สรุปงานรายสัปดาห์ได้ ทุกข้อความจาก AI ต้องกดยอมรับก่อนถึงจะใช้ (ปุ่ม AI จะขึ้นเมื่อผู้ดูแลระบบเปิดใช้)' },
  { tab: 'approvals', target: 'approvals', title: '7. ติดตามการรับรอง', body: 'พี่เลี้ยงจะรับรองบันทึกเป็นรอบรายสัปดาห์หรือรายเดือน หน้านี้แสดงว่ารอบไหนรับรองแล้ว ถ้ามีค้างให้เตือนพี่เลี้ยง' },
  { tab: 'report', target: 'report', title: '8. เขียนรายงาน', body: 'เขียนเนื้อหา 4 บทจากงานที่ทำจริง ไม่ต้องใส่ข้อมูลลับของบริษัท เขียนเสร็จแต่ละช่วงแล้วกดบันทึกรายงานด้านล่าง' },
  { tab: 'documents', target: 'documents', title: '9. ดาวน์โหลดเอกสาร', body: 'กดดูตัวอย่างก่อนได้ทุกเมื่อ พร้อมแล้วดาวน์โหลดเป็น DOCX หรือ PDF เพื่อส่งอาจารย์ แบบประเมินจะดาวน์โหลดได้หลังพี่เลี้ยงส่งผล' },
  { tab: 'overview', target: 'tasks', title: 'งานถัดไป', body: 'หน้าภาพรวมจะสรุปว่ายังเหลืออะไรต้องทำ กดที่รายการเพื่อไปยังหน้านั้นได้เลย' },
  { tab: 'overview', target: 'reminder-toggle', title: 'กันลืมบันทึก ⏰', body: 'ถ้ามีวันทำงานที่ยังไม่ได้บันทึก จะขึ้นแถบเตือนที่หน้านี้ กดที่วันที่เพื่อบันทึกย้อนหลังได้เลย เปิดแจ้งเตือนในเบราว์เซอร์ไว้ ระบบจะเตือนตอน 17:30 ถ้าวันนั้นยังไม่บันทึก' },
  { tab: 'overview', target: 'tour-button', title: 'จบคู่มือแล้ว 🎉', body: 'ถ้าลืมขั้นตอนไหน กดปุ่ม “คู่มือ” ตรงนี้เพื่อเปิดดูอีกครั้งได้ตลอด' },
]

// Step-by-step tour: switches tab, spotlights the target element, and shows a card beside it.
function Tour({ steps, go, onClose }) {
  const [index, setIndex] = useState(0)
  const [rect, setRect] = useState(null)
  const [viewport, setViewport] = useState({ w: window.innerWidth, h: window.innerHeight })
  const elementRef = useRef(null)
  const step = steps[index]
  const last = index === steps.length - 1
  useEffect(() => {
    let cancelled = false
    let frame = 0
    let tries = 0
    elementRef.current = null
    setRect(null)
    go(step.tab)
    if (!step.target) { scrollTo({ top: 0, behavior: 'smooth' }); return }
    const find = () => {
      if (cancelled) return
      const el = document.querySelector(`[data-tour="${step.target}"]`)
      if (el && el.getClientRects().length) {
        elementRef.current = el
        const tall = el.getBoundingClientRect().height > window.innerHeight * 0.55
        el.scrollIntoView({ block: tall ? 'start' : 'center', behavior: 'smooth' })
        setTimeout(() => { if (!cancelled) setRect(el.getBoundingClientRect()) }, 380)
      } else if (tries++ < 90) frame = requestAnimationFrame(find)
    }
    frame = requestAnimationFrame(find)
    return () => { cancelled = true; cancelAnimationFrame(frame) }
  }, [index])
  useEffect(() => {
    const update = () => {
      setViewport({ w: window.innerWidth, h: window.innerHeight })
      if (elementRef.current) setRect(elementRef.current.getBoundingClientRect())
    }
    const keys = event => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowRight') setIndex(i => Math.min(i + 1, steps.length - 1))
      if (event.key === 'ArrowLeft') setIndex(i => Math.max(i - 1, 0))
    }
    window.addEventListener('resize', update)
    window.addEventListener('scroll', update, true)
    window.addEventListener('keydown', keys)
    return () => { window.removeEventListener('resize', update); window.removeEventListener('scroll', update, true); window.removeEventListener('keydown', keys) }
  }, [])
  const pad = 8
  const hole = rect && {
    left: Math.max(rect.left - pad, 4), top: Math.max(rect.top - pad, 4),
    right: Math.min(rect.right + pad, viewport.w - 4), bottom: Math.min(rect.bottom + pad, viewport.h - 4),
  }
  const cardWidth = Math.min(360, viewport.w - 24)
  const cardHeight = 230
  let cardStyle = { width: cardWidth, left: (viewport.w - cardWidth) / 2, top: Math.max((viewport.h - cardHeight) / 2, 12) }
  if (hole) {
    const left = Math.min(Math.max(hole.left, 12), viewport.w - cardWidth - 12)
    if (viewport.h - hole.bottom >= cardHeight + 12) cardStyle = { width: cardWidth, left, top: hole.bottom + 12 }
    else if (hole.top >= cardHeight + 12) cardStyle = { width: cardWidth, left, top: hole.top - cardHeight - 12 }
    else if (viewport.w - hole.right >= cardWidth + 24) cardStyle = { width: cardWidth, left: hole.right + 12, top: Math.min(Math.max(hole.top, 12), viewport.h - cardHeight - 12) }
    else cardStyle = { width: cardWidth, left: (viewport.w - cardWidth) / 2, top: viewport.h - cardHeight - 16 }
  }
  return <div className="fixed inset-0 z-[60]" role="dialog" aria-modal="true" aria-label="คู่มือการใช้งาน">
    {hole
      ? <div className="pointer-events-none fixed rounded-xl ring-2 ring-blue-500 transition-all duration-300" style={{ left: hole.left, top: hole.top, width: hole.right - hole.left, height: hole.bottom - hole.top, boxShadow: '0 0 0 9999px rgba(2, 6, 23, 0.6)' }} />
      : <div className="fixed inset-0 bg-slate-950/60" />}
    <div className="fixed flex flex-col rounded-2xl bg-white p-5 shadow-2xl transition-all duration-300" style={{ ...cardStyle, minHeight: cardHeight - 40 }}>
      <div className="mb-2 flex items-center justify-between"><span className="text-[11px] font-semibold tracking-widest text-blue-700">ขั้นตอน {index + 1} / {steps.length}</span><button type="button" aria-label="ปิดคู่มือ" className="grid place-items-center rounded-md p-1 text-slate-500 hover:bg-slate-100" onClick={onClose}><X size={16} aria-hidden="true" /></button></div>
      <h3 className="text-base font-semibold text-blue-950">{step.title}</h3>
      <p className="mt-2 text-sm leading-relaxed text-slate-600">{step.body}</p>
      <div className="mt-4 flex gap-1" aria-hidden="true">{steps.map((_, i) => <span key={i} className={`h-1 flex-1 rounded-full ${i <= index ? 'bg-blue-600' : 'bg-slate-200'}`} />)}</div>
      <div className="mt-4 flex items-center justify-between gap-2">
        {last ? <span /> : <button type="button" className="text-xs font-semibold text-slate-500 hover:text-slate-800" onClick={onClose}>ข้ามคู่มือ</button>}
        <div className="flex gap-2">
          {index > 0 && <button type="button" className="rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50" onClick={() => setIndex(i => i - 1)}>ย้อนกลับ</button>}
          <button type="button" autoFocus className="rounded-lg bg-blue-700 px-4 py-2 text-xs font-semibold text-white hover:bg-blue-800" onClick={() => last ? onClose() : setIndex(i => i + 1)}>{last ? 'เริ่มใช้งาน' : 'ถัดไป'}</button>
        </div>
      </div>
    </div>
  </div>
}

function Overview({ data, student, go, startLog, notify, toggleNotify }) {
  const logs = data.logs || []
  const approved = logs.filter(x => x.approval_id).length
  const pending = logs.length - approved
  const hours = logs.reduce((a, x) => a + x.hours, 0)
  const firstName = data.user.name?.split(/\s+/)[0] || data.user.name
  const today = new Intl.DateTimeFormat('th-TH', { dateStyle: 'full', timeZone: 'Asia/Bangkok' }).format(new Date())
  const { missed, todayMissing } = student ? missedDays(data) : { missed: [], todayMissing: false }
  const tasks = student ? [
    missed.length > 0 && [`ยังไม่บันทึก ${missed.length} วัน`, 'วันทำงานที่ผ่านมาแล้วยังไม่มีบันทึก', 'logs'],
    !data.internship?.company && ['กรอกข้อมูลการฝึกงาน', 'ชื่อบริษัท ช่วงเวลา และพี่เลี้ยง', 'profile'],
    !logs.length && ['เริ่มบันทึกงาน', 'ใช้เวลาเพียงไม่กี่นาทีต่อวัน', 'logs'],
    pending > 0 && [`รอรับรอง ${pending} รายการ`, 'ติดตามรอบการรับรองกับพี่เลี้ยง', 'approvals'],
    !Object.values(data.internship?.report || {}).some(Boolean) && ['เริ่มร่างรายงาน', 'เขียนจากงานที่ทำจริง', 'report'],
  ].filter(Boolean) : [
    pending > 0 && [`ตรวจบันทึก ${pending} รายการ`, 'เลือกรอบสัปดาห์หรือเดือนที่ต้องการรับรอง', 'approvals'],
    !data.evaluation?.submitted_at && ['กรอกแบบประเมิน', 'ประเมิน 22 ข้อและส่งผลเมื่อพร้อม', 'evaluation'],
  ].filter(Boolean)
  return <div className="dashboard">
    <section className="overview-intro"><div><p className="date-overline">{today}</p><h2>สวัสดี, {firstName}</h2><p>{student ? 'วันนี้มีอะไรที่ต้องบันทึกไว้บ้าง' : `กำลังดูแลการฝึกงานที่ ${data.internship?.company || 'สถานประกอบการ'}`}</p></div><button className="primary" onClick={() => go('logs')}>{student && <Plus size={17} aria-hidden="true" />}{student ? 'บันทึกงาน' : 'ดูบันทึกนักศึกษา'}</button></section>
    {student && (missed.length > 0 || todayMissing) && <section className="flex flex-col gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 sm:flex-row sm:items-start sm:justify-between" role="alert" data-tour="reminder">
      <div className="flex gap-2.5"><TriangleAlert size={18} className="mt-0.5 shrink-0" aria-hidden="true" /><div>
        <strong className="block">{missed.length ? `ลืมบันทึก ${missed.length} วันทำงาน` : 'วันนี้ยังไม่ได้บันทึกงาน'}</strong>
        {missed.length > 0 && <div className="mt-2 flex flex-wrap gap-1.5">{missed.slice(0, 8).map(d => <button key={d} type="button" className="rounded-full border border-amber-300 bg-white px-2.5 py-0.5 text-xs font-medium hover:bg-amber-100" onClick={() => startLog(d)}>{thaiShortDate(d)}</button>)}{missed.length > 8 && <span className="text-xs">และอีก {missed.length - 8} วัน</span>}</div>}
        <small className="mt-1.5 block text-xs text-amber-800">กดที่วันที่เพื่อบันทึกย้อนหลัง ถ้าเป็นวันหยุด ให้เพิ่มในวันหยุดที่หน้าข้อมูลส่วนตัว</small>
      </div></div>
      <div className="flex shrink-0 gap-2">{todayMissing && <button type="button" className="rounded-md bg-amber-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-amber-700" onClick={() => startLog(todayISO())}>บันทึกวันนี้</button>}</div>
    </section>}
    {student && <div className="-mt-1 mb-3 flex justify-end"><button type="button" className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold ${notify ? 'border-blue-200 bg-blue-50 text-blue-800' : 'border-slate-300 bg-white text-slate-600'}`} data-tour="reminder-toggle" onClick={toggleNotify} title="แจ้งเตือนตอน 17:30 ถ้ายังไม่บันทึก (ต้องเปิดเว็บค้างไว้)"><BellRing size={14} aria-hidden="true" />{notify ? 'แจ้งเตือน 17:30 เปิดอยู่' : 'เปิดแจ้งเตือนลืมบันทึก'}</button></div>}
    <section className="summary-band" aria-label="สรุปการฝึกงาน"><div><CalendarDays size={18} className="summary-icon" aria-hidden="true" /><span>วันที่บันทึก</span><strong>{logs.length}</strong><small>วัน</small></div><div><Clock3 size={18} className="summary-icon" aria-hidden="true" /><span>ชั่วโมงปฏิบัติงาน</span><strong>{hours.toFixed(1)}</strong><small>ชั่วโมง</small></div><div><CheckCircle2 size={18} className="summary-icon" aria-hidden="true" /><span>รับรองแล้ว</span><strong>{approved}</strong><small>รายการ</small></div>{student ? <div><ListChecks size={18} className="summary-icon" aria-hidden="true" /><span>แผนการฝึก</span><strong className="summary-text">{data.internship?.plan_type === 'cooperative' ? 'สหกิจศึกษา' : 'ฝึกงาน'}</strong></div> : <div><ListChecks size={18} className="summary-icon" aria-hidden="true" /><span>แบบประเมิน</span><strong className="summary-text">{data.evaluation?.submitted_at ? 'ส่งแล้ว' : 'ยังไม่ส่ง'}</strong></div>}</section>
    <div className="dashboard-columns"><section className="card tasks-panel" data-tour="tasks"><div className="section-head"><div><span className="section-index">01 / สิ่งที่ต้องทำ</span><h2>งานถัดไป</h2></div></div>{tasks.length ? <div className="task-list">{tasks.map(([title, detail, target], i) => <button key={i} onClick={() => go(target)}><span className="task-number">{String(i + 1).padStart(2, '0')}</span><span><strong>{title}</strong><small>{detail}</small></span><ArrowUpRight size={16} aria-hidden="true" /></button>)}</div> : <p className="empty">{student ? 'รายการสำคัญครบแล้ว คุณสามารถตรวจเอกสารก่อนส่งได้' : 'ไม่มีงานที่ต้องรับรองในตอนนี้'}</p>}</section><section className="card recent-panel"><div className="section-head"><div><span className="section-index">02 / งานที่ผ่านมา</span><h2>บันทึกล่าสุด</h2></div><button className="text-button" onClick={() => go('logs')}><span>ดูทั้งหมด</span><ArrowUpRight size={14} aria-hidden="true" /></button></div>{logs.length ? <div className="recent-table">{logs.slice(0, 5).map(l => <div key={l.id}><span>{formatDate(l.date)}</span><strong>{l.work || kindName(l.kind)}</strong><small>{l.hours.toFixed(1)} ชม.</small><em className={l.approval_id ? 'is-approved' : ''}>{l.approval_id ? 'รับรองแล้ว' : 'รอรับรอง'}</em></div>)}</div> : <p className="empty">ยังไม่มีบันทึกประจำวัน</p>}</section></div>
  </div>
}

function InvitePanel({ internship, invite, onCreate, canInvite }) {
  const inputRef = useRef(null)
  const localOnly = ['localhost', '127.0.0.1', '::1', '[::1]'].includes(location.hostname)
  async function copyLink() {
    let copied = false
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(invite)
        copied = true
      }
    } catch { copied = false }
    if (!copied && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
      try { copied = document.execCommand('copy') } catch { copied = false }
    }
    if (copied) toast.success('คัดลอกลิงก์แล้ว')
    else toast.error('เลือกลิงก์ไว้แล้ว กด Ctrl+C เพื่อคัดลอก')
  }
  const mailSubject = encodeURIComponent('คำเชิญเป็นพี่เลี้ยงฝึกงาน')
  const mailBody = encodeURIComponent(`กรุณาเปิดลิงก์นี้เพื่อสมัครเป็นพี่เลี้ยงและดูแลการฝึกงาน:\n${invite}\n\nกรุณาสมัครด้วยอีเมล ${internship.mentor_email}`)
  return <>
    <div className="invite-box"><div><strong>เชื่อมพี่เลี้ยง</strong><p>{internship.mentor_linked ? `เชื่อมแล้ว: ${internship.mentor_name} · ไม่ต้องส่งคำเชิญซ้ำ` : canInvite ? 'นักศึกษาสร้างลิงก์ แล้วส่งให้พี่เลี้ยงสมัครด้วยอีเมลที่บันทึกไว้' : 'กรอกอีเมลพี่เลี้ยงและบันทึกข้อมูลก่อนสร้างลิงก์'}</p></div>{!internship.mentor_linked && <button type="button" onClick={onCreate} disabled={!canInvite}><Plus size={15} aria-hidden="true" />สร้างลิงก์คำเชิญ</button>}</div>
    {invite && <div className="invite-link"><input ref={inputRef} readOnly value={invite} aria-label="ลิงก์คำเชิญ" onFocus={event => event.target.select()} /><button type="button" onClick={copyLink}><Copy size={14} aria-hidden="true" />คัดลอก</button>{!localOnly && <a className="inline-flex items-center rounded-md border border-slate-300 px-3 py-2 text-xs font-semibold text-blue-900 hover:bg-slate-50" href={`mailto:${internship.mentor_email}?subject=${mailSubject}&body=${mailBody}`}>เปิดอีเมลเพื่อส่ง</a>}<small>ลิงก์หมดอายุใน 7 วัน ระบบไม่ส่งให้อัตโนมัติ</small></div>}
    {localOnly && !internship.mentor_linked && <p className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-950">เว็บนี้เปิดผ่าน 127.0.0.1 ซึ่งใช้ได้เฉพาะเครื่องนี้ ลิงก์ที่คัดลอกจะเปิดจากเครื่องพี่เลี้ยงไม่ได้ ต้องนำเว็บขึ้นโฮสต์ก่อนส่งให้คนอื่น</p>}
  </>
}

function Profile({ data, student, action, refresh, avatarVersion, onPhotoChanged }) {
  const [user, setUser] = useState(data.user)
  const [internship, setInternship] = useState(data.internship || {})
  const [invite, setInvite] = useState('')
  const dataRef = useRef(data)
  dataRef.current = data
  // Only sync fields the server controls; never overwrite what the user is typing.
  useEffect(() => { setInternship(x => ({ ...x, mentor_linked: data.internship?.mentor_linked, student_name: data.internship?.student_name })) }, [data.internship?.mentor_linked, data.internship?.student_name])
  const set = key => value => setInternship(x => ({ ...x, [key]: value }))
  const setUserField = key => value => setUser(x => ({ ...x, [key]: value }))
  const recordedDates = (data.logs || []).map(log => log.date)
  const firstRecordedDate = recordedDates.length ? recordedDates.reduce((a, b) => a < b ? a : b) : ''
  const lastRecordedDate = recordedDates.length ? recordedDates.reduce((a, b) => a > b ? a : b) : ''
  async function upload(event) {
    const file = event.target.files?.[0]
    if (!file) return
    if ((file.type && !['image/jpeg', 'image/png'].includes(file.type) && !/\.(png|jpe?g)$/i.test(file.name)) || file.size > 2 * 1024 * 1024) {
      toast.error('เลือกรูป JPG หรือ PNG ขนาดไม่เกิน 2 MB')
      event.target.value = ''
      return
    }
    const form = new FormData()
    form.append('photo', file)
    const ok = await action(() => request('/profile/photo', { method: 'POST', body: form }), 'อัปโหลดรูปแล้ว')
    if (ok) onPhotoChanged?.()
  }
  async function makeInvite() {
    await action(async () => {
      const value = await post('/invite', {})
      setInvite(location.origin + value.invite_url)
    }, 'สร้างลิงก์คำเชิญแล้ว ส่งให้พี่เลี้ยงด้วยช่องทางที่คุณเลือก')
  }
  const internshipSave = useAutosave(internship, {
    validate: value => {
      const current = dataRef.current.internship
      if (current?.mentor_linked && value.mentor_email !== current.mentor_email) return 'อีเมลพี่เลี้ยงที่เชื่อมแล้วแก้ไขไม่ได้'
      return validateInternship(value, dataRef.current.logs || [])
    },
    save: async value => {
      const previousMentorEmail = dataRef.current.internship?.mentor_email || ''
      await post('/internship', { ...value, university: 'มหาวิทยาลัยราชภัฏนครปฐม', year: '2569', term: '2' }, 'PUT')
      if (previousMentorEmail !== value.mentor_email) setInvite('')
      refresh()
    },
  })
  const profileSave = useAutosave(user, {
    validate: value => validateProfile(value, student, provinces, blankLog().date),
    save: async value => { await post('/profile', value, 'PUT'); refresh() },
  })
  const submitNow = save => event => { event.preventDefault(); save.flush() }
  return <div className="stack">
    <section className="card" data-tour="profile-form">
      <div className="section-head"><div><p className="eyebrow">ข้อมูลสำหรับเอกสาร</p><h2>ข้อมูลส่วนตัว</h2></div><SaveStatus state={profileSave} /></div>
      <div className="profile-row" data-tour="photo"><div className="profile-photo"><img src={`${data.user.photo_url}?v=${avatarVersion}`} onError={e => e.currentTarget.style.display = 'none'} onLoad={e => e.currentTarget.style.display = 'block'} alt="รูปโปรไฟล์" /><span>{data.user.name?.slice(0, 1)}</span></div><label className="upload-button"><ImagePlus size={16} aria-hidden="true" />อัปโหลดรูป JPG หรือ PNG<input type="file" accept="image/png,image/jpeg" onChange={upload} hidden /></label><small>ไม่เกิน 2 MB</small></div>
      <form onSubmit={submitNow(profileSave)}>
        <div className="form-grid"><Field label="ชื่อ–นามสกุล" value={user.name} onChange={setUserField('name')} maxLength={150} required /><Field label="เบอร์โทรศัพท์" value={user.phone} onChange={setUserField('phone')} maxLength={40} /><Field label="ตำแหน่ง" value={user.position} onChange={setUserField('position')} maxLength={120} placeholder={student ? 'นักศึกษาฝึกงาน' : 'Software Engineer'} />{student && <Field label="วันเดือนปีเกิด" type="date" value={user.birth_date} onChange={setUserField('birth_date')} max={blankLog().date} />}</div>
        {student && <><div className="mb-4 mt-3 border-t border-slate-200 pt-5"><h3 className="text-base font-semibold text-blue-950">ที่อยู่และข้อมูลสำหรับหน้าประวัติ</h3><p className="mt-1 text-xs text-slate-500">ใช้ในแบบบันทึกและลงเวลาตามฟอร์มมหาวิทยาลัย</p></div><Field label="บ้านเลขที่ ถนน/ซอย" value={user.home_address} onChange={setUserField('home_address')} maxLength={300} /><div className="form-grid"><Field label="ตำบล/แขวง" value={user.subdistrict} onChange={setUserField('subdistrict')} maxLength={100} /><Field label="อำเภอ/เขต" value={user.district} onChange={setUserField('district')} maxLength={100} /><label className="field"><span>จังหวัด</span><select value={user.province || ''} onChange={event => setUser(x => ({ ...x, province: event.target.value, country: event.target.value === 'other' ? x.country : 'ประเทศไทย' }))}><option value="">เลือกจังหวัด</option>{provinces.map(name => <option key={name} value={name}>{name}</option>)}<option value="other">อื่น ๆ — ต่างประเทศ</option></select><small>มีครบ 77 จังหวัด รวมกรุงเทพมหานคร</small></label>{user.province === 'other' && <Field label="ชื่อประเทศ" value={user.country === 'ประเทศไทย' ? '' : user.country} onChange={setUserField('country')} maxLength={100} required placeholder="เช่น ญี่ปุ่น" />}</div><div className="form-grid"><TextArea label="ความรู้ความสามารถพิเศษ" value={user.skills} onChange={setUserField('skills')} rows={3} maxLength={1000} /><TextArea label="ความสนใจพิเศษ" value={user.interests} onChange={setUserField('interests')} rows={3} maxLength={1000} /></div></>}
      </form>
    </section>
    {student ? <section className="card" data-tour="internship-form"><div className="section-head"><div><p className="eyebrow">สถานประกอบการและช่วงฝึก</p><h2>ข้อมูลการฝึกงาน</h2></div><SaveStatus state={internshipSave} /></div><form onSubmit={submitNow(internshipSave)}><div className="form-grid"><Field label="มหาวิทยาลัย" value="มหาวิทยาลัยราชภัฏนครปฐม" disabled note="มหาวิทยาลัยกำหนดไว้แล้ว" /><Field label="คณะ" value={internship.faculty} onChange={set('faculty')} /><Field label="สาขาวิชา" value={internship.major} onChange={set('major')} /><Field label="รหัสนักศึกษา" value={internship.student_number} onChange={value => set('student_number')(value.replace(/\D/g, '').slice(0, 9))} maxLength={9} pattern="6642590[0-9]{2}" title="รหัสต้องเป็น 6642590xx" placeholder="6642590xx" required /><Field label="รหัสรุ่น" value="66" disabled /><Field label="หมู่เรียน" value={internship.class_group} onChange={set('class_group')} maxLength={40} /><Field label="ปีการศึกษา" value="2569" disabled /><Field label="ภาคเรียน" value="2" disabled /><Field label="บริษัท/หน่วยงาน" value={internship.company} onChange={set('company')} maxLength={200} /><Field label="แผนก" value={internship.department} onChange={set('department')} /><Field label="ตำแหน่งฝึกงาน" value={internship.position} onChange={set('position')} /><Field label="ที่อยู่บริษัท" value={internship.company_address} onChange={set('company_address')} maxLength={400} /><Field label="อีเมลพี่เลี้ยง" type="email" value={internship.mentor_email} onChange={set('mentor_email')} disabled={internship.mentor_linked} note={internship.mentor_linked ? 'เชื่อมบัญชีแล้ว จึงเปลี่ยนอีเมลไม่ได้' : 'บันทึกก่อนสร้างคำเชิญ'} /></div><PlanSetup internship={internship} setInternship={setInternship} set={set} actualHours={(data.logs || []).reduce((sum, log) => sum + log.hours, 0)} firstRecordedDate={firstRecordedDate} lastRecordedDate={lastRecordedDate} /></form><div data-tour="invite"><InvitePanel internship={internship} invite={invite} onCreate={makeInvite} canInvite={Boolean(internship.mentor_email && internship.mentor_email === data.internship?.mentor_email)} /></div></section> : <section className="card"><h2>การฝึกงานที่ดูแล</h2><p className="muted">นักศึกษา: {internship.student_name || '—'}</p><p><strong>{internship.company || 'ยังไม่ระบุบริษัท'}</strong></p><p className="muted">แผน: {internship.plan_type === 'cooperative' ? 'สหกิจศึกษา' : 'ฝึกงานทั่วไป'} · {calculatePlan(internship.start_date, internship.end_date, internship.plan_type, internship.holiday_dates)?.hours ?? '—'} ชั่วโมงตามแผน</p><p className="muted">นักศึกษาจะบันทึกข้อมูลหลัก คุณสามารถตรวจและรับรองบันทึกในเมนูการรับรอง</p></section>}
    <ChangePassword />
  </div>
}

function PlanSetup({ internship, setInternship, set, actualHours, firstRecordedDate, lastRecordedDate }) {
  const [holidayInput, setHolidayInput] = useState('')
  const holidays = internship.holiday_dates || []
  const planType = internship.plan_type || 'internship'
  const summary = calculatePlan(internship.start_date, internship.end_date, planType, holidays)
  function addHoliday() {
    if (!holidayInput) return
    if (holidays.length >= 100) { toast.error('เพิ่มวันหยุดได้สูงสุด 100 วัน'); return }
    if (!summary) { toast.error('กรอกช่วงวันที่ฝึกงานก่อน'); return }
    if (holidayInput < internship.start_date || holidayInput > internship.end_date) { toast.error('วันหยุดต้องอยู่ในช่วงฝึกงาน'); return }
    if (new Date(`${holidayInput}T00:00:00Z`).getUTCDay() % 6 === 0) { toast.error('วันเสาร์–อาทิตย์ถูกหักออกแล้ว'); return }
    if (holidays.includes(holidayInput)) { toast.error('เพิ่มวันหยุดนี้แล้ว'); return }
    setInternship(x => ({ ...x, holiday_dates: [...(x.holiday_dates || []), holidayInput].sort() }))
    setHolidayInput('')
  }
  return <section className="mt-6 border-t border-slate-200 pt-6" aria-label="แผนและระยะเวลาการฝึกงาน" data-tour="plan">
    <div className="mb-4"><p className="eyebrow">แผนการฝึก</p><h3 className="text-lg font-semibold text-blue-950">เลือกประเภทและช่วงเวลา</h3><p className="mt-1 text-sm text-slate-500">ระบบนับวันจันทร์–ศุกร์ วันละ 8 ชั่วโมง และหักวันหยุดที่เพิ่มด้านล่าง</p></div>
    <div className="mb-5 grid gap-3 sm:grid-cols-2">
      {[['internship', 'ฝึกงานทั่วไป', 'ไม่น้อยกว่า 450 ชั่วโมง'], ['cooperative', 'สหกิจศึกษา', 'ไม่น้อยกว่า 16 สัปดาห์ และ 540 ชั่วโมง']].map(([value, label, detail]) => <label key={value} className={`flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors ${planType === value ? 'border-blue-700 bg-blue-50' : 'border-slate-200 bg-white hover:border-blue-300'}`}><input className="mt-1 accent-blue-700" type="radio" name="plan_type" value={value} checked={planType === value} onChange={() => set('plan_type')(value)} /><span><strong className="block text-sm text-slate-900">{label}</strong><small className="text-xs text-slate-500">{detail}</small></span></label>)}
    </div>
    <div className="form-grid"><Field label="เริ่มฝึกงาน" type="date" value={internship.start_date} onChange={set('start_date')} max={firstRecordedDate || internship.end_date || undefined} required={Boolean(firstRecordedDate)} note={firstRecordedDate ? `มีบันทึกแรกวันที่ ${formatDate(firstRecordedDate)} — เลื่อนวันเริ่มให้ช้ากว่านี้ไม่ได้` : ''} /><Field label="สิ้นสุดฝึกงาน" type="date" value={internship.end_date} onChange={set('end_date')} min={lastRecordedDate || internship.start_date || undefined} required={Boolean(lastRecordedDate)} note={lastRecordedDate ? `มีบันทึกล่าสุดวันที่ ${formatDate(lastRecordedDate)} — เลื่อนวันสิ้นสุดให้เร็วกว่านี้ไม่ได้` : ''} /></div>
    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4"><label className="mb-2 block text-xs font-semibold text-slate-700" htmlFor="holiday-date">วันหยุดนักขัตฤกษ์/วันหยุดของสถานประกอบการ</label><div className="flex flex-wrap gap-2"><input id="holiday-date" disabled={!summary || holidays.length >= 100} className="min-w-44 flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-blue-600" type="date" value={holidayInput} min={internship.start_date || undefined} max={internship.end_date || undefined} onChange={e => setHolidayInput(e.target.value)} /><button type="button" onClick={addHoliday} disabled={!summary || holidays.length >= 100} className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-semibold text-blue-900 hover:bg-blue-50"><Plus size={14} />เพิ่มวันหยุด</button></div><p className="mt-2 text-xs text-slate-500">วันหยุดต่างกันตามปีและบริษัท จึงเพิ่มวันที่จริงก่อนดูชั่วโมงสุทธิ</p>{holidays.length > 0 && <div className="mt-3 flex flex-wrap gap-2">{holidays.map(date => <span key={date} className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1 text-xs text-slate-700">{formatDate(date)}<button type="button" aria-label={`ลบวันหยุด ${formatDate(date)}`} className="border-0 bg-transparent p-0 text-slate-500 hover:text-red-700" onClick={() => setInternship(x => ({ ...x, holiday_dates: (x.holiday_dates || []).filter(d => d !== date) }))}>×</button></span>)}</div>}</div>
    {summary ? <div className="mt-4 rounded-xl border border-blue-200 bg-blue-50 p-4"><div className="grid gap-3 sm:grid-cols-3"><div><span className="block text-xs text-blue-800">จันทร์–ศุกร์ในช่วงที่เลือก</span><strong className="text-xl text-blue-950">{summary.weekdays} วัน</strong></div><div><span className="block text-xs text-blue-800">หักวันหยุด</span><strong className="text-xl text-blue-950">{summary.holidays} วัน</strong></div><div><span className="block text-xs text-blue-800">ชั่วโมงตามแผน</span><strong className="text-xl text-blue-950">{summary.hours} ชม.</strong></div></div><p className={`mt-3 border-t border-blue-200 pt-3 text-sm font-semibold ${summary.meetsRequirement ? 'text-blue-800' : 'text-amber-800'}`}>{summary.meetsRequirement ? 'ช่วงวันที่นี้ผ่านเกณฑ์ตามแผน' : `ช่วงวันที่นี้ยังไม่ถึงเกณฑ์ ${planType === 'cooperative' ? '80 วันทำงาน (16 สัปดาห์) และ ' : ''}${summary.requiredHours} ชั่วโมง`}</p><p className="mt-1 text-xs text-blue-900">บันทึกงานจริงแล้ว {actualHours.toFixed(1)} ชั่วโมง · ชั่วโมงตามแผนเป็นการประมาณ ไม่ใช่ชั่วโมงที่ปฏิบัติงานจริง</p></div> : <p className="mt-4 rounded-xl border border-dashed border-slate-300 p-4 text-sm text-slate-500">เลือกวันเริ่มและวันสิ้นสุดเพื่อดูจำนวนวันและชั่วโมงโดยอัตโนมัติ</p>}
  </section>
}

function CSVImport({ action }) {
  const inputRef = useRef(null)
  const [file, setFile] = useState(null)
  const [preview, setPreview] = useState(null)
  const [busy, setBusy] = useState(false)
  const requestId = useRef(0)
  async function inspect(selected) {
    const id = ++requestId.current
    setFile(selected)
    setPreview(null)
    setBusy(false)
    if (!selected) return
    if (selected.size > 2 * 1024 * 1024) { toast.error('ไฟล์ CSV ต้องไม่เกิน 2 MB'); return }
    const body = new FormData()
    body.append('file', selected)
    setBusy(true)
    try {
      const result = await request('/logs/import?preview=1', { method: 'POST', body })
      if (id === requestId.current) setPreview(result)
    } catch (error) { if (id === requestId.current) toast.error(error.message) }
    finally { if (id === requestId.current) setBusy(false) }
  }
  async function importFile() {
    if (!file || !preview || preview.errors.length || busy) return
    const body = new FormData()
    body.append('file', file)
    setBusy(true)
    const ok = await action(() => request('/logs/import', { method: 'POST', body }), `นำเข้าบันทึก ${preview.count} วันแล้ว`)
    setBusy(false)
    if (ok) {
      requestId.current++
      setFile(null)
      setPreview(null)
      if (inputRef.current) inputRef.current.value = ''
    }
  }
  return <section className="card" aria-label="นำเข้าบันทึก CSV" data-tour="csv-import">
    <div className="section-head"><div><p className="eyebrow">นำเข้าจากไฟล์</p><h2>นำเข้าบันทึกหลายวัน</h2></div><FileUp size={19} className="text-blue-800" aria-hidden="true" /></div>
    <p className="mb-4 text-xs text-slate-500">ไฟล์ UTF-8 ไม่เกิน 2 MB หรือ 500 แถว วันที่ใช้รูปแบบ YYYY-MM-DD ระบบจะตรวจวันที่ซ้ำและข้อมูลผิดก่อนบันทึก</p>
    <div className="flex flex-wrap items-center gap-3"><input ref={inputRef} type="file" accept=".csv,text/csv" aria-label="เลือกไฟล์ CSV" className="max-w-full text-xs file:mr-3 file:rounded-md file:border file:border-slate-300 file:bg-white file:px-3 file:py-2 file:font-semibold file:text-blue-900" onChange={event => inspect(event.target.files?.[0] || null)} /><a className="inline-flex items-center gap-1.5 text-xs font-semibold text-blue-800 underline" href="/log-import-template.csv" download><Download size={14} />ดาวน์โหลดแม่แบบ CSV</a></div>
    {busy && <p className="mt-4 text-xs text-slate-500">กำลังตรวจไฟล์…</p>}
    {preview && <div className="mt-5 border-t border-slate-200 pt-4"><p className="text-sm font-semibold">พบ {preview.count} แถว · {preview.errors.length ? `ต้องแก้ ${preview.errors.length} จุด` : 'พร้อมนำเข้า'}</p>{preview.errors.length > 0 && <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-xs text-red-800"><strong>แก้ไฟล์แล้วเลือกใหม่</strong><ul className="mt-2 list-disc pl-5">{preview.errors.slice(0, 10).map((issue, i) => <li key={i}>แถว {issue.line}: {issue.message}</li>)}</ul>{preview.errors.length > 10 && <p>ยังมีอีก {preview.errors.length - 10} จุด</p>}</div>}<div className="mt-3 max-h-60 overflow-auto"><table className="w-full border-collapse text-left text-xs"><thead><tr className="border-b border-slate-200 text-slate-500"><th className="py-2">แถว</th><th>วันที่</th><th>ประเภท</th><th>งานที่ปฏิบัติ</th></tr></thead><tbody>{preview.rows.slice(0, 10).map(row => <tr key={row.line} className="border-b border-slate-100"><td className="py-2">{row.line}</td><td>{row.date}</td><td>{kindName(row.kind)}</td><td className="max-w-64 truncate">{row.work || '—'}</td></tr>)}</tbody></table></div>{preview.count > 10 && <p className="mt-2 text-[11px] text-slate-500">แสดง 10 แถวแรกจากทั้งหมด {preview.count} แถว</p>}<button type="button" className="primary mt-4" disabled={busy || preview.errors.length > 0 || preview.count === 0} onClick={importFile}>นำเข้า {preview.count} วัน</button></div>}
  </section>
}

function Logs({ data, student, action, ai, initialDate, onDateUsed }) {
  const [form, setForm] = useState(blankLog())
  const [editing, setEditing] = useState(null)
  const set = key => value => setForm(x => ({ ...x, [key]: value }))
  useEffect(() => {
    if (!initialDate) return
    setEditing(null)
    setForm({ ...blankLog(), date: initialDate })
    onDateUsed?.()
    setTimeout(() => document.querySelector('[data-tour="log-form"]')?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 50)
  }, [initialDate])
  const polish = field => ai && <AiAssist run={() => post('/ai/polish', { field, text: form[field], date: form.date })} onApply={text => set(field)(text)} disabled={!form[field]?.trim()} hint="พิมพ์โน้ตสั้นๆ ก่อน" />
  async function submit(e) {
    e.preventDefault()
    const error = validateLog(form, data.internship, data.logs || [], editing, blankLog().date)
    if (error) { toast.error(error); return }
    const ok = await action(() => post(editing ? `/logs/${editing}` : '/logs', form, editing ? 'PUT' : 'POST'), 'บันทึกงานแล้ว')
    if (ok) { setForm(blankLog()); setEditing(null) }
  }
  function edit(l) { setEditing(l.id); setForm({ ...l }); scrollTo({ top: 0, behavior: 'smooth' }) }
  return <div className="stack">{student && <><CSVImport action={action} /><section className="card" data-tour="log-form"><div className="section-head"><div><p className="eyebrow">บันทึกงาน</p><h2>{editing ? 'แก้ไขบันทึก' : 'เพิ่มบันทึกประจำวัน'}</h2></div>{editing && <button onClick={() => { setEditing(null); setForm(blankLog()) }}>ยกเลิก</button>}</div><form onSubmit={submit}><div className="form-grid"><Field label="วันที่" type="date" value={form.date} onChange={set('date')} min={data.internship?.start_date || undefined} max={[data.internship?.end_date, blankLog().date].filter(Boolean).sort()[0]} required /><label className="field"><span>ประเภท</span><select value={form.kind} onChange={e => set('kind')(e.target.value)}><option value="work">ปฏิบัติงาน</option><option value="sick">ลาป่วย</option><option value="personal">ลากิจ</option><option value="holiday">วันหยุด</option></select></label>{form.kind === 'work' && <><Field label="เวลาเริ่ม" type="time" value={form.start_time} onChange={set('start_time')} required /><Field label="เวลาสิ้นสุด" type="time" value={form.end_time} onChange={set('end_time')} required /><Field label="พัก (นาที)" type="number" value={form.break_minutes} onChange={v => set('break_minutes')(Number(v))} min={0} max={600} step={1} /></>}</div>{form.kind === 'work' && <label className="late-check"><input type="checkbox" checked={Boolean(form.late)} onChange={e => set('late')(e.target.checked)} /> มาสายวันนี้</label>}<TextArea label="งานที่ปฏิบัติ" value={form.work} onChange={set('work')} rows={4} placeholder="วันนี้ทำอะไร ได้ผลอย่างไร" required={form.kind === 'work'} maxLength={4000} />{polish('work')}<div className="form-grid"><TextArea label="ปัญหาที่พบ" value={form.blockers} onChange={set('blockers')} rows={3} maxLength={2000} />{polish('blockers')}<TextArea label="สิ่งที่เรียนรู้" value={form.learning} onChange={set('learning')} rows={3} maxLength={2000} />{polish('learning')}</div><button className="primary">{editing ? 'บันทึกการแก้ไข' : 'เพิ่มบันทึก'}</button></form></section></>}{student && ai && <WeeklySummary />}<section className="card" data-tour="log-history"><div className="section-head"><div><p className="eyebrow">ประวัติบันทึก</p><h2>{student ? 'รายการที่บันทึก' : 'งานที่นักศึกษาบันทึก'}</h2></div><span className="pill">{data.logs.length} รายการ</span></div>{!student && <p className="mb-4 text-sm text-slate-600">บันทึกของ {data.internship?.student_name || 'นักศึกษา'} · ดูรายละเอียดได้อย่างเดียว การแก้ไขเป็นสิทธิ์ของนักศึกษา</p>}{data.logs.length ? <div className="log-list">{data.logs.map(l => <article className="log-card" key={l.id}><div className="log-date"><strong>{new Date(l.date + 'T12:00:00').getDate()}</strong><span>{new Intl.DateTimeFormat('th-TH', { month: 'short' }).format(new Date(l.date + 'T12:00:00'))}</span></div><div className="log-body"><div className="log-top"><span className="tag">{kindName(l.kind)}</span><span className={'status ' + (l.approval_id ? 'approved' : 'pending')}>{l.approval_id ? 'รับรองแล้ว' : 'รอรับรอง'}</span></div><p>{l.work || 'ไม่มีรายละเอียด'}</p><small>{l.kind === 'work' ? `${l.start_time}–${l.end_time} · ${l.hours.toFixed(1)} ชั่วโมง` : ''}</small>{l.blockers && <small>ปัญหา: {l.blockers}</small>}{l.learning && <small>เรียนรู้: {l.learning}</small>}</div>{student && <div className="log-actions"><button disabled={Boolean(l.approval_id)} title={l.approval_id ? 'พี่เลี้ยงรับรองแล้ว แก้ไขไม่ได้' : undefined} onClick={() => edit(l)}>แก้ไข</button><button className="danger-text" disabled={Boolean(l.approval_id)} title={l.approval_id ? 'พี่เลี้ยงรับรองแล้ว ลบไม่ได้' : undefined} onClick={() => { if (confirm('ลบบันทึกนี้?')) action(() => request(`/logs/${l.id}`, { method: 'DELETE' }), 'ลบบันทึกแล้ว') }}>ลบ</button></div>}</article>)}</div> : <p className="empty">ยังไม่มีบันทึกประจำวัน</p>}</section></div>
}

function period(kind) {
  const [yy, mm, dd] = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Bangkok' }).format(new Date()).split('-').map(Number)
  const now = new Date(yy, mm - 1, dd), y = now.getFullYear(), m = now.getMonth(), d = now.getDate()
  const local = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  if (kind === 'month') return { from: local(new Date(y, m, 1)), to: local(now) }
  const monday = new Date(y, m, d - ((now.getDay() + 6) % 7))
  return { from: local(monday), to: local(now) }
}

function shiftISO(iso, days) {
  const d = new Date(iso + 'T00:00:00Z')
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}
function WeeklySummary() {
  const thisWeek = period('week')
  const [range, setRange] = useState(thisWeek)
  const [busy, setBusy] = useState(false)
  const [text, setText] = useState('')
  const lastWeek = { from: shiftISO(thisWeek.from, -7), to: shiftISO(thisWeek.from, -1) }
  async function generate() {
    setBusy(true)
    try { const result = await post('/ai/weekly', range); setText(result.text) }
    catch (e) { toast.error(e.message) }
    finally { setBusy(false) }
  }
  async function copy() {
    try { await navigator.clipboard.writeText(text); toast.success('คัดลอกแล้ว') } catch { toast.error('คัดลอกไม่สำเร็จ') }
  }
  return <section className="card" data-tour="weekly-summary">
    <div className="section-head"><div><p className="eyebrow">ผู้ช่วย AI</p><h2>สรุปงานรายสัปดาห์</h2></div><Sparkles size={19} className="text-violet-700" aria-hidden="true" /></div>
    <p className="mb-4 text-xs text-slate-500">สรุปจากบันทึกรายวันในช่วงที่เลือก ใช้กรอก Weekly Planning / action plan ได้ ตรวจข้อเท็จจริงก่อนส่งทุกครั้ง</p>
    <div className="flex flex-wrap items-end gap-3">
      <div className="segmented"><button type="button" className={range.from === thisWeek.from && range.to === thisWeek.to ? 'selected' : ''} onClick={() => setRange(thisWeek)}>สัปดาห์นี้</button><button type="button" className={range.from === lastWeek.from && range.to === lastWeek.to ? 'selected' : ''} onClick={() => setRange(lastWeek)}>สัปดาห์ที่แล้ว</button></div>
      <Field label="ตั้งแต่" type="date" value={range.from} onChange={v => setRange(r => ({ ...r, from: v }))} />
      <Field label="ถึง" type="date" value={range.to} onChange={v => setRange(r => ({ ...r, to: v }))} />
      <button type="button" className="mb-4 inline-flex items-center gap-1.5 rounded-lg bg-violet-700 px-4 py-2 text-xs font-semibold text-white hover:bg-violet-800 disabled:opacity-50" onClick={generate} disabled={busy}><Sparkles size={14} aria-hidden="true" />{busy ? 'AI กำลังสรุป…' : 'สรุปด้วย AI'}</button>
    </div>
    {text && <div className="mt-2"><textarea className="w-full rounded-md border border-violet-200 bg-violet-50/40 p-3 text-sm leading-relaxed" value={text} onChange={e => setText(e.target.value)} rows={Math.min(18, text.split('\n').length + 2)} aria-label="สรุปรายสัปดาห์" /><div className="mt-2 flex gap-2"><button type="button" className="inline-flex items-center gap-1.5 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold" onClick={copy}><Copy size={13} aria-hidden="true" />คัดลอก</button><button type="button" className="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600" onClick={() => setText('')}>ล้าง</button></div></div>}
  </section>
}

const thaiDateTime = iso => iso ? new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Bangkok' }).format(new Date(iso)) : '—'
function timeAgo(iso) {
  if (!iso) return 'ยังไม่มีข้อมูล'
  const minutes = Math.floor((Date.now() - new Date(iso).getTime()) / 60000)
  if (minutes < 1) return 'เมื่อสักครู่'
  if (minutes < 60) return `${minutes} นาทีที่แล้ว`
  if (minutes < 60 * 24) return `${Math.floor(minutes / 60)} ชม.ที่แล้ว`
  const days = Math.floor(minutes / 1440)
  return days < 30 ? `${days} วันที่แล้ว` : thaiDateTime(iso)
}

function AdminUsers({ me }) {
  const [rows, setRows] = useState(null)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [role, setRole] = useState('all')
  const [loading, setLoading] = useState(false)
  const [confirmId, setConfirmId] = useState(null)
  const [resetting, setResetting] = useState(null)
  const [resetResult, setResetResult] = useState(null)
  async function resetPassword(row) {
    setResetting(row.id)
    try {
      const v = await post(`/admin/users/${row.id}/reset-password`, {})
      setResetResult({ name: row.name, email: v.email, password: v.password })
      setConfirmId(null)
      toast.success('รีเซ็ตรหัสผ่านแล้ว')
    } catch (e) { toast.error(e.message) }
    finally { setResetting(null) }
  }
  async function copyReset() {
    try { await navigator.clipboard.writeText(`อีเมล: ${resetResult.email}\nรหัสผ่านชั่วคราว: ${resetResult.password}\nเข้าสู่ระบบแล้วเปลี่ยนรหัสผ่านที่หน้าข้อมูลส่วนตัว`); toast.success('คัดลอกแล้ว') } catch { toast.error('คัดลอกไม่สำเร็จ') }
  }
  async function load() {
    setLoading(true)
    try { const v = await request('/admin/users'); setRows(v.users); setError('') }
    catch (e) { setError(e.message) }
    finally { setLoading(false) }
  }
  useEffect(() => { load() }, [])
  if (error) return <section className="card"><p className="error">{error}</p></section>
  if (!rows) return <section className="card"><p className="muted">กำลังโหลดรายชื่อผู้ใช้…</p></section>
  const weekAgo = Date.now() - 7 * 24 * 3600 * 1000
  const stats = [
    ['ผู้ใช้ทั้งหมด', rows.length, 'บัญชี'],
    ['นักศึกษา', rows.filter(r => r.role === 'student').length, 'คน'],
    ['พี่เลี้ยง', rows.filter(r => r.role === 'mentor').length, 'คน'],
    ['เข้าใช้ใน 7 วัน', rows.filter(r => r.last_login_at && new Date(r.last_login_at).getTime() >= weekAgo).length, 'คน'],
  ]
  const q = query.trim().toLowerCase()
  const shown = rows.filter(r => (role === 'all' || r.role === role) && (!q || [r.name, r.email, r.company, r.mentor_name, r.student_name].some(v => (v || '').toLowerCase().includes(q))))
  return <div className="stack">
    <section className="summary-band" aria-label="สรุปผู้ใช้งาน">{stats.map(([label, value, unit]) => <div key={label}><Users size={18} className="summary-icon" aria-hidden="true" /><span>{label}</span><strong>{value}</strong><small>{unit}</small></div>)}</section>
    <section className="card">
      <div className="section-head"><div><p className="eyebrow">ผู้ดูแลระบบ</p><h2>ผู้ใช้งานทั้งหมด</h2></div><button type="button" className="inline-flex items-center gap-1.5 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50" onClick={load} disabled={loading}><RefreshCw size={13} aria-hidden="true" />{loading ? 'กำลังโหลด…' : 'รีเฟรช'}</button></div>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="relative min-w-[220px] flex-1"><Search size={15} className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-slate-400" aria-hidden="true" /><input className="w-full rounded-lg border border-slate-300 py-2 pr-3 pl-9 text-sm" placeholder="ค้นหาชื่อ อีเมล หรือบริษัท" value={query} onChange={e => setQuery(e.target.value)} aria-label="ค้นหาผู้ใช้" /></label>
        <div className="segmented">{[['all', 'ทั้งหมด'], ['student', 'นักศึกษา'], ['mentor', 'พี่เลี้ยง']].map(([key, label]) => <button key={key} type="button" className={role === key ? 'selected' : ''} onClick={() => setRole(key)}>{label}</button>)}</div>
      </div>
      {resetResult && <div className="mb-4 flex flex-col gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900 sm:flex-row sm:items-center sm:justify-between" role="status">
        <div><strong className="block">รหัสผ่านชั่วคราวของ {resetResult.name || resetResult.email}</strong><span className="text-xs">ส่งให้เจ้าของบัญชี แล้วให้เปลี่ยนรหัสผ่านเองหลังเข้าสู่ระบบ · ข้อความนี้จะแสดงครั้งเดียว</span><div className="mt-1.5 font-mono text-base font-semibold tracking-wider" data-testid="temp-password">{resetResult.password}</div></div>
        <div className="flex shrink-0 gap-2"><button type="button" className="inline-flex items-center gap-1.5 rounded-md bg-emerald-700 px-3 py-1.5 text-xs font-semibold text-white" onClick={copyReset}><Copy size={13} aria-hidden="true" />คัดลอก</button><button type="button" className="rounded-md border border-emerald-300 bg-white px-3 py-1.5 text-xs font-semibold" onClick={() => setResetResult(null)}>ปิด</button></div>
      </div>}
      <div className="overflow-x-auto">
        <table className="w-full min-w-[960px] text-left text-sm">
          <thead><tr className="border-b border-slate-200 text-[11px] font-semibold tracking-wider text-slate-500 uppercase"><th className="py-2 pr-3">ผู้ใช้</th><th className="py-2 pr-3">บทบาท</th><th className="py-2 pr-3">บริษัท / การเชื่อมโยง</th><th className="py-2 pr-3 text-right">บันทึก</th><th className="py-2 pr-3">บันทึกล่าสุด</th><th className="py-2 pr-3">สมัครเมื่อ</th><th className="py-2 pr-3">เข้าใช้ล่าสุด</th><th className="py-2 text-right">จัดการ</th></tr></thead>
          <tbody>{shown.map(r => <tr key={r.id} className="border-b border-slate-100 align-top">
            <td className="py-3 pr-3"><div className="flex items-center gap-2.5"><Avatar user={r} className="avatar" /><div><div className="font-semibold text-slate-900">{r.name || '—'}{r.id === me.id && <span className="ml-1.5 text-[11px] font-normal text-slate-400">(คุณ)</span>}</div><div className="text-xs text-slate-500">{r.email}</div></div></div></td>
            <td className="py-3 pr-3"><span className={`rounded-full px-2 py-0.5 text-[11px] font-semibold ${r.role === 'student' ? 'bg-blue-50 text-blue-800' : 'bg-emerald-50 text-emerald-800'}`}>{r.role === 'student' ? 'นักศึกษา' : 'พี่เลี้ยง'}</span>{r.is_admin && <span className="ml-1 rounded-full bg-slate-800 px-2 py-0.5 text-[11px] font-semibold text-white">admin</span>}</td>
            <td className="py-3 pr-3 text-xs text-slate-600"><div className="font-medium text-slate-800">{r.company || '—'}</div>{r.role === 'student' ? <div>พี่เลี้ยง: {r.mentor_name || 'ยังไม่เชื่อม'}</div> : <div>ดูแล {r.mentees} คน{r.student_name ? ` · ${r.student_name}` : ''}</div>}</td>
            <td className="py-3 pr-3 text-right">{r.role === 'student' ? <><div className="font-semibold">{r.log_count}</div><div className="text-[11px] text-slate-500">รับรอง {r.approved}</div></> : '—'}</td>
            <td className="py-3 pr-3 text-xs">{r.last_log_date ? thaiShortDate(r.last_log_date) : '—'}</td>
            <td className="py-3 pr-3 text-xs text-slate-600">{thaiDateTime(r.created_at)}</td>
            <td className="py-3 pr-3 text-xs"><div className="flex items-center gap-1.5">{r.active_session && <span className="h-2 w-2 rounded-full bg-emerald-500" title="มี session ที่ยังไม่หมดอายุ" />}<span title={thaiDateTime(r.last_login_at)}>{timeAgo(r.last_login_at)}</span></div></td>
            <td className="py-3 text-right text-xs">{r.id === me.id ? <span className="text-slate-400">—</span> : confirmId === r.id
              ? <div className="inline-flex items-center gap-1.5"><span className="text-slate-600">รีเซ็ต?</span><button type="button" className="rounded-md bg-red-600 px-2.5 py-1 font-semibold text-white disabled:opacity-50" disabled={resetting === r.id} onClick={() => resetPassword(r)}>{resetting === r.id ? 'กำลังทำ…' : 'ยืนยัน'}</button><button type="button" className="rounded-md border border-slate-300 bg-white px-2.5 py-1 font-semibold" onClick={() => setConfirmId(null)}>ยกเลิก</button></div>
              : <button type="button" className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 font-semibold text-slate-700 hover:bg-slate-50" onClick={() => setConfirmId(r.id)}><KeyRound size={12} aria-hidden="true" />รีเซ็ตรหัสผ่าน</button>}</td>
          </tr>)}</tbody>
        </table>
        {!shown.length && <p className="empty py-6 text-center">ไม่พบผู้ใช้ที่ตรงกับเงื่อนไข</p>}
      </div>
      <p className="mt-4 text-[11px] text-slate-400">แสดงเฉพาะข้อมูลสรุปการใช้งาน ไม่แสดงรหัสผ่านหรือเนื้อหาบันทึก · รีเซ็ตรหัสผ่านแล้วผู้ใช้จะถูกออกจากระบบทุกอุปกรณ์ · วันสมัคร/เข้าใช้ล่าสุดเริ่มเก็บตั้งแต่อัปเดตนี้ บัญชีเก่าจะขึ้นเมื่อ login ครั้งถัดไป</p>
    </section>
  </div>
}

function Approvals({ data, student, action }) {
  const [frequency, setFrequency] = useState('week')
  const [range, setRange] = useState(period('week'))
  const [typed, setTyped] = useState('')
  const pending = data.logs.filter(l => !l.approval_id && l.date >= range.from && l.date <= range.to)
  async function approve(e) {
    e.preventDefault()
    const error = validateApproval(range, typed, data.user.name, pending.length, blankLog().date)
    if (error) { toast.error(error); return }
    const ok = await action(() => post('/approvals', { ...range, typed_name: typed }), `${pending.length} รายการได้รับการรับรองแล้ว`)
    if (ok) setTyped('')
  }
  return <div className="stack">{!student && <section className="card"><div className="section-head"><div><p className="eyebrow">ตรวจและรับรอง</p><h2>รับรองบันทึกเป็นรอบ</h2></div></div><div className="segmented"><button className={frequency === 'week' ? 'selected' : ''} onClick={() => { setFrequency('week'); setRange(period('week')) }}>รายสัปดาห์</button><button className={frequency === 'month' ? 'selected' : ''} onClick={() => { setFrequency('month'); setRange(period('month')) }}>รายเดือน</button></div><form onSubmit={approve}><div className="form-grid"><Field label="ตั้งแต่วันที่" type="date" value={range.from} onChange={v => setRange({ ...range, from: v })} max={blankLog().date} required /><Field label="ถึงวันที่" type="date" value={range.to} onChange={v => setRange({ ...range, to: v })} min={range.from || undefined} max={blankLog().date} required /></div><p className="muted">รายการที่ยังไม่รับรองในช่วงนี้: <strong>{pending.length}</strong></p>{pending.length > 0 && <div className="approval-preview">{pending.map(l => <div key={l.id}><strong>{formatDate(l.date)}</strong><span>{l.work || kindName(l.kind)}</span><small>{l.hours.toFixed(1)} ชม.</small></div>)}</div>}<Field label="พิมพ์ชื่อ–นามสกุลเพื่อรับรอง" value={typed} onChange={setTyped} placeholder={data.user.name} note={`ต้องตรงกับชื่อในโปรไฟล์: ${data.user.name}`} /><Button className="primary" disabled={!pending.length || typed !== data.user.name}>รับรอง {pending.length} รายการ</Button></form></section>}<section className="card" data-tour="approvals"><div className="section-head"><div><p className="eyebrow">รอบที่รับรองแล้ว</p><h2>ประวัติการรับรอง</h2></div></div>{data.approvals.length ? <div className="approval-history">{data.approvals.map(a => <div key={a.id}><span className="seal">✓</span><div><strong>{formatDate(a.from)} – {formatDate(a.to)}</strong><p>{a.typed_name} · {a.count} รายการ</p><small>ยืนยันเมื่อ {formatDateTime(a.signed_at)}</small></div></div>)}</div> : <p className="empty">ยังไม่มีรอบที่พี่เลี้ยงรับรอง</p>}</section></div>
}

function Evaluation({ data, student, action }) {
  const initial = data.evaluation || { scores: {}, improvement: '', strengths: '', suggestions: '', typed_name: '', submitted_at: '' }
  const [form, setForm] = useState(initial)
  const submitted = Boolean(data.evaluation?.submitted_at)
  const total = criteria.reduce((n, [, items]) => n + items.length, 0)
  const completed = Object.values(form.scores || {}).filter(Boolean).length
  const updateScore = (key, value) => setForm(v => ({ ...v, scores: { ...v.scores, [key]: value } }))
  async function save(submit) {
    if (submitted || student) return
    if (submit && (completed !== total || form.typed_name?.trim() !== data.user.name)) { toast.error('ให้คะแนนครบทุกข้อและพิมพ์ชื่อให้ตรงกับโปรไฟล์'); return }
    if (submit && !confirm('ส่งแบบประเมินแล้วจะแก้ไขไม่ได้ ยืนยันหรือไม่?')) return
    await action(() => post('/evaluation', { ...form, submit }), submit ? 'ส่งแบบประเมินแล้ว สามารถส่งออกเอกสารได้' : 'บันทึกแบบร่างแล้ว')
  }
  return <div className="stack"><section className="card"><div className="section-head"><div><p className="eyebrow">ผลการฝึกงาน</p><h2>แบบประเมินการฝึกงาน</h2></div><span className="pill">{submitted ? 'ส่งผลแล้ว' : `${completed}/${total} ข้อ`}</span></div><p className="muted">คะแนน 4 ดีมาก · 3 ดี · 2 พอใช้ · 1 ต้องปรับปรุง · 0 ไม่ผ่าน · * ประเมินไม่ได้</p>{student && !submitted && <p className="info-box">พี่เลี้ยงเป็นผู้กรอกแบบประเมิน เมื่อส่งผลแล้วจึงส่งออกเอกสารได้</p>}{submitted && <p className="success-box">พี่เลี้ยง {data.evaluation.typed_name} ส่งผลเมื่อ {formatDateTime(data.evaluation.submitted_at)}</p>}{criteria.map(([group, items]) => <div className="criteria-group" key={group}><h3>{group}</h3>{items.map(([key, label]) => <div className="criterion" key={key}><div><strong>{key}</strong> {label}</div><div className="score-options">{['4', '3', '2', '1', '0', '*'].map(score => <label key={score} className={form.scores?.[key] === score ? 'chosen' : ''}><input type="radio" name={key} value={score} checked={form.scores?.[key] === score} onChange={() => updateScore(key, score)} disabled={student || submitted} /><span>{score}</span></label>)}</div></div>)}</div>)}<div className="form-grid"><TextArea label="ข้อบกพร่องที่ควรแก้ไข ปรับปรุง" value={form.improvement} onChange={v => setForm({ ...form, improvement: v })} disabled={student || submitted} /><TextArea label="ความสามารถพิเศษ" value={form.strengths} onChange={v => setForm({ ...form, strengths: v })} disabled={student || submitted} /></div><TextArea label="ข้อเสนอแนะอื่น ๆ" value={form.suggestions} onChange={v => setForm({ ...form, suggestions: v })} disabled={student || submitted} />{!student && !submitted && <><Field label="พิมพ์ชื่อผู้ประเมินเพื่อส่งผล" value={form.typed_name} onChange={v => setForm({ ...form, typed_name: v })} placeholder={data.user.name} /><div className="button-row"><button onClick={() => save(false)}>บันทึกแบบร่าง</button><button className="primary" onClick={() => save(true)} disabled={completed !== total || form.typed_name !== data.user.name}>ส่งแบบประเมิน</button></div></>}</section></div>
}

function Report({ data, student, action, ai }) {
  const [draft, setDraft] = useState(data.internship || {})
  const report = draft.report || {}
  function set(key, value) { setDraft(v => ({ ...v, report: { ...(v.report || {}), [key]: value } })) }
  return <div className="stack"><section className="card" data-tour="report"><div className="section-head"><div><p className="eyebrow">รายงานส่งอาจารย์</p><h2>เขียนเนื้อหารายงาน</h2></div></div><p className="info-box">โครงบทอ้างอิงจากรายงานตัวอย่างของอาจารย์ เนื้อหาทุกช่องเป็นข้อมูลฝึกงานของคุณ ระบบจะนำวันและชั่วโมงไปสรุปในภาคผนวก</p>{reportFields.map(([group, fields]) => <div className="report-group" key={group}><h3>{group}</h3>{fields.map(([key, label]) => <React.Fragment key={key}><TextArea label={label} value={report[key] || ''} onChange={v => set(key, v)} rows={5} readOnly={!student} placeholder="เขียนจากงานที่ทำจริง โดยไม่ใส่ข้อมูลลับของบริษัท" />{student && ai && <AiAssist label="ร่างด้วย AI จากบันทึก" run={() => post('/ai/report', { key })} onApply={(text, mode) => set(key, mode === 'append' && (report[key] || '').trim() ? report[key].trimEnd() + '\n\n' + text : text)} modes={[['replace', 'แทนที่'], ['append', 'ต่อท้าย']]} disabled={!(data.logs || []).length} hint="ต้องมีบันทึกรายวันก่อน" />}</React.Fragment>)}</div>)}{student && <button className="primary" onClick={() => action(async () => { const latest = await request('/data'); return post('/internship', { ...latest.internship, report: draft.report }, 'PUT') }, 'บันทึกเนื้อหารายงานแล้ว')}>บันทึกรายงาน</button>}</section></div>
}

function Documents({ data, student, go }) {
  const [preview, setPreview] = useState(null)
  useEffect(() => {
    if (!preview) return
    const close = event => { if (event.key === 'Escape') setPreview(null) }
    window.addEventListener('keydown', close)
    return () => window.removeEventListener('keydown', close)
  }, [preview])
  const docs = [
    { kind: 'log', title: 'บันทึกและลงเวลาปฏิบัติงาน', detail: 'วันที่ เวลา งานที่ทำ ชั่วโมง และการรับรอง', ready: true, Icon: CalendarDays },
    { kind: 'evaluation', title: 'แบบประเมินการฝึกงาน', detail: 'คะแนน 22 ข้อและข้อเสนอแนะของพี่เลี้ยง', ready: Boolean(data.evaluation?.submitted_at), Icon: ClipboardCheck },
    { kind: 'report', title: 'รายงานผลการฝึกงาน', detail: 'เนื้อหา 4 บท พร้อมภาคผนวกสรุปชั่วโมง', ready: true, Icon: FileText },
  ]
  return <div className="documents-page">
    <div className="documents-intro"><p className="section-index">ไฟล์สำหรับส่งอาจารย์</p><h2>เอกสารของการฝึกงานครั้งนี้</h2><p>เปิดดูแบบร่างจากข้อมูลล่าสุดก่อนดาวน์โหลดได้ แม้จะยังกรอกไม่ครบ</p></div>
    <section className="card documents-list" data-tour="documents"><div className="doc-head"><span>เอกสาร</span><span>ดูตัวอย่างและดาวน์โหลด</span></div>{docs.map((d, i) => <div className="doc-row" key={d.kind}><span className="doc-number">{String(i + 1).padStart(2, '0')}</span><div className="doc-main"><div className="doc-title"><d.Icon size={18} strokeWidth={1.8} aria-hidden="true" /><h3>{d.title}</h3></div><p>{d.detail}</p></div><div className="doc-actions">{(!student || d.kind !== 'evaluation' || d.ready) && <button onClick={() => setPreview(d)}><Eye size={14} aria-hidden="true" />Preview</button>}{d.ready ? <><button onClick={() => downloadDocument(d.kind, 'docx')}><Download size={14} aria-hidden="true" />DOCX</button><button onClick={() => downloadDocument(d.kind, 'pdf')}><Download size={14} aria-hidden="true" />PDF</button></> : <span className="muted">รอพี่เลี้ยงส่งผล</span>}</div></div>)}</section>
    <div className="document-note"><strong>ตรวจข้อมูลก่อนส่ง</strong><span>ชื่อ มหาวิทยาลัย บริษัท และวันที่ใช้ข้อมูลจาก <button className="inline-link" onClick={() => go('profile')}>โปรไฟล์</button> ส่วนเนื้อหารายงานแก้ได้ใน <button className="inline-link" onClick={() => go('report')}>หน้ารายงาน</button></span></div>
    {preview && <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-3 sm:p-6" role="presentation" onClick={() => setPreview(null)}><div className="flex h-[min(92vh,980px)] w-full max-w-5xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl" role="dialog" aria-modal="true" aria-label={`ตัวอย่าง${preview.title}`} onClick={event => event.stopPropagation()}><div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-3"><div><p className="text-[11px] font-semibold uppercase tracking-widest text-blue-700">Document preview</p><h2 className="text-base font-semibold text-slate-900">{preview.title}</h2></div><div className="flex items-center gap-2">{preview.ready && <button onClick={() => downloadDocument(preview.kind, 'pdf')} className="inline-flex items-center gap-2 rounded-lg bg-blue-900 px-3 py-2 text-xs font-semibold text-white hover:bg-blue-800"><Download size={14} />PDF</button>}<button onClick={() => setPreview(null)} className="rounded-lg border border-slate-200 p-2 text-slate-600 hover:bg-slate-100" aria-label="ปิดตัวอย่าง"><X size={18} /></button></div></div>{!preview.ready && <p className="border-b border-amber-200 bg-amber-50 px-5 py-2 text-xs text-amber-900">ดูแบบร่างได้ แต่ดาวน์โหลดแบบประเมินหลังพี่เลี้ยงส่งผลแล้ว</p>}<div className="min-h-0 flex-1 bg-slate-200 p-2 sm:p-5"><iframe key={preview.kind} title={`ตัวอย่าง${preview.title}`} src={`/api/preview/${preview.kind}`} sandbox="" className="h-full w-full rounded-lg border border-slate-300 bg-white" /></div></div></div>}
  </div>
}

function kindName(kind) { return { work: 'ปฏิบัติงาน', sick: 'ลาป่วย', personal: 'ลากิจ', holiday: 'วันหยุด' }[kind] || kind }
function formatDate(s) { if (!s) return '—'; return new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(s + 'T12:00:00')) }
function formatDateTime(s) { if (!s) return '—'; return new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'Asia/Bangkok' }).format(new Date(s)) }

createRoot(document.getElementById('root')).render(<><Toaster position="top-right" toastOptions={{ duration: 3600, className: '!rounded-lg !border !border-slate-200 !bg-white !font-sans !text-blue-950 !shadow-lg', success: { iconTheme: { primary: '#2d7952', secondary: '#fff' } }, error: { iconTheme: { primary: '#b55448', secondary: '#fff' } } }} /><App /></>)
