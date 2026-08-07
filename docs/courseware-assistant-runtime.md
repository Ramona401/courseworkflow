# TE-DNA课件原生教学智能体运行时交接文档

## 一、文档信息

- 项目：TE-DNA 2.0
- 功能：课件原生教学智能体
- 当前应用版本：0.43.0
- 后端：Go 1.22、net/http、pgx/v5
- 前端：React 19、TypeScript 5.9、Vite 8
- 数据库：PostgreSQL 16
- 生产域名：https://workflow.pkuailab.com
- systemd服务：tedna
- 标准部署脚本：/www/wwwroot/tedna/deploy.sh
- 本文档固定路径：
  /www/wwwroot/tedna/docs/courseware-assistant-runtime.md

本文档只记录课件教学智能体MVP的正式范围、运行边界、状态机、
API、安全规则、计费规则、功能开关、部署方式、回滚方式和当前待办。

---

## 二、当前总体状态

当前技术状态：

1. 开发单元01—25已经完成代码实现和生产部署。
2. 开发单元26前置缺口已于2026年8月3日完成生产部署和技术验收。
3. 教师端功能开关当前为true，教师端保留路由已经开放并受JWT保护。
4. 公开运行开关当前为false，embed和external新会话创建均返回503。
5. 已签发external令牌会在运行授权服务中按session_kind即时拒绝，
   不再等待短时令牌自然过期。
6. teacher_preview继续复用会话读取和聊天路径，并要求有效短时令牌。
7. 真实后端健康接口为/api/v1/health，直连和HTTPS均已验证返回200。
8. 尚未创建用于真实学生业务验收的active试点部署，也尚未完成
   真实AI调用、真实积分扣费、真实使用流水和外部平台试点。
9. 开发单元26—29仍未完成真实业务验收。
10. 当前准确状态为：
    “MVP基础设施和单元26前置安全缺口已上线，
    真实业务验收与公开试点尚未开放。”

禁止把当前状态描述为：

- 已正式扩大上线；
- 真实试点已通过；
- 计费闭环已完成线上真实验证；
- 原始IDE PRD全部完成。

---

## 三、MVP固定范围

### 3.1 教师端能力

教师能够：

1. 为指定课件稳定页面配置一个教学智能体。
2. 每个页面最多绑定一个智能体插槽。
3. 选择已有AI助手。
4. 根据当前页面上下文生成教学方案。
5. 编辑名称、欢迎语、教学角色、教学目标、问题链、
   错误认知分支和答案泄露策略。
6. 查看受限页面上下文回执。
7. 在TE-DNA内部模拟学生对话。
8. 发布不可变部署版本。
9. 设置每日调用额度、单会话最大轮数、允许Origin和有效期。
10. 暂停、恢复或撤销部署。
11. 将可运行智能体入口随课件导出到ZIP。

### 3.2 学生端能力

外部学生能够：

1. 在课件页面中打开悬浮智能体。
2. 不登录TE-DNA。
3. 使用匿名短时会话。
4. 接收流式教学回复。
5. 查看剩余轮数和会话状态。
6. 在部署暂停、额度耗尽、会话过期或网络异常时看到明确提示。
7. 不接触教师JWT、模型Key、完整提示词或内部账户信息。

### 3.3 第一版明确不做

当前MVP不包含：

- 通用个人知识库；
- 学校知识库向量检索；
- 任意URL抓取；
- 任意工具调用；
- 智能体直接修改课件页面；
- 学生真实姓名；
- 外部平台用户体系；
- 长期学生学习档案；
- 多智能体协作；
- 主动弹出；
- 连续答错自动触发；
- 一页多个智能体；
- 离线AI运行；
- LTI；
- 外部平台签名launch；
- 学校级统一计费。

---

## 四、数据库表

### 4.1 courseware_assistant_slots

教师编辑态插槽。

主要字段：

- courseware_id
- page_id
- assistant_id
- created_by
- display_mode
- position
- title
- welcome_message
- teaching_role
- teaching_goal
- guidance_plan
- context_config
- status

核心规则：

- 每个课件稳定页面最多一个插槽。
- page_id是稳定关联，不使用页码作为主关联。
- 插槽是编辑态，不直接作为外部运行事实源。
- 删除插槽不得删除已发布的部署版本。

### 4.2 assistant_deployments

可运行部署主记录。

主要字段：

- id
- public_id
- courseware_id
- page_id
- owner_user_id
- school_id
- education_domain
- current_version
- status
- daily_call_limit
- max_session_turns
- allowed_origins
- effective_at
- expires_at

状态：

- active
- paused
- revoked

核心规则：

- public_id由服务端随机生成且不可预测。
- 只有active部署可运行。
- active与paused可相互转换。
- active或paused可以转为revoked。
- revoked是永久终态，不能恢复。

### 4.3 assistant_deployment_versions

不可变发布版本。

保存：

- assistant_id
- 助手提示词快照
- 助手提示词哈希
- 教学方案快照
- 页面上下文快照
- 页面HTML哈希
- 课件事实快照
- 创建者
- 创建时间

核心规则：

- 版本只能追加，不能更新旧版本。
- 外部运行始终使用发布时的不可变快照。
- 浏览器响应不得返回完整提示词、完整上下文或内部身份。

### 4.4 assistant_runtime_sessions

匿名学生和教师预览运行会话。

主要字段：

- id
- deployment_id
- deployment_version
- runtime_token_jti_hash
- anonymous_client_hmac
- parent_origin
- ip_hmac
- session_kind
- status
- turn_count
- max_turns
- active_turn_id
- messages
- expires_at
- last_active_at

会话类型：

- external
- teacher_preview

会话状态：

- active
- completed
- expired
- revoked

核心规则：

- 数据库只保存JTI哈希，不保存原始令牌。
- 不保存原始匿名客户端ID。
- 不保存学生原始IP。
- 只保存student和assistant正式可见消息。
- 不保存系统提示词、tool消息或隐藏推理。
- active_turn_id保证单会话同一时刻只有一个主轮次。

### 4.5 assistant_runtime_usage

运行使用流水。

记录：

- deployment_id
- session_id
- turn_id
- owner_user_id
- school_id
- courseware_id
- page_id
- session_kind
- input_chars
- output_chars
- input_tokens
- output_tokens
- credits_consumed
- model
- provider
- status
- error_code
- latency_ms

核心规则：

- turn_id是结算幂等键。
- 成功结算扣积分、写成功流水、追加正式消息并推进轮数。
- 失败结算只写失败流水并释放主轮次。
- 模型失败不得重复扣费。
- 使用流水只用于审计和计费追溯，不作为学生画像。

---

## 五、后端状态机

### 5.1 部署状态机

允许：

- active → paused
- paused → active
- active → revoked
- paused → revoked

禁止：

- revoked → active
- revoked → paused
- revoked →任何可运行状态

### 5.2 会话状态机

正常流程：

- active → completed
- active → expired
- active → revoked

以下情况应使会话不可继续：

- 短时令牌过期；
- 部署暂停；
- 部署撤销；
- 部署当前版本变化；
- 会话达到最大轮数；
- 会话达到有效期；
- 会话被服务端撤销。

### 5.3 主轮次状态

领取顺序固定：

1. 锁部署。
2. 锁会话。
3. 锁创建者个人积分账户。
4. 检查部署实时状态。
5. 检查版本一致性。
6. 检查单会话轮数。
7. external会话检查每日部署额度。
8. 检查创建者积分。
9. 设置active_turn_id。
10. 调用AI。
11. 成功或失败二选一结算。
12. 释放active_turn_id。

---

## 六、教师端API

所有教师端接口必须经过生产CORS和教师JWT。

### 6.1 插槽与方案

- GET /api/v1/coursewares/{courseware_id}/assistant-slots
- GET /api/v1/coursewares/{courseware_id}/pages/{page_id}/assistant-slot
- POST /api/v1/coursewares/{courseware_id}/pages/{page_id}/assistant-slot
- PUT /api/v1/coursewares/{courseware_id}/assistant-slots/{slot_id}
- DELETE /api/v1/coursewares/{courseware_id}/assistant-slots/{slot_id}
- GET /api/v1/coursewares/{courseware_id}/pages/{page_id}/assistant-context
- POST /api/v1/coursewares/{courseware_id}/pages/{page_id}/assistant-plan

### 6.2 部署管理

- GET /api/v1/coursewares/{courseware_id}/assistant-deployments
- POST /api/v1/coursewares/{courseware_id}/pages/{page_id}/assistant-deployment
- GET /api/v1/assistant-deployments/{deployment_id}/versions
- POST /api/v1/assistant-deployments/{deployment_id}/versions
- POST /api/v1/assistant-deployments/{deployment_id}/pause
- POST /api/v1/assistant-deployments/{deployment_id}/resume
- POST /api/v1/assistant-deployments/{deployment_id}/revoke
- PUT /api/v1/assistant-deployments/{deployment_id}/policy
- POST /api/v1/assistant-deployments/{deployment_id}/preview-session

### 6.3 教师端授权

- 只有课件作者本人可以创建、修改和删除插槽。
- admin不自动获得他人课件写权限。
- 集体备课参与者不自动获得教学智能体发布权限。
- owner_user_id、school_id和education_domain只由服务端解析。
- prompt_protected助手可使用，但提示词原文不得返回浏览器。
- submitted和in_pipeline状态下禁止普通编辑。
- 暂停和撤销可绕过普通编辑锁，用于紧急止损。

---

## 七、公开运行API

公开端点不使用教师JWT。

- GET /embed/assistant/{public_id}
- POST /api/v1/assistant-runtime/deployments/{public_id}/session
- GET /api/v1/assistant-runtime/sessions/{session_id}
- POST /api/v1/assistant-runtime/sessions/{session_id}/chat

运行认证：

- 会话创建使用官方embed页面上下文。
- 会话读取和聊天使用Authorization Bearer短时运行令牌。
- 运行令牌不得放入URL。
- 短时令牌TTL必须在5—15分钟。
- 当前推荐15分钟。

---

## 八、Origin和浏览器安全边界

external会话创建必须同时满足：

1. HTTP Origin等于TE-DNA官方运行站点。
2. Referer同源且精确指向当前public_id的官方embed页面。
3. 请求正文parent_origin命中部署allowed_origins。

allowed_origins规则：

- 必须是精确Origin。
- 外部来源必须使用HTTPS。
- HTTP只允许localhost或回环地址。
- 禁止通配符。
- 禁止路径。
- 禁止查询参数。
- 禁止片段。
- 禁止用户名和密码。
- 规范化后去重。

iframe安全属性：

- sandbox="allow-scripts allow-same-origin"
- referrerpolicy="origin"

postMessage规则：

- 只发送到document.referrer解析出的精确父Origin。
- 不使用星号目标。
- 父页面只接受TE-DNA精确Origin。
- 必须核对目标iframe窗口。
- 必须核对消息类型。
- 必须核对public_id。
- 必须校验高度范围。

---

## 九、动态CSP

embed页面按部署allowed_origins动态生成frame-ancestors。

主要策略：

- default-src 'none'
- script-src 'self'
- connect-src 'self'
- object-src 'none'
- base-uri 'none'
- form-action 'none'
- frame-ancestors只包含部署保存的精确Origin
- 不隐式加入self
- X-Content-Type-Options: nosniff
- Referrer-Policy: same-origin
- 禁止摄像头
- 禁止麦克风
- 禁止定位
- 禁止支付
- 禁止USB

允许来源不得出现在HTML正文中。

---

## 十、计费规则

固定规则：

1. external和teacher_preview都由部署创建者的个人积分账户付费。
2. 匿名学生不能以空UserID调用AI。
3. SchoolID取部署发布时固化值，用于模型分流。
4. external会话消耗部署每日额度。
5. teacher_preview不消耗部署每日额度。
6. 两类会话都消耗单会话轮数。
7. 两类会话都检查部署创建者积分。
8. 公开运行绕过普通AI积分钩子，由运行计费桥精确结算。
9. 模型失败不重复扣费。
10. 成功结算后自动补足只作为旁路，不影响AI主结果。

---

## 十一、功能开关

### 11.1 教师端总开关

环境变量：

COURSEWARE_ASSISTANT_ENABLED

默认值：

true

true时开放：

- 教学智能体Tab对应后端接口；
- 插槽查询和保存；
- 上下文预览；
- 方案生成；
- 部署管理；
- 教师内部预览会话创建。

false时：

- 教师端教学智能体保留路径统一返回404；
- 不构造教师端教学智能体服务依赖；
- 课件、漫画、Style Studio及其他功能继续正常下沉。

### 11.2 公开运行开关

环境变量：

COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED

默认值：

false

true时开放：

- /embed/assistant/{public_id}
- external会话创建

false时阻断：

- 公开embed页面；
- external新会话创建。

false时仍保留：

- 会话读取；
- 会话聊天。

保留原因：

- 教师内部预览使用teacher_preview短时令牌；
- 教师预览复用相同的GetSession和Chat端点。

当前生产行为：

- 公开开关关闭时，embed和external新会话创建立即返回503；
- 已签发external令牌在运行令牌验证阶段按session_kind即时拒绝；
- external旧会话不会继续工作至短时令牌自然过期；
- teacher_preview短时令牌仍可使用会话读取和聊天端点；
- 未显式启用公开运行的会话服务按false失败关闭；
- 如需重新开放公开运行，必须先完成单元26真实业务验收，
  配置单一明确HTTPS Origin，并重启tedna使配置生效。

### 11.3 开关依赖

当：

COURSEWARE_ASSISTANT_ENABLED=false

即使配置：

COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=true

配置加载层也会强制把公开运行开关收敛为false。

---

## 十二、生产环境变量

生产环境配置文件约定：

/www/wwwroot/tedna/private/config/assistant-runtime.env

必须包含独立隐私盐：

ASSISTANT_RUNTIME_PRIVACY_SALT=独立64字符随机十六进制字符串

令牌TTL：

ASSISTANT_RUNTIME_TOKEN_TTL_MINUTES=15

教师端和公开开关：

COURSEWARE_ASSISTANT_ENABLED=true
COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=false

生成隐私盐：

openssl rand -hex 32

禁止复用：

- JWT_SECRET
- AES_KEY
- 数据库密码
- AI API Key
- 其他业务盐值

修改环境变量后不需要重新构建代码，但需要重启tedna服务，
使新进程重新加载配置。

---

## 十三、学生端短时令牌自动恢复

学生端运行令牌只保存在React内存。

自动恢复流程：

1. 会话同步请求返回401。
2. 客户端标记正在恢复，防止重复触发。
3. 关闭旧SSE连接。
4. 中止旧启动请求。
5. 递增generation。
6. 清空旧session_id和旧runtime_token。
7. 生成新的随机匿名客户端ID。
8. 使用同一public_id和父页面Origin重新创建会话。
9. 使用新令牌读取新会话。
10. 成功后提示学生重新发送上一条消息。

聊天请求返回401时：

1. SSE错误回调不会取得HTTP状态对象。
2. 客户端立即读取旧会话。
3. 旧令牌读取返回401。
4. 进入统一自动恢复流程。

防重复规则：

- 不自动重新提交学生消息。
- 不复用旧session_id。
- 不复用旧runtime_token。
- 不复用旧匿名客户端ID。
- 旧generation的chunk、done、error和finally回调全部失效。
- 原学生消息在新会话创建时被清除。
- 学生必须主动重新发送，避免重复AI调用和重复扣费。

单元26仍需在真实浏览器验证：

- 聊天请求401后的恢复；
- 会话读取401后的恢复；
- 恢复期间旧SSE是否被关闭；
- 是否只创建一个新会话；
- 是否没有自动重发；
- 是否没有重复扣费；
- 自动恢复失败提示；
- 移动端恢复体验。

---

## 十四、课件导出行为

导出ZIP时：

- 按稳定page_id映射部署。
- 只注入导出时刻active且有效的部署。
- ZIP中只保存public_id。
- 不保存内部deployment_id。
- 不保存教师身份。
- 不保存学校ID。
- 不保存提示词。
- 不保存上下文正文。
- 不保存运行令牌。
- 不修改数据库原始页面HTML。
- 注入必须幂等。
- 悬浮入口位于1920×1080课件舞台之外。
- file协议下不创建在线iframe。
- 离线模式显示联网和HTTPS托管提示。
- 公网HTTP拒绝运行。
- localhost或回环地址允许HTTP。

---

## 十五、Nginx边界

生产Nginx必须保持：

- /embed/assistant/交给Go动态生成。
- embed路径不得进入SPA静态回退。
- 公开运行API关闭代理缓冲。
- SSE关闭Nginx缓冲。
- embed页面禁止缓存。
- /assets/assistant-embed.js禁止缓存。
- 教师端API保留现有CORS和JWT边界。
- 不设置全局Access-Control-Allow-Origin: *。

---

## 十六、标准部署

部署前必须：

1. 数据库完整备份。
2. 修改文件带时间戳备份。
3. 环境变量文件备份。
4. Nginx配置备份。
5. 检查磁盘空间。
6. 检查PostgreSQL。
7. 检查Nginx。
8. 检查tedna systemd服务。

标准部署命令：

cd /www/wwwroot/tedna
./deploy.sh

deploy.sh负责：

- 数据库自动备份；
- Go依赖同步；
- go vet；
- Go生产编译；
- 后端二进制原子替换；
- 前端Vite生产构建；
- Nginx配置检查；
- Nginx reload；
- systemd服务重启；
- 后端和代理健康检查；
- 真实后端健康接口为/api/v1/health；
- 失败自动回滚。

禁止：

- 执行与本次业务无关的全量测试；
- 私自执行Git提交或推送；
- 绕过deploy.sh直接替换生产二进制；
- 在数据库未备份时执行迁移或写操作。

---

## 十七、回滚原则

### 17.1 代码回滚

所有修改文件必须使用同一时间戳备份。

回滚时：

1. 停止继续修改。
2. 确认要恢复的backup时间戳。
3. 将对应backup文件完整复制回原路径。
4. 使用标准deploy.sh重新构建和部署。
5. 验证健康端点。
6. 验证教师端原功能。
7. 验证公开入口状态。

禁止使用sed、awk或局部拼接回滚。

### 17.2 公开运行紧急止损

优先顺序：

1. 设置：
   COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=false
2. 重启tedna服务，使配置生效。
3. 暂停所有active试点部署。
4. 如确认永久失效，撤销部署。
5. 检查运行会话和使用流水。
6. 核对积分变化。
7. 检查Nginx和journalctl日志。

### 17.3 教师端紧急关闭

设置：

COURSEWARE_ASSISTANT_ENABLED=false

该设置会同时强制公开运行开关关闭。

---

## 十八、开发单元26验收闸门

进入试点前必须完成：

### 18.1 教师端

- 作者创建插槽。
- 非作者不能创建。
- 选择现有助手。
- 生成当前页面教学方案。
- 保存后刷新仍存在。
- 教师内部模拟对话。
- 发布版本1。
- 修改但不发布时外部仍使用版本1。
- 发布版本2后外部切换版本2。
- 暂停后外部不能创建新会话。
- 恢复paused部署。
- revoked部署不能恢复。

### 18.2 外部端

- 允许Origin加载iframe。
- 非允许Origin不能创建会话。
- iframe源码无API Key。
- 短时令牌过期自动创建新会话。
- 不自动重发过期前消息。
- 超过单会话轮数被拒绝。
- 超过每日额度被拒绝。
- 网络断开显示降级提示。
- 离线课件其他内容正常。
- 刷新不重复扣除未发生的调用。
- iframe不能调用教师管理接口。
- 顶层打开embed不能创建会话。

### 18.3 安全

- 修改public_id不能读取内部信息。
- 修改session_id不能串会话。
- 修改page_id不能扩大上下文。
- 修改assistant_id不能更换助手。
- 伪造school_id无效。
- 伪造education_domain无效。
- 提示词不出现在浏览器。
- 完整页面HTML不出现在运行响应。
- 撤销部署后旧令牌失效。
- 旧发布版本继续使用固化快照。
- Origin和Referer任一错误均拒绝。
- parent_origin不在白名单时拒绝。
- Authorization不进入URL。
- JSON未知字段拒绝。
- 同会话并发消息只有一个主轮次。

### 18.4 计费

- 调用归属部署创建者。
- SchoolID用于正确模型分流。
- 积分不足明确拒绝。
- 模型失败不重复扣费。
- 使用流水与AI调用一致。
- external消耗每日额度。
- teacher_preview不消耗每日额度。
- 两类会话都消耗单会话轮数。
- AI失败不增加成功轮数。
- AI失败写失败流水。

---

## 十九、开发单元27—29

### 单元27：公开运行关闭状态部署

目标：

COURSEWARE_ASSISTANT_ENABLED=true
COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=false

验证：

- 教师端开放。
- 内部教师预览开放。
- embed关闭。
- external新会话创建关闭。
- 现有课件功能不受影响。

当前生产环境已经达到上述技术配置状态。
但单元26真实业务验收尚未完成，因此不得将单元27描述为业务结项。

### 单元28：单一试点域名

目标：

COURSEWARE_ASSISTANT_ENABLED=true
COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED=true

限制：

- 只允许一个明确HTTPS Origin。
- 只允许极少数测试课件。
- 使用小额度。
- 设置明确有效期。
- 每日核对日志、流水和积分。

### 单元29：正式扩大上线

前置条件：

- 单域名试点稳定。
- 无来源绕过。
- 无重复扣费。
- 无会话串线。
- 无提示词泄漏。
- 无未解释流水。
- 暂停和撤销可靠。
- 功能开关可靠。
- 回滚流程已验证。

---

## 二十、当前待办

已完成的前置项（2026年8月3日）：

1. 部署并验证两个功能开关。
2. 完整检查运行授权服务。
3. 增加public开关关闭时按session_kind即时拒绝external旧会话。
4. 保持teacher_preview在公开开关关闭时可运行。
5. 完成学生端401自动恢复代码和防自动重发协议检查。
6. 验证教师端路由开启、公开入口关闭和教师预览共用路径保留。
7. 验证真实后端健康接口为/api/v1/health。
8. 完成相关Go测试、TypeScript检查、go vet和生产入口编译。
9. 使用标准deploy.sh完成数据库备份、构建、重启和生产部署。

P0：

1. 在真实浏览器验证学生端401自动恢复。
2. 准备真实active测试部署。
3. 准备真实HTTPS试点父页面。
4. 验证真实AI调用。
5. 验证真实积分扣费。
6. 验证真实使用流水。
7. 验证真实导出ZIP。
8. 测试结束后暂停或撤销部署。

P1：

1. 完成教师端、外部端、安全和计费验收。
2. 验证版本1、版本2和未发布草稿隔离。
3. 验证paused、resumed和revoked状态。
4. 验证模型失败不重复扣费。
5. 验证会话并发保护。
6. 验证令牌和版本实时失效。
7. 验证移动端与网络异常体验。

---

## 二十一、当前已知限制

1. 当前没有用于真实试点验收的active部署，尚无真实外部学生会话。
2. 尚未完成真实AI扣费闭环验证。
3. 尚未完成外部平台端到端试点。
4. 公开运行开关当前阻断embed和新external会话。
5. external旧会话已经由服务层按session_kind即时硬阻断。
6. teacher_preview继续复用会话读取和聊天端点，不能把这两个端点整体关闭。
7. 令牌自动恢复已进入代码，但尚未完成真实浏览器验收。
8. 当前应用版本仍为0.43.0。
9. 单元26—29尚未完成真实业务验收。
10. 第二阶段知识库、事件桥、主动触发和运行分析不得提前启动。

---

## 二十二、最近更新

本次更新内容：

- 新增COURSEWARE_ASSISTANT_ENABLED配置。
- 新增COURSEWARE_ASSISTANT_PUBLIC_RUNTIME_ENABLED配置。
- 教师端总开关关闭时，教学智能体保留路径返回404。
- 公开开关关闭时，embed和新external会话返回503。
- 保留teacher_preview所需的会话读取和聊天路径。
- external旧令牌按session_kind在服务层即时拒绝。
- 学生端会话读取401时自动创建新会话。
- 学生端不自动重发过期前消息，避免重复AI调用和重复扣费。
- 建立固定路径交接文档和生产专用环境配置。
- 修复无关的课件AI审核全局讨论治理语法阻断，并完成纯函数回归测试。

生产部署记录：

- 部署日期：2026年8月3日。
- 标准脚本：/www/wwwroot/tedna/deploy.sh。
- 数据库备份：tedna_20260803_104822.sql.gz。
- 应用版本：0.43.0。
- 教师端路由无JWT返回401，证明教师端功能已开放。
- embed和external新会话返回503，证明公开运行保持关闭。
- 会话读取和聊天无运行令牌返回401，证明教师预览共用路径保留。
- /api/v1/health直连和HTTPS均返回200 application/json。
- 部署后journalctl没有warning及以上日志。

本次更新已完成生产部署和技术验收。
尚未完成真实active部署、真实学生浏览器会话、真实AI调用、
真实积分扣费、真实流水核对和外部HTTPS试点。
