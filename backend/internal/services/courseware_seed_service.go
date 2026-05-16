package services

import (
	"context"
	"log"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 课件工坊种子数据服务 ====================
// Phase 2: 批量填充课件组件（~24个）和风格模板（12套）
// admin通过接口一键调用，幂等执行（先清空再写入）

// CoursewareSeedService 种子数据服务
type CoursewareSeedService struct{}

// NewCoursewareSeedService 创建种子数据服务
func NewCoursewareSeedService() *CoursewareSeedService {
	return &CoursewareSeedService{}
}

// ==================== 种子数据填充入口 ====================

// SeedResult 种子数据填充结果
type SeedResult struct {
	ComponentsCreated int `json:"components_created"` // 创建的组件数
	TemplatesCreated  int `json:"templates_created"`  // 创建的模板数
	Errors            []string `json:"errors,omitempty"`
}

// SeedAll 一键填充全部种子数据（组件+模板）
// 幂等操作：先检查数量，已有数据时跳过（除非force=true则先清空）
func (s *CoursewareSeedService) SeedAll(ctx context.Context, force bool) (*SeedResult, error) {
	result := &SeedResult{}

	// 组件种子
	compCount, err := s.seedComponents(ctx, force)
	if err != nil {
		result.Errors = append(result.Errors, "组件种子失败: "+err.Error())
	}
	result.ComponentsCreated = compCount

	// 模板种子
	tmplCount, err := s.seedTemplates(ctx, force)
	if err != nil {
		result.Errors = append(result.Errors, "模板种子失败: "+err.Error())
	}
	result.TemplatesCreated = tmplCount

	return result, nil
}

// ==================== 组件种子数据 ====================

func (s *CoursewareSeedService) seedComponents(ctx context.Context, force bool) (int, error) {
	// 检查已有数据
	existing, total, err := repository.ListCWComponents(ctx, "", "", "", nil, 1, 0)
	_ = existing
	if err == nil && total > 0 && !force {
		log.Printf("[CW种子] 已有 %d 个组件，跳过（可传 force=true 强制重建）", total)
		return 0, nil
	}

	// force模式先清空（通过逐个删除实现，避免直接TRUNCATE）
	if force && total > 0 {
		log.Printf("[CW种子] force模式，清空现有 %d 个组件", total)
		allComps, _, _ := repository.ListCWComponents(ctx, "", "", "", nil, 500, 0)
		for _, c := range allComps {
			_ = repository.DeleteCWComponent(ctx, c.ID)
		}
	}

	components := buildSeedComponents()
	created := 0
	for _, comp := range components {
		if err := repository.CreateCWComponent(ctx, comp); err != nil {
			log.Printf("[CW种子] 创建组件失败 %s: %v", comp.Name, err)
			continue
		}
		created++
	}
	log.Printf("[CW种子] 组件创建完成: %d/%d", created, len(components))
	return created, nil
}

// buildSeedComponents 构建全部种子组件
// 6大类 × 每类3-5个 = 约24个核心组件
func buildSeedComponents() []*models.CoursewareComponent {
	il := func(v int) *int { return &v }
	var comps []*models.CoursewareComponent

	// ============ 1. 布局模板 (layout) — 5个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "标题+正文布局", Description: "经典的大标题+居中正文布局，适用于课程导入和知识讲解页面",
		ComponentType: "layout", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FR", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;display:flex;flex-direction:column;justify-content:center;align-items:center;padding:60px 80px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC)}
.title{font-size:42px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:32px;text-align:center;line-height:1.3;font-family:var(--cw-font-heading,system-ui,sans-serif)}
.content{font-size:28px;color:var(--cw-text,#1E293B);line-height:1.8;text-align:center;max-width:800px}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="content">{{CONTENT}}</div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;display:flex;flex-direction:column;justify-content:center;align-items:center;padding:20px;background:#F8FAFC"><h1 style="font-size:18px;color:#2563EB;margin-bottom:12px">标题区域</h1><p style="font-size:12px;color:#1E293B;text-align:center">正文内容区域，支持多行文本展示</p></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "左图右文布局", Description: "左侧展示图片/插图，右侧展示文字说明，适用于概念讲解",
		ComponentType: "layout", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CP", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;display:flex;align-items:center;padding:40px 60px;gap:50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC)}
.img-area{flex:1;height:380px;border-radius:var(--cw-radius,12px);background:var(--cw-secondary,#60A5FA);display:flex;align-items:center;justify-content:center;color:#fff;font-size:28px;overflow:hidden}
.img-area img{width:100%;height:100%;object-fit:cover}
.text-area{flex:1;display:flex;flex-direction:column;gap:20px}
.text-area h2{font-size:36px;font-weight:700;color:var(--cw-primary,#2563EB);font-family:var(--cw-font-heading,system-ui,sans-serif)}
.text-area p{font-size:24px;color:var(--cw-text,#1E293B);line-height:1.8}
</style></head><body>
<div class="page">
  <div class="img-area">{{IMG_PLACEHOLDER_01}}</div>
  <div class="text-area">
    <h2>{{PAGE_TITLE}}</h2>
    <p>{{CONTENT}}</p>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;display:flex;align-items:center;padding:12px;gap:12px;background:#F8FAFC"><div style="flex:1;height:80%;border-radius:8px;background:linear-gradient(135deg,#60A5FA,#2563EB);display:flex;align-items:center;justify-content:center;color:#fff;font-size:11px">图片区</div><div style="flex:1"><div style="font-size:14px;font-weight:600;color:#2563EB;margin-bottom:6px">标题</div><div style="font-size:10px;color:#64748B;line-height:1.6">文字说明内容区域</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "双栏对比布局", Description: "左右两栏对比展示，适用于概念对比、优劣分析等",
		ComponentType: "layout", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CP", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.page-title{font-size:36px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:30px;font-family:var(--cw-font-heading,system-ui,sans-serif)}
.compare{display:flex;gap:30px;flex:1}
.col{flex:1;background:#fff;border-radius:var(--cw-radius,12px);padding:28px;box-shadow:var(--cw-shadow,0 2px 8px rgba(0,0,0,0.08))}
.col h3{font-size:28px;font-weight:600;margin-bottom:16px;padding-bottom:12px;border-bottom:3px solid var(--cw-primary,#2563EB)}
.col.right h3{border-color:var(--cw-accent,#F59E0B)}
.col p{font-size:22px;color:var(--cw-text,#1E293B);line-height:1.7}
</style></head><body>
<div class="page">
  <h1 class="page-title">{{PAGE_TITLE}}</h1>
  <div class="compare">
    <div class="col"><h3>{{LEFT_TITLE}}</h3><p>{{LEFT_CONTENT}}</p></div>
    <div class="col right"><h3>{{RIGHT_TITLE}}</h3><p>{{RIGHT_CONTENT}}</p></div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:12px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:14px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:10px">对比标题</div><div style="display:flex;gap:8px;flex:1"><div style="flex:1;background:#fff;border-radius:8px;padding:8px;box-shadow:0 1px 4px rgba(0,0,0,0.06)"><div style="font-size:11px;font-weight:600;color:#2563EB;border-bottom:2px solid #2563EB;padding-bottom:4px;margin-bottom:4px">左栏</div><div style="font-size:9px;color:#64748B">内容A</div></div><div style="flex:1;background:#fff;border-radius:8px;padding:8px;box-shadow:0 1px 4px rgba(0,0,0,0.06)"><div style="font-size:11px;font-weight:600;color:#F59E0B;border-bottom:2px solid #F59E0B;padding-bottom:4px;margin-bottom:4px">右栏</div><div style="font-size:9px;color:#64748B">内容B</div></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "卡片网格布局", Description: "多个卡片网格排列，适用于知识点罗列、分类展示",
		ComponentType: "layout", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "GD", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC)}
.page-title{font-size:36px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:30px}
.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:20px}
.card{background:#fff;border-radius:var(--cw-radius,12px);padding:24px;box-shadow:var(--cw-shadow,0 2px 8px rgba(0,0,0,0.08));text-align:center}
.card-icon{font-size:36px;margin-bottom:12px}
.card-title{font-size:22px;font-weight:600;color:var(--cw-primary,#2563EB);margin-bottom:8px}
.card-desc{font-size:18px;color:var(--cw-text,#1E293B);line-height:1.6}
</style></head><body>
<div class="page">
  <h1 class="page-title">{{PAGE_TITLE}}</h1>
  <div class="grid">
    <div class="card"><div class="card-icon">📌</div><div class="card-title">要点一</div><div class="card-desc">说明文字</div></div>
    <div class="card"><div class="card-icon">💡</div><div class="card-title">要点二</div><div class="card-desc">说明文字</div></div>
    <div class="card"><div class="card-icon">🔍</div><div class="card-title">要点三</div><div class="card-desc">说明文字</div></div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC"><div style="font-size:13px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:8px">网格标题</div><div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6px"><div style="background:#fff;border-radius:6px;padding:8px;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,0.06)"><div style="font-size:16px">📌</div><div style="font-size:9px;font-weight:600;color:#2563EB">要点一</div></div><div style="background:#fff;border-radius:6px;padding:8px;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,0.06)"><div style="font-size:16px">💡</div><div style="font-size:9px;font-weight:600;color:#2563EB">要点二</div></div><div style="background:#fff;border-radius:6px;padding:8px;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,0.06)"><div style="font-size:16px">🔍</div><div style="font-size:9px;font-weight:600;color:#2563EB">要点三</div></div></div></div>`,
	})

	// ============ 2. 交互功能 (interaction) — 5个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "单选题交互", Description: "4选1单选题，点击选项即时反馈对错，支持3次重试，适用于课堂练习",
		ComponentType: "interaction", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(3), IdxVisualFormat: "CG", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.question{font-size:30px;font-weight:600;color:var(--cw-text,#1E293B);margin-bottom:30px;line-height:1.5}
.options{display:flex;flex-direction:column;gap:14px;flex:1}
.opt{padding:18px 24px;border-radius:var(--cw-radius,12px);border:2px solid #E2E8F0;background:#fff;font-size:24px;color:var(--cw-text,#1E293B);cursor:pointer;transition:all .2s;display:flex;align-items:center;gap:14px}
.opt:hover{border-color:var(--cw-primary,#2563EB);background:rgba(37,99,235,0.04)}
.opt.correct{border-color:#22C55E;background:#F0FDF4;color:#166534}
.opt.wrong{border-color:#EF4444;background:#FEF2F2;color:#991B1B}
.opt.disabled{pointer-events:none;opacity:.6}
.opt-label{width:32px;height:32px;border-radius:50%;background:var(--cw-primary,#2563EB);color:#fff;display:flex;align-items:center;justify-content:center;font-weight:600;font-size:16px;flex-shrink:0}
.feedback{margin-top:16px;padding:14px 20px;border-radius:8px;font-size:22px;text-align:center;display:none}
.feedback.show{display:block}
.feedback.ok{background:#F0FDF4;color:#166534}
.feedback.fail{background:#FEF2F2;color:#991B1B}
</style></head><body>
<div class="page">
  <div class="question">{{QUESTION}}</div>
  <div class="options" id="opts">
    <div class="opt" data-idx="0" onclick="check(this,0)"><span class="opt-label">A</span><span>{{OPTION_A}}</span></div>
    <div class="opt" data-idx="1" onclick="check(this,1)"><span class="opt-label">B</span><span>{{OPTION_B}}</span></div>
    <div class="opt" data-idx="2" onclick="check(this,2)"><span class="opt-label">C</span><span>{{OPTION_C}}</span></div>
    <div class="opt" data-idx="3" onclick="check(this,3)"><span class="opt-label">D</span><span>{{OPTION_D}}</span></div>
  </div>
  <div class="feedback" id="fb"></div>
</div>
<script>
var correctIdx={{CORRECT_INDEX}},tries=0,maxTries=3,done=false;
function check(el,idx){
  if(done)return;
  tries++;
  var fb=document.getElementById('fb');
  fb.classList.add('show');
  if(idx===correctIdx){
    el.classList.add('correct');done=true;
    fb.className='feedback show ok';fb.textContent='✅ 回答正确！';
    disableAll();
  }else{
    el.classList.add('wrong','disabled');
    if(tries>=maxTries){
      done=true;fb.className='feedback show fail';fb.textContent='已用完'+maxTries+'次机会，正确答案是 '+['A','B','C','D'][correctIdx];
      document.querySelectorAll('.opt')[correctIdx].classList.add('correct');disableAll();
    }else{
      fb.className='feedback show fail';fb.textContent='❌ 不对哦，还有'+(maxTries-tries)+'次机会';
    }
  }
}
function disableAll(){document.querySelectorAll('.opt').forEach(function(o){o.classList.add('disabled')})}
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:12px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:12px;font-weight:600;color:#1E293B;margin-bottom:10px">❓ 题目文本区域</div><div style="display:flex;flex-direction:column;gap:5px;flex:1"><div style="padding:6px 10px;border-radius:6px;border:1.5px solid #E2E8F0;background:#fff;font-size:10px;display:flex;align-items:center;gap:6px"><span style="width:18px;height:18px;border-radius:50%;background:#2563EB;color:#fff;display:flex;align-items:center;justify-content:center;font-size:8px;flex-shrink:0">A</span>选项A</div><div style="padding:6px 10px;border-radius:6px;border:1.5px solid #E2E8F0;background:#fff;font-size:10px;display:flex;align-items:center;gap:6px"><span style="width:18px;height:18px;border-radius:50%;background:#2563EB;color:#fff;display:flex;align-items:center;justify-content:center;font-size:8px;flex-shrink:0">B</span>选项B</div><div style="padding:6px 10px;border-radius:6px;border:1.5px solid #E2E8F0;background:#fff;font-size:10px;display:flex;align-items:center;gap:6px"><span style="width:18px;height:18px;border-radius:50%;background:#2563EB;color:#fff;display:flex;align-items:center;justify-content:center;font-size:8px;flex-shrink:0">C</span>选项C</div><div style="padding:6px 10px;border-radius:6px;border:1.5px solid #E2E8F0;background:#fff;font-size:10px;display:flex;align-items:center;gap:6px"><span style="width:18px;height:18px;border-radius:50%;background:#2563EB;color:#fff;display:flex;align-items:center;justify-content:center;font-size:8px;flex-shrink:0">D</span>选项D</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "拖拽排序练习", Description: "将打乱的选项拖拽到正确顺序，适用于步骤排序、时间线排列",
		ComponentType: "interaction", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(3), IdxVisualFormat: "TL", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:10px}
.hint{font-size:22px;color:#64748B;margin-bottom:24px}
.items{display:flex;flex-direction:column;gap:12px;flex:1}
.item{padding:16px 24px;background:#fff;border-radius:var(--cw-radius,12px);border:2px solid #E2E8F0;font-size:24px;color:var(--cw-text,#1E293B);cursor:grab;user-select:none;display:flex;align-items:center;gap:14px;transition:all .2s}
.item:active{cursor:grabbing;box-shadow:0 4px 16px rgba(0,0,0,0.12)}
.item.dragging{opacity:.5}
.item .num{width:32px;height:32px;border-radius:50%;background:#E2E8F0;display:flex;align-items:center;justify-content:center;font-weight:700;font-size:16px;color:#64748B;flex-shrink:0}
.check-btn{margin-top:16px;padding:14px 40px;border:none;border-radius:8px;background:var(--cw-primary,#2563EB);color:#fff;font-size:22px;font-weight:600;cursor:pointer;align-self:center}
.feedback{margin-top:12px;text-align:center;font-size:22px;min-height:30px}
</style></head><body>
<div class="page">
  <div class="title">{{PAGE_TITLE}}</div>
  <div class="hint">拖拽下方选项排列成正确顺序</div>
  <div class="items" id="sortable">
    <div class="item" draggable="true" data-order="{{ORDER_0}}"><span class="num">1</span>{{ITEM_0}}</div>
    <div class="item" draggable="true" data-order="{{ORDER_1}}"><span class="num">2</span>{{ITEM_1}}</div>
    <div class="item" draggable="true" data-order="{{ORDER_2}}"><span class="num">3</span>{{ITEM_2}}</div>
    <div class="item" draggable="true" data-order="{{ORDER_3}}"><span class="num">4</span>{{ITEM_3}}</div>
  </div>
  <button class="check-btn" onclick="checkOrder()">检查答案</button>
  <div class="feedback" id="fb"></div>
</div>
<script>
var container=document.getElementById('sortable'),dragEl=null;
container.addEventListener('dragstart',function(e){dragEl=e.target.closest('.item');dragEl.classList.add('dragging')});
container.addEventListener('dragend',function(){if(dragEl)dragEl.classList.remove('dragging');dragEl=null});
container.addEventListener('dragover',function(e){e.preventDefault();var after=getDragAfterElement(e.clientY);if(after){container.insertBefore(dragEl,after)}else{container.appendChild(dragEl)}updateNums()});
function getDragAfterElement(y){var els=[].slice.call(container.querySelectorAll('.item:not(.dragging)'));var closest=null,offset=Number.NEGATIVE_INFINITY;els.forEach(function(el){var box=el.getBoundingClientRect();var o=y-box.top-box.height/2;if(o<0&&o>offset){offset=o;closest=el}});return closest}
function updateNums(){container.querySelectorAll('.item').forEach(function(el,i){el.querySelector('.num').textContent=i+1})}
function checkOrder(){var items=container.querySelectorAll('.item');var correct=true;items.forEach(function(el,i){if(parseInt(el.dataset.order)!==i)correct=false});var fb=document.getElementById('fb');fb.textContent=correct?'✅ 顺序正确！':'❌ 顺序还不对，再试试';fb.style.color=correct?'#166534':'#991B1B'}
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:12px;font-weight:600;color:#2563EB;margin-bottom:4px">拖拽排序</div><div style="font-size:9px;color:#94A3B8;margin-bottom:8px">拖拽排列正确顺序</div><div style="display:flex;flex-direction:column;gap:4px;flex:1"><div style="padding:5px 8px;background:#fff;border-radius:4px;border:1px solid #E2E8F0;font-size:9px;display:flex;align-items:center;gap:4px"><span style="width:14px;height:14px;border-radius:50%;background:#E2E8F0;display:flex;align-items:center;justify-content:center;font-size:7px;font-weight:700;color:#64748B">1</span>项目一</div><div style="padding:5px 8px;background:#fff;border-radius:4px;border:1px solid #E2E8F0;font-size:9px;display:flex;align-items:center;gap:4px"><span style="width:14px;height:14px;border-radius:50%;background:#E2E8F0;display:flex;align-items:center;justify-content:center;font-size:7px;font-weight:700;color:#64748B">2</span>项目二</div><div style="padding:5px 8px;background:#fff;border-radius:4px;border:1px solid #E2E8F0;font-size:9px;display:flex;align-items:center;gap:4px"><span style="width:14px;height:14px;border-radius:50%;background:#E2E8F0;display:flex;align-items:center;justify-content:center;font-size:7px;font-weight:700;color:#64748B">3</span>项目三</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "翻卡片记忆", Description: "点击卡片翻转显示背面答案，适用于词汇记忆、概念配对",
		ComponentType: "interaction", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(2), IdxVisualFormat: "CG", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC)}
.title{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:30px}
.cards{display:grid;grid-template-columns:repeat(3,1fr);gap:20px}
.flip-card{height:180px;perspective:1000px;cursor:pointer}
.flip-inner{width:100%;height:100%;transition:transform .6s;transform-style:preserve-3d;position:relative}
.flip-card.flipped .flip-inner{transform:rotateY(180deg)}
.flip-front,.flip-back{position:absolute;width:100%;height:100%;backface-visibility:hidden;border-radius:var(--cw-radius,12px);display:flex;align-items:center;justify-content:center;padding:16px;text-align:center}
.flip-front{background:var(--cw-primary,#2563EB);color:#fff;font-size:26px;font-weight:600}
.flip-back{background:#fff;border:2px solid var(--cw-primary,#2563EB);color:var(--cw-text,#1E293B);font-size:22px;transform:rotateY(180deg);line-height:1.5}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="cards">
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_1}}</div><div class="flip-back">{{BACK_1}}</div></div></div>
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_2}}</div><div class="flip-back">{{BACK_2}}</div></div></div>
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_3}}</div><div class="flip-back">{{BACK_3}}</div></div></div>
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_4}}</div><div class="flip-back">{{BACK_4}}</div></div></div>
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_5}}</div><div class="flip-back">{{BACK_5}}</div></div></div>
    <div class="flip-card" onclick="this.classList.toggle('flipped')"><div class="flip-inner"><div class="flip-front">{{FRONT_6}}</div><div class="flip-back">{{BACK_6}}</div></div></div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:8px">翻卡片记忆</div><div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6px"><div style="height:45px;background:#2563EB;border-radius:6px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:9px;font-weight:600">正面1</div><div style="height:45px;background:#2563EB;border-radius:6px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:9px;font-weight:600">正面2</div><div style="height:45px;background:#2563EB;border-radius:6px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:9px;font-weight:600">正面3</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "填空题交互", Description: "在文段中留空让学生填写答案，支持即时校验和提示",
		ComponentType: "interaction", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(3), IdxVisualFormat: "FR", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:50px 80px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column;justify-content:center}
.title{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:30px;text-align:center}
.passage{font-size:26px;color:var(--cw-text,#1E293B);line-height:2.2;text-align:center}
.blank{display:inline-block;min-width:120px;border-bottom:3px solid var(--cw-primary,#2563EB);padding:2px 8px;font-size:26px;text-align:center;outline:none;color:var(--cw-primary,#2563EB);font-weight:600;background:transparent;transition:border-color .2s}
.blank.correct{border-color:#22C55E;color:#166534}
.blank.wrong{border-color:#EF4444;color:#991B1B}
.check-btn{margin-top:30px;padding:14px 40px;border:none;border-radius:8px;background:var(--cw-primary,#2563EB);color:#fff;font-size:22px;font-weight:600;cursor:pointer;align-self:center}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="passage">{{TEXT_BEFORE}} <input class="blank" id="b1" data-answer="{{ANSWER}}" placeholder="?" /> {{TEXT_AFTER}}</div>
  <button class="check-btn" onclick="checkBlanks()">检查答案</button>
</div>
<script>
function checkBlanks(){
  document.querySelectorAll('.blank').forEach(function(el){
    var ans=el.dataset.answer.trim().toLowerCase();
    var val=el.value.trim().toLowerCase();
    el.classList.remove('correct','wrong');
    el.classList.add(val===ans?'correct':'wrong');
  });
}
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:14px;background:#F8FAFC;display:flex;flex-direction:column;justify-content:center;align-items:center"><div style="font-size:12px;font-weight:600;color:#2563EB;margin-bottom:10px">填空题</div><div style="font-size:10px;color:#1E293B;text-align:center">文本内容 <span style="display:inline-block;min-width:50px;border-bottom:2px solid #2563EB;color:#2563EB;font-weight:600">____</span> 后续文本</div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "步骤引导交互", Description: "分步展示内容，点击按钮逐步推进，适用于实验步骤、操作流程",
		ComponentType: "interaction", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(2), IdxVisualFormat: "TL", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:20px;text-align:center}
.progress{display:flex;gap:8px;justify-content:center;margin-bottom:24px}
.dot{width:12px;height:12px;border-radius:50%;background:#E2E8F0;transition:all .3s}
.dot.active{background:var(--cw-primary,#2563EB);transform:scale(1.3)}
.dot.done{background:#22C55E}
.step-content{flex:1;display:flex;align-items:center;justify-content:center;background:#fff;border-radius:var(--cw-radius,12px);padding:30px;box-shadow:var(--cw-shadow,0 2px 8px rgba(0,0,0,0.08));text-align:center}
.step-content h3{font-size:28px;font-weight:600;color:var(--cw-primary,#2563EB);margin-bottom:12px}
.step-content p{font-size:24px;color:var(--cw-text,#1E293B);line-height:1.7}
.nav{display:flex;justify-content:center;gap:20px;margin-top:20px}
.nav button{padding:12px 32px;border-radius:8px;font-size:20px;font-weight:600;cursor:pointer;border:2px solid var(--cw-primary,#2563EB);transition:all .2s}
.nav .prev{background:transparent;color:var(--cw-primary,#2563EB)}
.nav .next{background:var(--cw-primary,#2563EB);color:#fff;border-color:var(--cw-primary,#2563EB)}
.nav button:disabled{opacity:.4;cursor:default}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="progress" id="prog"></div>
  <div class="step-content" id="content"></div>
  <div class="nav"><button class="prev" id="prevBtn" onclick="go(-1)">← 上一步</button><button class="next" id="nextBtn" onclick="go(1)">下一步 →</button></div>
</div>
<script>
var steps=[{title:"步骤一",text:"{{STEP1_TEXT}}"},{title:"步骤二",text:"{{STEP2_TEXT}}"},{title:"步骤三",text:"{{STEP3_TEXT}}"},{title:"步骤四",text:"{{STEP4_TEXT}}"}];
var cur=0;
function render(){
  var prog=document.getElementById('prog');prog.innerHTML='';
  steps.forEach(function(_,i){var d=document.createElement('div');d.className='dot'+(i===cur?' active':i<cur?' done':'');prog.appendChild(d)});
  var c=document.getElementById('content');c.innerHTML='<div><h3>'+steps[cur].title+'</h3><p>'+steps[cur].text+'</p></div>';
  document.getElementById('prevBtn').disabled=cur===0;
  document.getElementById('nextBtn').disabled=cur===steps.length-1;
  if(cur===steps.length-1)document.getElementById('nextBtn').textContent='完成 ✓';
  else document.getElementById('nextBtn').textContent='下一步 →';
}
function go(d){cur=Math.max(0,Math.min(steps.length-1,cur+d));render()}
render();
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:6px">步骤引导</div><div style="display:flex;gap:4px;justify-content:center;margin-bottom:8px"><div style="width:8px;height:8px;border-radius:50%;background:#2563EB"></div><div style="width:8px;height:8px;border-radius:50%;background:#E2E8F0"></div><div style="width:8px;height:8px;border-radius:50%;background:#E2E8F0"></div></div><div style="flex:1;background:#fff;border-radius:6px;display:flex;align-items:center;justify-content:center;font-size:10px;color:#64748B;box-shadow:0 1px 4px rgba(0,0,0,0.06)">步骤内容区</div></div>`,
	})

	// ============ 3. 数据可视化 (data_viz) — 4个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "时间线展示", Description: "水平时间线，展示历史事件或发展阶段，适用于历史、科技发展",
		ComponentType: "data_viz", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "TL", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:34px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:40px}
.timeline{position:relative;display:flex;justify-content:space-between;align-items:flex-start;flex:1;padding-top:30px}
.timeline::before{content:'';position:absolute;top:42px;left:40px;right:40px;height:4px;background:linear-gradient(90deg,var(--cw-primary,#2563EB),var(--cw-accent,#F59E0B))}
.tl-item{display:flex;flex-direction:column;align-items:center;width:180px;position:relative;z-index:1}
.tl-dot{width:20px;height:20px;border-radius:50%;background:var(--cw-primary,#2563EB);border:4px solid #fff;box-shadow:0 0 0 3px var(--cw-primary,#2563EB);margin-bottom:16px}
.tl-year{font-size:20px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:8px}
.tl-text{font-size:18px;color:var(--cw-text,#1E293B);text-align:center;line-height:1.5}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="timeline">
    <div class="tl-item"><div class="tl-dot"></div><div class="tl-year">{{YEAR_1}}</div><div class="tl-text">{{EVENT_1}}</div></div>
    <div class="tl-item"><div class="tl-dot"></div><div class="tl-year">{{YEAR_2}}</div><div class="tl-text">{{EVENT_2}}</div></div>
    <div class="tl-item"><div class="tl-dot"></div><div class="tl-year">{{YEAR_3}}</div><div class="tl-text">{{EVENT_3}}</div></div>
    <div class="tl-item"><div class="tl-dot"></div><div class="tl-year">{{YEAR_4}}</div><div class="tl-text">{{EVENT_4}}</div></div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:12px">时间线</div><div style="position:relative;display:flex;justify-content:space-around;padding-top:10px"><div style="position:absolute;top:14px;left:15%;right:15%;height:2px;background:linear-gradient(90deg,#2563EB,#F59E0B)"></div><div style="text-align:center;position:relative;z-index:1"><div style="width:8px;height:8px;border-radius:50%;background:#2563EB;margin:0 auto 4px"></div><div style="font-size:8px;font-weight:700;color:#2563EB">2020</div></div><div style="text-align:center;position:relative;z-index:1"><div style="width:8px;height:8px;border-radius:50%;background:#2563EB;margin:0 auto 4px"></div><div style="font-size:8px;font-weight:700;color:#2563EB">2022</div></div><div style="text-align:center;position:relative;z-index:1"><div style="width:8px;height:8px;border-radius:50%;background:#2563EB;margin:0 auto 4px"></div><div style="font-size:8px;font-weight:700;color:#2563EB">2024</div></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "进度/比例展示", Description: "圆形或条形进度指示器，展示完成度、比例数据",
		ComponentType: "data_viz", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CH", IdxTechTag: "SVG",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column;align-items:center}
.title{font-size:34px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:40px}
.charts{display:flex;gap:60px;justify-content:center;align-items:center;flex:1}
.chart-item{text-align:center}
.ring{transform:rotate(-90deg)}
.ring-bg{fill:none;stroke:#E2E8F0;stroke-width:10}
.ring-fg{fill:none;stroke:var(--cw-primary,#2563EB);stroke-width:10;stroke-linecap:round;transition:stroke-dashoffset 1s ease}
.chart-label{font-size:22px;font-weight:600;color:var(--cw-text,#1E293B);margin-top:12px}
.chart-value{font-size:36px;font-weight:700;color:var(--cw-primary,#2563EB);position:absolute;top:50%;left:50%;transform:translate(-50%,-50%)}
.chart-wrap{position:relative;display:inline-block}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="charts">
    <div class="chart-item">
      <div class="chart-wrap"><svg class="ring" width="140" height="140"><circle class="ring-bg" cx="70" cy="70" r="55"/><circle class="ring-fg" cx="70" cy="70" r="55" stroke-dasharray="345.6" stroke-dashoffset="103.7"/></svg><div class="chart-value">70%</div></div>
      <div class="chart-label">{{LABEL_1}}</div>
    </div>
    <div class="chart-item">
      <div class="chart-wrap"><svg class="ring" width="140" height="140"><circle class="ring-bg" cx="70" cy="70" r="55"/><circle class="ring-fg" cx="70" cy="70" r="55" stroke-dasharray="345.6" stroke-dashoffset="172.8" style="stroke:var(--cw-accent,#F59E0B)"/></svg><div class="chart-value">50%</div></div>
      <div class="chart-label">{{LABEL_2}}</div>
    </div>
    <div class="chart-item">
      <div class="chart-wrap"><svg class="ring" width="140" height="140"><circle class="ring-bg" cx="70" cy="70" r="55"/><circle class="ring-fg" cx="70" cy="70" r="55" stroke-dasharray="345.6" stroke-dashoffset="34.6" style="stroke:#22C55E"/></svg><div class="chart-value">90%</div></div>
      <div class="chart-label">{{LABEL_3}}</div>
    </div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC;display:flex;flex-direction:column;align-items:center"><div style="font-size:12px;font-weight:600;color:#2563EB;margin-bottom:8px">进度展示</div><div style="display:flex;gap:16px;justify-content:center"><div style="text-align:center"><svg width="40" height="40" style="transform:rotate(-90deg)"><circle cx="20" cy="20" r="15" fill="none" stroke="#E2E8F0" stroke-width="4"/><circle cx="20" cy="20" r="15" fill="none" stroke="#2563EB" stroke-width="4" stroke-dasharray="94.2" stroke-dashoffset="28.3" stroke-linecap="round"/></svg><div style="font-size:8px;color:#64748B;margin-top:2px">70%</div></div><div style="text-align:center"><svg width="40" height="40" style="transform:rotate(-90deg)"><circle cx="20" cy="20" r="15" fill="none" stroke="#E2E8F0" stroke-width="4"/><circle cx="20" cy="20" r="15" fill="none" stroke="#F59E0B" stroke-width="4" stroke-dasharray="94.2" stroke-dashoffset="47.1" stroke-linecap="round"/></svg><div style="font-size:8px;color:#64748B;margin-top:2px">50%</div></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "柱状图对比", Description: "CSS纯实现柱状图，展示数据对比，无外部依赖",
		ComponentType: "data_viz", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CH", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:34px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:30px}
.chart{display:flex;align-items:flex-end;justify-content:center;gap:40px;flex:1;padding-bottom:40px;border-bottom:3px solid #E2E8F0;position:relative}
.bar-group{display:flex;flex-direction:column;align-items:center;gap:8px}
.bar{width:60px;border-radius:8px 8px 0 0;transition:height 1s ease;min-height:10px}
.bar-val{font-size:20px;font-weight:700;color:var(--cw-text,#1E293B)}
.bar-label{font-size:18px;color:#64748B;margin-top:12px}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="chart">
    <div class="bar-group"><div class="bar-val">{{VAL_1}}</div><div class="bar" style="height:{{HEIGHT_1}}px;background:var(--cw-primary,#2563EB)"></div><div class="bar-label">{{LABEL_1}}</div></div>
    <div class="bar-group"><div class="bar-val">{{VAL_2}}</div><div class="bar" style="height:{{HEIGHT_2}}px;background:var(--cw-secondary,#60A5FA)"></div><div class="bar-label">{{LABEL_2}}</div></div>
    <div class="bar-group"><div class="bar-val">{{VAL_3}}</div><div class="bar" style="height:{{HEIGHT_3}}px;background:var(--cw-accent,#F59E0B)"></div><div class="bar-label">{{LABEL_3}}</div></div>
    <div class="bar-group"><div class="bar-val">{{VAL_4}}</div><div class="bar" style="height:{{HEIGHT_4}}px;background:#22C55E"></div><div class="bar-label">{{LABEL_4}}</div></div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:8px">柱状图</div><div style="display:flex;align-items:flex-end;justify-content:center;gap:12px;height:60%;border-bottom:2px solid #E2E8F0;padding-bottom:4px"><div style="width:20px;height:70%;background:#2563EB;border-radius:3px 3px 0 0"></div><div style="width:20px;height:50%;background:#60A5FA;border-radius:3px 3px 0 0"></div><div style="width:20px;height:85%;background:#F59E0B;border-radius:3px 3px 0 0"></div><div style="width:20px;height:40%;background:#22C55E;border-radius:3px 3px 0 0"></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "思维导图/树形展示", Description: "中心发散的思维导图布局，适用于知识结构梳理",
		ComponentType: "data_viz", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FR", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:30px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;align-items:center;justify-content:center}
.mind{position:relative;width:100%;height:100%}
.center{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);background:var(--cw-primary,#2563EB);color:#fff;padding:20px 32px;border-radius:50%;font-size:24px;font-weight:700;text-align:center;z-index:2;min-width:160px;min-height:100px;display:flex;align-items:center;justify-content:center}
.branch{position:absolute;background:#fff;padding:14px 20px;border-radius:var(--cw-radius,12px);box-shadow:var(--cw-shadow,0 2px 8px rgba(0,0,0,0.08));font-size:20px;color:var(--cw-text,#1E293B);border-left:4px solid;text-align:center;z-index:2}
.b1{top:8%;left:10%;border-color:var(--cw-primary,#2563EB)}.b2{top:8%;right:10%;border-color:var(--cw-accent,#F59E0B)}
.b3{bottom:15%;left:8%;border-color:#22C55E}.b4{bottom:15%;right:8%;border-color:#EC4899}
.b5{top:50%;left:2%;transform:translateY(-50%);border-color:#8B5CF6}.b6{top:50%;right:2%;transform:translateY(-50%);border-color:#0891B2}
.line{position:absolute;z-index:1}
</style></head><body>
<div class="page">
  <div class="mind">
    <div class="center">{{CENTER}}</div>
    <div class="branch b1">{{BRANCH_1}}</div>
    <div class="branch b2">{{BRANCH_2}}</div>
    <div class="branch b3">{{BRANCH_3}}</div>
    <div class="branch b4">{{BRANCH_4}}</div>
  </div>
</div>
<script>
// 连线用SVG动态绘制（省略，AI生成时会补充连线逻辑）
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:6px;background:#F8FAFC;display:flex;align-items:center;justify-content:center;position:relative"><div style="background:#2563EB;color:#fff;padding:6px 12px;border-radius:50%;font-size:9px;font-weight:700;z-index:2">中心</div><div style="position:absolute;top:8px;left:8px;background:#fff;padding:3px 6px;border-radius:4px;font-size:7px;border-left:2px solid #2563EB;box-shadow:0 1px 3px rgba(0,0,0,0.06)">分支1</div><div style="position:absolute;top:8px;right:8px;background:#fff;padding:3px 6px;border-radius:4px;font-size:7px;border-left:2px solid #F59E0B;box-shadow:0 1px 3px rgba(0,0,0,0.06)">分支2</div><div style="position:absolute;bottom:8px;left:8px;background:#fff;padding:3px 6px;border-radius:4px;font-size:7px;border-left:2px solid #22C55E;box-shadow:0 1px 3px rgba(0,0,0,0.06)">分支3</div><div style="position:absolute;bottom:8px;right:8px;background:#fff;padding:3px 6px;border-radius:4px;font-size:7px;border-left:2px solid #EC4899;box-shadow:0 1px 3px rgba(0,0,0,0.06)">分支4</div></div>`,
	})

	// ============ 4. 动画效果 (animation) — 3个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "淡入逐条展示", Description: "内容列表逐条淡入动画展示，适用于要点逐步揭示",
		ComponentType: "animation", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(2), IdxVisualFormat: "FR", IdxTechTag: "AN",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:34px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:30px}
.items{display:flex;flex-direction:column;gap:16px;flex:1}
.anim-item{opacity:0;transform:translateY(20px);transition:all .6s ease;padding:18px 24px;background:#fff;border-radius:var(--cw-radius,12px);border-left:5px solid var(--cw-primary,#2563EB);box-shadow:var(--cw-shadow,0 2px 8px rgba(0,0,0,0.08));font-size:24px;color:var(--cw-text,#1E293B);line-height:1.5}
.anim-item.show{opacity:1;transform:translateY(0)}
.trigger-btn{align-self:center;padding:12px 32px;border:none;border-radius:8px;background:var(--cw-primary,#2563EB);color:#fff;font-size:20px;font-weight:600;cursor:pointer;margin-top:16px}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="items" id="items">
    <div class="anim-item">{{ITEM_1}}</div>
    <div class="anim-item">{{ITEM_2}}</div>
    <div class="anim-item">{{ITEM_3}}</div>
    <div class="anim-item">{{ITEM_4}}</div>
  </div>
  <button class="trigger-btn" onclick="showNext()">展示下一条</button>
</div>
<script>
var items=document.querySelectorAll('.anim-item'),cur=0;
function showNext(){if(cur<items.length){items[cur].classList.add('show');cur++}if(cur>=items.length)document.querySelector('.trigger-btn').textContent='全部展示完毕'}
showNext(); // 自动展示第一条
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:8px">逐条展示</div><div style="display:flex;flex-direction:column;gap:4px;flex:1"><div style="padding:6px 8px;background:#fff;border-radius:4px;border-left:3px solid #2563EB;font-size:9px;box-shadow:0 1px 3px rgba(0,0,0,0.06)">要点一</div><div style="padding:6px 8px;background:#fff;border-radius:4px;border-left:3px solid #2563EB;font-size:9px;box-shadow:0 1px 3px rgba(0,0,0,0.06);opacity:0.3">要点二</div><div style="padding:6px 8px;background:#fff;border-radius:4px;border-left:3px solid #2563EB;font-size:9px;box-shadow:0 1px 3px rgba(0,0,0,0.06);opacity:0.1">要点三</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "计数动画展示", Description: "数字从0滚动到目标值的动画效果，适用于数据亮点展示",
		ComponentType: "animation", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CH", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:50px 60px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column;align-items:center}
.title{font-size:34px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:50px}
.stats{display:flex;gap:60px;justify-content:center}
.stat{text-align:center}
.stat-num{font-size:64px;font-weight:800;color:var(--cw-primary,#2563EB);font-variant-numeric:tabular-nums}
.stat-unit{font-size:28px;color:var(--cw-accent,#F59E0B);font-weight:600}
.stat-label{font-size:22px;color:#64748B;margin-top:8px}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="stats">
    <div class="stat"><span class="stat-num" data-target="{{NUM_1}}">0</span><span class="stat-unit">{{UNIT_1}}</span><div class="stat-label">{{LABEL_1}}</div></div>
    <div class="stat"><span class="stat-num" data-target="{{NUM_2}}">0</span><span class="stat-unit">{{UNIT_2}}</span><div class="stat-label">{{LABEL_2}}</div></div>
    <div class="stat"><span class="stat-num" data-target="{{NUM_3}}">0</span><span class="stat-unit">{{UNIT_3}}</span><div class="stat-label">{{LABEL_3}}</div></div>
  </div>
</div>
<script>
document.querySelectorAll('.stat-num').forEach(function(el){
  var target=parseInt(el.dataset.target)||0,cur=0,step=Math.ceil(target/60),start=null;
  function anim(ts){if(!start)start=ts;cur=Math.min(cur+step,target);el.textContent=cur;if(cur<target)requestAnimationFrame(anim)}
  requestAnimationFrame(anim);
});
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:12px;background:#F8FAFC;display:flex;flex-direction:column;align-items:center;justify-content:center"><div style="font-size:12px;font-weight:600;color:#2563EB;margin-bottom:12px">数据亮点</div><div style="display:flex;gap:20px"><div style="text-align:center"><div style="font-size:24px;font-weight:800;color:#2563EB">98</div><div style="font-size:8px;color:#64748B">指标一</div></div><div style="text-align:center"><div style="font-size:24px;font-weight:800;color:#2563EB">256</div><div style="font-size:8px;color:#64748B">指标二</div></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "粒子/气泡背景", Description: "CSS动画粒子背景装饰效果，适用于标题页、封面页",
		ComponentType: "animation", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FS", IdxTechTag: "AN",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;position:relative;overflow:hidden;font-family:var(--cw-font-body,system-ui,sans-serif);background:linear-gradient(135deg,var(--cw-primary,#2563EB),#1E40AF);display:flex;align-items:center;justify-content:center;flex-direction:column}
.bubble{position:absolute;border-radius:50%;background:rgba(255,255,255,0.08);animation:float 6s infinite ease-in-out}
@keyframes float{0%,100%{transform:translateY(0) scale(1)}50%{transform:translateY(-30px) scale(1.1)}}
.b1{width:120px;height:120px;top:10%;left:5%;animation-delay:0s}.b2{width:80px;height:80px;top:60%;left:15%;animation-delay:1s}
.b3{width:160px;height:160px;top:20%;right:10%;animation-delay:2s}.b4{width:60px;height:60px;bottom:10%;right:20%;animation-delay:0.5s}
.b5{width:100px;height:100px;bottom:20%;left:40%;animation-delay:3s}.b6{width:40px;height:40px;top:40%;left:30%;animation-delay:1.5s}
.center-text{position:relative;z-index:2;text-align:center;color:#fff}
.center-text h1{font-size:48px;font-weight:800;margin-bottom:16px;text-shadow:0 2px 20px rgba(0,0,0,0.2)}
.center-text p{font-size:28px;opacity:0.9}
</style></head><body>
<div class="page">
  <div class="bubble b1"></div><div class="bubble b2"></div><div class="bubble b3"></div>
  <div class="bubble b4"></div><div class="bubble b5"></div><div class="bubble b6"></div>
  <div class="center-text"><h1>{{PAGE_TITLE}}</h1><p>{{SUBTITLE}}</p></div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;background:linear-gradient(135deg,#2563EB,#1E40AF);display:flex;align-items:center;justify-content:center;border-radius:6px"><div style="text-align:center;color:#fff;z-index:2"><div style="font-size:16px;font-weight:800">标题页</div><div style="font-size:9px;opacity:0.8">副标题文字</div></div></div>`,
	})

	// ============ 5. 多媒体容器 (multimedia) — 3个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "图片画廊展示", Description: "多图轮播/网格画廊，点击可放大查看，适用于图片展示页",
		ComponentType: "multimedia", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(2), IdxVisualFormat: "GD", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);text-align:center;margin-bottom:24px}
.gallery{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;flex:1}
.gallery-item{border-radius:var(--cw-radius,12px);overflow:hidden;cursor:pointer;position:relative;background:var(--cw-secondary,#60A5FA);display:flex;align-items:center;justify-content:center}
.gallery-item img{width:100%;height:100%;object-fit:cover}
.gallery-item .placeholder{color:#fff;font-size:28px}
.gallery-item:hover{transform:scale(1.02);transition:transform .2s}
.lightbox{display:none;position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.8);z-index:999;align-items:center;justify-content:center;cursor:pointer}
.lightbox.show{display:flex}
.lightbox img{max-width:90%;max-height:90%;border-radius:12px}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="gallery">
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图1</span></div>
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图2</span></div>
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图3</span></div>
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图4</span></div>
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图5</span></div>
    <div class="gallery-item" onclick="openLB(this)"><span class="placeholder">📷 图6</span></div>
  </div>
</div>
<div class="lightbox" id="lb" onclick="this.classList.remove('show')"><img id="lbImg" src="" alt=""/></div>
<script>
function openLB(el){var img=el.querySelector('img');if(img){document.getElementById('lbImg').src=img.src;document.getElementById('lb').classList.add('show')}}
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC"><div style="font-size:12px;font-weight:600;color:#2563EB;text-align:center;margin-bottom:8px">图片画廊</div><div style="display:grid;grid-template-columns:repeat(3,1fr);gap:4px;flex:1"><div style="background:#60A5FA;border-radius:4px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:10px;min-height:30px">📷</div><div style="background:#60A5FA;border-radius:4px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:10px;min-height:30px">📷</div><div style="background:#60A5FA;border-radius:4px;display:flex;align-items:center;justify-content:center;color:#fff;font-size:10px;min-height:30px">📷</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "视频播放器容器", Description: "带样式的视频播放器占位容器，支持标题和说明",
		ComponentType: "multimedia", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FS", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:30px 50px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column}
.title{font-size:30px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:16px}
.video-wrap{flex:1;border-radius:var(--cw-radius,12px);overflow:hidden;background:#000;position:relative;display:flex;align-items:center;justify-content:center}
.video-wrap video{width:100%;height:100%;object-fit:contain}
.play-btn{width:80px;height:80px;border-radius:50%;background:rgba(255,255,255,0.9);display:flex;align-items:center;justify-content:center;cursor:pointer;position:absolute;transition:transform .2s}
.play-btn:hover{transform:scale(1.1)}
.play-btn::after{content:'';width:0;height:0;border-left:28px solid var(--cw-primary,#2563EB);border-top:18px solid transparent;border-bottom:18px solid transparent;margin-left:4px}
.desc{font-size:20px;color:#64748B;margin-top:12px;text-align:center}
</style></head><body>
<div class="page">
  <div class="title">{{PAGE_TITLE}}</div>
  <div class="video-wrap"><div class="play-btn" onclick="this.style.display='none'"></div></div>
  <div class="desc">{{VIDEO_DESC}}</div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:8px;background:#F8FAFC;display:flex;flex-direction:column"><div style="font-size:11px;font-weight:600;color:#2563EB;margin-bottom:6px">视频播放</div><div style="flex:1;background:#000;border-radius:6px;display:flex;align-items:center;justify-content:center"><div style="width:30px;height:30px;border-radius:50%;background:rgba(255,255,255,0.9);display:flex;align-items:center;justify-content:center"><div style="width:0;height:0;border-left:10px solid #2563EB;border-top:6px solid transparent;border-bottom:6px solid transparent;margin-left:2px"></div></div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "图片热区标注", Description: "在图片上标注可点击的热区，点击显示说明弹窗",
		ComponentType: "multimedia", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(3), IdxVisualFormat: "FR", IdxTechTag: "JS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:30px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column;align-items:center}
.title{font-size:30px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:16px}
.img-container{position:relative;flex:1;width:100%;max-width:800px;background:var(--cw-secondary,#60A5FA);border-radius:var(--cw-radius,12px);overflow:hidden}
.img-container img{width:100%;height:100%;object-fit:cover}
.hotspot{position:absolute;width:36px;height:36px;border-radius:50%;background:var(--cw-accent,#F59E0B);border:3px solid #fff;cursor:pointer;display:flex;align-items:center;justify-content:center;color:#fff;font-weight:700;font-size:16px;box-shadow:0 2px 8px rgba(0,0,0,0.2);animation:pulse 2s infinite}
@keyframes pulse{0%,100%{box-shadow:0 0 0 0 rgba(245,158,11,0.4)}50%{box-shadow:0 0 0 12px rgba(245,158,11,0)}}
.tooltip{display:none;position:absolute;background:#fff;border-radius:8px;padding:14px 18px;box-shadow:0 4px 16px rgba(0,0,0,0.15);font-size:18px;color:var(--cw-text,#1E293B);max-width:260px;z-index:10;line-height:1.5}
.tooltip.show{display:block}
</style></head><body>
<div class="page">
  <h1 class="title">{{PAGE_TITLE}}</h1>
  <div class="img-container" id="imgC">
    <div class="hotspot" style="top:20%;left:25%" onclick="toggleTip(this,'tip1')">1</div>
    <div class="tooltip" id="tip1" style="top:20%;left:calc(25% + 44px)">{{TIP_1}}</div>
    <div class="hotspot" style="top:50%;left:60%" onclick="toggleTip(this,'tip2')">2</div>
    <div class="tooltip" id="tip2" style="top:50%;left:calc(60% + 44px)">{{TIP_2}}</div>
    <div class="hotspot" style="top:70%;left:35%" onclick="toggleTip(this,'tip3')">3</div>
    <div class="tooltip" id="tip3" style="top:70%;left:calc(35% + 44px)">{{TIP_3}}</div>
  </div>
</div>
<script>
function toggleTip(el,id){document.querySelectorAll('.tooltip').forEach(function(t){t.classList.remove('show')});document.getElementById(id).classList.toggle('show')}
</script>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:8px;background:#F8FAFC;display:flex;flex-direction:column;align-items:center"><div style="font-size:11px;font-weight:600;color:#2563EB;margin-bottom:6px">图片热区</div><div style="flex:1;width:90%;background:#60A5FA;border-radius:6px;position:relative"><div style="position:absolute;top:20%;left:25%;width:16px;height:16px;border-radius:50%;background:#F59E0B;border:2px solid #fff;display:flex;align-items:center;justify-content:center;color:#fff;font-size:7px;font-weight:700">1</div><div style="position:absolute;top:55%;left:60%;width:16px;height:16px;border-radius:50%;background:#F59E0B;border:2px solid #fff;display:flex;align-items:center;justify-content:center;color:#fff;font-size:7px;font-weight:700">2</div></div></div>`,
	})

	// ============ 6. 样式主题 (style) — 4个 ============

	comps = append(comps, &models.CoursewareComponent{
		Name: "渐变标题装饰", Description: "带渐变背景和装饰元素的标题区域，适用于章节封面",
		ComponentType: "style", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FS", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;background:linear-gradient(135deg,var(--cw-primary,#2563EB) 0%,var(--cw-secondary,#60A5FA) 50%,var(--cw-accent,#F59E0B) 100%);display:flex;align-items:center;justify-content:center;position:relative;overflow:hidden;font-family:var(--cw-font-heading,system-ui,sans-serif)}
.deco{position:absolute;border-radius:50%;background:rgba(255,255,255,0.06)}
.d1{width:400px;height:400px;top:-100px;right:-100px}.d2{width:300px;height:300px;bottom:-80px;left:-60px}.d3{width:200px;height:200px;top:50%;left:20%;opacity:.04}
.center{text-align:center;color:#fff;z-index:2}
.center h1{font-size:52px;font-weight:800;margin-bottom:16px;text-shadow:0 2px 20px rgba(0,0,0,0.15)}
.center .subtitle{font-size:28px;opacity:.9;font-weight:400}
.center .tag{display:inline-block;margin-top:20px;padding:8px 24px;border:2px solid rgba(255,255,255,0.4);border-radius:24px;font-size:20px}
</style></head><body>
<div class="page">
  <div class="deco d1"></div><div class="deco d2"></div><div class="deco d3"></div>
  <div class="center">
    <h1>{{PAGE_TITLE}}</h1>
    <div class="subtitle">{{SUBTITLE}}</div>
    <div class="tag">{{TAG}}</div>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;background:linear-gradient(135deg,#2563EB,#60A5FA,#F59E0B);display:flex;align-items:center;justify-content:center;border-radius:6px"><div style="text-align:center;color:#fff"><div style="font-size:16px;font-weight:800">章节标题</div><div style="font-size:9px;opacity:0.8;margin-top:4px">副标题</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "卡片圆角阴影样式", Description: "带圆角和柔和阴影的内容卡片样式，可嵌套其他组件",
		ComponentType: "style", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "CG", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;align-items:center;justify-content:center}
.card{background:#fff;border-radius:var(--cw-radius,16px);padding:40px;box-shadow:var(--cw-shadow,0 4px 24px rgba(0,0,0,0.06));max-width:700px;width:100%;text-align:center;border:1px solid rgba(0,0,0,0.04)}
.card h2{font-size:32px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:20px}
.card p{font-size:24px;color:var(--cw-text,#1E293B);line-height:1.8}
.card .divider{width:60px;height:4px;background:var(--cw-accent,#F59E0B);border-radius:2px;margin:20px auto}
</style></head><body>
<div class="page">
  <div class="card">
    <h2>{{TITLE}}</h2>
    <div class="divider"></div>
    <p>{{CONTENT}}</p>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:12px;background:#F8FAFC;display:flex;align-items:center;justify-content:center"><div style="background:#fff;border-radius:10px;padding:14px;box-shadow:0 2px 12px rgba(0,0,0,0.06);text-align:center;width:80%"><div style="font-size:13px;font-weight:700;color:#2563EB;margin-bottom:6px">标题</div><div style="width:24px;height:2px;background:#F59E0B;margin:0 auto 6px"></div><div style="font-size:9px;color:#64748B;line-height:1.5">内容文字</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "分隔线+小标题装饰", Description: "带装饰性分隔线和小标题的内容分区样式",
		ComponentType: "style", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FR", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:40px 80px;font-family:var(--cw-font-body,system-ui,sans-serif);background:var(--cw-bg,#F8FAFC);display:flex;flex-direction:column;justify-content:center}
.section{margin-bottom:36px}
.section-header{display:flex;align-items:center;gap:16px;margin-bottom:16px}
.section-line{flex:1;height:2px;background:linear-gradient(90deg,var(--cw-primary,#2563EB),transparent)}
.section-tag{font-size:18px;font-weight:600;color:var(--cw-primary,#2563EB);background:var(--cw-bg,#F8FAFC);padding:0 12px;white-space:nowrap}
.section h2{font-size:30px;font-weight:700;color:var(--cw-text,#1E293B);margin-bottom:12px}
.section p{font-size:24px;color:#64748B;line-height:1.7}
</style></head><body>
<div class="page">
  <div class="section">
    <div class="section-header"><div class="section-line"></div><span class="section-tag">01</span><div class="section-line" style="background:linear-gradient(90deg,transparent,var(--cw-primary,#2563EB))"></div></div>
    <h2>{{TITLE_1}}</h2>
    <p>{{CONTENT_1}}</p>
  </div>
  <div class="section">
    <div class="section-header"><div class="section-line"></div><span class="section-tag">02</span><div class="section-line" style="background:linear-gradient(90deg,transparent,var(--cw-primary,#2563EB))"></div></div>
    <h2>{{TITLE_2}}</h2>
    <p>{{CONTENT_2}}</p>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;padding:10px;background:#F8FAFC;display:flex;flex-direction:column;justify-content:center"><div style="margin-bottom:10px"><div style="display:flex;align-items:center;gap:6px;margin-bottom:4px"><div style="flex:1;height:1px;background:linear-gradient(90deg,#2563EB,transparent)"></div><span style="font-size:8px;font-weight:600;color:#2563EB">01</span><div style="flex:1;height:1px;background:linear-gradient(90deg,transparent,#2563EB)"></div></div><div style="font-size:11px;font-weight:700;color:#1E293B;margin-bottom:2px">小标题</div><div style="font-size:8px;color:#64748B">内容文字</div></div></div>`,
	})

	comps = append(comps, &models.CoursewareComponent{
		Name: "背景图案网格纹", Description: "CSS背景图案装饰（点阵/格纹/斜线），用于丰富页面背景层次",
		ComponentType: "style", SubjectScope: "ALL", GradeScope: "ALL",
		IdxInteractionLevel: il(1), IdxVisualFormat: "FS", IdxTechTag: "CSS",
		ReviewStatus: "approved", IsActive: true,
		CodeContent: `<!DOCTYPE html><html><head><meta charset="UTF-8"><style>
*{margin:0;padding:0;box-sizing:border-box}
.page{width:960px;height:540px;padding:50px 80px;font-family:var(--cw-font-body,system-ui,sans-serif);display:flex;flex-direction:column;justify-content:center;align-items:center;position:relative;background-color:var(--cw-bg,#F8FAFC);background-image:radial-gradient(circle,rgba(37,99,235,0.08) 1px,transparent 1px);background-size:24px 24px}
.content-box{background:rgba(255,255,255,0.9);border-radius:var(--cw-radius,16px);padding:40px;text-align:center;box-shadow:var(--cw-shadow,0 4px 24px rgba(0,0,0,0.06));max-width:700px;backdrop-filter:blur(8px)}
.content-box h1{font-size:38px;font-weight:700;color:var(--cw-primary,#2563EB);margin-bottom:20px}
.content-box p{font-size:24px;color:var(--cw-text,#1E293B);line-height:1.8}
</style></head><body>
<div class="page">
  <div class="content-box">
    <h1>{{PAGE_TITLE}}</h1>
    <p>{{CONTENT}}</p>
  </div>
</div>
</body></html>`,
		PreviewHTML: `<div style="width:100%;height:100%;background-color:#F8FAFC;background-image:radial-gradient(circle,rgba(37,99,235,0.1) 1px,transparent 1px);background-size:10px 10px;display:flex;align-items:center;justify-content:center;border-radius:6px"><div style="background:rgba(255,255,255,0.9);border-radius:8px;padding:10px;text-align:center;box-shadow:0 2px 8px rgba(0,0,0,0.06)"><div style="font-size:12px;font-weight:700;color:#2563EB">标题</div><div style="font-size:8px;color:#64748B;margin-top:3px">带背景图案</div></div></div>`,
	})

	return comps
}

// ==================== 风格模板种子数据 ====================

func (s *CoursewareSeedService) seedTemplates(ctx context.Context, force bool) (int, error) {
	// 检查已有数据
	existing, err := repository.ListCWTemplates(ctx, false)
	if err == nil && len(existing) > 0 && !force {
		log.Printf("[CW种子] 已有 %d 套模板，跳过", len(existing))
		return 0, nil
	}
	if force && len(existing) > 0 {
		log.Printf("[CW种子] force模式，清空现有 %d 套模板", len(existing))
		for _, t := range existing {
			_ = repository.DeleteCWTemplate(ctx, t.ID)
		}
	}

	templates := buildSeedTemplates()
	created := 0
	for _, t := range templates {
		if err := repository.CreateCWTemplate(ctx, t); err != nil {
			log.Printf("[CW种子] 创建模板失败 %s: %v", t.Name, err)
			continue
		}
		created++
	}
	log.Printf("[CW种子] 模板创建完成: %d/%d", created, len(templates))
	return created, nil
}

// buildSeedTemplates 构建全部种子模板（5种风格 × 2-3配色 = 12套）
// v2.1 升级：更时尚的配色方案 + 每套模板含真实样例页面HTML
func buildSeedTemplates() []*models.CoursewareTemplate {
	var templates []*models.CoursewareTemplate
	order := 0

	add := func(name, desc, category, colorScheme, cssVars, samplePages string) {
		order += 10
		templates = append(templates, &models.CoursewareTemplate{
			Name: name, Description: desc, StyleCategory: category,
			ColorScheme: colorScheme, CSSVariables: cssVars,
			SamplePages: samplePages, IsActive: true, SortOrder: order,
		})
	}

	// ===== 简约清新 (minimalist) — 3套 =====
	add("简约清新-天际蓝", "天空蓝与云白色的极简搭配，干净透气，适用于通用课程",
		"minimalist",
		`{"primary":"#0EA5E9","secondary":"#38BDF8","background":"#F0F9FF","accent":"#F97316","text":"#0C4A6E"}`,
		`{"--cw-primary":"#0EA5E9","--cw-secondary":"#38BDF8","--cw-bg":"#F0F9FF","--cw-accent":"#F97316","--cw-text":"#0C4A6E","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"16px","--cw-shadow":"0 4px 24px rgba(14,165,233,0.08)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#F0F9FF 0%,#E0F2FE 100%);display:flex;align-items:center;justify-content:center;padding:80px;font-family:Inter,system-ui\"><div style=\"text-align:center\"><h1 style=\"font-size:52px;font-weight:800;color:#0EA5E9;margin-bottom:24px;letter-spacing:-1px\">认识人工智能</h1><div style=\"width:80px;height:4px;background:linear-gradient(90deg,#0EA5E9,#F97316);border-radius:2px;margin:0 auto 24px\"></div><p style=\"font-size:24px;color:#0C4A6E;opacity:0.7;line-height:1.8\">探索AI的奇妙世界</p></div></div>"]`,
	)
	add("简约清新-薄荷绿", "清新薄荷绿配色，自然舒适，适用于生物、健康课程",
		"minimalist",
		`{"primary":"#10B981","secondary":"#34D399","background":"#ECFDF5","accent":"#F59E0B","text":"#064E3B"}`,
		`{"--cw-primary":"#10B981","--cw-secondary":"#34D399","--cw-bg":"#ECFDF5","--cw-accent":"#F59E0B","--cw-text":"#064E3B","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"16px","--cw-shadow":"0 4px 24px rgba(16,185,129,0.08)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(135deg,#ECFDF5,#D1FAE5);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui\"><div style=\"text-align:center\"><div style=\"font-size:64px;margin-bottom:20px\">🌿</div><h1 style=\"font-size:48px;font-weight:800;color:#10B981;margin-bottom:16px\">生命的奥秘</h1><p style=\"font-size:22px;color:#064E3B;opacity:0.6\">探索自然界的神奇</p></div></div>"]`,
	)
	add("简约清新-石墨灰", "高级灰配银色点缀，商务质感，适用于通识、商业课程",
		"minimalist",
		`{"primary":"#334155","secondary":"#64748B","background":"#F8FAFC","accent":"#6366F1","text":"#1E293B"}`,
		`{"--cw-primary":"#334155","--cw-secondary":"#64748B","--cw-bg":"#F8FAFC","--cw-accent":"#6366F1","--cw-text":"#1E293B","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"12px","--cw-shadow":"0 2px 16px rgba(0,0,0,0.06)"}`,
		`["<div style=\"width:960px;height:540px;background:#F8FAFC;display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui\"><div style=\"background:#fff;border-radius:20px;padding:60px;box-shadow:0 8px 40px rgba(0,0,0,0.06);text-align:center;max-width:680px\"><h1 style=\"font-size:44px;font-weight:800;color:#334155;margin-bottom:16px\">商业思维导论</h1><div style=\"width:48px;height:3px;background:#6366F1;margin:0 auto 20px;border-radius:2px\"></div><p style=\"font-size:20px;color:#64748B;line-height:1.8\">从0到1理解商业逻辑</p></div></div>"]`,
	)

	// ===== 活泼趣味 (playful) — 3套 =====
	add("活泼趣味-缤纷糖果", "彩色糖果配色+圆润形态，充满童趣，适用于小学低年级",
		"playful",
		`{"primary":"#F472B6","secondary":"#A78BFA","background":"#FDF2F8","accent":"#FBBF24","text":"#1F2937"}`,
		`{"--cw-primary":"#F472B6","--cw-secondary":"#A78BFA","--cw-bg":"#FDF2F8","--cw-accent":"#FBBF24","--cw-text":"#1F2937","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"24px","--cw-shadow":"0 8px 32px rgba(244,114,182,0.15)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(135deg,#FDF2F8,#EDE9FE,#FEF3C7);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui\"><div style=\"text-align:center\"><div style=\"display:flex;gap:16px;justify-content:center;margin-bottom:24px\"><span style=\"font-size:56px;animation:bounce 1s infinite\">🍭</span><span style=\"font-size:56px;animation:bounce 1s infinite 0.2s\">🎨</span><span style=\"font-size:56px;animation:bounce 1s infinite 0.4s\">✨</span></div><h1 style=\"font-size:48px;font-weight:800;background:linear-gradient(135deg,#F472B6,#A78BFA);-webkit-background-clip:text;-webkit-text-fill-color:transparent\">欢乐学堂</h1></div></div>"]`,
	)
	add("活泼趣味-阳光橙", "温暖阳光橙+活力黄，元气满满，适用于小学中高段",
		"playful",
		`{"primary":"#F97316","secondary":"#FB923C","background":"#FFF7ED","accent":"#3B82F6","text":"#1F2937"}`,
		`{"--cw-primary":"#F97316","--cw-secondary":"#FB923C","--cw-bg":"#FFF7ED","--cw-accent":"#3B82F6","--cw-text":"#1F2937","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"20px","--cw-shadow":"0 6px 28px rgba(249,115,22,0.12)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#FFF7ED,#FFEDD5);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui\"><div style=\"text-align:center\"><div style=\"width:100px;height:100px;border-radius:50%;background:linear-gradient(135deg,#F97316,#FBBF24);margin:0 auto 24px;display:flex;align-items:center;justify-content:center;font-size:48px;box-shadow:0 8px 32px rgba(249,115,22,0.3)\">☀️</div><h1 style=\"font-size:46px;font-weight:800;color:#F97316\">科学小实验</h1><p style=\"font-size:22px;color:#9A3412;opacity:0.6;margin-top:12px\">动手发现大自然的秘密</p></div></div>"]`,
	)
	add("活泼趣味-梦幻紫", "渐变紫色系，梦幻浪漫，适用于创意课程",
		"playful",
		`{"primary":"#8B5CF6","secondary":"#C084FC","background":"#FAF5FF","accent":"#EC4899","text":"#1F2937"}`,
		`{"--cw-primary":"#8B5CF6","--cw-secondary":"#C084FC","--cw-bg":"#FAF5FF","--cw-accent":"#EC4899","--cw-text":"#1F2937","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"20px","--cw-shadow":"0 6px 28px rgba(139,92,246,0.12)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(135deg,#FAF5FF,#F3E8FF,#FDF2F8);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui;position:relative;overflow:hidden\"><div style=\"position:absolute;width:300px;height:300px;border-radius:50%;background:rgba(139,92,246,0.06);top:-50px;right:-50px\"></div><div style=\"position:absolute;width:200px;height:200px;border-radius:50%;background:rgba(236,72,153,0.06);bottom:-30px;left:-30px\"></div><div style=\"text-align:center;z-index:1\"><h1 style=\"font-size:50px;font-weight:800;background:linear-gradient(135deg,#8B5CF6,#EC4899);-webkit-background-clip:text;-webkit-text-fill-color:transparent\">创意无限</h1><p style=\"font-size:22px;color:#7C3AED;opacity:0.6;margin-top:16px\">释放你的想象力</p></div></div>"]`,
	)

	// ===== 科技感 (tech) — 2套 =====
	add("科技感-深空蓝", "深色底+霓虹蓝高光+微光粒子，赛博风格，适用于信息科技、AI课程",
		"tech",
		`{"primary":"#3B82F6","secondary":"#1D4ED8","background":"#030712","accent":"#22D3EE","text":"#E2E8F0"}`,
		`{"--cw-primary":"#3B82F6","--cw-secondary":"#1D4ED8","--cw-bg":"#030712","--cw-accent":"#22D3EE","--cw-text":"#E2E8F0","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"12px","--cw-shadow":"0 8px 32px rgba(59,130,246,0.2)"}`,
		`["<div style=\"width:960px;height:540px;background:#030712;display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui;position:relative;overflow:hidden\"><div style=\"position:absolute;width:600px;height:600px;border-radius:50%;background:radial-gradient(circle,rgba(59,130,246,0.15),transparent 70%);top:50%;left:50%;transform:translate(-50%,-50%)\"></div><div style=\"text-align:center;z-index:1\"><div style=\"font-size:14px;letter-spacing:6px;color:#22D3EE;text-transform:uppercase;margin-bottom:20px;font-weight:600\">ARTIFICIAL INTELLIGENCE</div><h1 style=\"font-size:56px;font-weight:800;color:#fff;text-shadow:0 0 60px rgba(59,130,246,0.4)\">人工智能导论</h1><div style=\"width:120px;height:2px;background:linear-gradient(90deg,transparent,#3B82F6,transparent);margin:24px auto\"></div><p style=\"font-size:20px;color:#94A3B8\">Exploring the Future of Technology</p></div></div>"]`,
	)
	add("科技感-赛博紫", "暗紫底+粉色霓虹，赛博朋克美学，适用于编程、创客课程",
		"tech",
		`{"primary":"#A855F7","secondary":"#7C3AED","background":"#0A0118","accent":"#F472B6","text":"#E2E8F0"}`,
		`{"--cw-primary":"#A855F7","--cw-secondary":"#7C3AED","--cw-bg":"#0A0118","--cw-accent":"#F472B6","--cw-text":"#E2E8F0","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"12px","--cw-shadow":"0 8px 32px rgba(168,85,247,0.2)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#0A0118,#1A0533);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui;position:relative;overflow:hidden\"><div style=\"position:absolute;width:100%;height:100%;background:repeating-linear-gradient(0deg,transparent,transparent 50px,rgba(168,85,247,0.03) 50px,rgba(168,85,247,0.03) 51px)\"></div><div style=\"text-align:center;z-index:1\"><div style=\"display:inline-block;padding:8px 24px;border:1px solid rgba(168,85,247,0.3);border-radius:24px;color:#A855F7;font-size:14px;letter-spacing:3px;margin-bottom:24px\">CODE LAB</div><h1 style=\"font-size:52px;font-weight:800;background:linear-gradient(135deg,#A855F7,#F472B6);-webkit-background-clip:text;-webkit-text-fill-color:transparent\">编程创造未来</h1><p style=\"font-size:20px;color:#C084FC;opacity:0.6;margin-top:16px\">从零开始的代码之旅</p></div></div>"]`,
	)

	// ===== 学术严谨 (academic) — 2套 =====
	add("学术严谨-牛津蓝", "深蓝+金色，经典学院风格，适用于高中理科、学术报告",
		"academic",
		`{"primary":"#1E3A5F","secondary":"#2563EB","background":"#FFFFFF","accent":"#B45309","text":"#1E293B"}`,
		`{"--cw-primary":"#1E3A5F","--cw-secondary":"#2563EB","--cw-bg":"#FFFFFF","--cw-accent":"#B45309","--cw-text":"#1E293B","--cw-font-heading":"Georgia,serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"4px","--cw-shadow":"0 2px 8px rgba(0,0,0,0.08)"}`,
		`["<div style=\"width:960px;height:540px;background:#fff;display:flex;font-family:Georgia,serif\"><div style=\"width:8px;background:linear-gradient(180deg,#1E3A5F,#B45309)\"></div><div style=\"flex:1;padding:60px 80px;display:flex;flex-direction:column;justify-content:center\"><div style=\"font-size:14px;color:#B45309;letter-spacing:4px;text-transform:uppercase;margin-bottom:16px;font-weight:600\">Chapter One</div><h1 style=\"font-size:48px;font-weight:700;color:#1E3A5F;line-height:1.2;margin-bottom:24px\">量子力学基础</h1><div style=\"width:60px;height:3px;background:#B45309;margin-bottom:24px\"></div><p style=\"font-size:20px;color:#64748B;line-height:1.8;font-family:Inter,system-ui\">Fundamentals of Quantum Mechanics</p></div></div>"]`,
	)
	add("学术严谨-复古棕", "暖棕+象牙白，古典书卷气息，适用于文史哲课程",
		"academic",
		`{"primary":"#78350F","secondary":"#92400E","background":"#FFFBEB","accent":"#B45309","text":"#1C1917"}`,
		`{"--cw-primary":"#78350F","--cw-secondary":"#92400E","--cw-bg":"#FFFBEB","--cw-accent":"#B45309","--cw-text":"#1C1917","--cw-font-heading":"Georgia,serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"4px","--cw-shadow":"0 2px 8px rgba(0,0,0,0.06)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#FFFBEB,#FEF3C7);display:flex;align-items:center;justify-content:center;font-family:Georgia,serif;position:relative\"><div style=\"position:absolute;top:30px;left:30px;right:30px;bottom:30px;border:2px solid rgba(120,53,15,0.15);border-radius:4px\"></div><div style=\"text-align:center;z-index:1\"><div style=\"font-size:48px;margin-bottom:20px\">📜</div><h1 style=\"font-size:44px;font-weight:700;color:#78350F;margin-bottom:16px\">唐诗宋词鉴赏</h1><div style=\"width:100px;height:1px;background:linear-gradient(90deg,transparent,#B45309,transparent);margin:0 auto 16px\"></div><p style=\"font-size:20px;color:#92400E;opacity:0.7;font-style:italic\">感受千年文化之美</p></div></div>"]`,
	)

	// ===== 自然有机 (organic) — 2套 =====
	add("自然有机-森林绿", "深绿+木质纹理感，自然有机风格，适用于生物、地理、环保课程",
		"organic",
		`{"primary":"#059669","secondary":"#10B981","background":"#ECFDF5","accent":"#D97706","text":"#064E3B"}`,
		`{"--cw-primary":"#059669","--cw-secondary":"#10B981","--cw-bg":"#ECFDF5","--cw-accent":"#D97706","--cw-text":"#064E3B","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"20px","--cw-shadow":"0 4px 24px rgba(5,150,105,0.1)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#ECFDF5,#D1FAE5);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui;position:relative;overflow:hidden\"><div style=\"position:absolute;bottom:0;left:0;right:0;height:120px;background:linear-gradient(180deg,transparent,rgba(5,150,105,0.08))\"></div><div style=\"text-align:center;z-index:1\"><div style=\"display:flex;gap:8px;justify-content:center;margin-bottom:24px;font-size:40px\">🌱🌍🌿</div><h1 style=\"font-size:48px;font-weight:800;color:#059669\">生态系统探秘</h1><p style=\"font-size:22px;color:#064E3B;opacity:0.6;margin-top:14px\">了解地球生命的循环与平衡</p></div></div>"]`,
	)
	add("自然有机-海洋蓝绿", "海洋蓝绿渐变，波浪感设计，适用于海洋、地球科学课程",
		"organic",
		`{"primary":"#0D9488","secondary":"#14B8A6","background":"#F0FDFA","accent":"#F59E0B","text":"#134E4A"}`,
		`{"--cw-primary":"#0D9488","--cw-secondary":"#14B8A6","--cw-bg":"#F0FDFA","--cw-accent":"#F59E0B","--cw-text":"#134E4A","--cw-font-heading":"'Inter',system-ui,sans-serif","--cw-font-body":"'Inter',system-ui,sans-serif","--cw-radius":"20px","--cw-shadow":"0 4px 24px rgba(13,148,136,0.1)"}`,
		`["<div style=\"width:960px;height:540px;background:linear-gradient(180deg,#F0FDFA,#CCFBF1,#99F6E4);display:flex;align-items:center;justify-content:center;font-family:Inter,system-ui;position:relative;overflow:hidden\"><div style=\"position:absolute;bottom:0;width:100%;height:80px;background:linear-gradient(180deg,transparent,rgba(13,148,136,0.1))\"></div><div style=\"text-align:center;z-index:1\"><div style=\"font-size:56px;margin-bottom:20px\">🌊</div><h1 style=\"font-size:48px;font-weight:800;background:linear-gradient(135deg,#0D9488,#14B8A6);-webkit-background-clip:text;-webkit-text-fill-color:transparent\">海洋世界</h1><p style=\"font-size:22px;color:#134E4A;opacity:0.6;margin-top:14px\">探索深蓝的奥秘</p></div></div>"]`,
	)

	return templates
}
