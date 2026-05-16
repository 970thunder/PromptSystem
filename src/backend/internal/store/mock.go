package store

import "sort"

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

func Categories() []Category {
	return append([]Category(nil), categories...)
}

func FilterPrompts(categoryID int, sortBy string) []Prompt {
	list := make([]Prompt, 0, len(prompts))
	for _, prompt := range prompts {
		if categoryID == 0 || prompt.CategoryID == categoryID {
			list = append(list, prompt)
		}
	}

	switch sortBy {
	case "popular":
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Likes > list[j].Likes
		})
	default:
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].CreatedAt > list[j].CreatedAt
		})
	}

	return list
}

func FindPromptByID(id int) (Prompt, bool) {
	for _, prompt := range prompts {
		if prompt.ID == id {
			return prompt, true
		}
	}

	return Prompt{}, false
}

var categories = []Category{
	{ID: 1, Name: "图片生成", Icon: "image", Count: 128},
	{ID: 2, Name: "文案写作", Icon: "edit", Count: 94},
	{ID: 3, Name: "编程开发", Icon: "code", Count: 76},
	{ID: 4, Name: "视频生成", Icon: "video", Count: 51},
	{ID: 5, Name: "Agent Prompt", Icon: "robot", Count: 33},
	{ID: 6, Name: "工作流", Icon: "workflow", Count: 27},
}

var prompts = []Prompt{
	{
		ID:           101,
		Title:        "品牌海报生成器",
		Description:  "把一句品牌口号扩展成一套可直接用于 Midjourney 和 SDXL 的高质量视觉提示词。",
		Cover:        "Aurora campaign board",
		Content:      "You are a senior visual director. Turn the input slogan into cinematic prompt variants...",
		SystemPrompt: "Act as a multidisciplinary art director and prompt engineer.",
		Model:        "Midjourney v6",
		Params:       PromptParams{Temperature: 0.7, TopP: 0.9, MaxTokens: 1200},
		CategoryID:   1,
		CategoryName: "图片生成",
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
		ID:           102,
		Title:        "SaaS 落地页文案重写",
		Description:  "输入产品定位和竞品特点，输出首页标题、副标题、功能卖点和 CTA 版本。",
		Cover:        "Copywriting desk",
		Content:      "Rewrite the landing page copy for a B2B SaaS company with a confident tone...",
		SystemPrompt: "You are a conversion-focused SaaS copywriter.",
		Model:        "GPT-4.1",
		Params:       PromptParams{Temperature: 0.5, TopP: 0.85, MaxTokens: 1600},
		CategoryID:   2,
		CategoryName: "文案写作",
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
		ID:           103,
		Title:        "代码评审助手",
		Description:  "针对 PR 变更输出风险优先级、回归点和测试建议，适合前后端日常 review。",
		Cover:        "Review terminal",
		Content:      "Review the code diff and prioritize bugs, regressions, and missing tests...",
		SystemPrompt: "You are a meticulous senior engineer who reviews for correctness first.",
		Model:        "GPT-4.1",
		Params:       PromptParams{Temperature: 0.3, TopP: 0.8, MaxTokens: 1400},
		CategoryID:   3,
		CategoryName: "编程开发",
		Tags:         []string{"Code Review", "工程效率", "PR"},
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
		Title:        "短视频脚本工厂",
		Description:  "从产品卖点自动生成 15 秒、30 秒和 60 秒短视频脚本，附镜头节奏建议。",
		Cover:        "Studio storyboard",
		Content:      "Generate three short-form video scripts with hooks, scene pacing, and voiceover...",
		SystemPrompt: "You are a short-form video creative strategist.",
		Model:        "Claude 3.7 Sonnet",
		Params:       PromptParams{Temperature: 0.8, TopP: 0.9, MaxTokens: 1800},
		CategoryID:   4,
		CategoryName: "视频生成",
		Tags:         []string{"短视频", "脚本", "种草"},
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
		Title:        "多 Agent 研究协调器",
		Description:  "把研究目标拆成并行子任务，生成 agent 分工、同步节奏和整合模板。",
		Cover:        "Coordination map",
		Content:      "Break down the research goal into parallel agents with a shared reporting contract...",
		SystemPrompt: "You are an operations-minded AI orchestration designer.",
		Model:        "o3",
		Params:       PromptParams{Temperature: 0.4, TopP: 0.9, MaxTokens: 1700},
		CategoryID:   5,
		CategoryName: "Agent Prompt",
		Tags:         []string{"Agent", "研究", "协作"},
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
		Title:        "客服 SOP 生成工作流",
		Description:  "从产品资料、FAQ 和语气规范自动生成客服回复模板与升级规则。",
		Cover:        "Support workflow board",
		Content:      "Create a customer support workflow with escalation rules and reusable reply templates...",
		SystemPrompt: "You are a CX operations architect.",
		Model:        "GPT-4o",
		Params:       PromptParams{Temperature: 0.45, TopP: 0.88, MaxTokens: 1500},
		CategoryID:   6,
		CategoryName: "工作流",
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
