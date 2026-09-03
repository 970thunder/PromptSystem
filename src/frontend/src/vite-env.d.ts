/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_APP_TITLE: string
  readonly VITE_ENABLE_PROMPT_API?: string
  readonly VITE_GITHUB_OAUTH_ENABLED?: string
  readonly VITE_EMAIL_AUTH_ENABLED?: string
  readonly VITE_SKILL_ENABLED?: string
  readonly VITE_PLAYGROUND_ENABLED?: string
  readonly VITE_CREATOR_ACADEMY_ENABLED?: string
  readonly VITE_MARKETPLACE_ENABLED?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
