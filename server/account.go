package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLen = 8

// validatePassword returns a Thai error message, or "" when the password is acceptable.
// bcrypt only uses the first 72 bytes, so longer passwords are rejected instead of silently truncated.
func validatePassword(p string) string {
	if utf8.RuneCountInString(p) < minPasswordLen {
		return fmt.Sprintf("รหัสผ่านต้องมีอย่างน้อย %d ตัวอักษร", minPasswordLen)
	}
	if len(p) > 72 {
		return "รหัสผ่านยาวเกินไป"
	}
	return ""
}

// ---- Login lockout -------------------------------------------------------

const (
	lockWindow       = 15 * time.Minute
	lockDuration     = 15 * time.Minute
	maxEmailFailures = 5
	maxIPFailures    = 20
)

type attemptState struct {
	failures    []time.Time
	lockedUntil time.Time
}

var loginAttempts = struct {
	sync.Mutex
	m map[string]*attemptState
}{m: map[string]*attemptState{}}

// clientIP prefers the first X-Forwarded-For hop (Render/Vercel sit in front of the app).
func clientIP(c *fiber.Ctx) string {
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	return c.IP()
}

func loginKeys(c *fiber.Ctx, email string) []string {
	return []string{"email:" + strings.ToLower(strings.TrimSpace(email)), "ip:" + clientIP(c)}
}

// lockedFor reports how long until any of the keys is unlocked (0 when not locked).
func lockedFor(keys []string, now time.Time) time.Duration {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	var wait time.Duration
	for _, k := range keys {
		if st := loginAttempts.m[k]; st != nil && now.Before(st.lockedUntil) {
			if d := st.lockedUntil.Sub(now); d > wait {
				wait = d
			}
		}
	}
	return wait
}

func recordLoginFailure(keys []string, now time.Time) {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	for _, k := range keys {
		st := loginAttempts.m[k]
		if st == nil {
			st = &attemptState{}
			loginAttempts.m[k] = st
		}
		kept := st.failures[:0]
		for _, t := range st.failures {
			if now.Sub(t) < lockWindow {
				kept = append(kept, t)
			}
		}
		st.failures = append(kept, now)
		limit := maxEmailFailures
		if strings.HasPrefix(k, "ip:") {
			limit = maxIPFailures
		}
		if len(st.failures) >= limit {
			st.lockedUntil = now.Add(lockDuration)
			st.failures = nil
		}
	}
	// Opportunistic cleanup so the map doesn't grow forever.
	if len(loginAttempts.m) > 5000 {
		for k, st := range loginAttempts.m {
			if now.After(st.lockedUntil) && (len(st.failures) == 0 || now.Sub(st.failures[len(st.failures)-1]) > lockWindow) {
				delete(loginAttempts.m, k)
			}
		}
	}
}

func clearLoginFailures(email string) {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	delete(loginAttempts.m, "email:"+strings.ToLower(strings.TrimSpace(email)))
}

func lockMessage(wait time.Duration) string {
	minutes := int(wait.Minutes() + 0.999)
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("ใส่รหัสผ่านผิดหลายครั้ง กรุณารอ %d นาทีแล้วลองใหม่", minutes)
}

// ---- Change own password ---------------------------------------------------

// PUT /api/profile/password {current, new}
func changePassword(c *fiber.Ctx) error {
	u := current(c)
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	var hash string
	if db.QueryRow(`SELECT password_hash FROM users WHERE id=?`, u.ID).Scan(&hash) != nil {
		return fail(c, 500, "เปลี่ยนรหัสผ่านไม่สำเร็จ")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Current)) != nil {
		return fail(c, 400, "รหัสผ่านปัจจุบันไม่ถูกต้อง")
	}
	if msg := validatePassword(in.New); msg != "" {
		return fail(c, 400, msg)
	}
	if in.New == in.Current {
		return fail(c, 400, "รหัสผ่านใหม่ต้องไม่ซ้ำกับรหัสเดิม")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(in.New), bcrypt.DefaultCost)
	if err != nil {
		return fail(c, 500, "เปลี่ยนรหัสผ่านไม่สำเร็จ")
	}
	if _, err = db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, string(newHash), u.ID); err != nil {
		return fail(c, 500, "เปลี่ยนรหัสผ่านไม่สำเร็จ")
	}
	// Sign out every other device; keep this session.
	db.Exec(`DELETE FROM sessions WHERE user_id=? AND token_hash<>?`, u.ID, hashToken(c.Cookies("session")))
	return c.JSON(fiber.Map{"ok": true})
}

// ---- Admin reset -------------------------------------------------------------

const tempAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

func tempPassword(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(tempAlphabet))))
		if err != nil {
			return "", err
		}
		b.WriteByte(tempAlphabet[idx.Int64()])
	}
	return b.String(), nil
}

// POST /api/admin/users/:id/reset-password → {password} shown once to the admin.
func adminResetPassword(c *fiber.Ctx) error {
	admin := current(c)
	if !admin.IsAdmin {
		return fail(c, 403, "เฉพาะผู้ดูแลระบบ")
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fail(c, 400, "รหัสผู้ใช้ไม่ถูกต้อง")
	}
	if id == admin.ID {
		return fail(c, 400, "เปลี่ยนรหัสผ่านของตัวเองได้ที่หน้าข้อมูลส่วนตัว")
	}
	var email string
	if db.QueryRow(`SELECT email FROM users WHERE id=?`, id).Scan(&email) != nil {
		return fail(c, 404, "ไม่พบผู้ใช้")
	}
	pass, err := tempPassword(10)
	if err != nil {
		return fail(c, 500, "สร้างรหัสผ่านไม่สำเร็จ")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return fail(c, 500, "สร้างรหัสผ่านไม่สำเร็จ")
	}
	if _, err = db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, string(hash), id); err != nil {
		return fail(c, 500, "รีเซ็ตรหัสผ่านไม่สำเร็จ")
	}
	db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	clearLoginFailures(email)
	return c.JSON(fiber.Map{"password": pass, "email": email})
}
