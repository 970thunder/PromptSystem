package store

type User struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Bio        string `json:"bio"`
	Level      int    `json:"level"`
	Experience int    `json:"experience"`
	Status     int    `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

type PromptParams struct {
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"topP"`
	MaxTokens   int     `json:"maxTokens"`
}

type Prompt struct {
	ID           int          `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Cover        string       `json:"cover"`
	Images       []string     `json:"images"`
	Content      string       `json:"content"`
	SystemPrompt string       `json:"systemPrompt"`
	Model        string       `json:"model"`
	Params       PromptParams `json:"params"`
	CategoryID   int          `json:"categoryId"`
	CategoryName string       `json:"categoryName"`
	Tags         []string     `json:"tags"`
	UserID       int          `json:"userId"`
	User         User         `json:"user"`
	Views        int          `json:"views"`
	Likes        int          `json:"likes"`
	Favorites    int          `json:"favorites"`
	Status       int          `json:"status"`
	CreatedAt    string       `json:"createdAt"`
	UpdatedAt    string       `json:"updatedAt"`
}

type Category struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Count int    `json:"count"`
}

type Comment struct {
	ID         int       `json:"id"`
	TargetType string    `json:"targetType"`
	TargetID   int       `json:"targetId"`
	UserID     int       `json:"userId"`
	User       User      `json:"user"`
	Content    string    `json:"content"`
	Likes      int       `json:"likes"`
	ParentID   *int      `json:"parentId"`
	Replies    []Comment `json:"replies"`
	CreatedAt  string    `json:"createdAt"`
}

type Report struct {
	ID         int    `json:"id"`
	UserID     int    `json:"userId"`
	TargetType string `json:"targetType"`
	TargetID   int    `json:"targetId"`
	Reason     string `json:"reason"`
	Detail     string `json:"detail"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

func Categories() []Category {
	return append([]Category(nil), categories...)
}

var prompts = []Prompt{
	{
		ID:          101,
		Title:       "Brand Poster Prompt Builder",
		Description: "Turn a short slogan into polished Midjourney and SDXL prompt variants for campaign visuals and ecommerce launches.",
		Cover:       "Aurora campaign board",
		Images: []string{
			"https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80",
			"https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80",
		},
		Content:      "You are a senior visual director. Turn the input slogan into cinematic prompt variants...",
		SystemPrompt: "Act as a multidisciplinary art director and prompt engineer.",
		Model:        "Midjourney v6",
		Params:       PromptParams{Temperature: 0.7, TopP: 0.9, MaxTokens: 1200},
		CategoryID:   1,
		CategoryName: "摄影",
		Tags:         []string{"品牌", "海报", "电商"},
		UserID:       1,
		User: User{
			ID:         1,
			Username:   "Astra Lab",
			Avatar:     "",
			Email:      "astra@example.com",
			Bio:        "Visual prompt designer",
			Level:      8,
			Experience: 1320,
			Status:     1,
			CreatedAt:  "2026-05-01",
		},
		Views:     12430,
		Likes:     893,
		Favorites: 516,
		Status:    1,
		CreatedAt: "2026-05-10",
		UpdatedAt: "2026-05-15",
	},
	{
		ID:          102,
		Title:       "SaaS Landing Page Copy Rewrite",
		Description: "Input product positioning and competitor context, then generate homepage headlines, supporting copy, feature points, and CTA options.",
		Cover:       "Copywriting desk",
		Images: []string{
			"https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80",
		},
		Content:      "Rewrite the landing page copy for a B2B SaaS company with a confident tone...",
		SystemPrompt: "You are a conversion-focused SaaS copywriter.",
		Model:        "GPT-4.1",
		Params:       PromptParams{Temperature: 0.5, TopP: 0.85, MaxTokens: 1600},
		CategoryID:   9,
		CategoryName: "海报",
		Tags:         []string{"SaaS", "转化", "营销"},
		UserID:       2,
		User: User{
			ID:         2,
			Username:   "Nora Chen",
			Avatar:     "",
			Email:      "nora@example.com",
			Bio:        "Growth copywriter",
			Level:      6,
			Experience: 910,
			Status:     1,
			CreatedAt:  "2026-04-20",
		},
		Views:     9780,
		Likes:     621,
		Favorites: 344,
		Status:    1,
		CreatedAt: "2026-05-11",
		UpdatedAt: "2026-05-15",
	},
	{
		ID:          103,
		Title:       "Code Review Assistant",
		Description: "Review a PR diff, prioritize risks, call out regressions, and suggest targeted follow-up tests for engineering teams.",
		Cover:       "Review terminal",
		Images: []string{
			"https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80",
		},
		Content:      "Review the code diff and prioritize bugs, regressions, and missing tests...",
		SystemPrompt: "You are a meticulous senior engineer who reviews for correctness first.",
		Model:        "GPT-4.1",
		Params:       PromptParams{Temperature: 0.3, TopP: 0.8, MaxTokens: 1400},
		CategoryID:   30,
		CategoryName: "其他",
		Tags:         []string{"代码审查", "工程", "PR"},
		UserID:       3,
		User: User{
			ID:         3,
			Username:   "Delta Forge",
			Avatar:     "",
			Email:      "delta@example.com",
			Bio:        "Dev tooling collective",
			Level:      9,
			Experience: 1680,
			Status:     1,
			CreatedAt:  "2026-03-18",
		},
		Views:     15120,
		Likes:     1012,
		Favorites: 673,
		Status:    1,
		CreatedAt: "2026-05-08",
		UpdatedAt: "2026-05-14",
	},
	{
		ID:           104,
		Title:        "Short Video Script Factory",
		Description:  "Generate 15s, 30s, and 60s short-form video scripts from product selling points, complete with hook and pacing guidance.",
		Cover:        "Studio storyboard",
		Content:      "Generate three short-form video scripts with hooks, scene pacing, and voiceover...",
		SystemPrompt: "You are a short-form video creative strategist.",
		Model:        "Claude 3.7 Sonnet",
		Params:       PromptParams{Temperature: 0.8, TopP: 0.9, MaxTokens: 1800},
		CategoryID:   28,
		CategoryName: "社交媒体",
		Tags:         []string{"短视频", "脚本", "增长"},
		UserID:       4,
		User: User{
			ID:         4,
			Username:   "Mica Studio",
			Avatar:     "",
			Email:      "mica@example.com",
			Bio:        "Video growth team",
			Level:      5,
			Experience: 740,
			Status:     1,
			CreatedAt:  "2026-04-02",
		},
		Views:     8420,
		Likes:     587,
		Favorites: 298,
		Status:    1,
		CreatedAt: "2026-05-09",
		UpdatedAt: "2026-05-13",
	},
	{
		ID:           105,
		Title:        "Multi-Agent Research Coordinator",
		Description:  "Break a research goal into parallel agent tasks, reporting contracts, and a practical merge plan for final synthesis.",
		Cover:        "Coordination map",
		Content:      "Break down the research goal into parallel agents with a shared reporting contract...",
		SystemPrompt: "You are an operations-minded AI orchestration designer.",
		Model:        "o3",
		Params:       PromptParams{Temperature: 0.4, TopP: 0.9, MaxTokens: 1700},
		CategoryID:   30,
		CategoryName: "其他",
		Tags:         []string{"智能体", "研究", "协作"},
		UserID:       5,
		User: User{
			ID:         5,
			Username:   "Ops Lantern",
			Avatar:     "",
			Email:      "ops@example.com",
			Bio:        "AI workflow operator",
			Level:      7,
			Experience: 1110,
			Status:     1,
			CreatedAt:  "2026-02-28",
		},
		Views:     6320,
		Likes:     441,
		Favorites: 257,
		Status:    1,
		CreatedAt: "2026-05-12",
		UpdatedAt: "2026-05-15",
	},
	{
		ID:           106,
		Title:        "Customer Support SOP Workflow",
		Description:  "Generate customer support flows, escalation rules, and reusable response templates from docs, FAQs, and tone guidance.",
		Cover:        "Support workflow board",
		Content:      "Create a customer support workflow with escalation rules and reusable reply templates...",
		SystemPrompt: "You are a CX operations architect.",
		Model:        "GPT-4o",
		Params:       PromptParams{Temperature: 0.45, TopP: 0.88, MaxTokens: 1500},
		CategoryID:   30,
		CategoryName: "其他",
		Tags:         []string{"客服", "SOP", "流程"},
		UserID:       6,
		User: User{
			ID:         6,
			Username:   "North Queue",
			Avatar:     "",
			Email:      "north@example.com",
			Bio:        "Service automation builder",
			Level:      4,
			Experience: 520,
			Status:     1,
			CreatedAt:  "2026-05-03",
		},
		Views:     5180,
		Likes:     322,
		Favorites: 199,
		Status:    1,
		CreatedAt: "2026-05-07",
		UpdatedAt: "2026-05-14",
	},
}

var comments = []Comment{
	{
		ID:         1001,
		TargetType: "prompt",
		TargetID:   101,
		UserID:     2,
		User: User{
			ID:         2,
			Username:   "Nora Chen",
			Avatar:     "",
			Email:      "nora@example.com",
			Bio:        "Growth copywriter",
			Level:      6,
			Experience: 910,
			Status:     1,
			CreatedAt:  "2026-04-20",
		},
		Content:   "The structure is solid. I would add one more constraint for product packshot composition.",
		Likes:     3,
		ParentID:  nil,
		Replies:   []Comment{},
		CreatedAt: "2026-06-07T10:30:00Z",
	},
	{
		ID:         1002,
		TargetType: "prompt",
		TargetID:   101,
		UserID:     1,
		User: User{
			ID:         1,
			Username:   "Astra Lab",
			Avatar:     "",
			Email:      "astra@example.com",
			Bio:        "Visual prompt designer",
			Level:      8,
			Experience: 1320,
			Status:     1,
			CreatedAt:  "2026-05-01",
		},
		Content:   "Agreed. I usually append packaging angle and lens language in the final line.",
		Likes:     1,
		ParentID:  intPtr(1001),
		Replies:   []Comment{},
		CreatedAt: "2026-06-07T12:00:00Z",
	},
}

func intPtr(value int) *int {
	return &value
}
