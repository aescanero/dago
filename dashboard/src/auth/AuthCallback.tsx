import { useEffect, useRef } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "./useAuth";

export function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { handleCallback } = useAuth();
  const calledRef = useRef(false);

  useEffect(() => {
    if (calledRef.current) return;
    calledRef.current = true;
    const code = searchParams.get("code");
    const state = searchParams.get("state");
    const storedState = sessionStorage.getItem("pkce_state");
    if (!code || state !== storedState) {
      void navigate("/");
      return;
    }
    handleCallback(code)
      .then(() => navigate("/graphs"))
      .catch(() => navigate("/"));
  }, [searchParams, navigate, handleCallback]);

  return <div>Completing sign in...</div>;
}
