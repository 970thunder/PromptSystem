// isoumao 统一登录弹窗：组件由身份中心托管，本站只负责打开与刷新会话。
// 组件地址：http://localhost:28310/widget/isoumao-login.js（生产 https://id.isoumao.cn/widget/isoumao-login.js）
import { identityCenterBase } from '@/utils/authUrl'

const WIDGET_SCRIPT_ID = 'isoumao-login-widget'

interface IsoumaoLoginWidget {
  open: (options: {
    loginUrl: string
    origins: string[]
    fallbackUrl?: string
    description?: string
    onSuccess?: () => void
    onError?: (message: string) => void
  }) => void
}

interface OpenLoginOptions {
  loginUrl: string
  onSuccess: () => void | Promise<void>
  description?: string
  fallbackUrl?: string
}

function ensureWidgetLoaded(): Promise<IsoumaoLoginWidget> {
  const existing = (window as unknown as { IsoumaoLogin?: IsoumaoLoginWidget }).IsoumaoLogin
  if (existing) return Promise.resolve(existing)

  return new Promise<IsoumaoLoginWidget>((resolve, reject) => {
    const current = document.getElementById(WIDGET_SCRIPT_ID) as HTMLScriptElement | null
    const onLoad = () => {
      const widget = (window as unknown as { IsoumaoLogin?: IsoumaoLoginWidget }).IsoumaoLogin
      if (widget) resolve(widget)
      else reject(new Error('登录组件初始化失败'))
    }
    if (current) {
      current.addEventListener('load', onLoad, { once: true })
      current.addEventListener('error', () => reject(new Error('登录组件加载失败')), { once: true })
      return
    }

    const script = document.createElement('script')
    script.id = WIDGET_SCRIPT_ID
    script.src = `${identityCenterBase.replace(/\/$/, '')}/widget/isoumao-login.js`
    script.async = true
    script.addEventListener('load', onLoad, { once: true })
    script.addEventListener('error', () => reject(new Error('登录组件加载失败，请检查身份中心是否可用')), { once: true })
    document.head.appendChild(script)
  })
}

export function useIsoumaoLogin() {
  async function openLoginDialog(options: OpenLoginOptions) {
    const widget = await ensureWidgetLoaded()
    // 回调收尾页由本站 API 源返回；本地开发时页面与 API 不同源，两者都登记。
    const apiOrigin = new URL(options.loginUrl, window.location.origin).origin
    const origins = Array.from(new Set([apiOrigin, window.location.origin]))
    widget.open({
      loginUrl: options.loginUrl,
      origins,
      fallbackUrl: options.fallbackUrl ?? options.loginUrl,
      description: options.description,
      onSuccess: () => { void options.onSuccess() }
    })
  }

  return { openLoginDialog }
}
