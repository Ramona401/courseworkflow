/**
 * coursewares.ai-review-groups.ts
 *
 * R-06正式问题组前端API。
 *
 * group是教师组织问题的业务集合，与pairwise relation结构事实严格并存。
 *
 * 安全边界：
 *   - 操作者身份、课件、会话、教育域全部由后端重新解析；
 *   - 浏览器只提交教师输入、稳定对象ID和后端最近返回的version；
 *   - 所有写操作最终由后端version/CAS及事务锁裁决；
 *   - 不修改页面、不确认整改指令、不提交审核决定。
 */

import apiClient from "./client";
import { extractData } from "./coursewares.types";

export type CWAIReviewItemGroupStatus =
  | "active"
  | "merged";

export type CWAIReviewItemGroupMemberStatus =
  | "active"
  | "removed";

export interface CWAIReviewItemGroupMember {
  id: string;
  group_id: string;
  item_id: string;

  status: CWAIReviewItemGroupMemberStatus;
  version: number;

  removed_at: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export interface CWAIReviewItemGroupEvent {
  id: string;
  group_version: number;
  event_type:
    | "created"
    | "renamed"
    | "primary_changed"
    | "member_added"
    | "member_removed"
    | "member_moved"
    | "merged"
    | "split";

  member_id: string | null;
  member_version: number | null;
  related_group_id: string | null;

  reason: string;
  metadata_json: string;
  created_at: string | null;
}

export interface CWAIReviewItemGroup {
  id: string;
  courseware_id: string;
  source_session_id: string;

  name: string;
  primary_item_id: string | null;

  status: CWAIReviewItemGroupStatus;
  version: number;

  merged_into_group_id: string | null;

  created_at: string | null;
  updated_at: string | null;

  members: CWAIReviewItemGroupMember[];
  events: CWAIReviewItemGroupEvent[];
}

export interface CWAIReviewItemGroupListResponse {
  groups: CWAIReviewItemGroup[];
}

export interface CWAIReviewItemGroupPairResponse {
  source: CWAIReviewItemGroup;
  target: CWAIReviewItemGroup;
}

export interface CreateCWAIReviewItemGroupRequest {
  name: string;
  item_ids: string[];
  primary_item_id: string;
}

export async function getCWAIReviewItemGroups(
  sessionId: string,
): Promise<CWAIReviewItemGroupListResponse> {
  const response = await apiClient.get(
    `/courseware-ai-reviews/${sessionId}/groups`,
  );

  return extractData<CWAIReviewItemGroupListResponse>(
    response,
  );
}

export async function createCWAIReviewItemGroup(
  sessionId: string,
  request: CreateCWAIReviewItemGroupRequest,
): Promise<CWAIReviewItemGroup> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups`,
    request,
  );

  return extractData<CWAIReviewItemGroup>(
    response,
  );
}

export async function renameCWAIReviewItemGroup(
  sessionId: string,
  groupId: string,
  expectedVersion: number,
  name: string,
): Promise<CWAIReviewItemGroup> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/${groupId}/rename`,
    {
      expected_version: expectedVersion,
      name,
    },
  );

  return extractData<CWAIReviewItemGroup>(
    response,
  );
}

export async function setCWAIReviewItemGroupPrimary(
  sessionId: string,
  groupId: string,
  expectedVersion: number,
  primaryItemId: string,
): Promise<CWAIReviewItemGroup> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/${groupId}/primary`,
    {
      expected_version: expectedVersion,
      primary_item_id: primaryItemId,
    },
  );

  return extractData<CWAIReviewItemGroup>(
    response,
  );
}

export async function addCWAIReviewItemGroupMember(
  sessionId: string,
  groupId: string,
  expectedGroupVersion: number,
  itemId: string,
): Promise<CWAIReviewItemGroup> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/${groupId}/members`,
    {
      expected_group_version: expectedGroupVersion,
      item_id: itemId,
    },
  );

  return extractData<CWAIReviewItemGroup>(
    response,
  );
}

export async function removeCWAIReviewItemGroupMember(
  sessionId: string,
  groupId: string,
  expectedGroupVersion: number,
  member: CWAIReviewItemGroupMember,
  reason: string,
): Promise<CWAIReviewItemGroup> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/${groupId}/remove-member`,
    {
      expected_group_version: expectedGroupVersion,
      expected_member_version: member.version,
      member_id: member.id,
      reason,
    },
  );

  return extractData<CWAIReviewItemGroup>(
    response,
  );
}

export async function moveCWAIReviewItemGroupMember(
  sessionId: string,
  source: CWAIReviewItemGroup,
  target: CWAIReviewItemGroup,
  member: CWAIReviewItemGroupMember,
  reason: string,
): Promise<CWAIReviewItemGroupPairResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/move-member`,
    {
      source_group_id: source.id,
      target_group_id: target.id,
      expected_source_version: source.version,
      expected_target_version: target.version,
      member_id: member.id,
      expected_member_version: member.version,
      reason,
    },
  );

  return extractData<CWAIReviewItemGroupPairResponse>(
    response,
  );
}

export async function mergeCWAIReviewItemGroups(
  sessionId: string,
  source: CWAIReviewItemGroup,
  target: CWAIReviewItemGroup,
  reason: string,
): Promise<CWAIReviewItemGroupPairResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/merge`,
    {
      source_group_id: source.id,
      target_group_id: target.id,
      expected_source_version: source.version,
      expected_target_version: target.version,
      reason,
    },
  );

  return extractData<CWAIReviewItemGroupPairResponse>(
    response,
  );
}

export async function splitCWAIReviewItemGroup(
  sessionId: string,
  source: CWAIReviewItemGroup,
  name: string,
  itemIds: string[],
  primaryItemId: string,
  reason: string,
): Promise<CWAIReviewItemGroupPairResponse> {
  const response = await apiClient.post(
    `/courseware-ai-reviews/${sessionId}/groups/split`,
    {
      source_group_id: source.id,
      expected_source_version: source.version,
      name,
      item_ids: itemIds,
      primary_item_id: primaryItemId,
      reason,
    },
  );

  return extractData<CWAIReviewItemGroupPairResponse>(
    response,
  );
}
