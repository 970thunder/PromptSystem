// User types
export interface User {
  id: number
  username: string
  avatar: string
  email: string
  bio: string
  level: number
  experience: number
  status: number
  createdAt: string
  hasGitHubBound?: boolean
}

export interface FollowStatus {
  userId: number
  following: boolean
  followerCount: number
  followingCount: number
}

export interface FollowActionResponse {
  status: FollowStatus
  applied: boolean
}

// Prompt types
export interface Prompt {
  id: number
  title: string
  description: string
  cover: string
  content: string
  systemPrompt: string
  model: string
  params: PromptParams
  categoryId: number
  categoryName: string
  tags: string[]
  userId: number
  user: User
  views: number
  likes: number
  favorites: number
  status: number
  createdAt: string
  updatedAt: string
}

export interface PromptParams {
  temperature?: number
  topP?: number
  maxTokens?: number
  system?: string
}

// Category types
export interface Category {
  id: number
  name: string
  icon: string
  count: number
}

// Skill types
export interface Skill {
  id: number
  name: string
  description: string
  cover: string
  workflow: WorkflowStep[]
  inputs: SkillInput[]
  outputs: SkillOutput[]
  userId: number
  user: User
  views: number
  likes: number
  favorites: number
  createdAt: string
}

export interface WorkflowStep {
  id: string
  type: 'prompt' | 'tool' | 'condition' | 'output'
  name: string
  config: Record<string, unknown>
}

export interface SkillInput {
  name: string
  type: 'string' | 'number' | 'boolean' | 'select'
  required: boolean
  options?: string[]
}

export interface SkillOutput {
  name: string
  type: 'string' | 'number' | 'boolean' | 'json'
}

// Comment types
export interface Comment {
  id: number
  targetType: 'prompt' | 'skill'
  targetId: number
  userId: number
  user: User
  content: string
  likes: number
  parentId: number | null
  replies: Comment[]
  createdAt: string
}

export interface CreateCommentRequest {
  content: string
  parentId?: number | null
}

// API response types
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PromptActionResponse {
  prompt: Prompt
  applied: boolean
}

export interface CommentActionResponse {
  comment: Comment
  applied: boolean
}

export interface Report {
  id: number
  userId: number
  targetType: string
  targetId: number
  reason: string
  detail: string
  status: string
  createdAt: string
}

export interface ReportActionResponse {
  report: Report
  applied: boolean
}

export interface PageResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

// Request types
export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  captcha: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface UploadImageResponse {
  url: string
}

export interface PublishPromptRequest {
  title: string
  description: string
  cover: string
  content: string
  systemPrompt: string
  model: string
  params: PromptParams
  categoryId: number
  tags: string[]
  status: number
}
