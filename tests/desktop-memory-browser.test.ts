import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  collectMemoryEntriesForWorkspace,
  readMemoryEntryDetail,
} from "../src/desktop/memory-browser.js";
import { MemoryStore } from "../src/memory/user.js";

describe("desktop memory browser", () => {
  let root: string;
  let mimoReasonixHome: string;

  beforeEach(() => {
    root = mkdtempSync(join(tmpdir(), "reasonix-memory-project-"));
    mimoReasonixHome = join(mkdtempSync(join(tmpdir(), "reasonix-memory-home-")), ".mimo-reasonix");
    mkdirSync(mimoReasonixHome, { recursive: true });
  });

  afterEach(() => {
    rmSync(root, { recursive: true, force: true });
    rmSync(mimoReasonixHome, { recursive: true, force: true });
  });

  it("lists project MIMO_REASONIX.md, global MIMO_REASONIX.md, and structured memory entries", () => {
    writeFileSync(join(root, "MIMO_REASONIX.md"), "project note", "utf8");
    writeFileSync(join(mimoReasonixHome, "MIMO_REASONIX.md"), "global note", "utf8");
    const store = new MemoryStore({ homeDir: mimoReasonixHome, projectRoot: root });
    store.write({
      name: "cli_pref",
      scope: "global",
      type: "user",
      description: "Use concise CLI output",
      body: "Keep command output short.",
    });
    store.write({
      name: "build_cmd",
      scope: "project",
      type: "project",
      description: "Use npm run verify",
      body: "Run npm run verify before release.",
    });

    const entries = collectMemoryEntriesForWorkspace(root, { mimoReasonixHome });

    expect(entries.map((e) => `${e.kind}:${e.scope}:${e.name}`)).toEqual([
      "project_file:project:MIMO_REASONIX.md",
      "global_file:global:MIMO_REASONIX.md",
      "structured:global:cli_pref",
      "structured:project:build_cmd",
    ]);
    expect(entries.every((e) => existsSync(e.path))).toBe(true);
    expect(entries.find((e) => e.name === "cli_pref")!.type).toBe("user");
  });

  it("reads details only for listed memory files", () => {
    writeFileSync(join(root, "MIMO_REASONIX.md"), "project note", "utf8");
    const entries = collectMemoryEntriesForWorkspace(root, { mimoReasonixHome });

    const detail = readMemoryEntryDetail({ path: entries[0]!.path }, root, { mimoReasonixHome });

    expect(detail).toMatchObject({
      kind: "project_file",
      scope: "project",
      name: "MIMO_REASONIX.md",
      body: "project note",
    });
    expect(() =>
      readMemoryEntryDetail({ path: join(mimoReasonixHome, "not-listed.md") }, root, {
        mimoReasonixHome,
      }),
    ).toThrow(/not available/);
  });
});
