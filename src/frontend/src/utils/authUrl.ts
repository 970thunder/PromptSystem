import { siteCapabilities } from '@/config/capabilities'

export function githubAuthUrl(): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')
  return `${apiBase}/auth/github`
}

export function oidcAuthUrl(): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')
  return `${apiBase}/auth/oidc`
}

export const oidcEnabled = String(import.meta.env.VITE_OIDC_ENABLED || 'false').toLowerCase() === 'true'

export const identityCenterBase = import.meta.env.VITE_IDENTITY_CENTER_BASE || 'https://id.isoumao.cn'
export const communityBase = import.meta.env.VITE_COMMUNITY_BASE || 'https://community.isoumao.cn'
export const nebulaBase = import.meta.env.VITE_NEBULA_BASE || 'https://nebula.isoumao.cn'

export { siteCapabilities }

export const githubOAuthEnabled = siteCapabilities.githubOAuth
