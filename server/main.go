package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *DB

type User struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Position    string `json:"position"`
	PhotoURL    string `json:"photo_url"`
	BirthDate   string `json:"birth_date"`
	HomeAddress string `json:"home_address"`
	Subdistrict string `json:"subdistrict"`
	District    string `json:"district"`
	Province    string `json:"province"`
	Country     string `json:"country"`
	Skills      string `json:"skills"`
	Interests   string `json:"interests"`
}

type Internship struct {
	ID             int64             `json:"id"`
	StudentID      int64             `json:"-"`
	StudentName    string            `json:"student_name"`
	MentorID       sql.NullInt64     `json:"-"`
	University     string            `json:"university"`
	Faculty        string            `json:"faculty"`
	Major          string            `json:"major"`
	StudentNumber  string            `json:"student_number"`
	ClassGroup     string            `json:"class_group"`
	Year           string            `json:"year"`
	Term           string            `json:"term"`
	Company        string            `json:"company"`
	CompanyAddress string            `json:"company_address"`
	Department     string            `json:"department"`
	Position       string            `json:"position"`
	StartDate      string            `json:"start_date"`
	EndDate        string            `json:"end_date"`
	PlanType       string            `json:"plan_type"`
	HolidayDates   []string          `json:"holiday_dates"`
	MentorEmail    string            `json:"mentor_email"`
	MentorName     string            `json:"mentor_name"`
	MentorPosition string            `json:"mentor_position"`
	MentorPhone    string            `json:"mentor_phone"`
	Report         map[string]string `json:"report"`
	MentorLinked   bool              `json:"mentor_linked"`
}

type DayLog struct {
	ID           int64   `json:"id"`
	Date         string  `json:"date"`
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	BreakMinutes int     `json:"break_minutes"`
	Kind         string  `json:"kind"`
	Late         bool    `json:"late"`
	Work         string  `json:"work"`
	Blockers     string  `json:"blockers"`
	Learning     string  `json:"learning"`
	Hours        float64 `json:"hours"`
	ApprovalID   int64   `json:"approval_id"`
}

type Approval struct {
	ID        int64  `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	TypedName string `json:"typed_name"`
	SignedAt  string `json:"signed_at"`
	Count     int    `json:"count"`
}

type Evaluation struct {
	Scores      map[string]string `json:"scores"`
	Improvement string            `json:"improvement"`
	Strengths   string            `json:"strengths"`
	Suggestions string            `json:"suggestions"`
	TypedName   string            `json:"typed_name"`
	SubmittedAt string            `json:"submitted_at"`
}

type Data struct {
	User       User        `json:"user"`
	Internship *Internship `json:"internship"`
	Logs       []DayLog    `json:"logs"`
	Approvals  []Approval  `json:"approvals"`
	Evaluation *Evaluation `json:"evaluation"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required (PostgreSQL connection string)")
	}
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	if err = sqlDB.Ping(); err != nil {
		log.Fatal("connect database: ", err)
	}
	db = &DB{sqlDB}
	if err = migrate(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{BodyLimit: 5 * 1024 * 1024})
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "same-origin")
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			origin := c.Get("Origin")
			if origin != "" && !sameOrigin(origin, c) {
				return fiber.NewError(403, "Origin ไม่ถูกต้อง")
			}
		}
		return c.Next()
	})
	api := app.Group("/api")
	api.Post("/auth/register", register)
	api.Post("/auth/login", login)
	api.Post("/auth/logout", logout)
	api.Get("/invite/:token", inviteInfo)
	api.Use(requireUser)
	api.Get("/me", me)
	api.Put("/profile", saveProfile)
	api.Post("/profile/photo", uploadPhoto)
	api.Get("/profile/photo/:id", photo)
	api.Get("/data", getData)
	api.Put("/internship", saveInternship)
	api.Post("/invite", createInvite)
	api.Post("/logs", saveLog)
	api.Post("/logs/import", importLogs)
	api.Put("/logs/:id", saveLog)
	api.Delete("/logs/:id", deleteLog)
	api.Post("/approvals", approve)
	api.Post("/evaluation", saveEvaluation)
	api.Get("/export/:kind.:format", exportDocument)
	api.Get("/preview/:kind", previewDocument)

	webDist := getenv("WEB_DIST", "../web/dist")
	if info, e := os.Stat(webDist); e == nil && info.IsDir() {
		app.Static("/", webDist)
		app.Get("*", func(c *fiber.Ctx) error { return c.SendFile(filepath.Join(webDist, "index.html")) })
	}
	addr := getenv("ADDR", "127.0.0.1:8080")
	log.Printf("Internship Journal listening on http://%s", addr)
	log.Fatal(app.Listen(addr))
}

func getenv(k, fallback string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return fallback
}
func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

// allowedOrigins lists extra trusted origins from APP_ORIGINS (comma-separated),
// e.g. a Vercel frontend that proxies /api to this server.
func allowedOrigins() []string {
	var out []string
	for _, o := range strings.Split(os.Getenv("APP_ORIGINS"), ",") {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func sameOrigin(origin string, c *fiber.Ctx) bool {
	for _, o := range allowedOrigins() {
		if strings.EqualFold(origin, o) {
			return true
		}
	}
	return origin == "http://"+c.Hostname() || origin == "https://"+c.Hostname() || origin == "http://"+c.Get("Host") || origin == "https://"+c.Get("Host") || (strings.HasPrefix(origin, "http://127.0.0.1:5173") && strings.HasPrefix(c.Get("Host"), "127.0.0.1:5173"))
}

func migrate() error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, role TEXT NOT NULL,
		name TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '', position TEXT NOT NULL DEFAULT '',
		photo BYTEA, photo_type TEXT NOT NULL DEFAULT '',
		birth_date TEXT NOT NULL DEFAULT '', home_address TEXT NOT NULL DEFAULT '', subdistrict TEXT NOT NULL DEFAULT '',
		district TEXT NOT NULL DEFAULT '', province TEXT NOT NULL DEFAULT '', country TEXT NOT NULL DEFAULT '',
		skills TEXT NOT NULL DEFAULT '', interests TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS sessions (token_hash TEXT PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, expires_at TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS internships (
		id BIGSERIAL PRIMARY KEY, student_id BIGINT UNIQUE NOT NULL REFERENCES users(id), mentor_id BIGINT REFERENCES users(id),
		university TEXT NOT NULL DEFAULT '', faculty TEXT NOT NULL DEFAULT '', major TEXT NOT NULL DEFAULT '', student_number TEXT NOT NULL DEFAULT '', class_group TEXT NOT NULL DEFAULT '',
		year TEXT NOT NULL DEFAULT '', term TEXT NOT NULL DEFAULT '', company TEXT NOT NULL DEFAULT '', company_address TEXT NOT NULL DEFAULT '',
		department TEXT NOT NULL DEFAULT '', position TEXT NOT NULL DEFAULT '', start_date TEXT NOT NULL DEFAULT '', end_date TEXT NOT NULL DEFAULT '',
		mentor_email TEXT NOT NULL DEFAULT '', report_json TEXT NOT NULL DEFAULT '{}',
		plan_type TEXT NOT NULL DEFAULT 'internship', holiday_dates_json TEXT NOT NULL DEFAULT '[]'
	);
	CREATE TABLE IF NOT EXISTS invites (
		token_hash TEXT PRIMARY KEY, internship_id BIGINT NOT NULL REFERENCES internships(id), email TEXT NOT NULL, expires_at TEXT NOT NULL, used_at TEXT
	);
	CREATE TABLE IF NOT EXISTS approvals (
		id BIGSERIAL PRIMARY KEY, internship_id BIGINT NOT NULL REFERENCES internships(id), mentor_id BIGINT NOT NULL REFERENCES users(id),
		from_date TEXT NOT NULL, to_date TEXT NOT NULL, typed_name TEXT NOT NULL, signed_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS day_logs (
		id BIGSERIAL PRIMARY KEY, internship_id BIGINT NOT NULL REFERENCES internships(id), date TEXT NOT NULL,
		start_time TEXT NOT NULL DEFAULT '', end_time TEXT NOT NULL DEFAULT '', break_minutes INTEGER NOT NULL DEFAULT 0,
		kind TEXT NOT NULL DEFAULT 'work', late BOOLEAN NOT NULL DEFAULT FALSE, work TEXT NOT NULL DEFAULT '', blockers TEXT NOT NULL DEFAULT '', learning TEXT NOT NULL DEFAULT '',
		approval_id BIGINT REFERENCES approvals(id), UNIQUE(internship_id,date)
	);
	CREATE TABLE IF NOT EXISTS evaluations (
		internship_id BIGINT PRIMARY KEY REFERENCES internships(id), scores_json TEXT NOT NULL DEFAULT '{}',
		improvement TEXT NOT NULL DEFAULT '', strengths TEXT NOT NULL DEFAULT '', suggestions TEXT NOT NULL DEFAULT '',
		typed_name TEXT NOT NULL DEFAULT '', submitted_at TEXT NOT NULL DEFAULT ''
	);`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE internships SET university='มหาวิทยาลัยราชภัฏนครปฐม', year='2569', term='2' WHERE university!='มหาวิทยาลัยราชภัฏนครปฐม' OR year!='2569' OR term!='2'`)
	if err != nil {
		return err
	}
	return nil
}

func token() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
func hashToken(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func validEmail(s string) bool {
	a, e := mail.ParseAddress(s)
	return e == nil && strings.EqualFold(a.Address, s)
}
func requestUser(c *fiber.Ctx) (User, error) {
	var u User
	t := c.Cookies("session")
	if t == "" {
		return u, errors.New("not logged in")
	}
	err := db.QueryRow(`SELECT u.id,u.email,u.role,u.name,u.phone,u.position,u.birth_date,u.home_address,u.subdistrict,u.district,u.province,u.country,u.skills,u.interests FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?`, hashToken(t), time.Now().UTC().Format(time.RFC3339)).Scan(&u.ID, &u.Email, &u.Role, &u.Name, &u.Phone, &u.Position, &u.BirthDate, &u.HomeAddress, &u.Subdistrict, &u.District, &u.Province, &u.Country, &u.Skills, &u.Interests)
	if err == nil {
		u.PhotoURL = fmt.Sprintf("/api/profile/photo/%d", u.ID)
	}
	return u, err
}
func requireUser(c *fiber.Ctx) error {
	u, e := requestUser(c)
	if e != nil {
		return fail(c, 401, "กรุณาเข้าสู่ระบบ")
	}
	c.Locals("user", u)
	return c.Next()
}
func current(c *fiber.Ctx) User { return c.Locals("user").(User) }
func setSession(c *fiber.Ctx, id int64) error {
	t, e := token()
	if e != nil {
		return e
	}
	expires := time.Now().UTC().Add(14 * 24 * time.Hour)
	if _, e = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, hashToken(t), id, expires.Format(time.RFC3339)); e != nil {
		return e
	}
	c.Cookie(&fiber.Cookie{Name: "session", Value: t, HTTPOnly: true, SameSite: "Strict", Secure: os.Getenv("APP_SECURE_COOKIE") == "1", Expires: expires, Path: "/"})
	return nil
}
func register(c *fiber.Ctx) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Invite   string `json:"invite"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if !validEmail(in.Email) || len(in.Password) < 10 || in.Name == "" {
		return fail(c, 400, "กรอกชื่อ อีเมล และรหัสผ่านอย่างน้อย 10 ตัวอักษร")
	}
	role := "student"
	var internshipID int64
	if in.Invite != "" {
		role = "mentor"
		err := db.QueryRow(`SELECT v.internship_id FROM invites v JOIN internships i ON i.id=v.internship_id WHERE v.token_hash=? AND v.email=? AND i.mentor_email=v.email AND i.mentor_id IS NULL AND v.used_at IS NULL AND v.expires_at>?`, hashToken(in.Invite), in.Email, time.Now().UTC().Format(time.RFC3339)).Scan(&internshipID)
		if err != nil {
			return fail(c, 400, "คำเชิญหมดอายุหรืออีเมลไม่ตรง")
		}
	}
	pass, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		return fail(c, 500, "สร้างบัญชีไม่สำเร็จ")
	}
	tx, e := db.Begin()
	if e != nil {
		return fail(c, 500, "ฐานข้อมูลไม่พร้อม")
	}
	defer tx.Rollback()
	var id int64
	e = tx.QueryRow(`INSERT INTO users(email,password_hash,role,name) VALUES(?,?,?,?) RETURNING id`, in.Email, string(pass), role, in.Name).Scan(&id)
	if e != nil {
		return fail(c, 409, "อีเมลนี้ถูกใช้งานแล้ว")
	}
	if role == "student" {
		_, e = tx.Exec(`INSERT INTO internships(student_id,university,year,term) VALUES(?,'มหาวิทยาลัยราชภัฏนครปฐม','2569','2')`, id)
	} else {
		res, ee := tx.Exec(`UPDATE internships SET mentor_id=? WHERE id=? AND mentor_email=? AND mentor_id IS NULL AND EXISTS (SELECT 1 FROM invites v WHERE v.token_hash=? AND v.internship_id=internships.id AND v.email=internships.mentor_email AND v.used_at IS NULL AND v.expires_at>?)`, id, internshipID, in.Email, hashToken(in.Invite), time.Now().UTC().Format(time.RFC3339))
		if ee != nil {
			return fail(c, 500, "เชื่อมพี่เลี้ยงไม่สำเร็จ")
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			return fail(c, 409, "มีพี่เลี้ยงเชื่อมแล้ว")
		}
		_, e = tx.Exec(`UPDATE invites SET used_at=? WHERE token_hash=?`, time.Now().UTC().Format(time.RFC3339), hashToken(in.Invite))
	}
	if e != nil {
		return fail(c, 500, "สร้างบัญชีไม่สำเร็จ")
	}
	if e = tx.Commit(); e != nil {
		return fail(c, 500, "สร้างบัญชีไม่สำเร็จ")
	}
	if e = setSession(c, id); e != nil {
		return fail(c, 500, "เข้าสู่ระบบไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func login(c *fiber.Ctx) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	var id int64
	var hash string
	e := db.QueryRow(`SELECT id,password_hash FROM users WHERE email=?`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash)
	if e != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		return fail(c, 401, "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
	}
	if e = setSession(c, id); e != nil {
		return fail(c, 500, "เข้าสู่ระบบไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func logout(c *fiber.Ctx) error {
	if t := c.Cookies("session"); t != "" {
		db.Exec(`DELETE FROM sessions WHERE token_hash=?`, hashToken(t))
	}
	c.Cookie(&fiber.Cookie{Name: "session", Value: "", HTTPOnly: true, SameSite: "Strict", Path: "/", Expires: time.Unix(0, 0)})
	return c.JSON(fiber.Map{"ok": true})
}
func me(c *fiber.Ctx) error { return c.JSON(current(c)) }
func saveProfile(c *fiber.Ctx) error {
	u := current(c)
	var in User
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 150 || len(in.Phone) > 40 || len(in.Position) > 120 {
		return fail(c, 400, "ข้อมูลโปรไฟล์ไม่ถูกต้อง")
	}
	if in.BirthDate != "" && !validDate(in.BirthDate) {
		return fail(c, 400, "วันเกิดไม่ถูกต้อง")
	}
	if len(in.HomeAddress) > 300 || len(in.Subdistrict) > 100 || len(in.District) > 100 || len(in.Province) > 100 || len(in.Country) > 100 || len(in.Skills) > 1000 || len(in.Interests) > 1000 {
		return fail(c, 400, "ข้อมูลโปรไฟล์ยาวเกินไป")
	}
	if u.Role == "student" && in.Province == "other" && strings.TrimSpace(in.Country) == "" {
		return fail(c, 400, "กรอกชื่อประเทศ")
	}
	if u.Role == "student" && in.Province != "" && in.Province != "other" && !validProvince(in.Province) {
		return fail(c, 400, "จังหวัดไม่ถูกต้อง")
	}
	if in.Province != "other" {
		in.Country = "ประเทศไทย"
	}
	_, e := db.Exec(`UPDATE users SET name=?,phone=?,position=?,birth_date=?,home_address=?,subdistrict=?,district=?,province=?,country=?,skills=?,interests=? WHERE id=?`, in.Name, in.Phone, in.Position, in.BirthDate, in.HomeAddress, in.Subdistrict, in.District, in.Province, in.Country, in.Skills, in.Interests, u.ID)
	if e != nil {
		return fail(c, 500, "บันทึกไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func uploadPhoto(c *fiber.Ctx) error {
	f, e := c.FormFile("photo")
	if e != nil || f.Size > 2*1024*1024 {
		return fail(c, 400, "เลือกรูปภาพขนาดไม่เกิน 2 MB")
	}
	file, e := f.Open()
	if e != nil {
		return fail(c, 400, "เปิดรูปไม่ได้")
	}
	defer file.Close()
	b, e := io.ReadAll(io.LimitReader(file, 2*1024*1024+1))
	if e != nil || len(b) > 2*1024*1024 {
		return fail(c, 400, "รูปใหญ่เกินไป")
	}
	content := http.DetectContentType(b)
	if content != "image/jpeg" && content != "image/png" {
		return fail(c, 400, "รองรับ JPG หรือ PNG")
	}
	_, e = db.Exec(`UPDATE users SET photo=?,photo_type=? WHERE id=?`, b, content, current(c).ID)
	if e != nil {
		return fail(c, 500, "บันทึกรูปไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func photo(c *fiber.Ctx) error {
	id, e := strconv.ParseInt(c.Params("id"), 10, 64)
	if e != nil {
		return fail(c, 400, "รหัสไม่ถูกต้อง")
	}
	u := current(c)
	if id != u.ID {
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM internships WHERE (student_id=? AND mentor_id=?) OR (student_id=? AND mentor_id=?)`, u.ID, id, id, u.ID).Scan(&n)
		if n == 0 {
			return fail(c, 403, "ไม่มีสิทธิ์ดูรูป")
		}
	}
	var b []byte
	var content string
	e = db.QueryRow(`SELECT photo,photo_type FROM users WHERE id=?`, id).Scan(&b, &content)
	if e != nil || len(b) == 0 {
		return fail(c, 404, "ไม่มีรูป")
	}
	c.Set("Content-Type", content)
	c.Set("Cache-Control", "private, max-age=60")
	return c.Send(b)
}
func internshipFor(u User) (*Internship, error) {
	var x Internship
	var reportJSON, holidayJSON string
	q := `SELECT i.id,i.student_id,s.name,i.mentor_id,i.university,i.faculty,i.major,i.student_number,i.class_group,i.year,i.term,i.company,i.company_address,i.department,i.position,i.start_date,i.end_date,i.plan_type,i.holiday_dates_json,i.mentor_email,i.report_json,COALESCE(m.name,''),COALESCE(m.position,''),COALESCE(m.phone,'') FROM internships i JOIN users s ON s.id=i.student_id LEFT JOIN users m ON m.id=i.mentor_id WHERE `
	if u.Role == "student" {
		q += "i.student_id=?"
	} else {
		q += "i.mentor_id=?"
	}
	e := db.QueryRow(q, u.ID).Scan(&x.ID, &x.StudentID, &x.StudentName, &x.MentorID, &x.University, &x.Faculty, &x.Major, &x.StudentNumber, &x.ClassGroup, &x.Year, &x.Term, &x.Company, &x.CompanyAddress, &x.Department, &x.Position, &x.StartDate, &x.EndDate, &x.PlanType, &holidayJSON, &x.MentorEmail, &reportJSON, &x.MentorName, &x.MentorPosition, &x.MentorPhone)
	if e != nil {
		return nil, e
	}
	x.MentorLinked = x.MentorID.Valid
	x.Report = map[string]string{}
	json.Unmarshal([]byte(reportJSON), &x.Report)
	x.HolidayDates = []string{}
	json.Unmarshal([]byte(holidayJSON), &x.HolidayDates)
	return &x, nil
}
func getData(c *fiber.Ctx) error {
	u := current(c)
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบข้อมูลการฝึกงาน")
	}
	d := Data{User: u, Internship: x, Logs: []DayLog{}, Approvals: []Approval{}}
	if u.Role == "mentor" {
		x.Report = nil
	}
	rows, e := db.Query(`SELECT id,date,start_time,end_time,break_minutes,kind,late,work,blockers,learning,COALESCE(approval_id,0) FROM day_logs WHERE internship_id=? ORDER BY date DESC`, x.ID)
	if e != nil {
		return fail(c, 500, "อ่านบันทึกไม่สำเร็จ")
	}
	for rows.Next() {
		var l DayLog
		if rows.Scan(&l.ID, &l.Date, &l.StartTime, &l.EndTime, &l.BreakMinutes, &l.Kind, &l.Late, &l.Work, &l.Blockers, &l.Learning, &l.ApprovalID) == nil {
			l.Hours = hours(l)
			d.Logs = append(d.Logs, l)
		}
	}
	rows.Close()
	rows, e = db.Query(`SELECT a.id,a.from_date,a.to_date,a.typed_name,a.signed_at,COUNT(l.id) FROM approvals a LEFT JOIN day_logs l ON l.approval_id=a.id WHERE a.internship_id=? GROUP BY a.id ORDER BY a.signed_at DESC`, x.ID)
	if e == nil {
		for rows.Next() {
			var a Approval
			if rows.Scan(&a.ID, &a.From, &a.To, &a.TypedName, &a.SignedAt, &a.Count) == nil {
				d.Approvals = append(d.Approvals, a)
			}
		}
		rows.Close()
	}
	var scoresJSON string
	var ev Evaluation
	e = db.QueryRow(`SELECT scores_json,improvement,strengths,suggestions,typed_name,submitted_at FROM evaluations WHERE internship_id=?`, x.ID).Scan(&scoresJSON, &ev.Improvement, &ev.Strengths, &ev.Suggestions, &ev.TypedName, &ev.SubmittedAt)
	if e == nil && (u.Role == "mentor" || ev.SubmittedAt != "") {
		ev.Scores = map[string]string{}
		json.Unmarshal([]byte(scoresJSON), &ev.Scores)
		d.Evaluation = &ev
	}
	return c.JSON(d)
}
func validDate(s string) bool  { _, e := time.Parse("2006-01-02", s); return e == nil }
func validClock(s string) bool { _, e := time.Parse("15:04", s); return e == nil }
func todayBangkok() string {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	return time.Now().In(loc).Format("2006-01-02")
}
func saveInternship(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	var x Internship
	if c.BodyParser(&x) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	if (x.StartDate != "" && !validDate(x.StartDate)) || (x.EndDate != "" && !validDate(x.EndDate)) || (x.StartDate != "" && x.EndDate != "" && x.StartDate > x.EndDate) {
		return fail(c, 400, "ช่วงวันที่ไม่ถูกต้อง")
	}
	x.University, x.Year, x.Term = "มหาวิทยาลัยราชภัฏนครปฐม", "2569", "2"
	if x.StudentNumber != "" && !regexp.MustCompile(`^6642590[0-9]{2}$`).MatchString(x.StudentNumber) {
		return fail(c, 400, "รหัสนักศึกษาต้องเป็น 6642590xx")
	}
	if len(x.ClassGroup) > 40 {
		return fail(c, 400, "หมู่เรียนยาวเกินไป")
	}
	if x.PlanType != "internship" && x.PlanType != "cooperative" {
		return fail(c, 400, "แผนการฝึกไม่ถูกต้อง")
	}
	if len(x.HolidayDates) > 100 {
		return fail(c, 400, "วันหยุดมากเกินไป")
	}
	seenHolidays := map[string]bool{}
	for _, date := range x.HolidayDates {
		if !validDate(date) || seenHolidays[date] {
			return fail(c, 400, "วันหยุดไม่ถูกต้องหรือซ้ำกัน")
		}
		seenHolidays[date] = true
	}
	if len(x.Company) > 200 || len(x.University) > 200 || len(x.CompanyAddress) > 400 {
		return fail(c, 400, "ข้อความยาวเกินไป")
	}
	if x.MentorEmail != "" && !validEmail(strings.ToLower(strings.TrimSpace(x.MentorEmail))) {
		return fail(c, 400, "อีเมลพี่เลี้ยงไม่ถูกต้อง")
	}
	old, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	x.MentorEmail = strings.ToLower(strings.TrimSpace(x.MentorEmail))
	if old.MentorLinked {
		var linkedEmail string
		if e = db.QueryRow(`SELECT email FROM users WHERE id=?`, old.MentorID.Int64).Scan(&linkedEmail); e != nil {
			return fail(c, 500, "ไม่พบข้อมูลพี่เลี้ยงที่เชื่อมอยู่")
		}
		if x.MentorEmail != linkedEmail {
			return fail(c, 409, "เชื่อมพี่เลี้ยงแล้ว เปลี่ยนอีเมลไม่ได้")
		}
	}
	tx, e := db.Begin()
	if e != nil {
		return fail(c, 500, "ฐานข้อมูลไม่พร้อม")
	}
	defer tx.Rollback()
	var firstLog, lastLog sql.NullString
	if e = tx.QueryRow(`SELECT MIN(date),MAX(date) FROM day_logs WHERE internship_id=?`, old.ID).Scan(&firstLog, &lastLog); e != nil {
		return fail(c, 500, "ตรวจสอบช่วงวันที่ไม่สำเร็จ")
	}
	if firstLog.Valid && ((x.StartDate != "" && x.StartDate > firstLog.String) || (x.EndDate != "" && x.EndDate < lastLog.String)) {
		return fail(c, 409, "ช่วงวันที่ใหม่ต้องครอบคลุมบันทึกรายวันที่มีอยู่")
	}
	if x.Report == nil {
		x.Report = map[string]string{}
	}
	b, _ := json.Marshal(x.Report)
	holidaysJSON, _ := json.Marshal(x.HolidayDates)
	if x.MentorEmail != old.MentorEmail {
		if _, e = tx.Exec(`DELETE FROM invites WHERE internship_id=? AND used_at IS NULL`, old.ID); e != nil {
			return fail(c, 500, "ยกเลิกคำเชิญเดิมไม่สำเร็จ")
		}
	}
	_, e = tx.Exec(`UPDATE internships SET university=?,faculty=?,major=?,student_number=?,class_group=?,year=?,term=?,company=?,company_address=?,department=?,position=?,start_date=?,end_date=?,plan_type=?,holiday_dates_json=?,mentor_email=?,report_json=? WHERE student_id=?`, x.University, x.Faculty, x.Major, x.StudentNumber, x.ClassGroup, x.Year, x.Term, x.Company, x.CompanyAddress, x.Department, x.Position, x.StartDate, x.EndDate, x.PlanType, string(holidaysJSON), x.MentorEmail, string(b), u.ID)
	if e != nil {
		return fail(c, 500, "บันทึกไม่สำเร็จ")
	}
	if e = tx.Commit(); e != nil {
		return fail(c, 500, "บันทึกไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func createInvite(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	x, e := internshipFor(u)
	if e != nil || x.MentorEmail == "" {
		return fail(c, 400, "กรอกอีเมลพี่เลี้ยงก่อน")
	}
	if x.MentorLinked {
		return fail(c, 409, "เชื่อมพี่เลี้ยงแล้ว")
	}
	t, e := token()
	if e != nil {
		return fail(c, 500, "สร้างคำเชิญไม่สำเร็จ")
	}
	_, e = db.Exec(`INSERT INTO invites(token_hash,internship_id,email,expires_at) VALUES(?,?,?,?)`, hashToken(t), x.ID, x.MentorEmail, time.Now().UTC().Add(7*24*time.Hour).Format(time.RFC3339))
	if e != nil {
		return fail(c, 500, "สร้างคำเชิญไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"invite_url": "/invite/" + t, "expires_in_days": 7})
}
func inviteInfo(c *fiber.Ctx) error {
	var email, student, company string
	e := db.QueryRow(`SELECT v.email,u.name,i.company FROM invites v JOIN internships i ON i.id=v.internship_id JOIN users u ON u.id=i.student_id WHERE v.token_hash=? AND v.email=i.mentor_email AND v.used_at IS NULL AND v.expires_at>? AND i.mentor_id IS NULL`, hashToken(c.Params("token")), time.Now().UTC().Format(time.RFC3339)).Scan(&email, &student, &company)
	if e != nil {
		return fail(c, 404, "คำเชิญไม่ถูกต้องหรือหมดอายุ")
	}
	return c.JSON(fiber.Map{"email": email, "student": student, "company": company})
}
func hours(l DayLog) float64 {
	if l.Kind != "work" || !validClock(l.StartTime) || !validClock(l.EndTime) {
		return 0
	}
	a, _ := time.Parse("15:04", l.StartTime)
	b, _ := time.Parse("15:04", l.EndTime)
	mins := int(b.Sub(a).Minutes()) - l.BreakMinutes
	if mins < 0 {
		return 0
	}
	return float64(mins) / 60
}
func saveLog(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	var l DayLog
	if c.BodyParser(&l) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	if !validDate(l.Date) || l.Kind != "work" && l.Kind != "sick" && l.Kind != "personal" && l.Kind != "holiday" {
		return fail(c, 400, "วันที่หรือประเภทไม่ถูกต้อง")
	}
	if l.Date > todayBangkok() {
		return fail(c, 400, "ยังบันทึกวันในอนาคตไม่ได้")
	}
	if x.StartDate != "" && l.Date < x.StartDate || x.EndDate != "" && l.Date > x.EndDate {
		return fail(c, 400, "วันที่อยู่นอกช่วงฝึกงาน")
	}
	if l.Kind == "work" {
		if !validClock(l.StartTime) || !validClock(l.EndTime) || l.StartTime >= l.EndTime || l.BreakMinutes < 0 || l.BreakMinutes > 600 || hours(l) <= 0 || strings.TrimSpace(l.Work) == "" {
			return fail(c, 400, "กรอกเวลาและงานที่ทำให้ครบ")
		}
	}
	if l.Kind != "work" {
		l.Late = false
	}
	if len(l.Work) > 4000 || len(l.Blockers) > 2000 || len(l.Learning) > 2000 {
		return fail(c, 400, "ข้อความยาวเกินไป")
	}
	if c.Method() == fiber.MethodPut {
		id, e := strconv.ParseInt(c.Params("id"), 10, 64)
		if e != nil {
			return fail(c, 400, "รหัสไม่ถูกต้อง")
		}
		r, e := db.Exec(`UPDATE day_logs SET date=?,start_time=?,end_time=?,break_minutes=?,kind=?,late=?,work=?,blockers=?,learning=? WHERE id=? AND internship_id=? AND approval_id IS NULL`, l.Date, l.StartTime, l.EndTime, l.BreakMinutes, l.Kind, l.Late, l.Work, l.Blockers, l.Learning, id, x.ID)
		if e != nil {
			return fail(c, 409, "วันที่นี้มีบันทึกแล้ว")
		}
		n, _ := r.RowsAffected()
		if n == 0 {
			return fail(c, 409, "รายการนี้รับรองแล้วหรือไม่พบ")
		}
	} else {
		_, e = db.Exec(`INSERT INTO day_logs(internship_id,date,start_time,end_time,break_minutes,kind,late,work,blockers,learning) VALUES(?,?,?,?,?,?,?,?,?,?)`, x.ID, l.Date, l.StartTime, l.EndTime, l.BreakMinutes, l.Kind, l.Late, l.Work, l.Blockers, l.Learning)
		if e != nil {
			return fail(c, 409, "วันที่นี้มีบันทึกแล้ว")
		}
	}
	return c.JSON(fiber.Map{"ok": true})
}
func deleteLog(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "student" {
		return fail(c, 403, "เฉพาะนักศึกษา")
	}
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	id, e := strconv.ParseInt(c.Params("id"), 10, 64)
	if e != nil {
		return fail(c, 400, "รหัสไม่ถูกต้อง")
	}
	r, e := db.Exec(`DELETE FROM day_logs WHERE id=? AND internship_id=? AND approval_id IS NULL`, id, x.ID)
	if e != nil {
		return fail(c, 500, "ลบไม่สำเร็จ")
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return fail(c, 409, "รายการนี้รับรองแล้วหรือไม่พบ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
func approve(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "mentor" {
		return fail(c, 403, "เฉพาะพี่เลี้ยง")
	}
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	var in struct {
		From      string `json:"from"`
		To        string `json:"to"`
		TypedName string `json:"typed_name"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	in.TypedName = strings.TrimSpace(in.TypedName)
	if !validDate(in.From) || !validDate(in.To) || in.From > in.To || in.To > todayBangkok() || in.TypedName != u.Name {
		return fail(c, 400, "ช่วงวันที่หรือชื่อผู้รับรองไม่ถูกต้อง")
	}
	from, _ := time.Parse("2006-01-02", in.From)
	to, _ := time.Parse("2006-01-02", in.To)
	if to.Sub(from) > 31*24*time.Hour {
		return fail(c, 400, "รับรองได้ครั้งละไม่เกิน 31 วัน")
	}
	tx, e := db.Begin()
	if e != nil {
		return fail(c, 500, "ฐานข้อมูลไม่พร้อม")
	}
	defer tx.Rollback()
	var count int
	tx.QueryRow(`SELECT COUNT(*) FROM day_logs WHERE internship_id=? AND date BETWEEN ? AND ? AND approval_id IS NULL`, x.ID, in.From, in.To).Scan(&count)
	if count == 0 {
		return fail(c, 400, "ไม่มีรายการที่รอรับรองในช่วงนี้")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var id int64
	e = tx.QueryRow(`INSERT INTO approvals(internship_id,mentor_id,from_date,to_date,typed_name,signed_at) VALUES(?,?,?,?,?,?) RETURNING id`, x.ID, u.ID, in.From, in.To, in.TypedName, now).Scan(&id)
	if e != nil {
		return fail(c, 500, "รับรองไม่สำเร็จ")
	}
	_, e = tx.Exec(`UPDATE day_logs SET approval_id=? WHERE internship_id=? AND date BETWEEN ? AND ? AND approval_id IS NULL`, id, x.ID, in.From, in.To)
	if e != nil {
		return fail(c, 500, "รับรองไม่สำเร็จ")
	}
	if e = tx.Commit(); e != nil {
		return fail(c, 500, "รับรองไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true, "count": count})
}

var criteria = []string{"1.1", "1.2", "1.3", "1.4", "2.1", "2.2", "2.3", "2.4", "2.5", "2.6", "3.1", "3.2", "3.3", "3.4", "4.1", "4.2", "4.3", "4.4", "5.1", "5.2", "5.3", "5.4"}

func saveEvaluation(c *fiber.Ctx) error {
	u := current(c)
	if u.Role != "mentor" {
		return fail(c, 403, "เฉพาะพี่เลี้ยง")
	}
	x, e := internshipFor(u)
	if e != nil {
		return fail(c, 404, "ไม่พบการฝึกงาน")
	}
	var ev Evaluation
	var in struct {
		Evaluation
		Submit bool `json:"submit"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	ev = in.Evaluation
	var submitted string
	db.QueryRow(`SELECT submitted_at FROM evaluations WHERE internship_id=?`, x.ID).Scan(&submitted)
	if submitted != "" {
		return fail(c, 409, "ผลประเมินส่งแล้ว แก้ไขไม่ได้")
	}
	if ev.Scores == nil {
		ev.Scores = map[string]string{}
	}
	for _, key := range criteria {
		s := ev.Scores[key]
		if s != "" && s != "0" && s != "1" && s != "2" && s != "3" && s != "4" && s != "*" {
			return fail(c, 400, "คะแนนไม่ถูกต้อง")
		}
		if in.Submit && s == "" {
			return fail(c, 400, "ให้คะแนนครบทุกข้อก่อนส่ง")
		}
	}
	if in.Submit {
		ev.TypedName = strings.TrimSpace(ev.TypedName)
		if ev.TypedName != u.Name {
			return fail(c, 400, "พิมพ์ชื่อให้ตรงกับโปรไฟล์พี่เลี้ยง")
		}
		ev.SubmittedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		ev.TypedName = ""
		ev.SubmittedAt = ""
	}
	b, _ := json.Marshal(ev.Scores)
	_, e = db.Exec(`INSERT INTO evaluations(internship_id,scores_json,improvement,strengths,suggestions,typed_name,submitted_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(internship_id) DO UPDATE SET scores_json=excluded.scores_json,improvement=excluded.improvement,strengths=excluded.strengths,suggestions=excluded.suggestions,typed_name=excluded.typed_name,submitted_at=excluded.submitted_at`, x.ID, string(b), ev.Improvement, ev.Strengths, ev.Suggestions, ev.TypedName, ev.SubmittedAt)
	if e != nil {
		return fail(c, 500, "บันทึกประเมินไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"ok": true})
}
