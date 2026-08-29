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
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
