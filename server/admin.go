package main

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Admins are configured by email, not stored in the database, so nobody can
// grant themselves admin through the app. ADMIN_EMAILS is comma-separated.
func isAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	for _, e := range strings.Split(os.Getenv("ADMIN_EMAILS"), ",") {
		if strings.ToLower(strings.TrimSpace(e)) == email {
			return true
		}
	}
	return false
}

func migrateAdmin() error {
	for _, q := range []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// markLogin records sign-in time (and first-seen time for accounts created before tracking).
func markLogin(userID int64) {
	now := time.Now().UTC().Format(time.RFC3339)
	db.Exec(`UPDATE users SET last_login_at=?, created_at=CASE WHEN created_at='' THEN ? ELSE created_at END WHERE id=?`, now, now, userID)
}

type adminUserRow struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	LastLoginAt string `json:"last_login_at"`
	Company     string `json:"company"`
	MentorName  string `json:"mentor_name"`
	StudentName string `json:"student_name"`
	Mentees     int    `json:"mentees"`
	LogCount    int    `json:"log_count"`
	LastLogDate string `json:"last_log_date"`
	Approved    int    `json:"approved"`
	ActiveNow   bool   `json:"active_session"`
	IsAdmin     bool   `json:"is_admin"`
	HasPhoto    bool   `json:"has_photo"`
}

// GET /api/admin/users — usage overview for admins. No passwords, photos, or log contents.
func adminUsers(c *fiber.Ctx) error {
	u := current(c)
	if !u.IsAdmin {
		return fail(c, 403, "เฉพาะผู้ดูแลระบบ")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := db.Query(`SELECT u.id, u.email, u.name, u.role, u.created_at, u.last_login_at,
		COALESCE(si.company, mi.company, ''),
		COALESCE(sm.name, ''),
		COALESCE(ms.name, ''),
		(SELECT COUNT(*) FROM internships x WHERE x.mentor_id=u.id),
		COALESCE((SELECT COUNT(*) FROM day_logs l WHERE l.internship_id=si.id), 0),
		COALESCE((SELECT MAX(l.date) FROM day_logs l WHERE l.internship_id=si.id), ''),
		COALESCE((SELECT COUNT(*) FROM day_logs l WHERE l.internship_id=si.id AND l.approval_id IS NOT NULL), 0),
		EXISTS (SELECT 1 FROM sessions s WHERE s.user_id=u.id AND s.expires_at>?),
		(u.photo IS NOT NULL AND length(u.photo)>0)
	FROM users u
	LEFT JOIN internships si ON si.student_id=u.id
	LEFT JOIN users sm ON sm.id=si.mentor_id
	LEFT JOIN LATERAL (SELECT company, student_id FROM internships WHERE mentor_id=u.id ORDER BY id LIMIT 1) mi ON true
	LEFT JOIN users ms ON ms.id=mi.student_id
	ORDER BY u.last_login_at DESC, u.id DESC`, now)
	if err != nil {
		return fail(c, 500, "อ่านข้อมูลผู้ใช้ไม่สำเร็จ")
	}
	defer rows.Close()
	list := []adminUserRow{}
	for rows.Next() {
		var r adminUserRow
		if err = rows.Scan(&r.ID, &r.Email, &r.Name, &r.Role, &r.CreatedAt, &r.LastLoginAt, &r.Company, &r.MentorName, &r.StudentName, &r.Mentees, &r.LogCount, &r.LastLogDate, &r.Approved, &r.ActiveNow, &r.HasPhoto); err != nil {
			return fail(c, 500, "อ่านข้อมูลผู้ใช้ไม่สำเร็จ")
		}
		r.IsAdmin = isAdminEmail(r.Email)
		list = append(list, r)
	}
	if rows.Err() != nil {
		return fail(c, 500, "อ่านข้อมูลผู้ใช้ไม่สำเร็จ")
	}
	return c.JSON(fiber.Map{"users": list})
}
