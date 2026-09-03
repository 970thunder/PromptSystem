// 文件作用：集中维护前后端共享的前端业务类型定义。
import type {
  ApiContractApiResponse,
  ApiContractCategory,
  ApiContractComment,
  ApiContractReport,
  ApiContractAuditEvent,
  ApiContractPageResponse,
  ApiContractPrompt,
  ApiContractPromptParams,
  ApiContractUser
} from './api-contract.generated'

export type User = ApiContractUser
export type Prompt = ApiContractPrompt
export type PromptParams = ApiContractPromptParams
export type Category = ApiContractCategory
export type Comment = ApiContractComment
export type Report = ApiContractReport
export type AuditEvent = ApiContractAuditEvent
export type ApiResponse<T> = ApiContractApiResponse<T>
export type PageResponse<T> = ApiContractPageResponse<T>

// User types
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
export interface PromptExample {
  title: string
  input: string
  output: string
}

export interface PromptWorkflowStep {
  title: string
  detail: string
}

// Category types
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

export interface CreateCommentRequest {
  content: string
  parentId?: number | null
}

export interface PromptActionResponse {
  prompt: Prompt
  applied: boolean
}

export interface CommentActionResponse {
  comment: Comment
  applied: boolean
}

export interface ReportActionResponse {
  report: Report
  applied: boolean
}

export interface UserDataExport {
  exportedAt: string
  user: User
  prompts: Prompt[]
  favorites: Prompt[]
  likes: Prompt[]
  history: Prompt[]
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

export interface ResetPasswordRequest {
  email: string
  password: string
  captcha: string
}

export interface LoginResponse {
  /** Legacy bearer clients may receive a token; browser sessions use a cookie. */
  token?: string
  user: User
}

export interface CaptchaResponse {
  expiresInSeconds: number
  devCode?: string
}

export interface UploadImageResponse {
  url: string
}

export interface PublishPromptRequest {
  title: string
  description: string
  cover: string
  images: string[]
  content: string
  systemPrompt: string
  model: string
  params: PromptParams
  categoryId: number
  tags: string[]
  status: number
}
