package handlers

// token_handler_helpers.go — Token处理器公共辅助函数
//
// 集中维护账户范围判断、分配来源判断、路径ID解析和统一错误响应。

import (
	"errors"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

func tokenOwnerInScope(
	scope *services.TokenScope,
	ownerID string,
) bool {
	if scope == nil {
		return false
	}

	if scope.OwnerIDs == nil {
		return true
	}

	for _, id :=
		range scope.OwnerIDs {
		if id == ownerID {
			return true
		}
	}

	return false
}

func tokenSourceAllowed(
	scope *services.TokenScope,
	accountType string,
	ownerID string,
) bool {
	if scope == nil {
		return false
	}

	if scope.IsAdmin {
		return true
	}

	if accountType ==
		models.AccountTypeRegion {
		for _, id :=
			range scope.AllowedRegionOwnerIDs {
			if id == ownerID {
				return true
			}
		}

		return false
	}

	return tokenOwnerInScope(
		scope,
		ownerID,
	)
}

func extractTokenPathID(
	path string,
) string {
	for index :=
		len(path) - 1;
		index >= 0;
		index-- {
		if path[index] != '/' {
			continue
		}

		id :=
			path[index+1:]

		for len(id) > 0 &&
			id[len(id)-1] == '/' {
			id =
				id[:len(id)-1]
		}

		if len(id) > 0 {
			return id
		}
	}

	return ""
}

func extractTokenMiddleID(
	path string,
	suffix string,
) string {
	if len(path) >= len(suffix) &&
		path[len(path)-len(suffix):] ==
			suffix {
		path =
			path[:len(path)-len(suffix)]
	}

	return extractTokenPathID(
		path,
	)
}

func handleTokenError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		repository.ErrTokenAccountNotFound,
	):
		utils.JSON(
			w,
			http.StatusNotFound,
			-1,
			"账户不存在",
			nil,
		)

	case errors.Is(
		err,
		repository.ErrInsufficientBalance,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"积分余额不足",
			nil,
		)

	case errors.Is(
		err,
		repository.ErrAccountSuspended,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"账户已冻结",
			nil,
		)

	case errors.Is(
		err,
		repository.ErrDuplicateAccount,
	):
		utils.JSON(
			w,
			http.StatusConflict,
			-1,
			"该实体已存在同类型账户",
			nil,
		)

	case errors.Is(
		err,
		services.ErrTokenInvalidAmount,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"积分数量必须大于0",
			nil,
		)

	case errors.Is(
		err,
		services.ErrTokenSelfAllocate,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"不能分配给自己",
			nil,
		)

	case errors.Is(
		err,
		services.ErrTokenNotParentChild,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"只能向下级账户分配积分",
			nil,
		)

	case errors.Is(
		err,
		services.ErrTokenAccountNotActive,
	):
		utils.JSON(
			w,
			http.StatusBadRequest,
			-1,
			"账户不在活跃状态",
			nil,
		)

	default:
		utils.JSON(
			w,
			http.StatusInternalServerError,
			-1,
			err.Error(),
			nil,
		)
	}
}
