/** escalationContract — model-aware contract so the system prompt names the actual tier (#582). */

import { describe, expect, it } from "vitest";
import { ESCALATION_CONTRACT, escalationContract } from "../src/prompt-fragments.js";

describe("escalationContract (#582)", () => {
  it("interpolates the actual model id for non-pro tiers", () => {
    const out = escalationContract("mimo-v2.5");
    expect(out).toContain("`mimo-v2.5`");
    expect(out).toContain("If asked which model you are, answer `mimo-v2.5`");
    expect(out).toContain("<<<NEEDS_PRO");
  });

  it("returns the no-escalation note for the pro tier instead of the full ladder", () => {
    const out = escalationContract("mimo-v2.5-pro");
    expect(out).toContain("`mimo-v2.5-pro`");
    expect(out).toContain("escalation tier");
    expect(out).toContain("If asked which model you are, answer `mimo-v2.5-pro`");
    expect(out).not.toContain("<<<NEEDS_PRO: <one-sentence reason>>>>");
  });

  it("never tells a pro session it is running on flash (regression for #582)", () => {
    const out = escalationContract("mimo-v2.5-pro");
    expect(out).not.toContain("running on `mimo-v2.5`:");
  });

  it("backward-compat const matches the default MiMo phrasing", () => {
    expect(ESCALATION_CONTRACT).toBe(escalationContract("mimo-v2.5"));
  });

  it("treats unknown future tiers as non-pro (full contract, name themselves)", () => {
    const out = escalationContract("deepseek-v5-experimental");
    expect(out).toContain("`deepseek-v5-experimental`");
    expect(out).toContain("<<<NEEDS_PRO");
  });
});
