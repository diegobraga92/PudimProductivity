import { request } from "./client";
import type { components } from "./generated/tasks-v1";

/** Collaboration API client; sharing and presence are user-scoped. */

export type TaskListMember = components["schemas"]["TaskListMember"];
export type ShareTaskListRequest = components["schemas"]["ShareTaskListRequest"];

/** Grants a user editor or viewer access to a task list. Owner-only. */
export async function shareTaskList(listId: string, req: ShareTaskListRequest): Promise<void> {
  return request<void>(`/task-lists/${listId}/share`, {
    method: "POST",
    body: JSON.stringify(req),
    action: "share task list",
  });
}

/** Revokes a user's access to a task list. Owner-only. */
export async function unshareTaskList(listId: string, userId: string): Promise<void> {
  return request<void>(`/task-lists/${listId}/share/${encodeURIComponent(userId)}`, {
    method: "DELETE",
    action: "unshare task list",
  });
}

/** Lists the shared (non-owner) members of a task list. */
export async function listTaskListMembers(listId: string): Promise<TaskListMember[]> {
  return request<TaskListMember[]>(`/task-lists/${listId}/members`, {
    action: "list members",
  });
}

export interface ListPresenceResponse {
  list_id: string;
  online: string[];
}

/** Users currently connected to a list; live changes arrive as presence events. */
export async function getListPresence(listId: string): Promise<ListPresenceResponse> {
  return request<ListPresenceResponse>(`/presence/${listId}`, { action: "fetch presence" });
}
