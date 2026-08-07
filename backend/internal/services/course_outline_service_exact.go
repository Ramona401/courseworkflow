package services

// course_outline_service_exact.go — 自动精确候选、手动学段候选与学制规范化
//
// 候选模式：
//   - exact：自动匹配使用，要求课程大纲与当前具体年级文本完全相等；
//   - manual：自动匹配没有唯一结果时使用，允许具体年级与学段相交；
//   - 两种模式都必须同时通过可信教育域、用户可见组织范围、active和同学科校验；
//   - mixed管理身份不进入普通备课候选链，固定返回安全空数组。

import (
        "context"
        "sort"
        "strings"

        "tedna/internal/models"
        "tedna/internal/repository"
)

// listCourseOutlineCandidates
// 统一解析可信Actor和可见范围，然后根据模式调用对应仓储。
func (s *CourseOutlineService) listCourseOutlineCandidates(
        ctx context.Context,
        userID string,
        subject string,
        grade string,
        manual bool,
) (
        []*models.CourseOutlineListItem,
        string,
        error,
) {
        subject = strings.TrimSpace(subject)
        grade = strings.TrimSpace(grade)

        if subject == "" || grade == "" {
                return []*models.CourseOutlineListItem{},
                        "",
                        nil
        }

        actor, err := resolveCourseOutlineActor(
                ctx,
                userID,
        )
        if err != nil {
                if isCourseOutlineSafeEmptyDomainError(err) {
                        return []*models.CourseOutlineListItem{},
                                "",
                                nil
                }

                return nil, "", err
        }

        if actor.MixedManagement {
                return []*models.CourseOutlineListItem{},
                        actor.EducationDomain,
                        nil
        }

        groupIDs, schoolIDs :=
                s.resolveUserVisibleScopeIDs(
                        ctx,
                        actor.Role,
                        actor.UserID,
                )

        var items []*models.CourseOutlineListItem

        if manual {
                items, err =
                        repository.ListVisibleManualCourseOutlineCandidates(
                                ctx,
                                false,
                                groupIDs,
                                schoolIDs,
                                actor.EducationDomain,
                                subject,
                                grade,
                        )
        } else {
                items, err =
                        repository.ListVisibleExactCourseOutlineCandidates(
                                ctx,
                                false,
                                groupIDs,
                                schoolIDs,
                                actor.EducationDomain,
                                subject,
                                grade,
                        )
        }

        if err != nil {
                return nil, "", err
        }

        if items == nil {
                items = []*models.CourseOutlineListItem{}
        }

        return items, actor.EducationDomain, nil
}

// ListExactCandidates
// 返回自动匹配使用的具体年级完全相等候选。
//
// 自动匹配是否成功由前端按照唯一候选原则判断：
//   - 恰好一条：可确定性自动选择；
//   - 零条或多条：不自动猜测，转入手动选择。
func (s *CourseOutlineService) ListExactCandidates(
        ctx context.Context,
        userID string,
        subject string,
        grade string,
) (
        []*models.CourseOutlineListItem,
        string,
        error,
) {
        return s.listCourseOutlineCandidates(
                ctx,
                userID,
                subject,
                grade,
                false,
        )
}

// ListManualCandidates
// 返回自动匹配失败后的手动选择候选。
//
// 仓储先召回同教育域、同学科且当前用户可见的active大纲；
// 本方法再使用统一年级或学段相交规则过滤：
//   - 具体年级可以选择覆盖自己的学段大纲；
//   - 学段可以选择其中的具体年级或相交学段大纲；
//   - 无关年级和无关学段不得出现在候选中。
func (s *CourseOutlineService) ListManualCandidates(
        ctx context.Context,
        userID string,
        subject string,
        grade string,
) (
        []*models.CourseOutlineListItem,
        string,
        error,
) {
        items, domain, err :=
                s.listCourseOutlineCandidates(
                        ctx,
                        userID,
                        subject,
                        grade,
                        true,
                )
        if err != nil {
                return nil, "", err
        }

        filtered := make(
                []*models.CourseOutlineListItem,
                0,
                len(items),
        )

        for _, item := range items {
                if item == nil {
                        continue
                }

                if !courseOutlineGradesMatch(
                        item.Grade,
                        grade,
                ) {
                        continue
                }

                filtered = append(
                        filtered,
                        item,
                )
        }

        // 手动列表仍把具体年级完全相等候选排在最前，
        // 其余学段相交候选保持仓储原有稳定排序。
        sort.SliceStable(
                filtered,
                func(i int, j int) bool {
                        leftExact :=
                                strings.TrimSpace(
                                        filtered[i].Grade,
                                ) ==
                                        strings.TrimSpace(
                                                grade,
                                        )

                        rightExact :=
                                strings.TrimSpace(
                                        filtered[j].Grade,
                                ) ==
                                        strings.TrimSpace(
                                                grade,
                                        )

                        if leftExact == rightExact {
                                return false
                        }

                        return leftExact
                },
        )

        return filtered, domain, nil
}

// ListAvailablePublishers
// 保留旧接口，但只聚合具体年级完全相等候选。
//
// 旧publisher-only链不得因为手动学段选择能力而扩大匹配范围。
func (s *CourseOutlineService) ListAvailablePublishers(
        ctx context.Context,
        userID string,
        subject string,
        grade string,
) ([]string, error) {
        items, domain, err := s.ListExactCandidates(
                ctx,
                userID,
                subject,
                grade,
        )
        if err != nil {
                return nil, err
        }

        if domain != models.EducationDomainK12 ||
                len(items) == 0 {
                return []string{}, nil
        }

        seen := make(
                map[string]struct{},
                len(items),
        )

        publishers := make(
                []string,
                0,
                len(items),
        )

        for _, item := range items {
                if item == nil {
                        continue
                }

                publisher := strings.TrimSpace(
                        item.Publisher,
                )

                if _, exists := seen[publisher]; exists {
                        continue
                }

                seen[publisher] = struct{}{}

                publishers = append(
                        publishers,
                        publisher,
                )
        }

        sort.SliceStable(
                publishers,
                func(i int, j int) bool {
                        if publishers[i] == "" {
                                return false
                        }

                        if publishers[j] == "" {
                                return true
                        }

                        return publishers[i] <
                                publishers[j]
                },
        )

        return publishers, nil
}

// normalizeCourseOutlineSchoolSystemForDomain 规范化学制。
//
// K12创建时空值按standard兼容旧客户端；
// K12更新时空值保留原学制；
// 非K12只允许空值或standard。
func normalizeCourseOutlineSchoolSystemForDomain(
        educationDomain string,
        requested string,
        fallback string,
) (string, error) {
        domain := strings.ToLower(
                strings.TrimSpace(educationDomain),
        )

        requested = strings.ToLower(
                strings.TrimSpace(requested),
        )

        fallback = strings.ToLower(
                strings.TrimSpace(fallback),
        )

        if !models.IsTeachingEducationDomain(
                domain,
        ) {
                return "",
                        ErrOutlineEducationDomainRequired
        }

        if domain != models.EducationDomainK12 {
                if requested != "" &&
                        requested !=
                                models.CourseOutlineSchoolSystemStandard {
                        return "",
                                ErrOutlineSchoolSystemInvalid
                }

                return models.
                                CourseOutlineSchoolSystemStandard,
                        nil
        }

        value := requested
        if value == "" {
                value = fallback
        }

        if value == "" {
                value =
                        models.CourseOutlineSchoolSystemStandard
        }

        if !models.IsValidCourseOutlineSchoolSystem(
                value,
        ) {
                return "",
                        ErrOutlineSchoolSystemInvalid
        }

        return value, nil
}
