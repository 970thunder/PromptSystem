// Package store 提供社区内容的领域校验与持久化抽象。
package store

import (
	"fmt"
	"strings"
)

type moderationField struct {
	Name  string
	Value string
}

var sensitiveRules = []string{
	"<script",
	"javascript:",
	"onerror=",
	"api_key",
	"begin private key",
	"钓鱼链接",
	"盗号",
}

func ValidatePromptModeration(input CreatePromptInput) error {
	return validateModerationFields([]moderationField{
		{Name: "标题", Value: input.Title},
		{Name: "描述", Value: input.Description},
		{Name: "提示词正文", Value: input.Content},
		{Name: "系统提示词", Value: input.SystemPrompt},
		{Name: "模型", Value: input.Model},
	})
}

func ValidateCommentModeration(content string) error {
	return validateModerationFields([]moderationField{
		{Name: "评论", Value: content},
	})
}

func validateModerationFields(fields []moderationField) error {
	for _, field := range fields {
		normalized := strings.ToLower(strings.TrimSpace(field.Value))
		for _, rule := range sensitiveRules {
			if strings.Contains(normalized, rule) {
				return fmt.Errorf("%s包含不符合平台规范的内容", field.Name)
			}
		}
	}

	return nil
}
