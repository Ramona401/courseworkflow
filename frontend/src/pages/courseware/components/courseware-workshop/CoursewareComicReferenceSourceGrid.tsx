/**
 * CoursewareComicReferenceSourceGrid.tsx
 *
 * 知识点漫画可选参考资料的展示网格。
 *
 * 本组件是纯展示与用户意图收集层：
 *   - 不访问后端；
 *   - 不保存正式参考资源；
 *   - 不上传文件；
 *   - 只把教师选择转换成待绑定请求。
 */

import type {
  ChangeEventHandler,
  ReactNode,
} from 'react'

import type {
  CoursewareAsset,
  CoursewareComicTextbookUnit,
  CoursewareListItem,
} from '@/api/coursewares'

import type {
  CourseOutlineListItem,
} from '@/api/course-outlines'

import type {
  CreateCoursewareComicReferenceInput,
} from '@/api/coursewares.comic.references'

import {
  addButtonStyle,
  cardBodyStyle,
  cardStyle,
  cardTitleStyle,
  controlStyle,
  defaultCoursewareComicReferenceImageName,
  emptyStyle,
  gridStyle,
  hiddenInputStyle,
  textareaStyle,
  uploadLabelStyle,
} from './coursewareComicReferencePickerHelpers'

interface CoursewareComicReferenceSourceGridProps {
  disabled: boolean
  loading: boolean
  working: boolean
  full: boolean

  gradeNumber: number

  publishers: string[]
  publisher: string

  onPublisherChange: (
    value: string,
  ) => void

  units:
    CoursewareComicTextbookUnit[]

  selectedUnitID: string

  onSelectedUnitIDChange: (
    value: string,
  ) => void

  coursewares:
    CoursewareListItem[]

  selectedCoursewareID: string

  onSelectedCoursewareIDChange: (
    value: string,
  ) => void

  outlines:
    CourseOutlineListItem[]

  selectedOutlineID: string

  onSelectedOutlineIDChange: (
    value: string,
  ) => void

  imageAssets:
    CoursewareAsset[]

  selectedAssetID: string

  onSelectedAssetIDChange: (
    value: string,
  ) => void

  otherTitle: string

  onOtherTitleChange: (
    value: string,
  ) => void

  otherText: string

  onOtherTextChange: (
    value: string,
  ) => void

  onAddRequestReference: (
    key: string,
    label: string,
    input:
      Omit<
        CreateCoursewareComicReferenceInput,
        'sort_order'
      >,
  ) => void

  onDocumentChange:
    ChangeEventHandler<HTMLInputElement>

  onImageChange:
    ChangeEventHandler<HTMLInputElement>
}

export default function CoursewareComicReferenceSourceGrid({
  disabled,
  loading,
  working,
  full,
  gradeNumber,
  publishers,
  publisher,
  onPublisherChange,
  units,
  selectedUnitID,
  onSelectedUnitIDChange,
  coursewares,
  selectedCoursewareID,
  onSelectedCoursewareIDChange,
  outlines,
  selectedOutlineID,
  onSelectedOutlineIDChange,
  imageAssets,
  selectedAssetID,
  onSelectedAssetIDChange,
  otherTitle,
  onOtherTitleChange,
  otherText,
  onOtherTextChange,
  onAddRequestReference,
  onDocumentChange,
  onImageChange,
}: CoursewareComicReferenceSourceGridProps) {
  return (
    <div
      style={gridStyle}
    >
      <SourceCard
        title="📚 课本单元"
      >
        {gradeNumber <= 0 ? (
          <div style={emptyStyle}>
            当前年级无法匹配教材年级
          </div>
        ) : (
          <>
            <select
              value={publisher}
              disabled={
                disabled ||
                loading
              }
              onChange={event =>
                onPublisherChange(
                  event.target.value,
                )
              }
              style={controlStyle}
            >
              <option value="">
                选择教材版本
              </option>

              {publishers.map(
                item => (
                  <option
                    key={item}
                    value={item}
                  >
                    {item}
                  </option>
                ),
              )}
            </select>

            <select
              value={
                selectedUnitID
              }
              disabled={
                disabled ||
                !publisher
              }
              onChange={event =>
                onSelectedUnitIDChange(
                  event.target.value,
                )
              }
              style={controlStyle}
            >
              <option value="">
                选择单元
              </option>

              {units.map(
                unit => (
                  <option
                    key={unit.id}
                    value={unit.id}
                  >
                    {unit.unit_title}
                    {unit.lesson_title
                      ? ` · ${unit.lesson_title}`
                      : ''}
                  </option>
                ),
              )}
            </select>

            <button
              type="button"
              disabled={
                disabled ||
                !selectedUnitID ||
                full
              }
              onClick={() => {
                const unit =
                  units.find(
                    item =>
                      item.id ===
                      selectedUnitID,
                  )

                if (!unit) {
                  return
                }

                onAddRequestReference(
                  `textbook:${unit.id}`,
                  `${unit.publisher} · ${unit.unit_title}`,
                  {
                    resource_type:
                      'textbook_unit',
                    source_id:
                      unit.id,
                  },
                )
              }}
              style={addButtonStyle}
            >
              添加课本单元
            </button>
          </>
        )}
      </SourceCard>

      <SourceCard
        title="🖥️ 已有课件"
      >
        <select
          value={
            selectedCoursewareID
          }
          disabled={
            disabled ||
              loading
          }
          onChange={event =>
            onSelectedCoursewareIDChange(
              event.target.value,
            )
          }
          style={controlStyle}
        >
          <option value="">
            选择自己的同年级课件
          </option>

          {coursewares.map(
            item => (
              <option
                key={item.id}
                value={item.id}
              >
                {item.title}
              </option>
            ),
          )}
        </select>

        <button
          type="button"
          disabled={
            disabled ||
              !selectedCoursewareID ||
              full
          }
          onClick={() => {
            const selected =
              coursewares.find(
                item =>
                  item.id ===
                  selectedCoursewareID,
              )

            if (!selected) {
              return
            }

            onAddRequestReference(
              `courseware:${selected.id}`,
              selected.title,
              {
                resource_type:
                  'courseware',
                source_id:
                  selected.id,
              },
            )
          }}
          style={addButtonStyle}
        >
          添加已有课件
        </button>
      </SourceCard>

      <SourceCard
        title="📋 课程大纲"
      >
        <select
          value={
            selectedOutlineID
          }
          disabled={
            disabled ||
              loading
          }
          onChange={event =>
            onSelectedOutlineIDChange(
              event.target.value,
            )
          }
          style={controlStyle}
        >
          <option value="">
            选择可见课程大纲
          </option>

          {outlines.map(
            item => (
              <option
                key={item.id}
                value={item.id}
              >
                {item.title}
              </option>
            ),
          )}
        </select>

        <button
          type="button"
          disabled={
            disabled ||
              !selectedOutlineID ||
              full
          }
          onClick={() => {
            const selected =
              outlines.find(
                item =>
                  item.id ===
                  selectedOutlineID,
              )

            if (!selected) {
              return
            }

            onAddRequestReference(
              `outline:${selected.id}`,
              selected.title,
              {
                resource_type:
                  'course_outline',
                source_id:
                  selected.id,
              },
            )
          }}
          style={addButtonStyle}
        >
          添加课程大纲
        </button>
      </SourceCard>

      <SourceCard
        title="🖼️ 图片资料"
      >
        <select
          value={
            selectedAssetID
          }
          disabled={
            disabled ||
              loading
          }
          onChange={event =>
            onSelectedAssetIDChange(
              event.target.value,
            )
          }
          style={controlStyle}
        >
          <option value="">
            选择当前课件已有图片
          </option>

          {imageAssets.map(
            (
              asset,
              index,
            ) => (
              <option
                key={asset.id}
                value={asset.id}
              >
                图片 {index + 1}
                {' · '}
                {asset.mime_type ||
                  'image'}
              </option>
            ),
          )}
        </select>

        <button
          type="button"
          disabled={
            disabled ||
              !selectedAssetID ||
              full
          }
          onClick={() => {
            const asset =
              imageAssets.find(
                item =>
                  item.id ===
                  selectedAssetID,
              )

            if (!asset) {
              return
            }

            onAddRequestReference(
              `asset:${asset.id}`,
              '当前课件参考图片',
              {
                resource_type:
                  'uploaded_image',
                asset_id:
                  asset.id,
                title:
                  '当前课件参考图片',
                file_name:
                  defaultCoursewareComicReferenceImageName(
                    asset.mime_type,
                  ),
                mime_type:
                  asset.mime_type ||
                  'image/png',
              },
            )
          }}
          style={addButtonStyle}
        >
          添加已有图片
        </button>

        <label
          style={uploadLabelStyle}
        >
          上传新图片

          <input
            type="file"
            accept="image/*"
            disabled={
              disabled ||
                full
            }
            onChange={
              onImageChange
            }
            style={hiddenInputStyle}
          />
        </label>
      </SourceCard>

      <SourceCard
        title="📄 文档资料"
      >
        <div
          style={emptyStyle}
        >
          支持DOCX和文字型PDF，文件只在浏览器中提取文字。
        </div>

        <label
          style={uploadLabelStyle}
        >
          选择文档

          <input
            type="file"
            accept=".docx,.pdf,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            disabled={
              disabled ||
                working ||
                full
            }
            onChange={
              onDocumentChange
            }
            style={hiddenInputStyle}
          />
        </label>
      </SourceCard>

      <SourceCard
        title="✍️ 其他文字"
      >
        <input
          value={otherTitle}
          disabled={disabled}
          onChange={event =>
            onOtherTitleChange(
              event.target.value,
            )
          }
          placeholder="资料标题，可不填"
          style={controlStyle}
        />

        <textarea
          value={otherText}
          disabled={disabled}
          onChange={event =>
            onOtherTextChange(
              event.target.value,
            )
          }
          rows={3}
          placeholder="粘贴补充说明、课堂案例或教学资料"
          style={textareaStyle}
        />

        <button
          type="button"
          disabled={
            disabled ||
              !otherText.trim() ||
              full
          }
          onClick={() => {
            const content =
              otherText.trim()

            if (!content) {
              return
            }

            const title =
              otherTitle.trim() ||
              '其他参考资料'

            onAddRequestReference(
              `text:${Date.now()}:${content.length}`,
              title,
              {
                resource_type:
                  'other_text',
                title,
                content_text:
                  content,
              },
            )

            onOtherTitleChange('')
            onOtherTextChange('')
          }}
          style={addButtonStyle}
        >
          添加文字资料
        </button>
      </SourceCard>
    </div>
  )
}

interface SourceCardProps {
  title: string
  children: ReactNode
}

function SourceCard({
  title,
  children,
}: SourceCardProps) {
  return (
    <div style={cardStyle}>
      <div style={cardTitleStyle}>
        {title}
      </div>

      <div style={cardBodyStyle}>
        {children}
      </div>
    </div>
  )
}
