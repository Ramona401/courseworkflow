/**
 * CoursewareComicReferencePicker.tsx
 *
 * 知识点漫画第一步可选参考资料状态容器。
 *
 * 本文件只负责：
 *   - 加载当前教师可用的正式资料；
 *   - 管理最多8项待绑定资料；
 *   - 在浏览器中提取DOCX与文字型PDF；
 *   - 校验待上传图片；
 *   - 将展示网格委托给独立组件。
 *
 * 各类来源的展示和添加按钮位于：
 * CoursewareComicReferenceSourceGrid.tsx。
 *
 * 通用校验、文件键和样式位于：
 * coursewareComicReferencePickerHelpers.ts。
 */

import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import type {
  ChangeEvent,
} from 'react'

import {
  getCoursewares,
  listCoursewareAssets,
  listCoursewareComicPublishers,
  listCoursewareComicTextbookUnits,
} from '@/api/coursewares'

import type {
  CoursewareAsset,
  CoursewareComicTextbookUnit,
  CoursewareDetail,
  CoursewareListItem,
} from '@/api/coursewares'

import {
  getCourseOutlines,
} from '@/api/course-outlines'

import type {
  CourseOutlineListItem,
} from '@/api/course-outlines'

import {
  extractDocFile,
} from '@/pages/lesson-plans/workshop/utils/docExtract'

import type {
  CreateCoursewareComicReferenceInput,
} from '@/api/coursewares.comic.references'

import type {
  CoursewareComicPendingReference,
} from './coursewareComicQuickCreate'

import CoursewareComicReferenceSourceGrid from './CoursewareComicReferenceSourceGrid'

import {
  MAX_COURSEWARE_COMIC_REFERENCE_IMAGE_BYTES,
  MAX_COURSEWARE_COMIC_REFERENCES,
  bodyStyle,
  containerStyle,
  countStyle,
  documentKey,
  hintStyle,
  imageFileKey,
  messageStyle,
  parseCoursewareComicGradeNumber,
  removeButtonStyle,
  resolveDocumentMimeType,
  resolveReferencePickerErrorMessage,
  selectedItemStyle,
  selectedListStyle,
  summaryStyle,
} from './coursewareComicReferencePickerHelpers'

interface CoursewareComicReferencePickerProps {
  coursewareId: string
  courseware: CoursewareDetail

  value:
    CoursewareComicPendingReference[]

  disabled: boolean

  onChange: (
    references:
      CoursewareComicPendingReference[],
  ) => void
}

export default function CoursewareComicReferencePicker({
  coursewareId,
  courseware,
  value,
  disabled,
  onChange,
}: CoursewareComicReferencePickerProps) {
  const valueRef =
    useRef<
      CoursewareComicPendingReference[]
    >(value)

  const [
    loading,
    setLoading,
  ] = useState(true)

  const [
    working,
    setWorking,
  ] = useState(false)

  const [
    message,
    setMessage,
  ] = useState('')

  const [
    publishers,
    setPublishers,
  ] = useState<string[]>([])

  const [
    publisher,
    setPublisher,
  ] = useState('')

  const [
    units,
    setUnits,
  ] = useState<
    CoursewareComicTextbookUnit[]
  >([])

  const [
    selectedUnitID,
    setSelectedUnitID,
  ] = useState('')

  const [
    coursewares,
    setCoursewares,
  ] = useState<
    CoursewareListItem[]
  >([])

  const [
    selectedCoursewareID,
    setSelectedCoursewareID,
  ] = useState('')

  const [
    outlines,
    setOutlines,
  ] = useState<
    CourseOutlineListItem[]
  >([])

  const [
    selectedOutlineID,
    setSelectedOutlineID,
  ] = useState('')

  const [
    imageAssets,
    setImageAssets,
  ] = useState<
    CoursewareAsset[]
  >([])

  const [
    selectedAssetID,
    setSelectedAssetID,
  ] = useState('')

  const [
    otherTitle,
    setOtherTitle,
  ] = useState('')

  const [
    otherText,
    setOtherText,
  ] = useState('')

  const gradeNumber =
    useMemo(
      () =>
        parseCoursewareComicGradeNumber(
          courseware.grade,
        ),
      [
        courseware.grade,
      ],
    )

  const firstPageNumber =
    courseware.pages?.[0]
      ?.page_number || 1

  const full =
    value.length >=
    MAX_COURSEWARE_COMIC_REFERENCES

  useEffect(
    () => {
      valueRef.current =
        value
    },
    [
      value,
    ],
  )

  useEffect(
    () => {
      let active = true

      const load =
        async () => {
          setLoading(true)
          setMessage('')

          const results =
            await Promise.allSettled([
              getCoursewares({
                subject:
                  courseware.subject,
                limit:
                  100,
                offset:
                  0,
              }),

              getCourseOutlines(),

              listCoursewareAssets(
                coursewareId,
              ),

              gradeNumber > 0
                ? listCoursewareComicPublishers(
                    courseware.subject,
                    gradeNumber,
                  )
                : Promise.resolve(
                    [] as string[],
                  ),
            ])

          if (!active) {
            return
          }

          const coursewareResult =
            results[0]

          if (
            coursewareResult.status ===
            'fulfilled'
          ) {
            setCoursewares(
              (
                coursewareResult.value
                  .coursewares || []
              ).filter(item =>
                item.id !==
                  coursewareId &&
                item.subject ===
                  courseware.subject &&
                item.grade ===
                  courseware.grade &&
                item.education_domain ===
                  courseware.education_domain,
              ),
            )
          }

          const outlineResult =
            results[1]

          if (
            outlineResult.status ===
            'fulfilled'
          ) {
            setOutlines(
              (
                outlineResult.value
                  .outlines || []
              ).filter(item =>
                item.subject ===
                  courseware.subject &&
                item.grade ===
                  courseware.grade,
              ),
            )
          }

          const assetResult =
            results[2]

          if (
            assetResult.status ===
            'fulfilled'
          ) {
            setImageAssets(
              (
                assetResult.value
                  .assets || []
              ).filter(asset =>
                asset.asset_type ===
                  'image' &&
                (
                  asset.status ===
                    'uploaded' ||
                  asset.status ===
                    'confirmed'
                ),
              ),
            )
          }

          const publisherResult =
            results[3]

          if (
            publisherResult.status ===
            'fulfilled'
          ) {
            const nextPublishers =
              publisherResult.value

            setPublishers(
              nextPublishers,
            )

            setPublisher(previous =>
              previous ||
              nextPublishers[0] ||
              '',
            )
          }

          if (
            results.some(
              result =>
                result.status ===
                'rejected',
            )
          ) {
            setMessage(
              '⚠️ 部分已有资料暂时加载失败，仍可上传文件、图片或粘贴文字。',
            )
          }

          setLoading(false)
        }

      void load()

      return () => {
        active = false
      }
    },
    [
      courseware.education_domain,
      courseware.grade,
      courseware.subject,
      coursewareId,
      gradeNumber,
    ],
  )

  useEffect(
    () => {
      let active = true

      if (
        !publisher ||
        gradeNumber <= 0
      ) {
        setUnits([])
        setSelectedUnitID('')

        return () => {
          active = false
        }
      }

      setSelectedUnitID('')

      void listCoursewareComicTextbookUnits({
        subject:
          courseware.subject,
        publisher,
        grade:
          gradeNumber,
      })
        .then(items => {
          if (!active) {
            return
          }

          setUnits(
            items,
          )
        })
        .catch(() => {
          if (!active) {
            return
          }

          setUnits([])

          setMessage(
            '⚠️ 教材单元加载失败，可改用课件、课程大纲或文件资料。',
          )
        })

      return () => {
        active = false
      }
    },
    [
      courseware.subject,
      gradeNumber,
      publisher,
    ],
  )

  const applyReferences =
    (
      next:
        CoursewareComicPendingReference[],
    ) => {
      valueRef.current =
        next

      onChange(
        next,
      )
    }

  const appendReference =
    (
      reference:
        CoursewareComicPendingReference,
    ) => {
      if (disabled) {
        return
      }

      const current =
        valueRef.current

      if (
        current.length >=
        MAX_COURSEWARE_COMIC_REFERENCES
      ) {
        setMessage(
          '⚠️ 每个漫画项目最多选择8项参考资料。',
        )
        return
      }

      if (
        current.some(
          item =>
            item.key ===
            reference.key,
        )
      ) {
        setMessage(
          '⚠️ 该资料已经选择。',
        )
        return
      }

      applyReferences([
        ...current,
        reference,
      ])

      setMessage(
        `✅ 已选择：${reference.label}`,
      )
    }

  const addRequestReference =
    (
      key: string,
      label: string,
      input:
        Omit<
          CreateCoursewareComicReferenceInput,
          'sort_order'
        >,
    ) => {
      appendReference({
        mode:
          'request',
        key,
        label,
        input,
      })
    }

  const removeReference =
    (
      key: string,
    ) => {
      if (disabled) {
        return
      }

      applyReferences(
        valueRef.current.filter(
          item =>
            item.key !==
            key,
        ),
      )
    }

  const handleDocument =
    async (
      event:
        ChangeEvent<HTMLInputElement>,
    ) => {
      const file =
        event.target.files?.[0]

      event.target.value = ''

      if (!file || disabled) {
        return
      }

      setWorking(true)

      setMessage(
        '⏳ 正在浏览器中提取文档文字…',
      )

      try {
        const extracted =
          await extractDocFile(
            file,
          )

        addRequestReference(
          documentKey(
            file,
          ),
          file.name,
          {
            resource_type:
              'uploaded_document',
            title:
              file.name,
            file_name:
              file.name,
            mime_type:
              resolveDocumentMimeType(
                file,
              ),
            content_text:
              extracted.text,
          },
        )
      } catch (error) {
        setMessage(
          '❌ ' +
          resolveReferencePickerErrorMessage(
            error,
            '文档文字提取失败',
          ),
        )
      } finally {
        setWorking(false)
      }
    }

  const handleImage =
    (
      event:
        ChangeEvent<HTMLInputElement>,
    ) => {
      const file =
        event.target.files?.[0]

      event.target.value = ''

      if (!file || disabled) {
        return
      }

      if (
        !file.type.startsWith(
          'image/',
        )
      ) {
        setMessage(
          '⚠️ 请选择图片文件。',
        )
        return
      }

      if (
        file.size >
        MAX_COURSEWARE_COMIC_REFERENCE_IMAGE_BYTES
      ) {
        setMessage(
          '⚠️ 图片不能超过5MB。',
        )
        return
      }

      appendReference({
        mode:
          'image_upload',
        key:
          imageFileKey(
            file,
          ),
        label:
          file.name,
        file,
        pageNumber:
          firstPageNumber,
      })
    }

  return (
    <details
      style={containerStyle}
    >
      <summary
        style={summaryStyle}
      >
        <span>
          📎 可选参考资料
        </span>

        <span
          style={countStyle}
        >
          已选 {value.length}/8
        </span>
      </summary>

      <div
        style={bodyStyle}
      >
        <div
          style={hintStyle}
        >
          不添加也能直接生成。参考资料只用于补充背景、例子、教学方式和视觉方向，知识点正文仍是核心事实。
        </div>

        {value.length > 0 && (
          <div
            style={selectedListStyle}
          >
            {value.map(
              item => (
                <div
                  key={item.key}
                  style={selectedItemStyle}
                >
                  <span>
                    {item.label}
                  </span>

                  <button
                    type="button"
                    disabled={
                      disabled
                    }
                    onClick={() =>
                      removeReference(
                        item.key,
                      )
                    }
                    style={removeButtonStyle}
                  >
                    ×
                  </button>
                </div>
              ),
            )}
          </div>
        )}

        <CoursewareComicReferenceSourceGrid
          disabled={disabled}
          loading={loading}
          working={working}
          full={full}
          gradeNumber={gradeNumber}
          publishers={publishers}
          publisher={publisher}
          onPublisherChange={setPublisher}
          units={units}
          selectedUnitID={selectedUnitID}
          onSelectedUnitIDChange={
            setSelectedUnitID
          }
          coursewares={coursewares}
          selectedCoursewareID={
            selectedCoursewareID
          }
          onSelectedCoursewareIDChange={
            setSelectedCoursewareID
          }
          outlines={outlines}
          selectedOutlineID={
            selectedOutlineID
          }
          onSelectedOutlineIDChange={
            setSelectedOutlineID
          }
          imageAssets={imageAssets}
          selectedAssetID={
            selectedAssetID
          }
          onSelectedAssetIDChange={
            setSelectedAssetID
          }
          otherTitle={otherTitle}
          onOtherTitleChange={
            setOtherTitle
          }
          otherText={otherText}
          onOtherTextChange={
            setOtherText
          }
          onAddRequestReference={
            addRequestReference
          }
          onDocumentChange={
            handleDocument
          }
          onImageChange={
            handleImage
          }
        />

        {(loading || working) && (
          <div style={messageStyle}>
            ⏳
            {working
              ? ' 正在处理文件…'
              : ' 正在加载可选资料…'}
          </div>
        )}

        {message && (
          <div style={messageStyle}>
            {message}
          </div>
        )}
      </div>
    </details>
  )
}
