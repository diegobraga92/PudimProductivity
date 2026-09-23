import config from "../config";
import { apiHeaders, apiIdentityHeaders } from "./client";

/**
 * Sound library management API.
 *
 *   GET    /api/v1/sounds      → catalog (built-in sounds + user-added sounds)
 *   POST   /api/v1/sounds      → multipart upload (label, icon, file)
 *   PUT    /api/v1/sounds/{id} → rename / re-icon a user-added sound
 *   DELETE /api/v1/sounds/{id} → remove a user-added sound
 *
 * Built-in sounds cannot be modified: the backend answers 409 for them.
 */

/** One entry in the backend sound catalog. */
export interface SoundEntry {
  id: string;
  file: string;
  mime: string;
  /** Name of a user-added sound (built-ins are named by the client). */
  label?: string;
  /** Emoji of a user-added sound. */
  icon?: string;
  /** True for sounds added by the user rather than shipped with the app. */
  custom?: boolean;
}

/** Largest audio file the backend accepts (mirrors maxSoundFileBytes in Go). */
export const MAX_SOUND_BYTES = 10 * 1024 * 1024;

/** Audio formats the backend accepts, as a file-input accept list. */
export const SOUND_FILE_ACCEPT = ".mp3,.ogg,.m4a,.wav";

async function handleError(response: Response, fallback: string): Promise<never> {
  // A reverse proxy in front of the backend answers 413 with an HTML page, so
  // the JSON body cannot be used to explain the failure.
  if (response.status === 413) {
    throw new Error("That file is too large (max 10 MB).");
  }
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

/** Adds a sound: its name, icon and audio file in one multipart request. */
export async function createSound(input: {
  label: string;
  icon: string;
  file: File;
}): Promise<SoundEntry> {
  const form = new FormData();
  form.append("label", input.label);
  form.append("icon", input.icon);
  form.append("file", input.file);

  const res = await fetch(`${config.apiBaseUrl}/sounds`, {
    method: "POST",
    // No Content-Type header: the browser sets it, boundary included.
    headers: apiIdentityHeaders(),
    body: form,
  });
  if (!res.ok) await handleError(res, `Failed to add sound: ${res.status}`);
  return res.json() as Promise<SoundEntry>;
}

/** Renames a sound the user added (and/or changes its icon). */
export async function updateSound(
  id: string,
  input: { label: string; icon: string },
): Promise<SoundEntry> {
  const res = await fetch(`${config.apiBaseUrl}/sounds/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(input),
  });
  if (!res.ok) await handleError(res, `Failed to update sound: ${res.status}`);
  return res.json() as Promise<SoundEntry>;
}

/** Deletes a sound the user added, along with its uploaded audio file. */
export async function deleteSound(id: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/sounds/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete sound: ${res.status}`);
}
