import React, { useEffect, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster, toast } from 'react-hot-toast'
import { LayoutDashboard, UserRound, CalendarDays, BadgeCheck, ClipboardCheck, BookOpenText, Files, LogOut, ChevronRight, Plus, ArrowUpRight, Download, Clock3, CheckCircle2, FileText, FileUp, ImagePlus, Copy, ListChecks, Building2, Eye, X } from 'lucide-react'
import { calculatePlan } from './plan.js'
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
      <form onSubmit={submit}>{mode === 'register' && <Field label="ชื่อ–นามสกุล" value={name} onChange={setName} maxLength={150} required />}<Field label="อีเมล" type="email" value={email} onChange={setEmail} disabled={Boolean(invite && info)} required note={invite && info ? 'อีเมลนี้ผูกกับคำเชิญ' : ''} /><Field label="รหัสผ่าน" type="password" value={password} onChange={setPassword} minLength={mode === 'register' ? 10 : undefined} required note={mode === 'register' ? 'อย่างน้อย 10 ตัวอักษร' : ''} />{error && <p className="error">{error}</p>}<Button className="primary full" type="submit" busy={busy} disabled={Boolean(invite && !info)}>{mode === 'login' ? 'เข้าสู่ระบบ' : 'สร้างบัญชี'}</Button></form>
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
  const headings = Object.fromEntries(tabs.map(([key, label]) => [key, label]))
  return <div className="app-shell">
    <aside className="sidebar">
      <div className="brand"><img className="brand-logo" src="/brand-mark.svg" alt="" /><div><strong>สมุดฝึกงาน</strong><small>ระบบบันทึกการฝึกงาน</small></div></div>
      <nav aria-label="เมนูหลัก">{tabs.map(([key, label, Icon]) => <button key={key} className={tab === key ? 'active' : ''} aria-current={tab === key ? 'page' : undefined} onClick={() => setTab(key)}><Icon size={17} strokeWidth={1.8} aria-hidden="true" /><span>{label}</span></button>)}</nav>
      <div className="sidebar-footer"><div className="avatar">{data.user.name?.slice(0, 1) || '?'}</div><div><strong>{data.user.name}</strong><small>{student ? 'นักศึกษา' : 'พี่เลี้ยง'}</small></div><button aria-label="ออกจากระบบ" title="ออกจากระบบ" onClick={signout}><LogOut size={17} aria-hidden="true" /></button></div>
    </aside>
    <div className="main-shell">
      <header className="topbar">
        <div className="topbar-title"><nav className="breadcrumb" aria-label="เส้นทางการนำทาง"><button onClick={() => setTab('overview')}>สมุดฝึกงาน</button><ChevronRight size={13} aria-hidden="true" /><span aria-current="page">{headings[tab]}</span></nav><h1>{headings[tab]}</h1></div>
        <div className="topbar-right"><span className="pill"><Building2 size={14} aria-hidden="true" />{data.internship?.company || 'ยังไม่ระบุบริษัท'}</span><div className="avatar small" title={data.user.name}>{data.user.name?.slice(0, 1) || '?'}</div></div>
      </header>
      <main className="content">{tab === 'overview' && <Overview data={data} student={student} go={setTab} />}{tab === 'profile' && <Profile data={data} student={student} action={action} />}{tab === 'logs' && <Logs data={data} student={student} action={action} />}{tab === 'approvals' && <Approvals data={data} student={student} action={action} />}{tab === 'evaluation' && !student && <Evaluation data={data} student={student} action={action} />}{tab === 'report' && student && <Report data={data} student={student} action={action} />}{tab === 'documents' && student && <Documents data={data} student={student} go={setTab} />}</main>
    </div>
  </div>
}

function Overview({ data, student, go }) {
  const logs = data.logs || []
  const approved = logs.filter(x => x.approval_id).length
  const pending = logs.length - approved
  const hours = logs.reduce((a, x) => a + x.hours, 0)
  const firstName = data.user.name?.split(/\s+/)[0] || data.user.name
  const today = new Intl.DateTimeFormat('th-TH', { dateStyle: 'full', timeZone: 'Asia/Bangkok' }).format(new Date())
  const tasks = student ? [
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
    <section className="summary-band" aria-label="สรุปการฝึกงาน"><div><CalendarDays size={18} className="summary-icon" aria-hidden="true" /><span>วันที่บันทึก</span><strong>{logs.length}</strong><small>วัน</small></div><div><Clock3 size={18} className="summary-icon" aria-hidden="true" /><span>ชั่วโมงปฏิบัติงาน</span><strong>{hours.toFixed(1)}</strong><small>ชั่วโมง</small></div><div><CheckCircle2 size={18} className="summary-icon" aria-hidden="true" /><span>รับรองแล้ว</span><strong>{approved}</strong><small>รายการ</small></div>{student ? <div><ListChecks size={18} className="summary-icon" aria-hidden="true" /><span>แผนการฝึก</span><strong className="summary-text">{data.internship?.plan_type === 'cooperative' ? 'สหกิจศึกษา' : 'ฝึกงาน'}</strong></div> : <div><ListChecks size={18} className="summary-icon" aria-hidden="true" /><span>แบบประเมิน</span><strong className="summary-text">{data.evaluation?.submitted_at ? 'ส่งแล้ว' : 'ยังไม่ส่ง'}</strong></div>}</section>
    <div className="dashboard-columns"><section className="card tasks-panel"><div className="section-head"><div><span className="section-index">01 / สิ่งที่ต้องทำ</span><h2>งานถัดไป</h2></div></div>{tasks.length ? <div className="task-list">{tasks.map(([title, detail, target], i) => <button key={i} onClick={() => go(target)}><span className="task-number">{String(i + 1).padStart(2, '0')}</span><span><strong>{title}</strong><small>{detail}</small></span><ArrowUpRight size={16} aria-hidden="true" /></button>)}</div> : <p className="empty">{student ? 'รายการสำคัญครบแล้ว คุณสามารถตรวจเอกสารก่อนส่งได้' : 'ไม่มีงานที่ต้องรับรองในตอนนี้'}</p>}</section><section className="card recent-panel"><div className="section-head"><div><span className="section-index">02 / งานที่ผ่านมา</span><h2>บันทึกล่าสุด</h2></div><button className="text-button" onClick={() => go('logs')}><span>ดูทั้งหมด</span><ArrowUpRight size={14} aria-hidden="true" /></button></div>{logs.length ? <div className="recent-table">{logs.slice(0, 5).map(l => <div key={l.id}><span>{formatDate(l.date)}</span><strong>{l.work || kindName(l.kind)}</strong><small>{l.hours.toFixed(1)} ชม.</small><em className={l.approval_id ? 'is-approved' : ''}>{l.approval_id ? 'รับรองแล้ว' : 'รอรับรอง'}</em></div>)}</div> : <p className="empty">ยังไม่มีบันทึกประจำวัน</p>}</section></div>
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

function Profile({ data, student, action }) {
  const [user, setUser] = useState(data.user)
  const [internship, setInternship] = useState(data.internship || {})
  const [invite, setInvite] = useState('')
  const [photoVersion, setPhotoVersion] = useState(Date.now())
  useEffect(() => { setUser(data.user); setInternship(data.internship || {}) }, [data.user, data.internship])
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
    if (ok) setPhotoVersion(Date.now())
  }
  async function makeInvite() {
    await action(async () => {
      const value = await post('/invite', {})
      setInvite(location.origin + value.invite_url)
    }, 'สร้างลิงก์คำเชิญแล้ว ส่งให้พี่เลี้ยงด้วยช่องทางที่คุณเลือก')
  }
  async function saveInternship(event) {
    event.preventDefault()
    if (data.internship?.mentor_linked && internship.mentor_email !== data.internship.mentor_email) { toast.error('อีเมลพี่เลี้ยงที่เชื่อมแล้วแก้ไขไม่ได้'); return }
    const error = validateInternship(internship, data.logs || [])
    if (error) { toast.error(error); return }
    const previousMentorEmail = data.internship?.mentor_email || ''
    const saved = await action(() => post('/internship', { ...internship, university: 'มหาวิทยาลัยราชภัฏนครปฐม', year: '2569', term: '2' }, 'PUT'), 'บันทึกข้อมูลการฝึกงานแล้ว')
    if (saved && previousMentorEmail !== internship.mentor_email) setInvite('')
  }
  function saveProfile(event) {
    event.preventDefault()
    const error = validateProfile(user, student, provinces, blankLog().date)
    if (error) { toast.error(error); return }
    action(() => post('/profile', user, 'PUT'), 'บันทึกข้อมูลส่วนตัวแล้ว')
  }
  return <div className="stack">
    <section className="card">
      <div className="section-head"><div><p className="eyebrow">ข้อมูลสำหรับเอกสาร</p><h2>ข้อมูลส่วนตัว</h2></div></div>
      <div className="profile-row"><div className="profile-photo"><img src={`${data.user.photo_url}?v=${photoVersion}`} onError={e => e.currentTarget.style.display = 'none'} onLoad={e => e.currentTarget.style.display = 'block'} alt="รูปโปรไฟล์" /><span>{data.user.name?.slice(0, 1)}</span></div><label className="upload-button"><ImagePlus size={16} aria-hidden="true" />อัปโหลดรูป JPG หรือ PNG<input type="file" accept="image/png,image/jpeg" onChange={upload} hidden /></label><small>ไม่เกิน 2 MB</small></div>
      <form onSubmit={saveProfile}>
        <div className="form-grid"><Field label="ชื่อ–นามสกุล" value={user.name} onChange={setUserField('name')} maxLength={150} required /><Field label="เบอร์โทรศัพท์" value={user.phone} onChange={setUserField('phone')} maxLength={40} /><Field label="ตำแหน่ง" value={user.position} onChange={setUserField('position')} maxLength={120} placeholder={student ? 'นักศึกษาฝึกงาน' : 'Software Engineer'} />{student && <Field label="วันเดือนปีเกิด" type="date" value={user.birth_date} onChange={setUserField('birth_date')} max={blankLog().date} />}</div>
        {student && <><div className="mb-4 mt-3 border-t border-slate-200 pt-5"><h3 className="text-base font-semibold text-blue-950">ที่อยู่และข้อมูลสำหรับหน้าประวัติ</h3><p className="mt-1 text-xs text-slate-500">ใช้ในแบบบันทึกและลงเวลาตามฟอร์มมหาวิทยาลัย</p></div><Field label="บ้านเลขที่ ถนน/ซอย" value={user.home_address} onChange={setUserField('home_address')} maxLength={300} /><div className="form-grid"><Field label="ตำบล/แขวง" value={user.subdistrict} onChange={setUserField('subdistrict')} maxLength={100} /><Field label="อำเภอ/เขต" value={user.district} onChange={setUserField('district')} maxLength={100} /><label className="field"><span>จังหวัด</span><select value={user.province || ''} onChange={event => setUser(x => ({ ...x, province: event.target.value, country: event.target.value === 'other' ? x.country : 'ประเทศไทย' }))}><option value="">เลือกจังหวัด</option>{provinces.map(name => <option key={name} value={name}>{name}</option>)}<option value="other">อื่น ๆ — ต่างประเทศ</option></select><small>มีครบ 77 จังหวัด รวมกรุงเทพมหานคร</small></label>{user.province === 'other' && <Field label="ชื่อประเทศ" value={user.country === 'ประเทศไทย' ? '' : user.country} onChange={setUserField('country')} maxLength={100} required placeholder="เช่น ญี่ปุ่น" />}</div><div className="form-grid"><TextArea label="ความรู้ความสามารถพิเศษ" value={user.skills} onChange={setUserField('skills')} rows={3} maxLength={1000} /><TextArea label="ความสนใจพิเศษ" value={user.interests} onChange={setUserField('interests')} rows={3} maxLength={1000} /></div></>}
        <button className="primary">บันทึกข้อมูลส่วนตัว</button>
      </form>
    </section>
    {student ? <section className="card"><div className="section-head"><div><p className="eyebrow">สถานประกอบการและช่วงฝึก</p><h2>ข้อมูลการฝึกงาน</h2></div></div><form onSubmit={saveInternship}><div className="form-grid"><Field label="มหาวิทยาลัย" value="มหาวิทยาลัยราชภัฏนครปฐม" disabled note="มหาวิทยาลัยกำหนดไว้แล้ว" /><Field label="คณะ" value={internship.faculty} onChange={set('faculty')} /><Field label="สาขาวิชา" value={internship.major} onChange={set('major')} /><Field label="รหัสนักศึกษา" value={internship.student_number} onChange={value => set('student_number')(value.replace(/\D/g, '').slice(0, 9))} maxLength={9} pattern="6642590[0-9]{2}" title="รหัสต้องเป็น 6642590xx" placeholder="6642590xx" required /><Field label="รหัสรุ่น" value="66" disabled /><Field label="หมู่เรียน" value={internship.class_group} onChange={set('class_group')} maxLength={40} /><Field label="ปีการศึกษา" value="2569" disabled /><Field label="ภาคเรียน" value="2" disabled /><Field label="บริษัท/หน่วยงาน" value={internship.company} onChange={set('company')} maxLength={200} /><Field label="แผนก" value={internship.department} onChange={set('department')} /><Field label="ตำแหน่งฝึกงาน" value={internship.position} onChange={set('position')} /><Field label="ที่อยู่บริษัท" value={internship.company_address} onChange={set('company_address')} maxLength={400} /><Field label="อีเมลพี่เลี้ยง" type="email" value={internship.mentor_email} onChange={set('mentor_email')} disabled={internship.mentor_linked} note={internship.mentor_linked ? 'เชื่อมบัญชีแล้ว จึงเปลี่ยนอีเมลไม่ได้' : 'บันทึกก่อนสร้างคำเชิญ'} /></div><PlanSetup internship={internship} setInternship={setInternship} set={set} actualHours={(data.logs || []).reduce((sum, log) => sum + log.hours, 0)} firstRecordedDate={firstRecordedDate} lastRecordedDate={lastRecordedDate} /><button className="primary">บันทึกข้อมูลการฝึกงาน</button></form><InvitePanel internship={internship} invite={invite} onCreate={makeInvite} canInvite={Boolean(internship.mentor_email && internship.mentor_email === data.internship?.mentor_email)} /></section> : <section className="card"><h2>การฝึกงานที่ดูแล</h2><p className="muted">นักศึกษา: {internship.student_name || '—'}</p><p><strong>{internship.company || 'ยังไม่ระบุบริษัท'}</strong></p><p className="muted">แผน: {internship.plan_type === 'cooperative' ? 'สหกิจศึกษา' : 'ฝึกงานทั่วไป'} · {calculatePlan(internship.start_date, internship.end_date, internship.plan_type, internship.holiday_dates)?.hours ?? '—'} ชั่วโมงตามแผน</p><p className="muted">นักศึกษาจะบันทึกข้อมูลหลัก คุณสามารถตรวจและรับรองบันทึกในเมนูการรับรอง</p></section>}
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
  return <section className="mt-6 border-t border-slate-200 pt-6" aria-label="แผนและระยะเวลาการฝึกงาน">
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
  return <section className="card" aria-label="นำเข้าบันทึก CSV">
    <div className="section-head"><div><p className="eyebrow">นำเข้าจากไฟล์</p><h2>นำเข้าบันทึกหลายวัน</h2></div><FileUp size={19} className="text-blue-800" aria-hidden="true" /></div>
    <p className="mb-4 text-xs text-slate-500">ไฟล์ UTF-8 ไม่เกิน 2 MB หรือ 500 แถว วันที่ใช้รูปแบบ YYYY-MM-DD ระบบจะตรวจวันที่ซ้ำและข้อมูลผิดก่อนบันทึก</p>
    <div className="flex flex-wrap items-center gap-3"><input ref={inputRef} type="file" accept=".csv,text/csv" aria-label="เลือกไฟล์ CSV" className="max-w-full text-xs file:mr-3 file:rounded-md file:border file:border-slate-300 file:bg-white file:px-3 file:py-2 file:font-semibold file:text-blue-900" onChange={event => inspect(event.target.files?.[0] || null)} /><a className="inline-flex items-center gap-1.5 text-xs font-semibold text-blue-800 underline" href="/log-import-template.csv" download><Download size={14} />ดาวน์โหลดแม่แบบ CSV</a></div>
    {busy && <p className="mt-4 text-xs text-slate-500">กำลังตรวจไฟล์…</p>}
    {preview && <div className="mt-5 border-t border-slate-200 pt-4"><p className="text-sm font-semibold">พบ {preview.count} แถว · {preview.errors.length ? `ต้องแก้ ${preview.errors.length} จุด` : 'พร้อมนำเข้า'}</p>{preview.errors.length > 0 && <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-xs text-red-800"><strong>แก้ไฟล์แล้วเลือกใหม่</strong><ul className="mt-2 list-disc pl-5">{preview.errors.slice(0, 10).map((issue, i) => <li key={i}>แถว {issue.line}: {issue.message}</li>)}</ul>{preview.errors.length > 10 && <p>ยังมีอีก {preview.errors.length - 10} จุด</p>}</div>}<div className="mt-3 max-h-60 overflow-auto"><table className="w-full border-collapse text-left text-xs"><thead><tr className="border-b border-slate-200 text-slate-500"><th className="py-2">แถว</th><th>วันที่</th><th>ประเภท</th><th>งานที่ปฏิบัติ</th></tr></thead><tbody>{preview.rows.slice(0, 10).map(row => <tr key={row.line} className="border-b border-slate-100"><td className="py-2">{row.line}</td><td>{row.date}</td><td>{kindName(row.kind)}</td><td className="max-w-64 truncate">{row.work || '—'}</td></tr>)}</tbody></table></div>{preview.count > 10 && <p className="mt-2 text-[11px] text-slate-500">แสดง 10 แถวแรกจากทั้งหมด {preview.count} แถว</p>}<button type="button" className="primary mt-4" disabled={busy || preview.errors.length > 0 || preview.count === 0} onClick={importFile}>นำเข้า {preview.count} วัน</button></div>}
  </section>
}

function Logs({ data, student, action }) {
  const [form, setForm] = useState(blankLog())
  const [editing, setEditing] = useState(null)
  const set = key => value => setForm(x => ({ ...x, [key]: value }))
  async function submit(e) {
    e.preventDefault()
    const error = validateLog(form, data.internship, data.logs || [], editing, blankLog().date)
    if (error) { toast.error(error); return }
    const ok = await action(() => post(editing ? `/logs/${editing}` : '/logs', form, editing ? 'PUT' : 'POST'), 'บันทึกงานแล้ว')
    if (ok) { setForm(blankLog()); setEditing(null) }
  }
  function edit(l) { setEditing(l.id); setForm({ ...l }); scrollTo({ top: 0, behavior: 'smooth' }) }
  return <div className="stack">{student && <><CSVImport action={action} /><section className="card"><div className="section-head"><div><p className="eyebrow">บันทึกงาน</p><h2>{editing ? 'แก้ไขบันทึก' : 'เพิ่มบันทึกประจำวัน'}</h2></div>{editing && <button onClick={() => { setEditing(null); setForm(blankLog()) }}>ยกเลิก</button>}</div><form onSubmit={submit}><div className="form-grid"><Field label="วันที่" type="date" value={form.date} onChange={set('date')} min={data.internship?.start_date || undefined} max={[data.internship?.end_date, blankLog().date].filter(Boolean).sort()[0]} required /><label className="field"><span>ประเภท</span><select value={form.kind} onChange={e => set('kind')(e.target.value)}><option value="work">ปฏิบัติงาน</option><option value="sick">ลาป่วย</option><option value="personal">ลากิจ</option><option value="holiday">วันหยุด</option></select></label>{form.kind === 'work' && <><Field label="เวลาเริ่ม" type="time" value={form.start_time} onChange={set('start_time')} required /><Field label="เวลาสิ้นสุด" type="time" value={form.end_time} onChange={set('end_time')} required /><Field label="พัก (นาที)" type="number" value={form.break_minutes} onChange={v => set('break_minutes')(Number(v))} min={0} max={600} step={1} /></>}</div>{form.kind === 'work' && <label className="late-check"><input type="checkbox" checked={Boolean(form.late)} onChange={e => set('late')(e.target.checked)} /> มาสายวันนี้</label>}<TextArea label="งานที่ปฏิบัติ" value={form.work} onChange={set('work')} rows={4} placeholder="วันนี้ทำอะไร ได้ผลอย่างไร" required={form.kind === 'work'} maxLength={4000} /><div className="form-grid"><TextArea label="ปัญหาที่พบ" value={form.blockers} onChange={set('blockers')} rows={3} maxLength={2000} /><TextArea label="สิ่งที่เรียนรู้" value={form.learning} onChange={set('learning')} rows={3} maxLength={2000} /></div><button className="primary">{editing ? 'บันทึกการแก้ไข' : 'เพิ่มบันทึก'}</button></form></section></>}<section className="card"><div className="section-head"><div><p className="eyebrow">ประวัติบันทึก</p><h2>{student ? 'รายการที่บันทึก' : 'งานที่นักศึกษาบันทึก'}</h2></div><span className="pill">{data.logs.length} รายการ</span></div>{!student && <p className="mb-4 text-sm text-slate-600">บันทึกของ {data.internship?.student_name || 'นักศึกษา'} · ดูรายละเอียดได้อย่างเดียว การแก้ไขเป็นสิทธิ์ของนักศึกษา</p>}{data.logs.length ? <div className="log-list">{data.logs.map(l => <article className="log-card" key={l.id}><div className="log-date"><strong>{new Date(l.date + 'T12:00:00').getDate()}</strong><span>{new Intl.DateTimeFormat('th-TH', { month: 'short' }).format(new Date(l.date + 'T12:00:00'))}</span></div><div className="log-body"><div className="log-top"><span className="tag">{kindName(l.kind)}</span><span className={'status ' + (l.approval_id ? 'approved' : 'pending')}>{l.approval_id ? 'รับรองแล้ว' : 'รอรับรอง'}</span></div><p>{l.work || 'ไม่มีรายละเอียด'}</p><small>{l.kind === 'work' ? `${l.start_time}–${l.end_time} · ${l.hours.toFixed(1)} ชั่วโมง` : ''}</small>{l.blockers && <small>ปัญหา: {l.blockers}</small>}{l.learning && <small>เรียนรู้: {l.learning}</small>}</div>{student && <div className="log-actions"><button disabled={Boolean(l.approval_id)} title={l.approval_id ? 'พี่เลี้ยงรับรองแล้ว แก้ไขไม่ได้' : undefined} onClick={() => edit(l)}>แก้ไข</button><button className="danger-text" disabled={Boolean(l.approval_id)} title={l.approval_id ? 'พี่เลี้ยงรับรองแล้ว ลบไม่ได้' : undefined} onClick={() => { if (confirm('ลบบันทึกนี้?')) action(() => request(`/logs/${l.id}`, { method: 'DELETE' }), 'ลบบันทึกแล้ว') }}>ลบ</button></div>}</article>)}</div> : <p className="empty">ยังไม่มีบันทึกประจำวัน</p>}</section></div>
}

function period(kind) {
  const [yy, mm, dd] = new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Bangkok' }).format(new Date()).split('-').map(Number)
  const now = new Date(yy, mm - 1, dd), y = now.getFullYear(), m = now.getMonth(), d = now.getDate()
  const local = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  if (kind === 'month') return { from: local(new Date(y, m, 1)), to: local(now) }
  const monday = new Date(y, m, d - ((now.getDay() + 6) % 7))
  return { from: local(monday), to: local(now) }
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
  return <div className="stack">{!student && <section className="card"><div className="section-head"><div><p className="eyebrow">ตรวจและรับรอง</p><h2>รับรองบันทึกเป็นรอบ</h2></div></div><div className="segmented"><button className={frequency === 'week' ? 'selected' : ''} onClick={() => { setFrequency('week'); setRange(period('week')) }}>รายสัปดาห์</button><button className={frequency === 'month' ? 'selected' : ''} onClick={() => { setFrequency('month'); setRange(period('month')) }}>รายเดือน</button></div><form onSubmit={approve}><div className="form-grid"><Field label="ตั้งแต่วันที่" type="date" value={range.from} onChange={v => setRange({ ...range, from: v })} max={blankLog().date} required /><Field label="ถึงวันที่" type="date" value={range.to} onChange={v => setRange({ ...range, to: v })} min={range.from || undefined} max={blankLog().date} required /></div><p className="muted">รายการที่ยังไม่รับรองในช่วงนี้: <strong>{pending.length}</strong></p>{pending.length > 0 && <div className="approval-preview">{pending.map(l => <div key={l.id}><strong>{formatDate(l.date)}</strong><span>{l.work || kindName(l.kind)}</span><small>{l.hours.toFixed(1)} ชม.</small></div>)}</div>}<Field label="พิมพ์ชื่อ–นามสกุลเพื่อรับรอง" value={typed} onChange={setTyped} placeholder={data.user.name} note={`ต้องตรงกับชื่อในโปรไฟล์: ${data.user.name}`} /><Button className="primary" disabled={!pending.length || typed !== data.user.name}>รับรอง {pending.length} รายการ</Button></form></section>}<section className="card"><div className="section-head"><div><p className="eyebrow">รอบที่รับรองแล้ว</p><h2>ประวัติการรับรอง</h2></div></div>{data.approvals.length ? <div className="approval-history">{data.approvals.map(a => <div key={a.id}><span className="seal">✓</span><div><strong>{formatDate(a.from)} – {formatDate(a.to)}</strong><p>{a.typed_name} · {a.count} รายการ</p><small>ยืนยันเมื่อ {formatDateTime(a.signed_at)}</small></div></div>)}</div> : <p className="empty">ยังไม่มีรอบที่พี่เลี้ยงรับรอง</p>}</section></div>
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

function Report({ data, student, action }) {
  const [draft, setDraft] = useState(data.internship || {})
  const report = draft.report || {}
  function set(key, value) { setDraft(v => ({ ...v, report: { ...(v.report || {}), [key]: value } })) }
  return <div className="stack"><section className="card"><div className="section-head"><div><p className="eyebrow">รายงานส่งอาจารย์</p><h2>เขียนเนื้อหารายงาน</h2></div></div><p className="info-box">โครงบทอ้างอิงจากรายงานตัวอย่างของอาจารย์ เนื้อหาทุกช่องเป็นข้อมูลฝึกงานของคุณ ระบบจะนำวันและชั่วโมงไปสรุปในภาคผนวก</p>{reportFields.map(([group, fields]) => <div className="report-group" key={group}><h3>{group}</h3>{fields.map(([key, label]) => <TextArea key={key} label={label} value={report[key] || ''} onChange={v => set(key, v)} rows={5} readOnly={!student} placeholder="เขียนจากงานที่ทำจริง โดยไม่ใส่ข้อมูลลับของบริษัท" />)}</div>)}{student && <button className="primary" onClick={() => action(async () => { const latest = await request('/data'); return post('/internship', { ...latest.internship, report: draft.report }, 'PUT') }, 'บันทึกเนื้อหารายงานแล้ว')}>บันทึกรายงาน</button>}</section></div>
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
    <section className="card documents-list"><div className="doc-head"><span>เอกสาร</span><span>ดูตัวอย่างและดาวน์โหลด</span></div>{docs.map((d, i) => <div className="doc-row" key={d.kind}><span className="doc-number">{String(i + 1).padStart(2, '0')}</span><div className="doc-main"><div className="doc-title"><d.Icon size={18} strokeWidth={1.8} aria-hidden="true" /><h3>{d.title}</h3></div><p>{d.detail}</p></div><div className="doc-actions">{(!student || d.kind !== 'evaluation' || d.ready) && <button onClick={() => setPreview(d)}><Eye size={14} aria-hidden="true" />Preview</button>}{d.ready ? <><button onClick={() => downloadDocument(d.kind, 'docx')}><Download size={14} aria-hidden="true" />DOCX</button><button onClick={() => downloadDocument(d.kind, 'pdf')}><Download size={14} aria-hidden="true" />PDF</button></> : <span className="muted">รอพี่เลี้ยงส่งผล</span>}</div></div>)}</section>
    <div className="document-note"><strong>ตรวจข้อมูลก่อนส่ง</strong><span>ชื่อ มหาวิทยาลัย บริษัท และวันที่ใช้ข้อมูลจาก <button className="inline-link" onClick={() => go('profile')}>โปรไฟล์</button> ส่วนเนื้อหารายงานแก้ได้ใน <button className="inline-link" onClick={() => go('report')}>หน้ารายงาน</button></span></div>
    {preview && <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/60 p-3 sm:p-6" role="presentation" onClick={() => setPreview(null)}><div className="flex h-[min(92vh,980px)] w-full max-w-5xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl" role="dialog" aria-modal="true" aria-label={`ตัวอย่าง${preview.title}`} onClick={event => event.stopPropagation()}><div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-5 py-3"><div><p className="text-[11px] font-semibold uppercase tracking-widest text-blue-700">Document preview</p><h2 className="text-base font-semibold text-slate-900">{preview.title}</h2></div><div className="flex items-center gap-2">{preview.ready && <button onClick={() => downloadDocument(preview.kind, 'pdf')} className="inline-flex items-center gap-2 rounded-lg bg-blue-900 px-3 py-2 text-xs font-semibold text-white hover:bg-blue-800"><Download size={14} />PDF</button>}<button onClick={() => setPreview(null)} className="rounded-lg border border-slate-200 p-2 text-slate-600 hover:bg-slate-100" aria-label="ปิดตัวอย่าง"><X size={18} /></button></div></div>{!preview.ready && <p className="border-b border-amber-200 bg-amber-50 px-5 py-2 text-xs text-amber-900">ดูแบบร่างได้ แต่ดาวน์โหลดแบบประเมินหลังพี่เลี้ยงส่งผลแล้ว</p>}<div className="min-h-0 flex-1 bg-slate-200 p-2 sm:p-5"><iframe key={preview.kind} title={`ตัวอย่าง${preview.title}`} src={`/api/preview/${preview.kind}`} sandbox="" className="h-full w-full rounded-lg border border-slate-300 bg-white" /></div></div></div>}
  </div>
}

function kindName(kind) { return { work: 'ปฏิบัติงาน', sick: 'ลาป่วย', personal: 'ลากิจ', holiday: 'วันหยุด' }[kind] || kind }
function formatDate(s) { if (!s) return '—'; return new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(s + 'T12:00:00')) }
function formatDateTime(s) { if (!s) return '—'; return new Intl.DateTimeFormat('th-TH', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'Asia/Bangkok' }).format(new Date(s)) }

createRoot(document.getElementById('root')).render(<><Toaster position="top-right" toastOptions={{ duration: 3600, className: '!rounded-lg !border !border-slate-200 !bg-white !font-sans !text-blue-950 !shadow-lg', success: { iconTheme: { primary: '#2d7952', secondary: '#fff' } }, error: { iconTheme: { primary: '#b55448', secondary: '#fff' } } }} /><App /></>)
