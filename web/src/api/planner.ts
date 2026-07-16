import config from "../config";

export interface PlannerEntry {
  id: string;
  title: string;
  days: string[];
  start_time: string;
  end_time: string;
  color: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePlannerEntryRequest {
  title: string;
  days: string[];
  start_time: string;
  end_time: string;
  color?: string;
}

export interface UpdatePlannerEntryRequest {
  title?: string;
  days?: string[];
  start_time?: string;
  end_time?: string;
  color?: string;
}

/**
 * Fetches all planner entries.
 */
export async function listPlannerEntries(): Promise<PlannerEntry[]> {
  const response = await fetch(`${config.apiBaseUrl}/planner`);
  if (!response.ok) {
    throw new Error(`Failed to list planner entries: ${response.status}`);
  }
  return response.json() as Promise<PlannerEntry[]>;
}

/**
 * Creates a new planner entry.
 */
export async function createPlannerEntry(req: CreatePlannerEntryRequest): Promise<PlannerEntry> {
  const response = await fetch(`${config.apiBaseUrl}/planner`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to create planner entry: ${response.status}`);
  }

  return response.json() as Promise<PlannerEntry>;
}

/**
 * Fetches a single planner entry by ID.
 */
export async function getPlannerEntry(entryId: string): Promise<PlannerEntry> {
  const response = await fetch(`${config.apiBaseUrl}/planner/${entryId}`);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Planner entry not found");
    }
    throw new Error(`Failed to get planner entry: ${response.status}`);
  }

  return response.json() as Promise<PlannerEntry>;
}

/**
 * Updates an existing planner entry.
 */
export async function updatePlannerEntry(
  entryId: string,
  req: UpdatePlannerEntryRequest
): Promise<PlannerEntry> {
  const response = await fetch(`${config.apiBaseUrl}/planner/${entryId}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Planner entry not found");
    }
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to update planner entry: ${response.status}`);
  }

  return response.json() as Promise<PlannerEntry>;
}

/**
 * Deletes a planner entry by ID.
 */
export async function deletePlannerEntry(entryId: string): Promise<void> {
  const response = await fetch(`${config.apiBaseUrl}/planner/${entryId}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Planner entry not found");
    }
    throw new Error(`Failed to delete planner entry: ${response.status}`);
  }
}