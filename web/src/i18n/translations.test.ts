import { describe, expect, it } from "vitest";
import { dictionaries } from "./translations";

const languages = Object.keys(dictionaries) as (keyof typeof dictionaries)[];

// Source files are loaded as raw text through Vite so the guard needs no node
// filesystem access. Test files are skipped because they contain sample keys.
const sources = Object.entries(
  import.meta.glob("/src/**/*.{ts,tsx}", {
    query: "?raw",
    import: "default",
    eager: true,
  }) as Record<string, string>
).filter(([path]) => !/\.test\.tsx?$/.test(path));

/** Every literal key passed to `t()`, mapped to the file that uses it. */
function referencedKeys(): Map<string, string> {
  const keys = new Map<string, string>();
  const pattern = /\bt\(\s*"([^"\\]+)"/g;

  for (const [path, source] of sources) {
    let match: RegExpExecArray | null;
    while ((match = pattern.exec(source)) !== null) {
      if (!keys.has(match[1])) keys.set(match[1], path);
    }
  }
  return keys;
}

describe("i18n dictionaries", () => {
  it("keeps every language on the same key set", () => {
    const [first, ...rest] = languages;
    const expected = Object.keys(dictionaries[first]).sort();

    for (const language of rest) {
      expect(Object.keys(dictionaries[language]).sort(), `locale ${language}`).toEqual(expected);
    }
  });

  it("defines every key referenced by the UI", () => {
    // A key that is used but missing falls back to printing its own name in the
    // interface, so the guard is the only thing that catches a dropped entry.
    const keys = referencedKeys();

    // Guards against a glob that silently matches nothing.
    expect(keys.size).toBeGreaterThan(100);

    const missing: string[] = [];

    for (const [key, file] of keys) {
      for (const language of languages) {
        if (!(key in dictionaries[language])) {
          missing.push(`${key} (${language}, used in ${file})`);
        }
      }
    }

    expect(missing).toEqual([]);
  });

  it("has no blank translations", () => {
    const blank: string[] = [];

    for (const language of languages) {
      for (const [key, value] of Object.entries(dictionaries[language])) {
        if (value.trim() === "") blank.push(`${key} (${language})`);
      }
    }

    expect(blank).toEqual([]);
  });
});
