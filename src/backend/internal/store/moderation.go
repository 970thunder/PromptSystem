// Package store 提供社区内容的领域校验与持久化抽象。
package store

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type moderationField struct {
	Name     string
	Value    string
	MaxRunes int
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

var (
	// ErrInvalidContent is returned for malformed UTF-8 or disallowed control
	// characters in user-authored content.
	ErrInvalidContent = errors.New("invalid content")
	// ErrContentTooLong is returned before a value reaches a database column or
	// an expensive downstream parser.
	ErrContentTooLong = errors.New("content is too long")
	// ErrUnsafeContent is returned when content contains executable markup or a
	// known dangerous URL/event-handler pattern.
	ErrUnsafeContent = errors.New("unsafe content")
)

const (
	MaxUsernameRunes  = 39
	MaxBioRunes       = 500
	MaxPromptTitle    = 200
	MaxPromptDesc     = 1000
	MaxPromptContent  = 65535
	MaxSystemPrompt   = 2000
	MaxPromptModel    = 50
	MaxCommentContent = 1000
)

func ValidatePromptModeration(input CreatePromptInput) error {
	return validateModerationFields([]moderationField{
		{Name: "标题", Value: input.Title, MaxRunes: MaxPromptTitle},
		{Name: "描述", Value: input.Description, MaxRunes: MaxPromptDesc},
		{Name: "提示词正文", Value: input.Content, MaxRunes: MaxPromptContent},
		{Name: "系统提示词", Value: input.SystemPrompt, MaxRunes: MaxSystemPrompt},
		{Name: "模型", Value: input.Model, MaxRunes: MaxPromptModel},
	})
}

func ValidateCommentModeration(content string) error {
	return validateModerationFields([]moderationField{
		{Name: "评论", Value: content, MaxRunes: MaxCommentContent},
	})
}

// ValidateUserProfile applies the same boundary checks to profile fields. It
// is shared by memory and MySQL stores so a caller cannot bypass API checks by
// selecting a different persistence implementation.
func ValidateUserProfile(username, bio string) error {
	return validateModerationFields([]moderationField{
		{Name: "用户名", Value: username, MaxRunes: MaxUsernameRunes},
		{Name: "简介", Value: bio, MaxRunes: MaxBioRunes},
	})
}

func validateModerationFields(fields []moderationField) error {
	for _, field := range fields {
		value := strings.TrimSpace(field.Value)
		if !utf8.ValidString(value) || containsDisallowedControl(value) {
			return fmt.Errorf("%w: %s", ErrInvalidContent, field.Name)
		}
		if field.MaxRunes > 0 && len([]rune(value)) > field.MaxRunes {
			return fmt.Errorf("%w: %s", ErrContentTooLong, field.Name)
		}

		normalized := strings.ToLower(value)
		for _, rule := range sensitiveRules {
			if strings.Contains(normalized, rule) {
				return fmt.Errorf("%w: %s", ErrUnsafeContent, field.Name)
			}
		}
		if containsExecutableMarkup(normalized) {
			return fmt.Errorf("%w: %s", ErrUnsafeContent, field.Name)
		}
	}

	return nil
}

func containsDisallowedControl(value string) bool {
	for _, r := range value {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return true
		}
	}
	return false
}

func containsExecutableMarkup(value string) bool {
	// Keep Markdown and ordinary text usable, but reject the forms that can
	// execute script when rendered by a permissive client.
	unsafeFragments := []string{
		"<script", "</script", "<iframe", "<object", "<embed", "data:text/html",
		"javascript:", "vbscript:",
	}
	for _, fragment := range unsafeFragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}

	// Detect HTML event attributes without enumerating every possible event
	// name (onclick, onload, onanimationstart, ...).
	for index := 0; index+3 < len(value); index++ {
		if value[index] != 'o' || value[index+1] != 'n' {
			continue
		}
		end := index + 2
		for end < len(value) && ((value[end] >= 'a' && value[end] <= 'z') || value[end] == '-' || value[end] == '_') {
			end++
		}
		for end < len(value) && (value[end] == ' ' || value[end] == '\t' || value[end] == '\n' || value[end] == '\r') {
			end++
		}
		if end < len(value) && value[end] == '=' {
			return true
		}
	}
	return false
}
