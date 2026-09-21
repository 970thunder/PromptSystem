// isoumao 统一登录弹窗：组件由身份中心托管，本站只负责打开与刷新会话。
// 组件地址：http://localhost:28310/widget/isoumao-login.js（生产 https://id.isoumao.cn/widget/isoumao-login.js）
import { identityCenterBase } from '@/utils/authUrl'

const WIDGET_SCRIPT_ID = 'isoumao-login-widget'

interface IsoumaoLoginWidget {
  open: (options: { loginUrl: string; origin: string; description?: string; onSuccess?: () => void }) => void
}

interface OpenLoginOptions {
  loginUrl: string
  onSuccess: () => void | Promise<void>
  description?: string
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
    // 回调收尾页由本站 API 源返回，postMessage 的消息来源即 API 源。
    const apiOrigin = new URL(options.loginUrl, window.location.origin).origin
    widget.open({
      loginUrl: options.loginUrl,
      origin: apiOrigin,
      description: options.description,
      onSuccess: () => { void options.onSuccess() }
    })
  }

  return { openLoginDialog }
}