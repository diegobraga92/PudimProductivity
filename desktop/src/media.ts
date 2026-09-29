/**
 * PudimProductivity desktop media control.
 *
 * Drives the active MPRIS player (Spotify, browsers, mpv, VLC) through the
 * `playerctl` command line tool. Linux only, and every function degrades to a
 * plain result object so a missing binary can never throw across IPC.
 */
import { execFile } from "node:child_process";

const COMMAND_TIMEOUT_MS = 2_000;

export type MediaAction = "play" | "pause";

/** Why a media command did not succeed. */
export type MediaFailure = "unavailable" | "no-players" | "error";

export interface MediaControlResult {
  ok: boolean;
  reason?: MediaFailure;
  message?: string;
}

interface ExecFailure {
  code?: string | number;
  stderr?: string | Buffer;
  message?: string;
}

/** True when this platform exposes MPRIS players through playerctl. */
export function isMediaControlSupported(): boolean {
  return process.platform === "linux";
}

// Wraps execFile so callers get a promise and a rejection they can inspect.
function runPlayerctl(args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    execFile(
      "playerctl",
      args,
      { timeout: COMMAND_TIMEOUT_MS, encoding: "utf8" },
      (err, stdout) => {
        if (err) {
          reject(err);
          return;
        }
        resolve(stdout);
      }
    );
  });
}

/** Returns false when playerctl is not installed on this machine. */
export async function isPlayerctlAvailable(): Promise<boolean> {
  if (!isMediaControlSupported()) return false;
  try {
    await runPlayerctl(["--version"]);
    return true;
  } catch {
    return false;
  }
}

/** Names of the MPRIS players playerctl can currently reach. */
export async function listPlayers(): Promise<string[]> {
  if (!isMediaControlSupported()) return [];
  try {
    return parsePlayerList(await runPlayerctl(["--list-all"]));
  } catch {
    return [];
  }
}

/** Splits the `--list-all` output into trimmed player names. */
export function parsePlayerList(stdout: string): string[] {
  return stdout
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

/**
 * Sends play or pause to one player, or to the playerctl default when no name
 * is given. Never throws, the outcome is always described by the result.
 */
export async function control(action: MediaAction, player?: string): Promise<MediaControlResult> {
  if (!isMediaControlSupported()) return { ok: false, reason: "unavailable" };

  const args = player ? [`--player=${player}`, action] : [action];

  try {
    await runPlayerctl(args);
    return { ok: true };
  } catch (err) {
    return mapExecFailure(err);
  }
}

/**
 * Translates an execFile failure into a result. A missing binary and an empty
 * player list are expected situations, so they get their own reason codes.
 */
export function mapExecFailure(err: unknown): MediaControlResult {
  const failure = err as ExecFailure;

  if (failure?.code === "ENOENT") return { ok: false, reason: "unavailable" };

  const message = String(failure?.stderr ?? failure?.message ?? "").trim();
  if (/no players found/i.test(message)) return { ok: false, reason: "no-players" };

  return { ok: false, reason: "error", message };
}
