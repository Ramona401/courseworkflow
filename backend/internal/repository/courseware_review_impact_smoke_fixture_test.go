package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tedna/internal/database"
	"tedna/internal/models"
)

type r07SmokeFixture struct {
	sessionID    string
	coursewareID string
	actorID      string
	ownerID      string
	reviewLevel  int
	baseItemID   string
	marker       string
}

func newR07SmokeFixture(
	t *testing.T,
	ctx context.Context,
) *r07SmokeFixture {
	t.Helper()

	if os.Getenv("TEDNA_R07_SMOKE_DB") != "1" {
		t.Fatal("R-07 smoke硬门禁失败：TEDNA_R07_SMOKE_DB必须显式为1")
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	user := strings.TrimSpace(os.Getenv("DB_USER"))
	password := os.Getenv("DB_PASSWORD")
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))

	if host == "" ||
		port == "" ||
		user == "" ||
		password == "" ||
		dbName == "" {
		t.Fatal("R-07 smoke数据库环境变量不完整")
	}

	if !strings.HasPrefix(dbName, "tedna_r07_smoke_") {
		t.Fatalf("R-07 smoke拒绝连接非临时库：%s", dbName)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		url.PathEscape(dbName),
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建R-07 smoke连接池失败: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("连接R-07 smoke数据库失败: %v", err)
	}

	database.DB = pool

	t.Cleanup(func() {
		pool.Close()
		database.DB = nil
	})

	var currentDB string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT current_database()`,
	).Scan(&currentDB); err != nil {
		t.Fatalf("读取当前数据库名失败: %v", err)
	}

	if currentDB != dbName ||
		!strings.HasPrefix(currentDB, "tedna_r07_smoke_") {
		t.Fatalf(
			"R-07 smoke数据库二次门禁失败：current=%s expected=%s",
			currentDB,
			dbName,
		)
	}

	var pgcryptoSchema string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT extnamespace::regnamespace::text
		 FROM pg_extension
		 WHERE extname = 'pgcrypto'`,
	).Scan(&pgcryptoSchema); err != nil {
		t.Fatalf("R-07 smoke临时库缺少pgcrypto: %v", err)
	}

	if pgcryptoSchema != "public" {
		t.Fatalf(
			"R-07 smoke pgcrypto schema=%s，期望public",
			pgcryptoSchema,
		)
	}

	var digestResult string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT encode(
			 public.digest(
				 convert_to('r07-smoke-fixture', 'UTF8'),
				 'sha256'
			 ),
			 'hex'
		 )`,
	).Scan(&digestResult); err != nil {
		t.Fatalf("R-07 smoke临时库digest不可用: %v", err)
	}

	if len(digestResult) != 64 {
		t.Fatalf(
			"R-07 smoke digest结果长度=%d，期望64",
			len(digestResult),
		)
	}

	fixture := &r07SmokeFixture{
		marker: fmt.Sprintf(
			"r07-smoke-%d",
			time.Now().UnixNano(),
		),
	}

	err = database.DB.QueryRow(
		ctx,
		`SELECT
			 session.id::text,
			 session.courseware_id::text,
			 session.reviewer_id::text,
			 courseware.user_id::text,
			 session.review_level,
			 item.id::text
		 FROM courseware_ai_review_sessions AS session
		 INNER JOIN coursewares AS courseware
		    ON courseware.id = session.courseware_id
		 INNER JOIN courseware_review_items AS item
		    ON item.source_session_id = session.id
		 WHERE session.status = 'done'
		   AND session.review_level > 0
		 ORDER BY
			 session.created_at DESC,
			 item.created_at DESC
		 LIMIT 1`,
	).Scan(
		&fixture.sessionID,
		&fixture.coursewareID,
		&fixture.actorID,
		&fixture.ownerID,
		&fixture.reviewLevel,
		&fixture.baseItemID,
	)
	if err != nil {
		t.Fatalf(
			"smoke副本中找不到可复用的已完成正式AI审核会话/整改项: %v",
			err,
		)
	}

	return fixture
}

func (fixture *r07SmokeFixture) newUUID(
	t *testing.T,
	ctx context.Context,
) string {
	t.Helper()

	var value string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT gen_random_uuid()::text`,
	).Scan(&value); err != nil {
		t.Fatalf("生成smoke UUID失败: %v", err)
	}

	return value
}

func (fixture *r07SmokeFixture) cloneDetectedItem(
	t *testing.T,
	ctx context.Context,
	suffix string,
) *models.CoursewareReviewItem {
	t.Helper()

	itemID := fixture.newUUID(t, ctx)
	sourceFindingID := fixture.marker + "-" + suffix

	var insertedID string

	/*
		这里刻意不覆盖origin_type和source_global_message_id。

		两者直接继承数据库中已经合法的来源整改项：
		  - ai_finding继续保持NULL source message；
		  - global_discussion_manual继续保持原可信assistant消息。

		Smoke测试目标是Atomic Apply，不应该人为制造一套新的来源事实。
	*/
	err := database.DB.QueryRow(
		ctx,
		`INSERT INTO courseware_review_items
		 SELECT (
			 jsonb_populate_record(
				 NULL::courseware_review_items,
				 to_jsonb(source_item)
				 ||
				 jsonb_build_object(
					 'id', $1::uuid,
					 'courseware_id', $2::uuid,
					 'source_session_id', $3::uuid,
					 'source_finding_id', $4::text,
					 'courseware_review_id', NULL::uuid,
					 'feedback_id', NULL::uuid,
					 'source_type', 'formal',
					 'review_level', $5::integer,
					 'review_round', 0,
					 'created_by', $6::uuid,
					 'owner_id', $7::uuid,
					 'page_id', NULL::uuid,
					 'page_number_snapshot', 0,
					 'page_title_snapshot', '',
					 'page_html_hash', '',
					 'page_updated_at_snapshot', NULL::timestamptz,
					 'severity', 'medium',
					 'dimension', 'manual_review',
					 'title', $8::text,
					 'description', $9::text,
					 'original_suggestion', '',
					 'confirmed_instruction', '',
					 'current_instruction_version_id', NULL::uuid,
					 'delivered_instruction_version_id', NULL::uuid,
					 'applied_instruction_version_id', NULL::uuid,
					 'status', 'detected',
					 'applied_page_hash', '',
					 'confirmed_at', NULL::timestamptz,
					 'applied_at', NULL::timestamptz,
					 'resolved_at', NULL::timestamptz,
					 'resolved_by', NULL::uuid,
					 'resolved_review_id', NULL::uuid,
					 'resolved_review_level', 0,
					 'resolved_review_round', 0,
					 'resolution_note', '',
					 'resubmitted_at', NULL::timestamptz,
					 'resubmitted_review_level', 0,
					 'resubmitted_review_round', 0,
					 'created_at', clock_timestamp(),
					 'updated_at', clock_timestamp()
				 )
			 )
		 ).*
		 FROM courseware_review_items AS source_item
		 WHERE source_item.id = $10::uuid
		 RETURNING id::text`,
		itemID,
		fixture.coursewareID,
		fixture.sessionID,
		sourceFindingID,
		fixture.reviewLevel,
		fixture.actorID,
		fixture.ownerID,
		fixture.marker+"-"+suffix,
		"R-07 Atomic Apply隔离smoke整改项",
		fixture.baseItemID,
	).Scan(&insertedID)
	if err != nil {
		t.Fatalf(
			"克隆隔离smoke整改项失败 suffix=%s: %v",
			suffix,
			err,
		)
	}

	item, err := scanCoursewareReviewItem(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewItemSelectColumns+`
			 FROM courseware_review_items
			 WHERE id = $1`,
			insertedID,
		),
	)
	if err != nil {
		t.Fatalf(
			"读取隔离smoke整改项失败 suffix=%s: %v",
			suffix,
			err,
		)
	}

	return item
}

func (fixture *r07SmokeFixture) createTrustedMessage(
	t *testing.T,
	ctx context.Context,
	suffix string,
) string {
	t.Helper()

	var messageID string

	err := database.DB.QueryRow(
		ctx,
		`INSERT INTO courseware_ai_review_messages (
			 session_id,
			 review_item_id,
			 user_id,
			 role,
			 content,
			 citations_json,
			 tokens_used,
			 model_used,
			 created_at
		 )
		 VALUES (
			 $1,
			 NULL,
			 $2,
			 'assistant',
			 $3,
			 '{}'::jsonb,
			 0,
			 'r07-atomic-smoke',
			 clock_timestamp()
		 )
		 RETURNING id::text`,
		fixture.sessionID,
		fixture.actorID,
		fixture.marker+"-"+suffix,
	).Scan(&messageID)
	if err != nil {
		t.Fatalf(
			"创建隔离smoke可信assistant消息失败: %v",
			err,
		)
	}

	return messageID
}

func (fixture *r07SmokeFixture) createPlan(
	t *testing.T,
	ctx context.Context,
	messageID string,
	operations []models.CoursewareReviewImpactOperation,
) *models.CoursewareReviewImpactPlan {
	t.Helper()

	raw, err := json.Marshal(operations)
	if err != nil {
		t.Fatalf(
			"序列化smoke operations失败: %v",
			err,
		)
	}

	plan, err := CreateCoursewareReviewImpactPlanDraft(
		ctx,
		fixture.coursewareID,
		fixture.sessionID,
		messageID,
		fixture.actorID,
		string(raw),
	)
	if err != nil {
		t.Fatalf(
			"创建隔离smoke impact plan失败: %v",
			err,
		)
	}

	return plan
}
