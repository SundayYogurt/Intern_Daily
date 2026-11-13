package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// AI helpers. Two providers:
//   - Typhoon / any OpenAI-compatible API (AI_PROVIDER=typhoon or openai):
//       AI_API_KEY (required), AI_BASE_URL (default https://api.opentyphoon.ai/v1),
//       AI_MODEL (default typhoon-v2.5-30b-a3b-instruct)
//   - Ollama native /api/chat (default when AI_PROVIDER is empty or "ollama"):
//       OLLAMA_API_KEY (required), OLLAMA_HOST (default https://ollama.com), OLLAMA_MODEL (default kimi-k2.6)

const aiSystemPrompt = `คุณเป็นผู้ช่วยเขียนเอกสารฝึกงานของนักศึกษามหาวิทยาลัยในประเทศไทย
- เขียนเป็นภาษาไทยแบบทางการ กระชับ อ่านง่าย คำศัพท์เทคนิคใช้ภาษาอังกฤษได้
- ใช้เฉพาะข้อเท็จจริงจากข้อมูลที่ให้มา ห้ามแต่งงาน ตัวเลข ชื่อระบบ หรือผลลัพธ์ที่ไม่มีในข้อมูล
- ห้ามใส่ข้อมูลลับของบริษัท รหัสผ่าน หรือข้อมูลส่วนบุคคลของผู้อื่น
- ตอบเฉพาะเนื้อหาที่ขอ ไม่ต้องมีคำนำ คำอธิบาย หรือหัวข้อ Markdown`

// Writing guide distilled from the faculty's sample internship report (no personal data).
//
//go:embed templates/report_guide.md
var reportGuideRaw string

// reportGuide returns the general style rules and the guidance for one report section key.
func reportGuide(key string) (general, section string) {
	for _, block := range strings.Split(reportGuideRaw, "\n## ")[1:] {
		name, body, _ := strings.Cut(block, "\n")
		switch strings.TrimSpace(name) {
		case "general":
			general = strings.TrimSpace(body)
		case key:
			section = strings.TrimSpace(body)
		}
	}
	return general, section
}

func env(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func aiProvider() string {
	switch strings.ToLower(env("AI_PROVIDER")) {
	case "typhoon", "openai":
		return "openai"
	default:
		return "ollama"
	}
}

func aiKey() string {
	if aiProvider() == "openai" {
		return env("AI_API_KEY")
	}
	if k := env("OLLAMA_API_KEY"); k != "" {
		return k
	}
	return env("AI_API_KEY")
}

func aiEnabled() bool { return aiKey() != "" }

func aiHost() string {
	if aiProvider() == "openai" {
		if h := strings.TrimRight(env("AI_BASE_URL"), "/"); h != "" {
			return h
		}
		return "https://api.opentyphoon.ai/v1"
	}
	if h := strings.TrimRight(env("OLLAMA_HOST"), "/"); h != "" {
		return h
	}
	return "https://ollama.com"
}

func aiModel() string {
	if aiProvider() == "openai" {
		if m := env("AI_MODEL"); m != "" {
			return m
		}
		return "typhoon-v2.5-30b-a3b-instruct"
	}
	if m := env("OLLAMA_MODEL"); m != "" {
		return m
	}
	if m := env("AI_MODEL"); m != "" {
		return m
	}
	return "kimi-k2.6"
}

var aiClient = &http.Client{Timeout: 120 * time.Second}

// aiChat sends one system+user turn to the configured provider and returns the reply text.
func aiChat(ctx context.Context, user string) (string, error) {
	messages := []map[string]string{{"role": "system", "content": aiSystemPrompt}, {"role": "user", "content": user}}
	var url string
	var payload map[string]any
	if aiProvider() == "openai" {
		url = aiHost() + "/chat/completions"
		payload = map[string]any{"model": aiModel(), "messages": messages, "temperature": 0.4, "max_tokens": 2048, "stream": false}
	} else {
		url = aiHost() + "/api/chat"
		payload = map[string]any{"model": aiModel(), "messages": messages, "stream": false, "options": map[string]any{"temperature": 0.4}}
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+aiKey())
	resp, err := aiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI ตอบกลับ %d: %s", resp.StatusCode, apiErrorMessage(raw))
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("อ่านคำตอบจาก AI ไม่ได้")
	}
	content := out.Message.Content
	if len(out.Choices) > 0 {
		content = out.Choices[0].Message.Content
	}
	text := strings.TrimSpace(stripThinking(content))
	if text == "" {
		return "", fmt.Errorf("AI ไม่ได้ส่งข้อความกลับมา")
	}
	return text, nil
}

// apiErrorMessage pulls a readable message out of {"error":"..."} or {"error":{"message":"..."}}.
func apiErrorMessage(raw []byte) string {
	var e struct {
		Error json.RawMessage `json:"error"`
	}
	msg := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &e) == nil && len(e.Error) > 0 {
		var asString string
		var asObject struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(e.Error, &asString) == nil && asString != "" {
			msg = asString
		} else if json.Unmarshal(e.Error, &asObject) == nil && asObject.Message != "" {
			msg = asObject.Message
		}
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return msg
}

// stripThinking drops <think>…</think> blocks some reasoning models include in content.
func stripThinking(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "</think>")
		if end < 0 {
			return s[:start]
		}
		s = s[:start] + s[start+end+len("</think>"):]
	}
}

// Simple per-user limiter so one account can't burn through the API key.
var aiLimiter = struct {
	sync.Mutex
	hits map[int64][]time.Time
}{hits: map[int64][]time.Time{}}

func aiAllow(userID int64) bool {
	const limit, window = 30, 10 * time.Minute
	aiLimiter.Lock()
	defer aiLimiter.Unlock()
	now := time.Now()
	kept := aiLimiter.hits[userID][:0]
	for _, t := range aiLimiter.hits[userID] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		aiLimiter.hits[userID] = kept
		return false
	}
	aiLimiter.hits[userID] = append(kept, now)
	return true
}

func aiStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"enabled": aiEnabled(), "provider": aiProvider(), "model": aiModel()})
}

// aiGuard checks the feature is on and the caller is a student under the rate limit.
func aiGuard(c *fiber.Ctx) (User, *Internship, error) {
	u := current(c)
	if !aiEnabled() {
		return u, nil, fail(c, 503, "ยังไม่ได้ตั้งค่า AI (AI_API_KEY)")
	}
	if u.Role != "student" {
		return u, nil, fail(c, 403, "เฉพาะนักศึกษา")
	}
	if !aiAllow(u.ID) {
		return u, nil, fail(c, 429, "ใช้ AI บ่อยเกินไป กรุณารอสักครู่")
	}
	x, err := internshipFor(u)
	if err != nil {
		return u, nil, fail(c, 404, "ไม่พบการฝึกงาน")
	}
	return u, x, nil
}

func runAI(c *fiber.Ctx, prompt string) error {
	ctx, cancel := context.WithTimeout(c.Context(), 110*time.Second)
	defer cancel()
	text, err := aiChat(ctx, prompt)
	if err != nil {
		return fail(c, 502, "AI ทำงานไม่สำเร็จ: "+err.Error())
	}
	return c.JSON(fiber.Map{"text": text})
}

func internshipContext(x *Internship) string {
	var b strings.Builder
	fmt.Fprintf(&b, "บริษัท/หน่วยงาน: %s\nแผนก: %s\nตำแหน่งฝึกงาน: %s\nช่วงฝึกงาน: %s ถึง %s\nสาขาวิชา: %s\n",
		dash(x.Company), dash(x.Department), dash(x.Position), dash(x.StartDate), dash(x.EndDate), dash(x.Major))
	return b.String()
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}

// logsDigest renders logs (oldest first) as compact lines, capped at maxBytes.
func logsDigest(internshipID int64, from, to string, maxBytes int) (string, int, error) {
	q := `SELECT date,kind,work,blockers,learning FROM day_logs WHERE internship_id=?`
	args := []any{internshipID}
	if from != "" && to != "" {
		q += ` AND date BETWEEN ? AND ?`
		args = append(args, from, to)
	}
	rows, err := db.Query(q+` ORDER BY date`, args...)
	if err != nil {
		return "", 0, err
	}
	defer rows.Close()
	kinds := map[string]string{"work": "ทำงาน", "sick": "ลาป่วย", "personal": "ลากิจ", "holiday": "วันหยุด"}
	var lines []string
	for rows.Next() {
		var date, kind, work, blockers, learning string
		if err = rows.Scan(&date, &kind, &work, &blockers, &learning); err != nil {
			return "", 0, err
		}
		line := date + " [" + kinds[kind] + "]"
		if strings.TrimSpace(work) != "" {
			line += " งาน: " + oneLine(work)
		}
		if strings.TrimSpace(blockers) != "" {
			line += " | ปัญหา: " + oneLine(blockers)
		}
		if strings.TrimSpace(learning) != "" {
			line += " | เรียนรู้: " + oneLine(learning)
		}
		lines = append(lines, line)
	}
	if err = rows.Err(); err != nil {
		return "", 0, err
	}
	count := len(lines)
	// Keep the most recent entries if the digest is too long.
	total := 0
	start := len(lines)
	for start > 0 && total+len(lines[start-1])+1 <= maxBytes {
		start--
		total += len(lines[start]) + 1
	}
	digest := strings.Join(lines[start:], "\n")
	if start > 0 {
		digest = fmt.Sprintf("(ตัดบันทึกเก่า %d วันออกเพื่อความยาว)\n", start) + digest
	}
	return digest, count, nil
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

var aiFieldLabels = map[string]string{"work": "งานที่ทำ", "blockers": "ปัญหาและอุปสรรค", "learning": "สิ่งที่ได้เรียนรู้"}

// POST /api/ai/polish {field, text, date}
func aiPolish(c *fiber.Ctx) error {
	_, x, err := aiGuard(c)
	if x == nil {
		return err
	}
	var in struct {
		Field string `json:"field"`
		Text  string `json:"text"`
		Date  string `json:"date"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	label, ok := aiFieldLabels[in.Field]
	if !ok {
		return fail(c, 400, "ช่องข้อมูลไม่ถูกต้อง")
	}
	if strings.TrimSpace(in.Text) == "" {
		return fail(c, 400, "พิมพ์โน้ตสั้นๆ ก่อน แล้วให้ AI ช่วยเกลา")
	}
	if len(in.Text) > 4000 {
		return fail(c, 400, "ข้อความยาวเกินไป")
	}
	prompt := fmt.Sprintf(`เกลาข้อความในช่อง "%s" ของบันทึกการฝึกงานประจำวันที่ %s ให้เป็นภาษาทางการ
- คงความหมายและข้อเท็จจริงเดิมทั้งหมด ห้ามเพิ่มงานที่ไม่ได้เขียนไว้
- ถ้ามีหลายเรื่อง ให้แยกเป็นข้อ ขึ้นต้นแต่ละข้อด้วย "- "
- ความยาวใกล้เคียงต้นฉบับ ไม่เกิน 6 ข้อ

ข้อมูลการฝึกงาน:
%s
ข้อความต้นฉบับ:
%s`, label, dash(in.Date), internshipContext(x), in.Text)
	return runAI(c, prompt)
}

// POST /api/ai/report {key}
func aiReport(c *fiber.Ctx) error {
	_, x, err := aiGuard(c)
	if x == nil {
		return err
	}
	var in struct {
		Key string `json:"key"`
	}
	if c.BodyParser(&in) != nil {
		return fail(c, 400, "ข้อมูลไม่ถูกต้อง")
	}
	var chapter, label string
	for _, ch := range reportChapters {
		for _, f := range ch.Fields {
			if f.Key == in.Key {
				chapter, label = ch.Title, f.Label
			}
		}
	}
	if label == "" {
		return fail(c, 400, "ไม่พบหัวข้อรายงาน")
	}
	digest, count, e := logsDigest(x.ID, "", "", 14000)
	if e != nil {
		return fail(c, 500, "อ่านบันทึกไม่สำเร็จ")
	}
	if count == 0 {
		return fail(c, 400, "ยังไม่มีบันทึกรายวันให้ AI ใช้ร่างรายงาน")
	}
	var others []string
	for _, ch := range reportChapters {
		for _, f := range ch.Fields {
			if v := strings.TrimSpace(x.Report[f.Key]); v != "" && f.Key != in.Key {
				if len(v) > 600 {
					v = v[:600] + "…"
				}
				others = append(others, f.Label+": "+oneLine(v))
			}
		}
	}
	existing := strings.TrimSpace(x.Report[in.Key])
	general, guide := reportGuide(in.Key)
	prompt := fmt.Sprintf(`ร่างเนื้อหารายงานผลการฝึกงาน หัวข้อ "%s" (อยู่ใน %s)
- ทำตามแนวการเขียนของคณะด้านล่าง (สรุปจากรายงานตัวอย่าง) ทั้งรูปแบบ ความยาว และสำนวน
- อ้างอิงจากบันทึกรายวันจริงด้านล่างเท่านั้น ถ้าข้อมูลไม่พอสำหรับบางส่วน ให้เขียนเท่าที่มีและใส่ [เติมข้อมูล: ...] ตรงที่นักศึกษาต้องเติมเอง
- ไม่ต้องใส่ชื่อหัวข้อซ้ำ

แนวการเขียนทั้งเล่ม:
%s

แนวการเขียนของหัวข้อนี้:
%s

ข้อมูลการฝึกงาน:
%s
เนื้อหาที่เขียนไว้แล้วในหัวข้ออื่น:
%s

ร่างเดิมของหัวข้อนี้ (ถ้ามี ให้ปรับปรุงต่อจากนี้):
%s

บันทึกรายวัน (%d วัน):
%s`, label, chapter, dash(general), dash(guide), internshipContext(x), dash(strings.Join(others, "\n")), dash(existing), count, digest)
	return runAI(c, prompt)
}

// POST /api/ai/weekly {from, to}
func aiWeekly(c *fiber.Ctx) error {
	_, x, err := aiGuard(c)
	if x == nil {
		return err
	}
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if c.BodyParser(&in) != nil || !validISODate(in.From) || !validISODate(in.To) || in.From > in.To {
		return fail(c, 400, "ช่วงวันที่ไม่ถูกต้อง")
	}
	digest, count, e := logsDigest(x.ID, in.From, in.To, 10000)
	if e != nil {
		return fail(c, 500, "อ่านบันทึกไม่สำเร็จ")
	}
	if count == 0 {
		return fail(c, 400, "ไม่มีบันทึกในช่วงนี้")
	}
	prompt := fmt.Sprintf(`สรุปงานฝึกงานช่วง %s ถึง %s สำหรับรายงานความคืบหน้ารายสัปดาห์ (action plan / weekly report)
ใช้รูปแบบนี้เท่านั้น:
งานที่ทำ:
- ...
ปัญหาและวิธีแก้:
- ... (ถ้าไม่มี เขียน "- ไม่มี")
สิ่งที่ได้เรียนรู้:
- ...
แผนสัปดาห์ถัดไป:
- [เติมข้อมูล: แผนงานถัดไป]

ข้อมูลการฝึกงาน:
%s
บันทึกรายวัน (%d วัน):
%s`, in.From, in.To, internshipContext(x), count, digest)
	return runAI(c, prompt)
}

func validISODate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
