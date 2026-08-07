package handlers

// courseware_assistant_handler.go
//
// 本文件只定义教师端课件教学智能体Handler的依赖结构和构造函数。
//
// HTTP方法按职责拆分：
//   - courseware_assistant_read_handler.go：
//     插槽列表、单页插槽和上下文安全预览；
//
//   - courseware_assistant_write_handler.go：
//     插槽创建、更新、删除和AI方案草稿生成。
//
// 请求解析、路径解析、响应模型和错误映射继续分别位于：
//   - courseware_assistant_request.go；
//   - courseware_assistant_response.go；
//   - courseware_assistant_handler_errors.go。
//
// 本单元只建立Handler，不在总路由中注册。
// 路由与AI配置依赖注入统一留到开发单元16。

import "tedna/internal/services"

// CoursewareAssistantHandler 是教师端课件教学智能体处理器。
type CoursewareAssistantHandler struct {
	slotService    *services.CoursewareAssistantSlotService
	contextService *services.CoursewareAssistantContextService
	planService    *services.CoursewareAssistantPlanService
}

// NewCoursewareAssistantHandler 创建教师端教学智能体处理器。
func NewCoursewareAssistantHandler(
	slotService *services.CoursewareAssistantSlotService,
	contextService *services.CoursewareAssistantContextService,
	planService *services.CoursewareAssistantPlanService,
) *CoursewareAssistantHandler {
	return &CoursewareAssistantHandler{
		slotService:    slotService,
		contextService: contextService,
		planService:    planService,
	}
}
