package services

// courseware_auto_assembly_launch.go — 自动装配启动票据与开始/取消竞态协调
//
// Tracked Handler会在登记后台任务前创建一个只属于本次请求的启动票据。
// 该票据解决数据库运行尚未创建前的短窗口：
//
//   1. 正常开始：版本包装入口消费本次票据，再领取数据库运行；
//   2. 启动前取消：取消入口只给当前真实票据绑定取消；
//   3. 空闲误点取消：没有启动票据时不会留下任何状态，不影响后续新任务；
//   4. 登记失败：Handler按原票据清理，不能误删另一请求的票据；
//   5. 部署排空：onDrain在800毫秒缓冲期内也能绑定当前票据并阻止启动；
//   6. 页面刷新：状态接口可以读取pending与skip_video，恢复正确交付模式。
//
// 为什么取消票据不再使用固定TTL：
//
// 启动和取消现在通过LaunchToken精确绑定，且票据会在消费、登记失败、
// 前置检查失败和进程退出时确定性清理。因此无需依赖10秒过期时间。
// 固定TTL反而会在数据库短暂变慢、前置检查超过10秒时让真实取消失效。
//
// 启动票据只存在于当前进程，数据库运行身份仍以assembly_version和run_id为准。

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrCoursewareAutoAssemblyLaunchPending 表示同一课件已有待启动装配请求。
	ErrCoursewareAutoAssemblyLaunchPending = errors.New(
		"课件自动装配正在启动",
	)

	// ErrCoursewareAutoAssemblyLaunchInvalid 表示启动票据缺失或不属于当前请求。
	ErrCoursewareAutoAssemblyLaunchInvalid = errors.New(
		"课件自动装配启动票据无效",
	)
)

// CoursewareAutoAssemblyLaunchState 是状态接口可读取的进程内启动快照。
//
// Token不对外暴露；这里只返回前端恢复所需的pending、交付模式和开始时间。
type CoursewareAutoAssemblyLaunchState struct {
	Pending   bool
	SkipVideo bool
	StartedAt time.Time
}

// coursewareAutoAssemblyLaunchTicket 是一次精确启动的进程内身份。
type coursewareAutoAssemblyLaunchTicket struct {
	Token     string
	SkipVideo bool
	StartedAt time.Time
}

// coursewareAutoAssemblyCancelTicket 与一次精确启动票据绑定。
type coursewareAutoAssemblyCancelTicket struct {
	LaunchToken string
}

var (
	cwAssemblyVersionLocks  sync.Map
	cwAssemblyPendingLaunch sync.Map
	cwAssemblyPendingCancel sync.Map
	cwAssemblyLaunchSeq     uint64
)

// coursewareAssemblyVersionLock 返回课件级开始/取消互斥锁。
func coursewareAssemblyVersionLock(
	coursewareID string,
) *sync.Mutex {
	lock, _ :=
		cwAssemblyVersionLocks.LoadOrStore(
			coursewareID,
			&sync.Mutex{},
		)

	return lock.(*sync.Mutex)
}

// PrepareCoursewareAutoAssemblyLaunch 为一次Tracked启动创建精确票据。
//
// 调用顺序必须是：
//
//	Prepare → TryStartExternal → 成功后启动goroutine；
//	TryStartExternal失败时立即Abort。
func PrepareCoursewareAutoAssemblyLaunch(
	coursewareID string,
	skipVideo bool,
) (string, error) {
	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	if coursewareID == "" {
		return "",
			ErrCoursewareAutoAssemblyLaunchInvalid
	}

	lock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lock.Lock()
	defer lock.Unlock()

	if _, exists :=
		cwAssemblyPendingLaunch.Load(
			coursewareID,
		); exists {
		return "",
			ErrCoursewareAutoAssemblyLaunchPending
	}

	sequence :=
		atomic.AddUint64(
			&cwAssemblyLaunchSeq,
			1,
		)

	token := fmt.Sprintf(
		"%d-%d",
		time.Now().UnixNano(),
		sequence,
	)

	// 新启动不能继承任何旧取消票据。
	cwAssemblyPendingCancel.Delete(
		coursewareID,
	)

	cwAssemblyPendingLaunch.Store(
		coursewareID,
		coursewareAutoAssemblyLaunchTicket{
			Token:     token,
			SkipVideo: skipVideo,
			StartedAt: time.Now().UTC(),
		},
	)

	return token, nil
}

// GetCoursewareAutoAssemblyLaunchState 返回当前进程的待启动快照。
//
// 数据库运行一旦创建，启动票据会先被消费，此函数随后返回Pending=false。
// 因此它只描述数据库running之前的短窗口，不与数据库状态重复。
func GetCoursewareAutoAssemblyLaunchState(
	coursewareID string,
) CoursewareAutoAssemblyLaunchState {
	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	if coursewareID == "" {
		return CoursewareAutoAssemblyLaunchState{}
	}

	lock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lock.Lock()
	defer lock.Unlock()

	value, exists :=
		cwAssemblyPendingLaunch.Load(
			coursewareID,
		)
	ticket, valid :=
		value.(coursewareAutoAssemblyLaunchTicket)

	if !exists ||
		!valid ||
		strings.TrimSpace(
			ticket.Token,
		) == "" {
		return CoursewareAutoAssemblyLaunchState{}
	}

	return CoursewareAutoAssemblyLaunchState{
		Pending:   true,
		SkipVideo: ticket.SkipVideo,
		StartedAt: ticket.StartedAt,
	}
}

// AbortCoursewareAutoAssemblyLaunch 按票据精确清理尚未消费的启动。
//
// token不匹配时保持现状，避免一个失败请求清掉另一个请求的票据。
func AbortCoursewareAutoAssemblyLaunch(
	coursewareID string,
	launchToken string,
) {
	coursewareID = strings.TrimSpace(
		coursewareID,
	)
	launchToken = strings.TrimSpace(
		launchToken,
	)
	if coursewareID == "" ||
		launchToken == "" {
		return
	}

	lock :=
		coursewareAssemblyVersionLock(
			coursewareID,
		)
	lock.Lock()
	defer lock.Unlock()

	currentValue, exists :=
		cwAssemblyPendingLaunch.Load(
			coursewareID,
		)
	currentTicket, valid :=
		currentValue.(coursewareAutoAssemblyLaunchTicket)

	if exists &&
		valid &&
		currentTicket.Token == launchToken {
		cwAssemblyPendingLaunch.Delete(
			coursewareID,
		)
	}

	cancelValue, cancelExists :=
		cwAssemblyPendingCancel.Load(
			coursewareID,
		)
	cancelTicket, cancelValid :=
		cancelValue.(coursewareAutoAssemblyCancelTicket)

	if cancelExists &&
		cancelValid &&
		cancelTicket.LaunchToken == launchToken {
		cwAssemblyPendingCancel.Delete(
			coursewareID,
		)
	}
}

// consumeCoursewareAutoAssemblyLaunchLocked 消费当前启动票据。
//
// 调用方必须已持有coursewareAssemblyVersionLock。
func consumeCoursewareAutoAssemblyLaunchLocked(
	coursewareID string,
	launchToken string,
) (
	cancelled bool,
	err error,
) {
	launchToken = strings.TrimSpace(
		launchToken,
	)

	// 无票据是保留的内部调用模式，不参与800毫秒启动竞态。
	if launchToken == "" {
		return false, nil
	}

	currentValue, exists :=
		cwAssemblyPendingLaunch.Load(
			coursewareID,
		)
	currentTicket, valid :=
		currentValue.(coursewareAutoAssemblyLaunchTicket)

	if !exists ||
		!valid ||
		currentTicket.Token != launchToken {
		return false,
			ErrCoursewareAutoAssemblyLaunchInvalid
	}

	cwAssemblyPendingLaunch.Delete(
		coursewareID,
	)

	cancelValue, cancelExists :=
		cwAssemblyPendingCancel.LoadAndDelete(
			coursewareID,
		)
	if !cancelExists {
		return false, nil
	}

	cancelTicket, cancelValid :=
		cancelValue.(coursewareAutoAssemblyCancelTicket)
	if !cancelValid ||
		cancelTicket.LaunchToken != launchToken {
		return false, nil
	}

	// 精确票据一旦命中即永久表示“本次启动已取消”，不受耗时影响。
	return true, nil
}

// markCoursewareAutoAssemblyPendingCancelLocked 给当前真实启动票据绑定取消。
//
// 调用方必须已持有coursewareAssemblyVersionLock。
// 返回false表示课件当前没有处于数据库运行前窗口的装配请求。
func markCoursewareAutoAssemblyPendingCancelLocked(
	coursewareID string,
) bool {
	launchValue, exists :=
		cwAssemblyPendingLaunch.Load(
			coursewareID,
		)
	launchTicket, valid :=
		launchValue.(coursewareAutoAssemblyLaunchTicket)

	if !exists ||
		!valid ||
		strings.TrimSpace(
			launchTicket.Token,
		) == "" {
		return false
	}

	cwAssemblyPendingCancel.Store(
		coursewareID,
		coursewareAutoAssemblyCancelTicket{
			LaunchToken: launchTicket.Token,
		},
	)

	return true
}
