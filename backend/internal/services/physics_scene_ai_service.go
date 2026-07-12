package services

// physics_scene_ai_service.go — 力学场景 Matter.js setup 代码 AI 生成服务
//
// 生成目标不是 HTML，而是一段操作 Matter / engine / world / W / H 的 setup 构造代码。
// 前端把这段代码包装成临时 PhysicsTemplate，继续走 PhysicsSceneModal 的预览、播放、重置、融入链路。

import (
        "context"
        "fmt"
        "strings"

        aiClient "tedna/internal/ai"
        "tedna/internal/repository"
)

const (
        physicsSceneDescMaxRunes  = 2400
        physicsSceneBaseMaxRunes  = 50000
        physicsSceneImageMaxChars = 12000000
)

type PhysicsSceneGenInput struct {
        Mode         string
        Description  string
        BaseCode     string
        TemplateName string
        Image        string
}

func buildPhysicsSceneSystemPrompt(mode string, hasImage bool) string {
        var b strings.Builder

        b.WriteString("你是一名精通 Matter.js 0.20 的 K12 力学互动课件工程师。你的任务是生成一段可直接执行的 Matter.js 场景 setup 构造代码。\n\n")
        b.WriteString("【运行环境，必须严格遵守】\n")
        b.WriteString("1. 代码运行时已有变量：Matter、engine、world、W、H。\n")
        b.WriteString("2. engine 是 Matter.Engine 实例，world=engine.world，W/H 是画布宽高像素。\n")
        b.WriteString("3. 外层系统会在每次重置前执行 Matter.Composite.clear(world,false)，所以你的代码只负责重新构造场景。\n")
        b.WriteString("4. 绝对禁止创建 Engine、Render、Runner；禁止调用 Matter.Engine.create / Matter.Render.create / Matter.Runner.create。\n")
        b.WriteString("5. 绝对禁止输出 HTML、CSS、<script>、React、DOM 操作、document/window/fetch/import/require。\n")
        b.WriteString("6. 只输出纯 JavaScript 语句本身，不要 Markdown 代码围栏，不要解释文字。\n\n")

        b.WriteString("【代码硬规范】\n")
        b.WriteString("1. 字符串一律用单引号，变量一律用 var。\n")
        b.WriteString("2. 禁止出现 </script 这一字符序列。\n")
        b.WriteString("3. 所有位置和尺寸尽量基于 W/H 相对计算，适配 560/680/800 三档画布。\n")
        b.WriteString("4. 必须设置合理重力：engine.gravity.y = 1; 或按场景设 0/0.5/1.5。\n")
        b.WriteString("5. 必须使用 Matter.Composite.add(world, ...) 添加刚体或约束。\n")
        b.WriteString("6. 静态边界/地面/墙体必须 isStatic:true。\n")
        b.WriteString("7. 代码必须是纯构造，不依赖上次运行残留状态。\n")
        b.WriteString("8. 控制在 120 行以内，注释用简短中文单行注释。\n\n")

        b.WriteString("【可用 Matter.js API 范式】\n")
        b.WriteString("- Matter.Bodies.rectangle(x,y,w,h,opts)\n")
        b.WriteString("- Matter.Bodies.circle(x,y,r,opts)\n")
        b.WriteString("- Matter.Constraint.create({pointA:{x,y}, bodyB:body, length:L, stiffness:1})\n")
        b.WriteString("- Matter.Body.setVelocity(body,{x:...,y:...})\n")
        b.WriteString("- Matter.Body.setMass(body,m)\n")
        b.WriteString("- Matter.Composite.add(world, bodyOrArray)\n\n")

        b.WriteString("【教学场景要求】\n")
        b.WriteString("1. 适合 K12 力学：运动与力、重力、摩擦、斜面、碰撞、动量、能量、单摆、弹簧、分子热运动、机械波的简化粒子模型等。\n")
        b.WriteString("2. 画面要有明显教学对象：球、滑块、斜面、弹簧、摆、赛道、墙体、碰撞体等。\n")
        b.WriteString("3. 用颜色区分对象：蓝 #6C9BF0、珊瑚红 #EE7B70、薄荷绿 #5BBFA5、薰衣草紫 #9B8AE6、静态体 #94A3B8。\n")
        b.WriteString("4. 对比实验中可用 collisionFilter:{group:-1} 让多个对比物体互不碰撞，但仍与地面/墙体碰撞。\n")
        b.WriteString("5. 地面/墙体 friction/restitution 要合理，避免物体穿墙、飞出画布或刚开始就静止。\n\n")

        if hasImage {
                b.WriteString("【图片输入要求】\n")
                b.WriteString("用户附了题目图片、教材截图或物理示意图。你要先读图，提取物体、约束、初速度、力学关系和演示目标，再生成对应 Matter.js 场景。图片模糊时基于可辨认部分生成，少画不要臆造。\n\n")
        }

        if mode == "adapt" {
                b.WriteString("【本次任务：模板改编】\n")
                b.WriteString("用户会给你一个现有 Matter.js setup 底稿和改编要求。你要做最小必要修改：\n")
                b.WriteString("1. 保留底稿的基本版式、边界、教学结构。\n")
                b.WriteString("2. 按要求替换或新增刚体、约束、初速度、摩擦、恢复系数等。\n")
                b.WriteString("3. 如果用户给的是报错信息，优先修复语法/API错误，其余保持不动。\n")
                b.WriteString("4. 输出修改后的完整 setup 代码，不是差异片段。\n")
        } else {
                b.WriteString("【本次任务：从零生成】\n")
                b.WriteString("用户会描述一个新的力学场景或上传题图。你要从零生成完整 setup 代码，确保打开后能直接播放演示。\n")
        }

        return b.String()
}

func buildPhysicsSceneUserPrompt(in *PhysicsSceneGenInput) string {
        var b strings.Builder
        hasImage := strings.TrimSpace(in.Image) != ""

        if in.Mode == "adapt" {
                if strings.TrimSpace(in.TemplateName) != "" {
                        b.WriteString("【底稿模板名称】" + strings.TrimSpace(in.TemplateName) + "\n\n")
                }
                b.WriteString("【底稿 setup 代码，基于它做最小必要修改】\n")
                b.WriteString(in.BaseCode)
                if hasImage {
                        b.WriteString("\n\n【用户补充说明，图片也要参考】\n")
                } else {
                        b.WriteString("\n\n【用户要求的变化】\n")
                }
                b.WriteString(in.Description)
                b.WriteString("\n\n请输出修改后的完整 Matter.js setup 构造代码。")
        } else {
                if hasImage {
                        b.WriteString("【用户补充说明，主要内容见附图】\n")
                } else {
                        b.WriteString("【用户描述的新力学场景】\n")
                }
                b.WriteString(in.Description)
                b.WriteString("\n\n请输出完整 Matter.js setup 构造代码。")
        }

        return b.String()
}

func validatePhysicsSceneSetupCode(code string) error {
        trimmed := strings.TrimSpace(code)
        low := strings.ToLower(trimmed)

        if trimmed == "" {
                return fmt.Errorf("生成结果为空")
        }
        if strings.Contains(low, "<script") || strings.Contains(low, "<div") || strings.Contains(low, "<html") || strings.Contains(low, "<body") {
                return fmt.Errorf("生成结果包含 HTML，已拦截")
        }
        if strings.Contains(low, "</script") {
                return fmt.Errorf("生成结果包含非法片段 </script，已拦截")
        }
        if strings.Contains(low, "document.") || strings.Contains(low, "window.") || strings.Contains(low, "fetch(") || strings.Contains(low, "xmlhttprequest") || strings.Contains(low, "import ") || strings.Contains(low, "require(") {
                return fmt.Errorf("生成结果包含越界浏览器/API操作，已拦截")
        }
        if strings.Contains(low, "engine.create") || strings.Contains(low, "render.create") || strings.Contains(low, "runner.create") || strings.Contains(low, "matter.engine.create") || strings.Contains(low, "matter.render.create") || strings.Contains(low, "matter.runner.create") {
                return fmt.Errorf("生成结果试图创建 Engine/Render/Runner，已拦截")
        }
        if !strings.Contains(trimmed, "Matter.") {
                return fmt.Errorf("生成结果未使用 Matter API")
        }
        if !strings.Contains(trimmed, "Composite.add") {
                return fmt.Errorf("生成结果缺少 Matter.Composite.add(world, ...)")
        }
        return nil
}

func (s *MathGraphAIService) GeneratePhysicsScene(ctx context.Context, callerID string, in *PhysicsSceneGenInput) (string, error) {
        if in == nil {
                return "", fmt.Errorf("请求参数为空")
        }

        in.Mode = strings.TrimSpace(in.Mode)
        if in.Mode != "adapt" && in.Mode != "create" {
                return "", fmt.Errorf("mode 必须是 adapt 或 create")
        }

        in.Description = strings.TrimSpace(in.Description)
        in.Image = strings.TrimSpace(in.Image)
        hasImage := in.Image != ""

        if in.Description == "" && !hasImage {
                return "", fmt.Errorf("力学场景描述为空")
        }
        if in.Description == "" {
                in.Description = "请根据附图生成对应的 Matter.js 力学演示场景。"
        }
        if descRunes := []rune(in.Description); len(descRunes) > physicsSceneDescMaxRunes {
                in.Description = string(descRunes[:physicsSceneDescMaxRunes])
        }

        if hasImage {
                if !strings.HasPrefix(in.Image, "data:image/") {
                        return "", fmt.Errorf("图片格式不正确，需为 data:image/... 的 base64 数据")
                }
                if len(in.Image) > physicsSceneImageMaxChars {
                        return "", fmt.Errorf("图片过大，请压缩后重试")
                }
        }

        in.BaseCode = strings.TrimSpace(in.BaseCode)
        if in.Mode == "adapt" {
                if in.BaseCode == "" {
                        return "", fmt.Errorf("模板改编模式必须提供底稿 setup 代码 base_code")
                }
                if baseRunes := []rune(in.BaseCode); len(baseRunes) > physicsSceneBaseMaxRunes {
                        return "", fmt.Errorf("底稿代码过长，已超过 %d 字符", physicsSceneBaseMaxRunes)
                }
        }

        aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), mathGraphSceneCode, "", "", "")
        if err != nil {
                return "", fmt.Errorf("AI配置加载失败: %w", err)
        }

        systemPrompt := buildPhysicsSceneSystemPrompt(in.Mode, hasImage)
        userPrompt := buildPhysicsSceneUserPrompt(in)

        schoolID, _ := repository.GetSchoolIDByUserID(ctx, callerID)
        uid := callerID
        traceCtx := &aiClient.TraceContext{
                SceneCode: mathGraphSceneCode,
                UserID:    &uid,
                SchoolID:  schoolIDPtr(schoolID),
        }

        var result *aiClient.CallResult
        if hasImage {
                result, err = aiClient.CallAIMultimodal(aiCfg, systemPrompt, userPrompt, in.Image, traceCtx)
        } else {
                result, err = aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
        }
        if err != nil {
                return "", fmt.Errorf("力学场景生成失败: %w", err)
        }

        code := stripMathGraphFences(result.Content)
        if err := validatePhysicsSceneSetupCode(code); err != nil {
                return "", err
        }

        mathGraphLog.Info("力学场景setup代码生成完成",
                "mode", in.Mode,
                "caller", callerID,
                "template", in.TemplateName,
                "with_image", hasImage,
                "desc_len", len([]rune(in.Description)),
                "code_len", len([]rune(code)),
                "tokens", result.TokensUsed,
        )

        return code, nil
}
