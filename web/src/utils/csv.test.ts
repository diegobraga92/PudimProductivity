import { describe, it, expect } from "vitest";
import {
  parseCsv,
  parseDoneValue,
  normalizeMediaType,
  parseYearValue,
  parseScoreValue,
  applyAutoScoreResults,
} from "./csv";

describe("parseCsv", () => {
  it("parses simple comma-separated rows", () => {
    expect(parseCsv("The Matrix,movie,1999\nInterstellar,movie,2014")).toEqual([
      ["The Matrix", "movie", "1999"],
      ["Interstellar", "movie", "2014"],
    ]);
  });

  it("handles quoted fields with commas", () => {
    expect(parseCsv('"The Matrix",movie,1999\n"Blade Runner, The",movie,1982')).toEqual([
      ["The Matrix", "movie", "1999"],
      ["Blade Runner, The", "movie", "1982"],
    ]);
  });

  it("handles escaped quotes", () => {
    expect(parseCsv('"He said ""hi""",book')).toEqual([['He said "hi"', "book"]]);
  });

  it("handles CRLF line endings and a UTF-8 BOM", () => {
    expect(parseCsv("\uFEFFname,type\r\nDune,book\r\n")).toEqual([
      ["name", "type"],
      ["Dune", "book"],
    ]);
  });

  it("handles newlines inside quoted fields", () => {
    expect(parseCsv('"Line 1\nLine 2",series')).toEqual([["Line 1\nLine 2", "series"]]);
  });

  it("drops trailing empty rows", () => {
    expect(parseCsv("a,b\n\n")).toEqual([["a", "b"]]);
  });

  it("returns an empty array for empty input", () => {
    expect(parseCsv("")).toEqual([]);
  });
});

describe("normalizeMediaType", () => {
  it("maps case-insensitive values to the enum", () => {
    expect(normalizeMediaType("Movie")).toBe("movie");
    expect(normalizeMediaType("SERIES")).toBe("series");
    expect(normalizeMediaType(" book ")).toBe("book");
  });

  it("returns null for unknown types", () => {
    expect(normalizeMediaType("vinyl")).toBeNull();
    expect(normalizeMediaType("")).toBeNull();
  });
});

describe("parseDoneValue", () => {
  it("treats common truthy tokens as done", () => {
    for (const v of ["true", "yes", "y", "1", "x", "done", "read", "watched", "played", "✔"]) {
      expect(parseDoneValue(v)).toBe(true);
    }
  });

  it("treats everything else as not done", () => {
    for (const v of ["false", "no", "0", "", "maybe"]) {
      expect(parseDoneValue(v)).toBe(false);
    }
  });
});

describe("parseYearValue", () => {
  it("parses integers", () => {
    expect(parseYearValue("1999")).toBe(1999);
    expect(parseYearValue(" 2024 ")).toBe(2024);
  });

  it("returns null for empty, garbage or out-of-range values", () => {
    expect(parseYearValue("")).toBeNull();
    expect(parseYearValue("n/a")).toBeNull();
    expect(parseYearValue("999")).toBeNull();
    expect(parseYearValue("3000")).toBeNull();
  });
});

describe("parseScoreValue", () => {
  it("parses integers and decimals on the 0-100 scale", () => {
    expect(parseScoreValue("96")).toBe(96);
    expect(parseScoreValue("8.7")).toBe(8.7);
    expect(parseScoreValue(" 42 ")).toBe(42);
  });

  it("returns null for empty, garbage or out-of-range values", () => {
    expect(parseScoreValue("")).toBeNull();
    expect(parseScoreValue("n/a")).toBeNull();
    expect(parseScoreValue("-1")).toBeNull();
    expect(parseScoreValue("101")).toBeNull();
  });
});

describe("applyAutoScoreResults", () => {
  it("fills rows from the top candidate and counts matches", () => {
    const { fill, found, noRating } = applyAutoScoreResults(
      [
        { requestIndex: 0, row: 2 },
        { requestIndex: 1, row: 5 },
      ],
      [
        { index: 0, candidates: [{ score: 96, score_source: "metacritic" }] },
        { index: 1, candidates: [{ score: 84, score_source: "metacritic" }] },
      ],
    );
    expect(fill).toEqual({ 2: { score: 96, score_source: "metacritic" }, 5: { score: 84, score_source: "metacritic" } });
    expect(found).toBe(2);
    expect(noRating).toBe(0);
  });

  it("counts rows without matches separately from failed lookups", () => {
    const { fill, found, noRating } = applyAutoScoreResults(
      [{ requestIndex: 0, row: 1 }],
      [
        { index: 0, candidates: [] },
        { index: 1, error: "provider boom" },
      ],
    );
    expect(fill).toEqual({});
    expect(found).toBe(0);
    expect(noRating).toBe(1); // request 0 → no candidates; request 1 isn't a requested row
  });

  it("ignores unknown request indices", () => {
    const { fill, found, noRating } = applyAutoScoreResults(
      [{ requestIndex: 3, row: 9 }],
      [{ index: 0, candidates: [{ score: 90, score_source: "imdb" }] }],
    );
    expect(fill).toEqual({});
    expect(found).toBe(0);
    expect(noRating).toBe(0);
  });
});
