package store

import "testing"

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
