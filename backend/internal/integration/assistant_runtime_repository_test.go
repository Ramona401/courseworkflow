package integration

// assistant_runtime_repository_test.go
//
// 使用真实tedna_test验证：
//   - 首发部署和版本1处于同一事务；
//   - 活动部署唯一约束；
//   - revoked后允许重建；
//   - 并发版本追加严格连续且无重复。
//
// 运行会话和结算测试位于assistant_runtime_settlement_test.go。

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"tedna/internal/database"
	"tedna/internal/repository"
)

// TestAssistantRuntimeRepositoryDeploymentTransaction 验证首发事务。
func TestAssistantRuntimeRepositoryDeploymentTransaction(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
	)
	CleanAndSeed(t)

	fixture := SeedAssistantRuntimeFixture(
		t,
	)

	// 让版本JSON转换失败，验证前一步部署INSERT随事务回滚。
	brokenDeployment,
		brokenVersion :=
		fixture.NewDeploymentRecords()

	brokenVersion.TeachingPlanJSON =
		"{invalid-json"

	err := repository.CreateAssistantDeploymentWithFirstVersion(
		context.Background(),
		brokenDeployment,
		brokenVersion,
	)
	if err == nil {
		t.Fatal(
			"非法版本JSON应使首发事务失败",
		)
	}

	var deploymentCount int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM assistant_deployments
		`,
	).Scan(
		&deploymentCount,
	); err != nil {
		t.Fatalf(
			"查询首发回滚结果失败: %v",
			err,
		)
	}

	if deploymentCount != 0 {
		t.Fatalf(
			"首发失败后留下半部署: %d",
			deploymentCount,
		)
	}

	deployment,
		version :=
		fixture.CreateDeployment(
			t,
		)

	if deployment.ID == "" ||
		deployment.PublicID == "" ||
		deployment.CurrentVersion != 1 ||
		version.DeploymentID != deployment.ID ||
		version.Version != 1 {
		t.Fatalf(
			"首发回填结果错误: deployment=%+v version=%+v",
			deployment,
			version,
		)
	}

	var (
		storedCurrentVersion int
		versionCount         int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			d.current_version,
			COUNT(v.version)::integer
		FROM assistant_deployments AS d
		JOIN assistant_deployment_versions AS v
		  ON v.deployment_id = d.id
		WHERE d.id = $1
		GROUP BY d.current_version
		`,
		deployment.ID,
	).Scan(
		&storedCurrentVersion,
		&versionCount,
	); err != nil {
		t.Fatalf(
			"查询首发部署和版本失败: %v",
			err,
		)
	}

	if storedCurrentVersion != 1 ||
		versionCount != 1 {
		t.Fatalf(
			"首发事务结果错误: current=%d versions=%d",
			storedCurrentVersion,
			versionCount,
		)
	}

	duplicateDeployment,
		duplicateVersion :=
		fixture.NewDeploymentRecords()

	err = repository.CreateAssistantDeploymentWithFirstVersion(
		context.Background(),
		duplicateDeployment,
		duplicateVersion,
	)
	if !errors.Is(
		err,
		repository.ErrAssistantDeploymentPageAlreadyLive,
	) {
		t.Fatalf(
			"同页重复活动部署应被唯一约束拒绝: %v",
			err,
		)
	}

	if _, err := database.DB.Exec(
		context.Background(),
		`
		UPDATE assistant_deployments
		SET
			status = 'revoked',
			updated_at = NOW()
		WHERE id = $1
		`,
		deployment.ID,
	); err != nil {
		t.Fatalf(
			"撤销测试部署失败: %v",
			err,
		)
	}

	replacementDeployment,
		replacementVersion :=
		fixture.NewDeploymentRecords()

	if err := repository.CreateAssistantDeploymentWithFirstVersion(
		context.Background(),
		replacementDeployment,
		replacementVersion,
	); err != nil {
		t.Fatalf(
			"撤销旧部署后无法建立新部署: %v",
			err,
		)
	}

	if replacementDeployment.ID ==
		deployment.ID {
		t.Fatal(
			"重建部署不应复用旧内部ID",
		)
	}

	if replacementDeployment.PublicID ==
		deployment.PublicID {
		t.Fatal(
			"重建部署不应复用旧public_id",
		)
	}
}

// TestAssistantRuntimeRepositoryConcurrentVersions 验证并发版本序列。
func TestAssistantRuntimeRepositoryConcurrentVersions(
	t *testing.T,
) {
	cfg := testConfig()

	initTestDB(
		t,
		cfg,
	)
	CleanAndSeed(t)

	fixture := SeedAssistantRuntimeFixture(
		t,
	)

	deployment,
		_ :=
		fixture.CreateDeployment(
			t,
		)

	const additionalVersions = 6

	errorChannel := make(
		chan error,
		additionalVersions,
	)

	var waitGroup sync.WaitGroup

	waitGroup.Add(
		additionalVersions,
	)

	for index := 0; index <
		additionalVersions; index++ {
		go func(versionIndex int) {
			defer waitGroup.Done()

			_,
				version :=
				fixture.NewDeploymentRecords()

			version.AssistantPromptSnapshot =
				fmt.Sprintf(
					"并发版本快照-%d",
					versionIndex+2,
				)

			_,
				err :=
				repository.AppendAssistantDeploymentVersion(
					context.Background(),
					deployment.ID,
					deployment.CoursewareID,
					deployment.PageID,
					deployment.OwnerUserID,
					version,
				)

			errorChannel <- err
		}(index)
	}

	waitGroup.Wait()

	close(
		errorChannel,
	)

	for err := range errorChannel {
		if err != nil {
			t.Fatalf(
				"并发追加部署版本失败: %v",
				err,
			)
		}
	}

	expectedVersion :=
		1 + additionalVersions

	var (
		currentVersion   int
		versionCount     int
		distinctVersions int
		minVersion       int
		maxVersion       int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			d.current_version,
			COUNT(v.version)::integer,
			COUNT(DISTINCT v.version)::integer,
			MIN(v.version),
			MAX(v.version)
		FROM assistant_deployments AS d
		JOIN assistant_deployment_versions AS v
		  ON v.deployment_id = d.id
		WHERE d.id = $1
		GROUP BY d.current_version
		`,
		deployment.ID,
	).Scan(
		&currentVersion,
		&versionCount,
		&distinctVersions,
		&minVersion,
		&maxVersion,
	); err != nil {
		t.Fatalf(
			"查询并发版本结果失败: %v",
			err,
		)
	}

	if currentVersion != expectedVersion ||
		versionCount != expectedVersion ||
		distinctVersions != expectedVersion ||
		minVersion != 1 ||
		maxVersion != expectedVersion {
		t.Fatalf(
			"并发版本异常: current=%d count=%d distinct=%d min=%d max=%d",
			currentVersion,
			versionCount,
			distinctVersions,
			minVersion,
			maxVersion,
		)
	}

	rows, err := database.DB.Query(
		context.Background(),
		`
		SELECT version
		FROM assistant_deployment_versions
		WHERE deployment_id = $1
		ORDER BY version
		`,
		deployment.ID,
	)
	if err != nil {
		t.Fatalf(
			"读取版本序列失败: %v",
			err,
		)
	}
	defer rows.Close()

	expected := 1

	for rows.Next() {
		var actual int

		if err := rows.Scan(
			&actual,
		); err != nil {
			t.Fatalf(
				"扫描版本号失败: %v",
				err,
			)
		}

		if actual != expected {
			t.Fatalf(
				"版本号不连续: expected=%d actual=%d",
				expected,
				actual,
			)
		}

		expected++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf(
			"遍历版本序列失败: %v",
			err,
		)
	}
}
