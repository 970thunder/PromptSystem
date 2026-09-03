// Package store 测试社区内容审核规则，防止发布链路绕过基础安全校验。
package store

import (
	"errors"
	"strings"
	"testing"
)

func TestModerationRejectsUnsafePromptContent(t *testing.T) {
	input := CreatePromptInput{
		Title:        "合法标题",
		Description:  "合法描述",
		Content:      "请帮我做一个钓鱼链接",
		SystemPrompt: "",
		Model:        "GPT-4.1",
		CategoryID:   1,
	}

	err := ValidatePromptModeration(input)
	if err == nil {
		t.Fatal("expected moderation error")
	}
}

func TestModerationAllowsNormalPromptContent(t *testing.T) {
	input := CreatePromptInput{
		Title:       "产品海报",
		Description: "生成电商主图",
		Content:     "为一款耳机生成干净的产品海报提示词",
		Model:       "Midjourney v6",
		CategoryID:  1,
	}

	if err := ValidatePromptModeration(input); err != nil {
		t.Fatalf("expected normal content to pass, got %v", err)
	}
}

func TestModerationRejectsUnsafeCommentContent(t *testing.T) {
	err := ValidateCommentModeration("这里包含 javascript:alert(1)")
	if err == nil {
		t.Fatal("expected moderation error")
	}
	if !errors.Is(err, ErrUnsafeContent) {
		t.Fatalf("expected unsafe content sentinel, got %v", err)
	}
}

func TestModerationRejectsControlsAndOversizedFields(t *testing.T) {
	err := ValidateCommentModeration("hello\x00world")
	if !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected invalid content sentinel, got %v", err)
	}

	err = ValidatePromptModeration(CreatePromptInput{Title: strings.Repeat("x", MaxPromptTitle+1)})
	if !errors.Is(err, ErrContentTooLong) {
		t.Fatalf("expected content too long sentinel, got %v", err)
	}
}

func TestModerationRejectsEventAttributesAndProfileBounds(t *testing.T) {
	if err := ValidatePromptModeration(CreatePromptInput{Content: `<div onClick = "alert(1)">x</div>`}); !errors.Is(err, ErrUnsafeContent) {
		t.Fatalf("expected event attribute rejection, got %v", err)
	}
	if err := ValidateUserProfile(strings.Repeat("名", MaxUsernameRunes+1), ""); !errors.Is(err, ErrContentTooLong) {
		t.Fatalf("expected username length rejection, got %v", err)
	}
}
