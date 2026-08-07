package services

// token_account_search_service.go — 积分账户用户名搜索服务
//
// 本文件给TokenHandler提供范围感知的账户搜索入口。
// 权限范围仍由ResolveTokenScope生成，本服务只消费OwnerIDs白名单，
// 不自行建立第二套角色或权限判断。

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ListAccountsSearchableScoped 查询支持登录用户名搜索的积分账户。
//
// scope语义：
//   - nil：兼容内部调用，不过滤；
//   - admin：OwnerIDs为nil，不过滤；
//   - 非admin：按OwnerIDs白名单收窄；
//   - Blocked：OwnerIDs为空切片，仓储返回空集。
func (s *TokenService) ListAccountsSearchableScoped(
	ctx context.Context,
	accountType string,
	parentAccountID string,
	status string,
	keyword string,
	scope *TokenScope,
	limit int,
	offset int,
) ([]*models.TokenAccountListItem, int, error) {
	var ownerIDs []string

	if scope != nil {
		ownerIDs = scope.OwnerIDs
	}

	return repository.ListTokenAccountsSearchable(
		ctx,
		accountType,
		parentAccountID,
		status,
		keyword,
		ownerIDs,
		limit,
		offset,
	)
}
