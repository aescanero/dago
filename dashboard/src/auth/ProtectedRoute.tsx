import { type ReactNode } from "react";
import { useAuth } from "./useAuth";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, startPKCE } = useAuth();
  if (isLoading) return <div>Loading...</div>;
  if (!isAuthenticated) {
    void startPKCE();
    return null;
  }
  return <>{children}</>;
}
