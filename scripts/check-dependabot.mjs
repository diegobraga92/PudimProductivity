#!/usr/bin/env node
/**
 * Guards .github/dependabot.yml against options that are valid for one package
 * ecosystem but rejected for another.
 *
 * GitHub validates that file server-side once it lands on the default branch and
 * refuses the *entire* file on a mismatch, so no update entry runs at all — both
 * version and security updates stop silently until someone reads the error:
 *
 *   The property '#/updates/4/cooldown/semver-major-days' is not supported
 *   for the package ecosystem 'github-actions'.
 *
 * The check is deliberately narrow and table-driven: it only asserts the
 * ecosystem/option pairs GitHub is known to reject, so it cannot invent false
 * alarms for combinations that were never exercised.
 *
 * Usage:
 *   node scripts/check-dependabot.mjs [path]   # defaults to .github/dependabot.yml
 */
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const target = resolve(process.argv[2] ?? join(root, ".github", "dependabot.yml"));

/**
 * Options GitHub refuses for a given ecosystem, keyed by ecosystem then by the
 * block they live in.
 *
 * `github-actions` versions actions by tag ref (actions/checkout@v7), not
 * semver, so the version-type based cooldown keys are meaningless there and the
 * whole config is rejected. Only `default-days` (plus `include`/`exclude`) is
 * accepted for it. Drop an entry here if GitHub starts supporting the key.
 */
const REJECTED_OPTIONS = {
  "github-actions": {
    cooldown: ["semver-major-days", "semver-minor-days", "semver-patch-days"],
  },
};

const BLOCKS = ["cooldown"];

let text;
try {
  text = readFileSync(target, "utf8");
} catch (error) {
  if (error.code === "ENOENT") {
    console.log(`[dependabot] ${target} not found — nothing to check`);
    process.exit(0);
  }
  throw error;
}

const problems = [];
let ecosystem = null; // ecosystem of the update entry being scanned
let block = null; // block key currently being scanned, null when at entry level
let blockIndent = null; // indentation of that block's key

text.split(/\r?\n/).forEach((line, index) => {
  const content = line.trim();
  if (content === "" || content.startsWith("#")) return;

  const lineNumber = index + 1;
  const indent = line.length - line.trimStart().length;

  // `- package-ecosystem: npm` — starts a new update entry.
  const ecosystemMatch = /^(?:-\s*)?package-ecosystem:\s*(\S.*)$/.exec(content);
  if (ecosystemMatch !== null) {
    ecosystem = ecosystemMatch[1].replace(/^["']|["']$/g, "").trim();
    block = null;
    return;
  }

  // Enter one of the tracked blocks (`cooldown:`), remembering its indentation.
  const entered = BLOCKS.find((name) => content === `${name}:`);
  if (entered !== undefined) {
    block = entered;
    blockIndent = indent;
    return;
  }

  if (block === null) return;

  // A sibling key (or the next list item) at or above that indentation ends it.
  if (indent <= blockIndent) {
    block = null;
    return;
  }

  const optionMatch = /^([A-Za-z0-9_-]+):/.exec(content);
  if (optionMatch === null) return;

  const rejected = REJECTED_OPTIONS[ecosystem]?.[block] ?? [];
  if (rejected.includes(optionMatch[1])) {
    problems.push(
      `${lineNumber}: ${block}.${optionMatch[1]} is not supported for ` +
        `package-ecosystem '${ecosystem}'`,
    );
  }
});

if (problems.length > 0) {
  console.error(`[dependabot] ${target} would be rejected by GitHub:`);
  for (const problem of problems) console.error(`  - ${problem}`);
  console.error(
    "[dependabot] a rejected config disables every entry (security updates included); " +
      "fix the option or update REJECTED_OPTIONS in scripts/check-dependabot.mjs",
  );
  process.exit(1);
}

console.log(`[dependabot] ecosystem/option combinations OK (${target})`);
