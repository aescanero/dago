export async function generateVerifier(): Promise<string> {
  const bytes = crypto.getRandomValues(new Uint8Array(32));
  return base64urlEncode(bytes);
}

export async function generateChallenge(verifier: string): Promise<string> {
  const bytes = new TextEncoder().encode(verifier);
  const hash = await crypto.subtle.digest("SHA-256", bytes);
  return base64urlEncode(new Uint8Array(hash));
}

function base64urlEncode(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}
