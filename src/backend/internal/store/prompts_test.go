package store

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizePromptTagsCollapsesWhitespaceAndDedupes(t *testing.T) {
	tags, err := NormalizePromptTags([]string{
		"  摄影  ",
		" 摄影 ",
		" 智能体 研究 ",
		" Brand ",
		"brand",
	})
	if err != nil {
		t.Fatalf("NormalizePromptTags() error = %v", err)
	}
	want := []string{"摄影", "智能体 研究", "brand"}
	if !reflect.DeepEqual(tags, want) {
		t.Fatalf("NormalizePromptTags() = %#v, want %#v", tags, want)
	}
}

func TestNormalizePromptTagsRejectsBlankTag(t *testing.T) {
	if _, err := NormalizePromptTags([]string{"   ", "   "}); err == nil {
		t.Fatal("expected blank tag to be rejected")
	}
}

func TestNormalizePromptTagsRejectsOversizedTag(t *testing.T) {
	long := strings.Repeat("字", MaxPromptTagLength+1)
	if _, err := NormalizePromptTags([]string{long}); err == nil {
		t.Fatal("expected tag longer than MaxPromptTagLength to be rejected")
	}
}

func TestNormalizePromptTagsCapsCount(t *testing.T) {
	input := make([]string, 0, MaxPromptTags+5)
	for i := 0; i < MaxPromptTags+5; i++ {
		input = append(input, "tag")
	}
	tags, err := NormalizePromptTags(input)
	if err != nil {
		t.Fatalf("NormalizePromptTags() error = %v", err)
	}
	if len(tags) > MaxPromptTags {
		t.Fatalf("expected at most %d tags, got %d", MaxPromptTags, len(tags))
	}
}

func TestMemoryPromptRejectsBlankTag(t *testing.T) {
	_, err := CreatePrompt(CreatePromptInput{
		Title:      "Title",
		Content:    "Content",
		Model:      "gpt-4",
		CategoryID: 1,
		User:       User{ID: 1, Username: "u", Email: "u@example.com", Status: 1},
		Status:     1,
		Tags:       []string{"  "},
	})
	if err == nil {
		t.Fatal("expected blank tag to be rejected on create")
	}
}

func TestMemoryPromptDraftLifecycle(t *testing.T) {
	prompt, err := CreatePrompt(CreatePromptInput{
		Title:      "Draft title",
		Content:    "Draft content",
		Model:      "Midjourney v6",
		CategoryID: 1,
		User: User{
			ID:       902,
			Username: "Draft Owner",
			Email:    "draft-owner@example.com",
			Status:   1,
		},
		Status: 0,
	})
	if err != nil {
		t.Fatalf("CreatePrompt() draft error = %v", err)
	}

	if _, found := FindPromptByID(prompt.ID); found {
		t.Fatal("draft should not be visible through public detail lookup")
	}
	if _, found := FindOwnedPromptByID(prompt.ID, 902); !found {
		t.Fatal("draft owner should be able to load the draft")
	}

	drafts := ListUserDraftPrompts(902)
	if len(drafts) == 0 || drafts[0].ID != prompt.ID {
		t.Fatalf("expected draft in owner draft list, got %+v", drafts)
	}

	published, err := UpdatePrompt(prompt.ID, 902, CreatePromptInput{
		Title:       "Published title",
		Description: "Published description",
		Cover:       "https://example.com/cover.png",
		Content:     "Published content",
		Model:       "Midjourney v6",
		CategoryID:  1,
		User:        prompt.User,
		Status:      1,
	})
	if err != nil {
		t.Fatalf("UpdatePrompt() publish draft error = %v", err)
	}
	if published.Status != 1 {
		t.Fatalf("expected published status 1, got %d", published.Status)
	}
	if _, found := FindPromptByID(prompt.ID); !found {
		t.Fatal("published draft should be visible through public detail lookup")
	}
}

func TestMemoryPromptReportIsIdempotent(t *testing.T) {
	promptStore := NewMemoryPromptStore()

	first, applied, err := promptStore.Report(101, 9201, ReportReasonSpam, "标题疑似误导")
	if err != nil {
		t.Fatalf("Report() first error = %v", err)
	}
	if !applied {
		t.Fatal("expected first report to be applied")
	}
	if first.TargetType != "prompt" || first.TargetID != 101 || first.UserID != 9201 {
		t.Fatalf("unexpected first report: %+v", first)
	}

	second, applied, err := promptStore.Report(101, 9201, ReportReasonSpam, "重复提交")
	if err != nil {
		t.Fatalf("Report() second error = %v", err)
	}
	if applied {
		t.Fatal("expected duplicate report to be idempotent")
	}
	if second.ID != first.ID {
		t.Fatalf("expected duplicate report to reuse id %d, got %d", first.ID, second.ID)
	}
}
