import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const storage = new Map<string, string>();
const storageStub = {
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

// config.ts reads window at module load, so the globals must exist first.
function stubBrowser(apiBaseUrl?: string) {
  vi.stubGlobal("window", {
    location: { origin: "http://localhost:3000" },
    desktop: apiBaseUrl ? { getApiBaseUrl: () => apiBaseUrl } : undefined,
  });
  vi.stubGlobal("localStorage", storageStub);
}

const uploadResponse = { url: "/api/v1/media/abc/pancakes.jpg", key: "abc/pancakes.jpg" };

function fetchReturning(body?: unknown) {
  return vi.fn<(input: RequestInfo | URL, init?: RequestInit) => Promise<unknown>>(async () => ({
    ok: true,
    status: 200,
    json: async () => body,
  }));
}

function teardown() {
  storage.clear();
  vi.unstubAllGlobals();
  stubBrowser();
  vi.resetModules();
}

stubBrowser();
const recipes = await import("./recipes");

describe("recipe media URLs", () => {
  beforeEach(() => {
    storage.clear();
  });

  it("returns an absolute upload URL while the API base stays relative", async () => {
    const fetchMock = fetchReturning(uploadResponse);
    vi.stubGlobal("fetch", fetchMock);

    const upload = await recipes.generateRecipeUploadURL("r1", { content_type: "image/jpeg" });

    expect(upload.url).toBe("http://localhost:3000/api/v1/media/abc/pancakes.jpg");
    expect(upload.key).toBe("abc/pancakes.jpg");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/recipes/r1/upload-url",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("PUTs a relative presigned URL to the API origin", async () => {
    const fetchMock = fetchReturning();
    vi.stubGlobal("fetch", fetchMock);

    await recipes.uploadToPresignedUrl(
      "/api/v1/media/abc/pancakes.jpg",
      new File(["x"], "pancakes.jpg", { type: "image/jpeg" }),
    );

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("http://localhost:3000/api/v1/media/abc/pancakes.jpg");
    expect(init?.method).toBe("PUT");
  });

  it("prefixes stored keys with the media base", () => {
    expect(recipes.resolveMediaUrl("abc/pancakes.jpg")).toBe(
      "http://localhost:3000/api/v1/media/abc/pancakes.jpg",
    );
  });

  it("passes through absolute and data URLs", () => {
    expect(recipes.resolveMediaUrl("https://cdn.example.com/a.jpg")).toBe(
      "https://cdn.example.com/a.jpg",
    );
    expect(recipes.resolveMediaUrl("data:image/png;base64,AAAA")).toBe(
      "data:image/png;base64,AAAA",
    );
  });

  it("returns null for empty values", () => {
    expect(recipes.resolveMediaUrl(null)).toBeNull();
    expect(recipes.resolveMediaUrl("")).toBeNull();
  });
});

describe("desktop absolute API base", () => {
  beforeEach(() => {
    vi.resetModules();
    storage.clear();
    stubBrowser("http://192.168.3.99:8086/api/v1");
  });

  afterEach(teardown);

  it("resolves stored keys against the backend origin", async () => {
    const desktopRecipes = await import("./recipes");

    expect(desktopRecipes.resolveMediaUrl("abc/pancakes.jpg")).toBe(
      "http://192.168.3.99:8086/api/v1/media/abc/pancakes.jpg",
    );
  });

  it("PUTs the presigned media URL to the backend origin", async () => {
    const desktopRecipes = await import("./recipes");
    const fetchMock = fetchReturning();
    vi.stubGlobal("fetch", fetchMock);

    await desktopRecipes.uploadToPresignedUrl(
      "/api/v1/media/abc/pancakes.jpg",
      new File(["x"], "pancakes.jpg", { type: "image/jpeg" }),
    );

    expect(fetchMock.mock.calls[0][0]).toBe(
      "http://192.168.3.99:8086/api/v1/media/abc/pancakes.jpg",
    );
  });
});
