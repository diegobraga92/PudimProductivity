import config from "../config";

/**
 * WebSocket event types pushed by the backend sync hub (see api/ws/events-v1.json).
 */
type WsEventType =
  | "task.created"
  | "task.updated"
  | "task.deleted"
  | "task.completed"
  | "task.uncompleted"
  | "task.merged"
  | "tasklist.shared"
  | "tasklist.unshared"
  | "presence.online"
  | "presence.offline"
  | "stale";

export interface WsEvent {
  type: WsEventType;
  seq: number;
  timestamp: string;
  payload?: Record<string, unknown> | null;
}

const LAST_SEQ_KEY = "pudim.ws.lastSeq";
const MAX_RECONNECT_DELAY_MS = 30_000;
const INITIAL_RECONNECT_DELAY_MS = 1_000;

/** Builds the ws(s):// URL from the configured API base URL. */
function buildWsUrl(lastSeq: number): string {
  const base = config.apiBaseUrl.replace(/\/+$/, "");
  // Relative base ("/api/v1") → same origin; absolute → http(s) → ws(s).
  const wsBase = base.startsWith("http")
    ? base.replace(/^http/, "ws")
    : `${window.location.origin}${base}`;
  return `${wsBase}/ws?last_seq=${lastSeq}`;
}

/**
 * Singleton WebSocket client for real-time task updates.
 *
 * - Auto-reconnects with exponential backoff.
 * - Tracks the last-seen sequence number (sessionStorage) so reconnects resume
 *   without missing events; a `stale` event triggers a full REST refresh.
 * - Reference-counted: `connect()`/`close()` are safe to call from multiple
 *   consumers; the socket is only torn down when the last consumer disconnects.
 */
export class SyncClient {
  private ws: WebSocket | null = null;
  private lastSeq = 0;
  private reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS;
  private reconnectTimer: number | null = null;
  private stopped = false;
  private refCount = 0;

  private handlers = new Map<WsEventType, Set<(event: WsEvent) => void>>();
  private statusHandlers = new Set<(connected: boolean) => void>();

  constructor() {
    const raw = sessionStorage.getItem(LAST_SEQ_KEY);
    this.lastSeq = raw ? (parseInt(raw, 10) || 0) : 0;
  }

  /** Registers a consumer. The first consumer opens the socket. */
  connect(): void {
    this.refCount += 1;
    if (this.refCount === 1) {
      this.stopped = false;
      // Cancel any pending reconnect before opening.
      if (this.reconnectTimer !== null) {
        window.clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
      this.open();
    }
  }

  /** Unregisters a consumer. The socket closes when the last one leaves. */
  close(): void {
    this.refCount = Math.max(0, this.refCount - 1);
    if (this.refCount === 0) {
      this.stopped = true;
      if (this.reconnectTimer !== null) {
        window.clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
      this.ws?.close();
      this.ws = null;
    }
  }

  /** Subscribes to an event type. Returns an unsubscribe function. */
  on(type: WsEventType, handler: (event: WsEvent) => void): () => void {
    if (!this.handlers.has(type)) this.handlers.set(type, new Set());
    this.handlers.get(type)!.add(handler);
    return () => {
      this.handlers.get(type)?.delete(handler);
    };
  }

  /** Observes connection status. Returns an unsubscribe function. */
  onStatus(handler: (connected: boolean) => void): () => void {
    this.statusHandlers.add(handler);
    return () => {
      this.statusHandlers.delete(handler);
    };
  }

  private open(): void {
    if (this.stopped) return;

    // A new connection replaces any existing one instead of joining it.
    this.ws?.close();

    let socket: WebSocket;
    try {
      socket = new WebSocket(buildWsUrl(this.lastSeq));
    } catch {
      this.scheduleReconnect();
      return;
    }
    this.ws = socket;

    socket.onopen = () => {
      this.reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS;
      this.emitStatus(true);
    };

    socket.onmessage = (message) => {
      this.handleMessage(message.data);
    };

    socket.onclose = () => {
      // Only the current socket may flip status and schedule a reconnect.
      if (this.ws !== socket) return;
      this.ws = null;
      this.emitStatus(false);
      if (!this.stopped) this.scheduleReconnect();
    };

    socket.onerror = () => {
      // onclose follows; closing here forces a clean teardown path.
      socket.close();
    };
  }

  private handleMessage(raw: string): void {
    let event: WsEvent;
    try {
      event = JSON.parse(raw) as WsEvent;
    } catch {
      console.error("Pudim: ignored malformed WebSocket message", raw);
      return;
    }

    if (typeof event.seq === "number" && event.seq > this.lastSeq) {
      this.lastSeq = event.seq;
      sessionStorage.setItem(LAST_SEQ_KEY, String(this.lastSeq));
    }

    const set = this.handlers.get(event.type);
    if (set) {
      set.forEach((handler) => handler(event));
    }
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.reconnectTimer !== null) return;
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, this.reconnectDelayMs);
    this.reconnectDelayMs = Math.min(
      this.reconnectDelayMs * 2,
      MAX_RECONNECT_DELAY_MS
    );
  }

  private emitStatus(connected: boolean): void {
    this.statusHandlers.forEach((handler) => handler(connected));
  }
}

/** Shared instance used by all pages. */
export const syncClient = new SyncClient();
