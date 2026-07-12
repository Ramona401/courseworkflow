package services

// class_profile_score_import.go — 班级学情·成绩单导入归并（批次2b / 2b-2）
//
// 挂在 ClassProfileService 上的成绩导入方法，与 class_profile_service.go 的学生 CRUD 分文件，
// 避免动已上线验证过的核心文件，且导入逻辑较重（解析/归并/自动建生/覆盖画像/错误收集）独立成文。
//
// 设计要点（2b-2，贴合"一次考试一个批次、对着成绩挨个学生填一行"的真实用法）：
//   1. 考试名称 + 考试日期整批统一（请求体顶层），一次导入就是一次考试，只校验一次。
//   2. 每行只带逐生不同的内容：学号代号 | 分数 | 薄弱点 | 备注。
//   3. 成绩归并：按学号归并进各学生 scores 数组，追加不覆盖（去重键=考试名+日期，
//      命中则更新该条分数，便于改错分）。归并算法在 models.MergeScoreInto。
//   4. 薄弱点/备注归并：非空则覆盖该生当前值（取老师最新判断），留空则不动（避免误清旧值）。
//   5. latest_score 自动取"考试日期最新"那条（models.LatestScoreOf）。
//   6. 学号不存在则自动新建学生（tier 空，等老师后续手动定 ABC）。
//   7. 成绩导入只动学生明细，不刷新班级卡四大段、不触发 AI（AI 总结是批次2c 的事）；
//      但会同步班级卡 student_count（best-effort），让卡片人数跟上自动建生。
//
// 合规红线：导入只写学号代号 + 成绩 + 薄弱点/备注，绝不写真名。前端模板与提示引导用学号代号、
// 薄弱点/备注不写姓名隐私；后端不做姓名检测（易误杀），靠界面文案 + 学生子页红色合规带兜住。
//
// 鉴权：复用 ensureProfileOwned（先确认班级卡归属当前老师），与学生 CRUD 同一道闸门。

import (
	"context"
	"strconv"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// 导入参数护栏
const (
	maxImportRows      = 2000 // 单次导入最多行数（防超大表拖死）
	maxImportErrorShow = 20   // 错误明细最多回显条数（更多只计数不逐条列）
)

// ImportScores 把一批成绩行归并进对应学生（成绩追加、薄弱点/备注覆盖）。
//
// 流程：
//   1. 校验班级卡归属（ensureProfileOwned）。
//   2. 校验整批统一的考试名/日期（必填，只校验一次）。
//   3. 一次性拉本班全部学生，建 student_code → *ClassStudent 索引。
//   4. 清洗每行（trim、学号/分数必填），非法行计入 SkippedRows + Errors（带行号）；
//      合法行按学号聚合"待归并内容"（成绩条目 + 薄弱点 + 备注）。
//   5. 对未知学号自动建生。
//   6. 逐学生归并：成绩 MergeScoreInto 进数组、重算 latest_score；薄弱点/备注非空则覆盖；
//      写回 UpdateClassStudent。
//   7. 同步班级卡 student_count（best-effort）。
//
// 返回 ImportScoresResult 供前端展示"追加 X 人 Y 条 / 新建 Z 人 / 更新画像 W 人"。
func (s *ClassProfileService) ImportScores(
	ctx context.Context, userID, classProfileID string, req *models.ImportScoresRequest,
) (*models.ImportScoresResult, error) {

	profile, err := s.ensureProfileOwned(ctx, userID, classProfileID)
	if err != nil {
		return nil, err
	}

	result := &models.ImportScoresResult{Errors: []string{}}

	if req == nil || len(req.Rows) == 0 {
		// 空导入：不算错误，原样返回零结果（前端会提示"没有可导入的成绩行"）
		return result, nil
	}

	// 整批统一的考试名/日期，必填且只校验一次（2b-2）
	examName := strings.TrimSpace(req.ExamName)
	examDate := strings.TrimSpace(req.ExamDate)
	if examName == "" || examDate == "" {
		result.Errors = append(result.Errors, "本批考试名称与考试日期不能为空")
		result.SkippedRows = len(req.Rows)
		return result, nil
	}

	if len(req.Rows) > maxImportRows {
		// 超量直接拒绝（service 层兜底，前端也会拦），返回业务可读错误
		result.Errors = append(result.Errors,
			"单次最多导入 "+strconv.Itoa(maxImportRows)+" 行，当前 "+strconv.Itoa(len(req.Rows))+" 行，请分批导入")
		result.SkippedRows = len(req.Rows)
		return result, nil
	}

	// ---------- 第一步：拉本班现有学生，建学号索引 ----------
	existing, err := repository.ListClassStudents(ctx, classProfileID)
	if err != nil {
		return nil, err
	}
	// 学号 → 学生实体（同一学号取首个；UNIQUE 约束保证库里不会重号）
	byCode := make(map[string]*models.ClassStudent, len(existing))
	for _, st := range existing {
		byCode[st.StudentCode] = st
	}

	// ---------- 第二步：清洗每行，按学号聚合"待归并内容" ----------
	// 该学号本次要归并的成绩条目（一次导入=一次考试，正常每学号一条；
	// 但若同一学号在表里出现多行，仍按行各算一条，去重交 MergeScoreInto）
	pendingScores := make(map[string][]models.ClassStudentScore)
	// 该学号本次要覆盖的薄弱点/备注（取该学号在表里最后一个非空值）
	pendingWeak := make(map[string]string)
	pendingNote := make(map[string]string)
	// 保持学号首次出现顺序，让处理稳定可复现
	codeOrder := make([]string, 0)
	seenCode := make(map[string]bool)

	addErr := func(rowNo int, msg string) {
		result.SkippedRows++
		if len(result.Errors) < maxImportErrorShow {
			result.Errors = append(result.Errors, "第"+strconv.Itoa(rowNo)+"行："+msg)
		}
	}

	for i, row := range req.Rows {
		rowNo := i + 1
		code := strings.TrimSpace(row.StudentCode)
		weak := strings.TrimSpace(row.WeakTopics)
		note := strings.TrimSpace(row.Note)

		if code == "" {
			addErr(rowNo, "学号代号为空")
			continue
		}

		// 成绩条目（考试名/日期取整批统一值）
		sc := models.ClassStudentScore{
			Name:  examName,
			Score: row.Score, // 前端已把"分数"解析为 number 传来
			Max:   0,         // v1 模板无满分列，统一 0=未提供
			At:    examDate,
		}
		pendingScores[code] = append(pendingScores[code], sc)

		// 薄弱点/备注：非空才登记（覆盖语义；同学号多行取最后一个非空）
		if weak != "" {
			pendingWeak[code] = weak
		}
		if note != "" {
			pendingNote[code] = note
		}

		if !seenCode[code] {
			seenCode[code] = true
			codeOrder = append(codeOrder, code)
		}
	}

	if len(pendingScores) == 0 {
		// 全部行非法，没有任何可归并的成绩，原样返回（含错误明细）
		result.TotalRows = len(req.Rows)
		return result, nil
	}

	// ---------- 第三步：对未知学号自动建生 ----------
	for _, code := range codeOrder {
		if _, ok := byCode[code]; ok {
			continue // 已存在，无需建
		}
		// 学号不存在 → 自动建一条空档案（tier 空、薄弱点空、备注空、成绩空）。
		// 注意：这里 code 是导入表里明确写了的学号，直接用它建（不走 NextStudentCode），
		// 因为老师在成绩单里写的学号就是他要的代号；薄弱点/备注随后在第四步统一覆盖写入。
		st := &models.ClassStudent{
			ClassProfileID: classProfileID,
			OwnerID:        userID,
			StudentCode:    code,
			Tier:           models.StudentTierNone,
			Scores:         "[]",
			LatestScore:    nil,
		}
		if err := repository.CreateClassStudent(ctx, st); err != nil {
			if isUniqueViolation(err) {
				// 并发或重复：理论上不会（同一导入内已去重），保守跳过该学号的归并
				addErr(0, "学号「"+code+"」建立冲突，已跳过")
				delete(pendingScores, code)
				continue
			}
			// 其它错误：建生失败则该学号成绩无处可归，跳过但不中断整体导入
			addErr(0, "学号「"+code+"」自动建立失败："+err.Error())
			delete(pendingScores, code)
			continue
		}
		byCode[code] = st
		result.CreatedStudents++
	}

	// ---------- 第四步：逐学生归并（成绩追加 + 薄弱点/备注覆盖）----------
	for _, code := range codeOrder {
		incomingList, ok := pendingScores[code]
		if !ok || len(incomingList) == 0 {
			continue // 该学号在建生阶段被剔除
		}
		st, ok := byCode[code]
		if !ok {
			continue // 理论不会（建生已并入），保险跳过
		}

		// 成绩：解析现有数组，逐条归并
		scores := models.ParseClassStudentScores(st.Scores)
		writtenForThis := 0
		for _, inc := range incomingList {
			scores = models.MergeScoreInto(scores, inc)
			writtenForThis++ // 无论追加还是更新都算处理一条
		}
		st.Scores = models.ScoresToJSON(scores)
		st.LatestScore = models.LatestScoreOf(scores)

		// 薄弱点/备注：非空则覆盖（取老师最新判断），留空不动旧值
		profileTouched := false
		if w, ok := pendingWeak[code]; ok && w != "" {
			st.WeakTopics = w
			profileTouched = true
		}
		if n, ok := pendingNote[code]; ok && n != "" {
			st.Note = n
			profileTouched = true
		}

		if err := repository.UpdateClassStudent(ctx, st); err != nil {
			// 单个学生写回失败：记错误但不中断其它学生
			addErr(0, "学号「"+code+"」成绩写入失败："+err.Error())
			continue
		}

		result.AffectedStudents++
		result.AppendedScores += writtenForThis
		if profileTouched {
			result.ProfileUpdated++
		}
	}

	// ---------- 第五步：同步班级卡人数（best-effort）----------
	s.syncStudentCount(ctx, profile)

	result.TotalRows = len(req.Rows)

	classProfileLog.Info("成绩单导入完成",
		"profile", classProfileID, "owner", userID,
		"exam", examName, "date", examDate,
		"total_rows", result.TotalRows, "appended", result.AppendedScores,
		"affected", result.AffectedStudents, "created", result.CreatedStudents,
		"profile_updated", result.ProfileUpdated, "skipped", result.SkippedRows)

	return result, nil
}
