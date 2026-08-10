import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/tasks-v1";

/**
 * Phase 8 collaboration API client. Backend contract: api/openapi/tasks-v1.yaml.
 * Sharing + presence are user-scoped via the dev identity headers.
 */

export type TaskListMember = components["schemas"]["TaskListMember"];
export type ShareTaskListRequest = components["schemas"]["ShareTaskListRequest"];

/**
 * Grants a user editor or viewer access to a task list. Owner-only.
 */
export async function shareTaskList(
  listId: string,
  req: ShareTaskListRequest
): Promise<void> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists/${listId}/share`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to share task list: ${response.status}`);
  }
}

/**
 * Revokes a user's access to a task list. Owner-only.
 */
export async function unshareTaskList(listId: string, userId: string): Promise<void> {
  const response = await fetch(
    `${config.apiBaseUrl}/task-lists/${listId}/share/${encodeURIComponent(userId)}`,
    { method: "DELETE", headers: apiHeaders() }
  );
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to unshare task list: ${response.status}`);
  }
}

/**
 * Lists the shared (non-owner) members of a task list.
 */
export async function listTaskListMembers(listId: string): Promise<TaskListMember[]> {
  const response = await fetch(
    `${config.apiBaseUrl}/task-lists/${listId}/members`,
    { headers: apiHeaders() }
  );
  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to list members: ${response.status}`);
  }
  return response.json() as Promise<TaskListMember[]>;
}

export interface ListPresenceResponse {
  list_id: string;
  online: string[];
}

/**
 * Snapshot of the users currently connected and able to access a list. Live
 * updates arrive via presence.online / presence.offline WebSocket events.
 */
export async function getListPresence(listId: string): Promise<ListPresenceResponse> {
  const response = await fetch(`${config.apiBaseUrl}/presence/${listId}`);
  if (!response.ok) {
    throw new Error(`Failed to fetch presence: ${response.status}`);
  }
  return response.json() as Promise<ListPresenceResponse>;
}

/**
 * Sends a CRDT merge for a task. On HTTP 409 the write lost and the server
 * returns the winning task state; the caller should reconcile against it.
 */
export async function mergeTask(
  taskId: string,
  fields: Record<string, unknown>
): Promise<{ task: import("./tasks").Task; conflict: boolean }> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}/merge`, {
    method: "PATCH",
    headers: apiHeaders(),
    body: JSON.stringify(fields),
  });
  const body = (await response.json().catch(() => null)) as import("./tasks").Task | null;
  if (!response.ok) {
    if (response.status === 409 && body) {
      // Lost the merge: server returned the winning state for reconciliation.
      return { task: body, conflict: true };
    }
    throw new Error(`Failed to merge task: ${response.status}`);
  }
  return { task: body as import("./tasks").Task, conflict: false };
}
