/**
 * DbCheck步骤调试面板
 * P4.5-B: 展示课程索引验证结果
 */
import { kvRow, kvLabel, kvValue, passStyle, failStyle } from './StepCommon'

/** DbCheck面板属性 */
interface DbCheckPanelProps {
  data: any
}

/** DbCheck步骤调试面板 */
export default function DbCheckPanel({ data }: DbCheckPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      <div style={kvRow}>
        <span style={kvLabel}>课程编号</span>
        <span style={kvValue}>{data.course_code || '-'}</span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>课程ID</span>
        <span style={kvValue} title={data.course_id}>
          {data.course_id ? data.course_id.substring(0, 8) + '...' : '-'}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>模块ID</span>
        <span style={kvValue}>{data.module_id ?? '-'}</span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>索引状态</span>
        <span style={kvValue}>
          {data.has_index
            ? <span style={passStyle}>已有索引</span>
            : <span style={failStyle}>无索引</span>}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>页面数</span>
        <span style={kvValue}>{data.page_count ?? '-'}</span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>总长度</span>
        <span style={kvValue}>{data.total_length ? data.total_length.toLocaleString() + ' 字符' : '-'}</span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>索引哈希</span>
        <span style={kvValue} title={data.index_hash}>
          {data.index_hash ? data.index_hash.substring(0, 16) + '...' : '-'}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>验证结果</span>
        <span style={kvValue}>
          {data.is_valid
            ? <span style={passStyle}>通过</span>
            : <span style={failStyle}>不通过 — {data.error_detail || '未知错误'}</span>}
        </span>
      </div>
    </div>
  )
}
