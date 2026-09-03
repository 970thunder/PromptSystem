type EnvFlag = string | boolean | undefined

const readFlag = (value: EnvFlag, fallback: boolean): boolean => {
  if (typeof value === 'boolean') {
    return value
  }
  if (value === undefined || value.trim() === '') {
    return fallback
  }
  return ['1', 'true', 'yes', 'on'].includes(value.trim().toLowerCase())
}

/** Build-time switches for routes and integrations that are actually ready. */
export const siteCapabilities = Object.freeze({
  emailAuth: readFlag(import.meta.env.VITE_EMAIL_AUTH_ENABLED, true),
  githubOAuth: readFlag(import.meta.env.VITE_GITHUB_OAUTH_ENABLED, false),
  skillRunner: readFlag(import.meta.env.VITE_SKILL_ENABLED, false),
  playground: readFlag(import.meta.env.VITE_PLAYGROUND_ENABLED, false),
  creatorAcademy: readFlag(import.meta.env.VITE_CREATOR_ACADEMY_ENABLED, false),
  marketplace: readFlag(import.meta.env.VITE_MARKETPLACE_ENABLED, false)
})

export type SiteCapabilities = typeof siteCapabilities
