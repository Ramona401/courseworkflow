package services

// courseware_add_page_discussion_service.go — 新增课件页的无状态AI需求讨论服务。
//
// 本服务只负责在真正建页之前帮助老师澄清需求并形成结构化页面方案：
//   - 不创建courseware_pages记录；
//   - 不生成HTML、CSS或JavaScript；
//   - 不修改课件状态、页码或总页数；
//   - 不保存模型隐藏推理；
//   - 对话历史和当前方案由前端受保护草稿保存，每轮随请求传回。
//
// 老师点击独立“按此方案生成并插入”按钮后，前端才调用既有新增页接口，
// 再调用既有单页重新生成接口。这样讨论失败或中途关闭弹窗都不会产生空白页。
//
// 权限边界：
//   服务端重新加载课件，执行作者本人、教育域快照和审核写锁校验；
//   客户端提交的用户、角色和教育域都不能替代可信Actor。

import (
        "context"
        "errors"
        "fmt"
        "strings"
        "unicode/utf8"

        "tedna/internal/ai"
        "tedna/internal/models"
        "tedna/internal/repository"
)

const (
        cwAddPageDiscussionMaxMessages        = 24
        cwAddPageDiscussionMaxMessageRunes    = 8000
        cwAddPageDiscussionMaxTranscriptRunes = 32000
        cwAddPageDiscussionMaxOutlineRunes    = 16000
)

var (
        // ErrCWAddPageDiscussionInvalidRequest 表示前端提交的数据不完整或插入位置无效。
        ErrCWAddPageDiscussionInvalidRequest = errors.New(
                "新增页讨论请求无效",
        )

        // ErrCWAddPageDiscussionTooLong 表示单条消息或历史上下文超过安全上限。
        ErrCWAddPageDiscussionTooLong = errors.New(
                "新增页讨论内容过长",
        )
)

// CoursewareAddPageDiscussionMessage 是一条由前端受保护草稿保存的正式对话消息。
//
// Role只允许teacher或assistant；不接受system等可改变系统指令的角色。
type CoursewareAddPageDiscussionMessage struct {
        Role    string `json:"role"`
        Content string `json:"content"`
}

// CoursewareAddPagePlan 是讨论过程中持续收敛的结构化页面方案。
//
// 字段与现有AddCWPageRequest保持同一语义，前端确认后可直接映射到新增页接口。
type CoursewareAddPagePlan struct {
        Title               string `json:"title"`
        Purpose             string `json:"purpose"`
        ContentSummary      string `json:"content_summary"`
        InteractionType     string `json:"interaction_type"`
        VisualFormat        string `json:"visual_format"`
        MediaRequirements   string `json:"media_requirements"`
        EstimatedComplexity int    `json:"estimated_complexity"`
}

// CoursewareAddPageDiscussionRequest 是新增页讨论单轮请求。
//
// Messages不包含本轮Message；服务端会在构建讨论记录时把本轮消息追加到末尾。
type CoursewareAddPageDiscussionRequest struct {
        Message     string                               `json:"message"`
        Messages    []CoursewareAddPageDiscussionMessage `json:"messages"`
        InsertAt    int                                  `json:"insert_at"`
        CurrentPlan CoursewareAddPagePlan                `json:"current_plan"`
}

// CoursewareAddPageDiscussionResponse 是前端可直接展示和保存的本轮讨论结果。
type CoursewareAddPageDiscussionResponse struct {
        Reply                string                `json:"reply"`
        Summary              string                `json:"summary"`
        ReadyForConfirmation bool                  `json:"ready_for_confirmation"`
        InsertAt             int                   `json:"insert_at"`
        Plan                 CoursewareAddPagePlan `json:"plan"`
}

// CoursewareAddPageDiscussionService 复用课件生成服务中的正式AI配置。
type CoursewareAddPageDiscussionService struct {
        genService *CoursewareGenService
}

// NewCoursewareAddPageDiscussionService 创建新增页讨论服务。
func NewCoursewareAddPageDiscussionService(
        genService *CoursewareGenService,
) *CoursewareAddPageDiscussionService {
        return &CoursewareAddPageDiscussionService{
                genService: genService,
        }
}

// Discuss 执行一轮新增页需求讨论。
//
// 本方法只返回讨论回复和页面方案，不创建页面、不生成HTML。
func (s *CoursewareAddPageDiscussionService) Discuss(
        ctx context.Context,
        coursewareID string,
        actor *CoursewareActorContext,
        request *CoursewareAddPageDiscussionRequest,
) (*CoursewareAddPageDiscussionResponse, error) {
        if request == nil {
                return nil, fmt.Errorf(
                        "%w: 请求体不能为空",
                        ErrCWAddPageDiscussionInvalidRequest,
                )
        }

        message := strings.TrimSpace(
                request.Message,
        )
        if message == "" {
                return nil, fmt.Errorf(
                        "%w: 讨论内容不能为空",
                        ErrCWAddPageDiscussionInvalidRequest,
                )
        }

        if utf8.RuneCountInString(message) >
                cwAddPageDiscussionMaxMessageRunes {
                return nil, fmt.Errorf(
                        "%w: 单次讨论不能超过%d个字符",
                        ErrCWAddPageDiscussionTooLong,
                        cwAddPageDiscussionMaxMessageRunes,
                )
        }

        if len(request.Messages) >
                cwAddPageDiscussionMaxMessages {
                return nil, fmt.Errorf(
                        "%w: 本次讨论最多保留%d条历史消息",
                        ErrCWAddPageDiscussionTooLong,
                        cwAddPageDiscussionMaxMessages,
                )
        }

        courseware, scopedActor, err :=
                (&CoursewareService{}).
                        loadOwnedCoursewareForControlMutation(
                                ctx,
                                strings.TrimSpace(
                                        coursewareID,
                                ),
                                actor,
                        )
        if err != nil {
                return nil, err
        }

        pages, err := repository.ListCoursewarePages(
                ctx,
                courseware.ID,
        )
        if err != nil {
                return nil, fmt.Errorf(
                        "读取课件页面方案失败: %w",
                        err,
                )
        }

        insertAt := request.InsertAt
        if insertAt <= 0 {
                insertAt = len(pages) + 1
        }

        if insertAt < 1 ||
                insertAt > len(pages)+1 {
                return nil, fmt.Errorf(
                        "%w: 当前共%d页，可插入第1至第%d页",
                        ErrCWAddPageDiscussionInvalidRequest,
                        len(pages),
                        len(pages)+1,
                )
        }

        history, err :=
                normalizeCWAddPageDiscussionMessages(
                        request.Messages,
                )
        if err != nil {
                return nil, err
        }

        currentPlan :=
                normalizeCWAddPagePlan(
                        request.CurrentPlan,
                )

        userPrompt, err :=
                buildCWAddPageDiscussionUserPrompt(
                        courseware,
                        pages,
                        insertAt,
                        history,
                        message,
                        currentPlan,
                )
        if err != nil {
                return nil, err
        }

        if s == nil ||
                s.genService == nil ||
                s.genService.cfg == nil {
                return nil, fmt.Errorf(
                        "新增页AI讨论服务未完成初始化",
                )
        }

        aiConfig, err := ai.GetEffectiveConfig(
                s.genService.cfg.GetAESKey(),
                models.SceneCWPageRefine,
                s.genService.cfg.AIAPIBaseURL,
                s.genService.cfg.AIAPIKey,
                s.genService.cfg.AIDefaultModel,
        )
        if err != nil {
                return nil, fmt.Errorf(
                        "获取新增页讨论AI配置失败: %w",
                        err,
                )
        }

        schoolID, _ :=
                repository.GetSchoolIDByUserID(
                        ctx,
                        scopedActor.UserID,
                )

        traceContext := &ai.TraceContext{
                SceneCode:
                        models.SceneCWPageRefine,
                UserID:
                        &scopedActor.UserID,
                SchoolID:
                        schoolIDPtr(
                                schoolID,
                        ),
        }

        result, err := ai.CallAI(
                aiConfig,
                cwAddPageDiscussionSystemPrompt,
                userPrompt,
                traceContext,
        )
        if err != nil {
                return nil, fmt.Errorf(
                        "新增页AI讨论失败: %w",
                        err,
                )
        }

        if result == nil ||
                strings.TrimSpace(
                        result.Content,
                ) == "" {
                return nil, fmt.Errorf(
                        "新增页AI讨论返回空内容",
                )
        }

        aiResponse, err :=
                parseCWAddPageDiscussionAIResponse(
                        result.Content,
                        currentPlan,
                )
        if err != nil {
                return nil, err
        }

        plan := mergeCWAddPagePlans(
                currentPlan,
                aiResponse.Plan,
        )

        plan = normalizeCWAddPagePlan(
                plan,
        )

        ready :=
                aiResponse.ReadyForConfirmation &&
                        isCWAddPagePlanReady(
                                plan,
                        )

        return &CoursewareAddPageDiscussionResponse{
                Reply: strings.TrimSpace(
                        aiResponse.Reply,
                ),
                Summary: strings.TrimSpace(
                        aiResponse.Summary,
                ),
                ReadyForConfirmation:
                        ready,
                InsertAt:
                        insertAt,
                Plan:
                        plan,
        }, nil
}

// normalizeCWAddPageDiscussionMessages 校验并规范化前端保存的历史消息。
func normalizeCWAddPageDiscussionMessages(
        messages []CoursewareAddPageDiscussionMessage,
) ([]CoursewareAddPageDiscussionMessage, error) {
        normalized := make(
                []CoursewareAddPageDiscussionMessage,
                0,
                len(messages),
        )

        for _, message := range messages {
                role := strings.ToLower(
                        strings.TrimSpace(
                                message.Role,
                        ),
                )

                switch role {
                case "teacher", "user":
                        role = "teacher"

                case "assistant":

                default:
                        return nil, fmt.Errorf(
                                "%w: 对话角色无效",
                                ErrCWAddPageDiscussionInvalidRequest,
                        )
                }

                content := strings.TrimSpace(
                        message.Content,
                )
                if content == "" {
                        continue
                }

                if utf8.RuneCountInString(content) >
                        cwAddPageDiscussionMaxMessageRunes {
                        return nil, fmt.Errorf(
                                "%w: 历史消息单条不能超过%d个字符",
                                ErrCWAddPageDiscussionTooLong,
                                cwAddPageDiscussionMaxMessageRunes,
                        )
                }

                normalized = append(
                        normalized,
                        CoursewareAddPageDiscussionMessage{
                                Role:
                                        role,
                                Content:
                                        content,
                        },
                )
        }

        return normalized, nil
}
