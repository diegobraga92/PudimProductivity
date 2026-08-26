import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  close = vi.fn(() => {});

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
}

const storage = new Map<string, string>();
const sessionStorageStub = {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => {
    storage.set(key, String(value));
  },
  removeItem: (key: string) => {
    storage.delete(key);
  },
  clear: () => {
    storage.clear();
  },
};

// `config.ts` reads `window` at module load, so the globals must exist before
// the module is imported (hence the dynamic import below).
vi.stubGlobal(
  "window",
  {
    location: { origin: "http://localhost:3000" },
    setTimeout: (fn: () => void, ms: number) => setTimeout(fn, ms),
    clearTimeout: (id: ReturnType<typeof setTimeout>) => clearTimeout(id),
  },
);

vi.stubGlobal("sessionStorage", sessionStorageStub);
vi.stubGlobal("WebSocket", MockWebSocket);

const { SyncClient } = await import("./sync");

describe("SyncClient connection lifecycle", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    storage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("opens a single socket for multiple consumers", () => {
    const client = new SyncClient();
    client.connect();
    client.connect();
    expect(MockWebSocket.instances).toHaveLength(1);

    client.close();
    client.close();
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it("ignores a superseded socket closing (StrictMode race) and delivers each event once", () => {
    const client = new SyncClient();
    const statuses: boolean[] = [];
    const received: string[] = [];
    client.onStatus((connected) => statuses.push(connected));
    client.on("task.created", (event) => {
      received.push((event.payload as { id: string }).id);
    });

    // Simulate React StrictMode dev double-mount: connect → close → connect.
    client.connect(); // socket A
    const socketA = MockWebSocket.instances[0];
    client.close(); // starts an async close of A
    client.connect(); // socket B (replaces A)
    expect(MockWebSocket.instances).toHaveLength(2);

    // Socket A's onclose fires only after B is already live. It must NOT
    // flip status offline or schedule a reconnect (which would open socket C).
    socketA.onclose?.();
    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances).toHaveLength(2);
    expect(statuses).toEqual([]);

    // The single live socket delivers the event exactly once.
    const socketB = MockWebSocket.instances[1];
    socketB.onmessage?.({
      data: JSON.stringify({
        type: "task.created",
        seq: 1,
        timestamp: "2026-01-01T00:00:00Z",
        payload: { id: "t1", title: "Buy milk" },
      }),
    });
    expect(received).toEqual(["t1"]);
  });

  it("still reconnects when the current socket closes", () => {
    const client = new SyncClient();
    client.connect();
    const socketA = MockWebSocket.instances[0];

    socketA.onclose?.();
    expect(MockWebSocket.instances).toHaveLength(1);

    // Initial reconnect delay is 1s; a fresh socket must be opened.
    vi.advanceTimersByTime(1_000);
    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it("does not reconnect once closed for good", () => {
    const client = new SyncClient();
    client.connect();
    const socketA = MockWebSocket.instances[0];

    client.close(); // last consumer left → stopped
    socketA.onclose?.();
    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances).toHaveLength(1);
  });
});
