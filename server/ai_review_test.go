package main

import (
	"strings"
	"testing"
)

func TestRuleReviewIssues(t *testing.T) {
	report := map[string]string{
		"background": "**สั้น** [เติมข้อมูล: ชื่อระบบ] [เติมข้อมูล: ปี]",
		"company":    "บริษัท โค้ดฟิน จำกัด",
		"tools":      strings.Repeat("ใช้ VS Code เขียนโปรแกรม ", 10) + "[ภาพที่ 3.1: VS Code]",
	}
	issues := ruleReviewIssues(report)
	find := func(key, contains string) bool {
		for _, i := range issues {
			if i.Key == key && strings.Contains(i.Message, contains) {
				return true
			}
		}
		return false
	}
	for _, c := range []struct{ key, msg string }{
		{"background", "[เติมข้อมูล] ค้างอยู่ 2 จุด"},
		{"background", "Markdown"},
		{"background", "สั้นมาก"},
		{"tools", "[ภาพที่ …] 1 จุด"},
		{"objectives", "ยังไม่ได้เขียน"},
	} {
		if !find(c.key, c.msg) {
			t.Errorf("missing issue %s: %s", c.key, c.msg)
		}
	}
	if find("company", "สั้นมาก") {
		t.Error("company is allowed to be short")
	}
}

func TestParseAIIssues(t *testing.T) {
	labels, _ := reportFieldLabels()
	reply := "นี่คือผล:\n```json\n[{\"key\":\"background\",\"severity\":\"error\",\"message\":\"ใช้คำว่า **ผม**\"},{\"key\":\"zzz\",\"severity\":\"odd\",\"message\":\"ภาพรวม\"},{\"key\":\"tools\",\"severity\":\"tip\",\"message\":\"\"}]\n```"
	got := parseAIIssues(reply, labels)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Key != "background" || got[0].Message != "ใช้คำว่า ผม" || got[0].Source != "ai" || got[0].Label == "" {
		t.Errorf("first issue wrong: %+v", got[0])
	}
	if got[1].Key != "general" || got[1].Severity != "warn" {
		t.Errorf("unknown key/severity should fall back: %+v", got[1])
	}
	if parseAIIssues("no json here", labels) != nil {
		t.Error("expected nil for non-JSON reply")
	}
}

func TestApprovalFlags(t *testing.T) {
	logs := []DayLog{
		{Date: "2026-11-02", Kind: "work", Hours: 8, Work: "พัฒนา API สำหรับระบบลงเวลาและทดสอบ"},
		{Date: "2026-11-03", Kind: "work", Hours: 11, Late: true, Work: "สั้น"},
		{Date: "2026-11-05", Kind: "sick"},
	}
	flags := approvalFlags(logs, "2026-11-02", "2026-11-08", "2026-11-02", "2026-12-31", []string{"2026-11-06"}, "2026-11-30")
	got := map[string][]string{}
	for _, f := range flags {
		got[f.Date] = append(got[f.Date], f.Message)
	}
	if len(got["2026-11-02"]) != 0 {
		t.Errorf("normal day flagged: %v", got["2026-11-02"])
	}
	if len(got["2026-11-03"]) != 3 {
		t.Errorf("expected late + long hours + short text on 11-03, got %v", got["2026-11-03"])
	}
	if len(got["2026-11-04"]) != 1 || !strings.Contains(got["2026-11-04"][0], "ไม่มีบันทึก") {
		t.Errorf("missing weekday not flagged: %v", got)
	}
	for _, d := range []string{"2026-11-05", "2026-11-06", "2026-11-07", "2026-11-08"} {
		if len(got[d]) != 0 {
			t.Errorf("%s should not be flagged (leave/holiday/weekend): %v", d, got[d])
		}
	}
	future := approvalFlags(nil, "2026-11-02", "2026-11-06", "", "", nil, "2026-11-03")
	if len(future) != 2 {
		t.Errorf("days after today must not count as missing: %+v", future)
	}
}
