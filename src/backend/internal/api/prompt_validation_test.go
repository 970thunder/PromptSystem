package api

import "testing"

// 封面改为可选：文字类提示词（写作/编程/工作流）没有“生成效果图”，
// 不应被封面必填卡在发布流程外。
func TestValidatePromptPayloadAllowsEmptyCover(t *testing.T) {
	status := 1
	payload := promptPayload{
		Title:       "周报总结提示词",
		Description: "把本周工作整理成结构化周报",
		Content:     "请根据以下要点生成一份周报…",
		Model:       "GPT-4o",
		CategoryID:  1,
		Status:      &status,
	}

	if message := validatePromptPayload(payload); message != "" {
		t.Fatalf("expected empty cover to be allowed, got validation error: %s", message)
	}
}

func TestValidatePromptPayloadStillRejectsMissingContent(t *testing.T) {
	status := 1
	payload := promptPayload{
		Title:       "无正文",
		Description: "缺少正文",
		Model:       "GPT-4o",
		CategoryID:  1,
		Status:      &status,
	}

	if message := validatePromptPayload(payload); message != "Prompt content is required" {
		t.Fatalf("expected content required error, got: %q", message)
	}
}
