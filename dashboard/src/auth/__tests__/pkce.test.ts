import { describe, test, expect } from "vitest";
import { generateVerifier, generateChallenge } from "../pkce";

describe("generateVerifier", () => {
  test("produces valid base64url string with 43+ characters", async () => {
    const verifier = await generateVerifier();
    expect(verifier.length).toBeGreaterThanOrEqual(43);
    expect(verifier.length).toBeLessThanOrEqual(128);
    expect(verifier).toMatch(/^[A-Za-z0-9\-._~]+$/);
  });

  test("produces unique values on each call", async () => {
    const a = await generateVerifier();
    const b = await generateVerifier();
    expect(a).not.toEqual(b);
  });
});

describe("generateChallenge", () => {
  test("is deterministic for a given verifier", async () => {
    const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
    const c1 = await generateChallenge(verifier);
    const c2 = await generateChallenge(verifier);
    expect(c1).toEqual(c2);
  });

  test("produces correct S256 challenge for known verifier", async () => {
    // RFC 7636 test vector
    const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk";
    const challenge = await generateChallenge(verifier);
    expect(challenge).toEqual("E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM");
  });

  test("uses base64url alphabet without padding", async () => {
    const verifier = await generateVerifier();
    const challenge = await generateChallenge(verifier);
    expect(challenge).not.toContain("=");
    expect(challenge).not.toContain("+");
    expect(challenge).not.toContain("/");
  });
});
