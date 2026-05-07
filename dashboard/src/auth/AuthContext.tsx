import { createContext } from "react";
import type { ApiClient } from "@/api/client";

export interface UserInfo {
  sub: string;
  email: string;
  tags: string[];
}

export interface AuthState {
  token: string | null;
  user: UserInfo | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

export interface AuthContextValue extends AuthState {
  setToken: (token: string) => void;
  logout: () => void;
  startPKCE: () => Promise<void>;
  handleCallback: (code: string) => Promise<void>;
  apiClient: ApiClient;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
