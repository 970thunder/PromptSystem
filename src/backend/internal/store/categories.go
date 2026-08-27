package store

// Image-focused prompt categories (Chinese labels for MVP publish flow).
var categories = []Category{
	{ID: 1, Name: "摄影", Icon: "camera", Count: 128},
	{ID: 2, Name: "插画", Icon: "brush", Count: 96},
	{ID: 3, Name: "3D", Icon: "cube", Count: 84},
	{ID: 4, Name: "电商", Icon: "shopping", Count: 112},
	{ID: 5, Name: "人像", Icon: "portrait", Count: 102},
	{ID: 6, Name: "建筑", Icon: "building", Count: 67},
	{ID: 7, Name: "动漫", Icon: "anime", Count: 91},
	{ID: 8, Name: "UI", Icon: "layout", Count: 73},
	{ID: 9, Name: "海报", Icon: "poster", Count: 88},
	{ID: 10, Name: "产品", Icon: "product", Count: 79},
	{ID: 11, Name: "风景", Icon: "landscape", Count: 95},
	{ID: 12, Name: "美食", Icon: "food", Count: 61},
	{ID: 13, Name: "时尚", Icon: "fashion", Count: 58},
	{ID: 14, Name: "游戏", Icon: "game", Count: 70},
	{ID: 15, Name: "图标", Icon: "icon", Count: 54},
	{ID: 16, Name: "LOGO", Icon: "logo", Count: 66},
	{ID: 17, Name: "室内设计", Icon: "interior", Count: 49},
	{ID: 18, Name: "汽车", Icon: "car", Count: 44},
	{ID: 19, Name: "宠物", Icon: "pet", Count: 52},
	{ID: 20, Name: "婚礼", Icon: "wedding", Count: 47},
	{ID: 21, Name: "科幻", Icon: "scifi", Count: 63},
	{ID: 22, Name: "水彩", Icon: "watercolor", Count: 41},
	{ID: 23, Name: "油画", Icon: "oil", Count: 38},
	{ID: 24, Name: "像素", Icon: "pixel", Count: 35},
	{ID: 25, Name: "线稿", Icon: "lineart", Count: 42},
	{ID: 26, Name: "表情包", Icon: "emoji", Count: 56},
	{ID: 27, Name: "壁纸", Icon: "wallpaper", Count: 89},
	{ID: 28, Name: "社交媒体", Icon: "social", Count: 76},
	{ID: 29, Name: "广告创意", Icon: "ads", Count: 82},
	{ID: 30, Name: "其他", Icon: "more", Count: 33},
}

const MaxPromptTags = 30

// MaxPromptTagLength is the maximum length (in runes) of a single tag. It must
// match the prompt_tags.tag column width (VARCHAR(50)) so persisted tags never
// exceed the column and never collide with the semantics of the unique index.
const MaxPromptTagLength = 50
