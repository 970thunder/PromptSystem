import { siteCapabilities } from '@/config/capabilities'

export function githubAuthUrl(): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')
  return `${apiBase}/auth/github`
}

export { siteCapabilities }

export const githubOAuthEnabled = siteCapabilities.githubOAuth
