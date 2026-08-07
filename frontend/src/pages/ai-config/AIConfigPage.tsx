/**
 * AIConfigPage — 历史AI配置入口兼容页
 *
 * 系统曾同时维护：
 *   - /workflow/ai-config
 *   - /ai-center
 *
 * 两套页面的场景编辑、Fallback和媒体网关能力不同，
 * 容易出现管理员在旧页面修改后误认为配置没有生效。
 *
 * 现在统一以 /ai-center 为唯一AI模型管理入口。
 * 旧书签和历史菜单仍可访问，但会立即跳转到新管理中心。
 */
import { Navigate } from 'react-router-dom'

export default function AIConfigPage() {
  return (
    <Navigate
      to="/ai-center"
      replace
    />
  )
}
