// Package store 测试社区内容审核规则，防止发布链路绕过基础安全校验。
package store

import "testing"

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
}
