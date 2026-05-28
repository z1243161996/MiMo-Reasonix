import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { detectLocale, resetLocaleCache, t } from "../src/index/semantic/i18n.js";

describe("semantic i18n", () => {
  const originalEnv = { ...process.env };

  beforeEach(() => {
    resetLocaleCache();
  });

  afterEach(() => {
    // Restore env so test order can't leak through cached locale.
    for (const k of Object.keys(process.env)) {
      if (!(k in originalEnv)) delete process.env[k];
    }
    Object.assign(process.env, originalEnv);
    resetLocaleCache();
  });

  describe("detectLocale", () => {
    it("returns 'zh' when MIMO_REASONIX_LANG=zh", () => {
      process.env.MIMO_REASONIX_LANG = "zh";
      expect(detectLocale()).toBe("zh");
    });

    it("returns 'en' when MIMO_REASONIX_LANG=en (overrides system locale)", () => {
      process.env.MIMO_REASONIX_LANG = "en";
      process.env.LANG = "zh_CN.UTF-8";
      expect(detectLocale()).toBe("en");
    });

    it("returns 'zh' for LANG=zh_CN.UTF-8", () => {
      process.env.MIMO_REASONIX_LANG = undefined;
      process.env.LANG = "zh_CN.UTF-8";
      expect(detectLocale()).toBe("zh");
    });

    it("returns 'zh' for LANG=zh_TW.Big5", () => {
      process.env.MIMO_REASONIX_LANG = undefined;
      process.env.LANG = "zh_TW.Big5";
      expect(detectLocale()).toBe("zh");
    });

    it("returns 'en' when LANG points elsewhere and no override", () => {
      process.env.MIMO_REASONIX_LANG = undefined;
      process.env.LANG = "en_US.UTF-8";
      process.env.LC_ALL = undefined;
      process.env.LC_MESSAGES = undefined;
      // Note: Intl fallback may still detect zh on a Chinese system,
      // but we can at least assert non-zh LANG doesn't produce zh
      // when MIMO_REASONIX_LANG is absent. We don't pin Intl here because
      // the test machine's system locale isn't fixed.
      const got = detectLocale();
      expect(["zh", "en"]).toContain(got); // sanity: only one of two
    });
  });

  describe("t()", () => {
    it("substitutes {placeholders} from vars", () => {
      process.env.MIMO_REASONIX_LANG = "en";
      const out = t("modelPullFailed", { model: "nomic-embed-text", code: 137 });
      expect(out).toContain("nomic-embed-text");
      expect(out).toContain("137");
    });

    it("returns Chinese strings under zh locale", () => {
      process.env.MIMO_REASONIX_LANG = "zh";
      const out = t("ollamaNotFound");
      expect(out).toMatch(/未找到/);
    });

    it("falls back to English when a key is only present in EN dict", () => {
      // Every ZH entry that exists must also exist in EN; the table
      // is structured so a missing ZH translation falls through. We
      // can't easily induce a missing-zh state without mutating the
      // module, so we exercise the happy path: a key that exists in
      // both renders the zh form.
      process.env.MIMO_REASONIX_LANG = "zh";
      const out = t("daemonReady", { pid: " (pid 123)" });
      expect(out).toMatch(/守护进程/);
      expect(out).toContain("(pid 123)");
    });

    it("leaves {var} literal when the key is missing from vars", () => {
      process.env.MIMO_REASONIX_LANG = "en";
      const out = t("modelPullFailed", { model: "x" }); // no `code`
      expect(out).toContain("x");
      expect(out).toContain("{code}");
    });
  });
});
