import { registerSW } from 'virtual:pwa-register';

// 作为 BCTools 的内嵌应用，um-react 不应反复弹出刷新提示。
// 使用 autoUpdate 策略让 Service Worker 在后台静默激活。
registerSW({
  onNeedRefresh() {},
  onOfflineReady() {},
});
