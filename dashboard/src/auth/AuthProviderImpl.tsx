import { useState, useRef, type ReactNode } from "react";
import { AuthContext, type AuthContextValue, type AuthState, type UserInfo } from "./AuthContext";
import { createApiClient } from "@/api/client";
import { generateVerifier, generateChallenge } from "./pkce";

const AUTH_URL = import.meta.env.VITE_AUTH_URL ?? "http://localhost:8081";
const CLIENT_ID = import.meta.env.VITE_AUTH_CLIENT_ID ?? "dago-dashboard";
const REDIRECT_URI = import.meta.env.VITE_AUTH_REDIRECT_URI ?? "http://localhost:5173/auth/callback";

function decodeJwtPayload(token: string): UserInfo | null {
  try {
    const payload = token.split(".")[1];
    const decoded = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
    return { sub: decoded.sub, email: decoded.email, tags: decoded.tags ?? [] };
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const tokenRef = useRef<string | null>(null);
  const [state, setState] = useState<AuthState>({
    token: null,
    user: null,
    isAuthenticated: false,
    isLoading: false,
  });

  const setToken = (token: string) => {
    tokenRef.current = token;
    setState({
      token,
      user: decodeJwtPayload(token),
      isAuthenticated: true,
      isLoading: false,
    });
  };

  const logout = () => {
    tokenRef.current = null;
    setState({ token: null, user: null, isAuthenticated: false, isLoading: false });
  };

  const startPKCE = async () => {
    const verifier = await generateVerifier();
    const challenge = await generateChallenge(verifier);
    const state = crypto.randomUUID();
    sessionStorage.setItem("pkce_verifier", verifier);
    sessionStorage.setItem("pkce_state", state);
    const url = new URL(`${AUTH_URL}/authorize`);
    url.searchParams.set("response_type", "code");
    url.searchParams.set("client_id", CLIENT_ID);
    url.searchParams.set("redirect_uri", REDIRECT_URI);
    url.searchParams.set("code_challenge", challenge);
    url.searchParams.set("code_challenge_method", "S256");
    url.searchParams.set("state", state);
    window.location.href = url.toString();
  };

  const handleCallback = async (code: string) => {
    const verifier = sessionStorage.getItem("pkce_verifier");
    if (!verifier) throw new Error("missing pkce_verifier in sessionStorage");
    const res = await fetch(`${AUTH_URL}/token`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        grant_type: "authorization_code",
        code,
        code_verifier: verifier,
        redirect_uri: REDIRECT_URI,
        client_id: CLIENT_ID,
      }),
    });
    if (!res.ok) throw new Error("token exchange failed");
    const data = (await res.json()) as { access_token: string };
    sessionStorage.removeItem("pkce_verifier");
    sessionStorage.removeItem("pkce_state");
    setToken(data.access_token);
  };

  const apiClient = createApiClient(() => tokenRef.current);

  const value: AuthContextValue = {
    ...state,
    setToken,
    logout,
    startPKCE,
    handleCallback,
    apiClient,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
