#!/bin/bash
# TE-DNA 全链路验证测试脚本
# 自动创建+启动4门课程Pipeline，结果写入日志

export PATH=$PATH:/usr/local/go/bin

API="https://workflow.pkuailab.com/api/v1"
LOG="/tmp/tedna-test-result.log"
COURSES=("G1-01" "G5-65" "G7-01" "G10-01")
CONFIG='{"threshold":9.0,"eval_rounds":3,"max_meta_retry":3,"max_tr_loop":3}'

echo "========================================" | tee $LOG
echo "TE-DNA 全链路验证测试" | tee -a $LOG
echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a $LOG
echo "模型: anthropic/claude-opus-4.6" | tee -a $LOG
echo "阈值: 9.0 | 评估轮数: 3 | Meta重试: 3 | TR循环: 3" | tee -a $LOG
echo "测试课程: ${COURSES[*]}" | tee -a $LOG
echo "========================================" | tee -a $LOG

get_token() {
    curl -s -X POST "$API/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])"
}

TOKEN=$(get_token)

echo "" | tee -a $LOG
echo "--- 清理旧Pipeline ---" | tee -a $LOG
for code in "${COURSES[@]}"; do
    OLD_IDS=$(PGPASSWORD=9fIbnkYABWXt3VGPv8Pn psql -U tedna_user -d tedna -h 127.0.0.1 -t -A -c \
      "SELECT id FROM pipelines WHERE course_code='$code' AND status IN ('pending','failed','cancelled');")
    for oid in $OLD_IDS; do
        if [ -n "$oid" ]; then
            curl -s -X DELETE "$API/pipelines/$oid" -H "Authorization: Bearer $TOKEN" > /dev/null 2>&1
            echo "  删除 $code 旧Pipeline: $oid" | tee -a $LOG
        fi
    done
done

TOTAL_START=$(date +%s)
PASS_COUNT=0
FAIL_COUNT=0

for i in "${!COURSES[@]}"; do
    code="${COURSES[$i]}"
    num=$((i+1))
    echo "" | tee -a $LOG
    echo "========================================" | tee -a $LOG
    echo "[$num/4] 测试课程: $code" | tee -a $LOG
    echo "开始时间: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a $LOG
    echo "========================================" | tee -a $LOG

    TOKEN=$(get_token)

    CREATE_RESP=$(curl -s -X POST "$API/pipelines" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"course_code\":\"$code\",\"config\":$CONFIG}")

    PL_ID=$(echo "$CREATE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    if [ -z "$PL_ID" ]; then
        echo "  创建失败: $CREATE_RESP" | tee -a $LOG
        FAIL_COUNT=$((FAIL_COUNT+1))
        continue
    fi
    echo "  Pipeline ID: $PL_ID" | tee -a $LOG

    echo "  启动中... (最长等待60分钟)" | tee -a $LOG
    STEP_START=$(date +%s)

    START_RESP=$(curl -s -X POST "$API/pipelines/$PL_ID/start" \
      -H "Authorization: Bearer $TOKEN" \
      --max-time 3600 2>&1)

    STEP_END=$(date +%s)
    ELAPSED=$(( STEP_END - STEP_START ))
    echo "  执行耗时: ${ELAPSED}秒 ($(( ELAPSED/60 ))分$(( ELAPSED%60 ))秒)" | tee -a $LOG

    echo "$START_RESP" | python3 -c "
import sys,json
try:
    data = json.load(sys.stdin)
    if data.get('code')==0:
        d = data['data']
        print(f\"  最终状态: {d['status']} ({d['status_name']})\")
        print(f\"  当前步骤: {d['current_step']} ({d['current_step_name']})\")
        if d.get('error_message'):
            print(f\"  错误信息: {d['error_message'][:200]}\")
        print(f\"  步骤明细:\")
        for s in d.get('steps',[]):
            dur = f\" {s['duration_ms']/1000:.0f}s\" if s.get('duration_ms',0)>0 else ''
            tok = f\" tok={s['tokens_used']}\" if s.get('tokens_used',0)>0 else ''
            err = f\" ERR={s.get('error_message','')[:80]}\" if s.get('error_message') else ''
            print(f\"    {s['step_order']}.{s['step_name']:12s} {s['status']:8s}{dur}{tok}{err}\")
    else:
        print(f\"  执行返回错误: {data.get('message','')}\")
except Exception as e:
    print(f\"  解析响应失败: {e}\")
" 2>&1 | tee -a $LOG

    echo "  --- 分数详情 ---" | tee -a $LOG
    PGPASSWORD=9fIbnkYABWXt3VGPv8Pn psql -U tedna_user -d tedna -h 127.0.0.1 -t -A -c "
    SELECT 'evaluator: avg=' || COALESCE(ps.step_data->>'avg_total','N/A') || ' e1=' || COALESCE(ps.step_data->>'avg_e1','N/A') || ' e2=' || COALESCE(ps.step_data->>'avg_e2','N/A') || ' e3=' || COALESCE(ps.step_data->>'avg_e3','N/A') || ' e4=' || COALESCE(ps.step_data->>'avg_e4','N/A') || ' variance=' || COALESCE(ps.step_data->>'variance','N/A')
    FROM pipeline_steps ps WHERE ps.pipeline_id='$PL_ID' AND ps.step_name='evaluator' AND ps.status='done'
    UNION ALL
    SELECT 'meta: total=' || COALESCE(ps.step_data->>'total_final','N/A') || ' e1=' || COALESCE(ps.step_data->>'e1_final','N/A') || ' e2=' || COALESCE(ps.step_data->>'e2_final','N/A') || ' e3=' || COALESCE(ps.step_data->>'e3_final','N/A') || ' e4=' || COALESCE(ps.step_data->>'e4_final','N/A') || ' hard=' || COALESCE(ps.step_data->>'hard_constraint','N/A') || ' grade=' || COALESCE(ps.step_data->>'grade','N/A') || ' passed=' || COALESCE(ps.step_data->>'passed','N/A')
    FROM pipeline_steps ps WHERE ps.pipeline_id='$PL_ID' AND ps.step_name='meta' AND ps.status='done'
    UNION ALL
    SELECT 'translator: score=' || COALESCE(ps.step_data->>'final_score','N/A') || ' gate=' || COALESCE(ps.step_data->>'final_quality_gate','N/A') || ' grade=' || COALESCE(ps.step_data->>'final_grade','N/A') || ' passed=' || COALESCE(ps.step_data->>'passed','N/A') || ' round=' || COALESCE(ps.step_data->>'final_round','N/A')
    FROM pipeline_steps ps WHERE ps.pipeline_id='$PL_ID' AND ps.step_name='translator' AND ps.status='done'
    UNION ALL
    SELECT 'generator: pages=' || COALESCE(ps.step_data->>'total_pages','N/A') || ' modify=' || COALESCE(ps.step_data->>'modified_pages','N/A') || ' create=' || COALESCE(ps.step_data->>'created_pages','N/A') || ' keep=' || COALESCE(ps.step_data->>'kept_pages','N/A')
    FROM pipeline_steps ps WHERE ps.pipeline_id='$PL_ID' AND ps.step_name='generator' AND ps.status='done';
    " 2>/dev/null | while read line; do echo "    $line" | tee -a $LOG; done

    echo "  --- Eval各轮 ---" | tee -a $LOG
    PGPASSWORD=9fIbnkYABWXt3VGPv8Pn psql -U tedna_user -d tedna -h 127.0.0.1 -t -A -c "
    SELECT 'R' || round_number || ': total=' || COALESCE(score_total::text,'N/A') || ' e1=' || COALESCE(score_e1::text,'N/A') || ' e2=' || COALESCE(score_e2::text,'N/A') || ' e3=' || COALESCE(score_e3::text,'N/A') || ' e4=' || COALESCE(score_e4::text,'N/A') || ' hard=' || COALESCE(dimensions->>'hard_constraint','N/A')
    FROM eval_rounds WHERE pipeline_id='$PL_ID' ORDER BY round_number;
    " 2>/dev/null | while read line; do echo "    $line" | tee -a $LOG; done

    FINAL_STATUS=$(PGPASSWORD=9fIbnkYABWXt3VGPv8Pn psql -U tedna_user -d tedna -h 127.0.0.1 -t -A -c \
      "SELECT status FROM pipelines WHERE id='$PL_ID';")
    if [ "$FINAL_STATUS" = "review_queue" ] || [ "$FINAL_STATUS" = "finalized" ]; then
        PASS_COUNT=$((PASS_COUNT+1))
        echo "  结果: PASS ($FINAL_STATUS)" | tee -a $LOG
    else
        FAIL_COUNT=$((FAIL_COUNT+1))
        echo "  结果: FAIL ($FINAL_STATUS)" | tee -a $LOG
    fi

    echo "完成时间: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a $LOG
done

TOTAL_END=$(date +%s)
TOTAL_ELAPSED=$(( TOTAL_END - TOTAL_START ))

echo "" | tee -a $LOG
echo "========================================" | tee -a $LOG
echo "测试汇总" | tee -a $LOG
echo "========================================" | tee -a $LOG
echo "总耗时: ${TOTAL_ELAPSED}秒 ($(( TOTAL_ELAPSED/60 ))分$(( TOTAL_ELAPSED%60 ))秒)" | tee -a $LOG
echo "通过: $PASS_COUNT / ${#COURSES[@]}" | tee -a $LOG
echo "失败: $FAIL_COUNT / ${#COURSES[@]}" | tee -a $LOG
echo "结束时间: $(date '+%Y-%m-%d %H:%M:%S')" | tee -a $LOG
echo "========================================" | tee -a $LOG
echo "详细日志: $LOG" | tee -a $LOG
