import { describe, test, expect } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "../../test/server";
import { AuthProvider } from "../AuthProvider";
import { useAuth } from "../useAuth";
import { ProtectedRoute } from "../ProtectedRoute";

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={makeQueryClient()}>
      <MemoryRouter>
        <AuthProvider>{children}</AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function AuthStatus() {
  const { isAuthenticated, token } = useAuth();
  return (
    <div>
      <span data-testid="auth">{isAuthenticated ? "authenticated" : "unauthenticated"}</span>
      <span data-testid="token">{token ?? "null"}</span>
    </div>
  );
}

describe("AuthProvider", () => {
  test("initial state is unauthenticated", () => {
    render(<AuthStatus />, { wrapper: Wrapper });
    expect(screen.getByTestId("auth")).toHaveTextContent("unauthenticated");
    expect(screen.getByTestId("token")).toHaveTextContent("null");
  });

  test("setToken makes isAuthenticated true", async () => {
    function SetTokenButton() {
      const { setToken } = useAuth();
      return (
        <button onClick={() => setToken("test-jwt")}>set token</button>
      );
    }

    render(
      <Wrapper>
        <SetTokenButton />
        <AuthStatus />
      </Wrapper>
    );

    const btn = screen.getByRole("button", { name: "set token" });
    await act(async () => btn.click());
    expect(screen.getByTestId("auth")).toHaveTextContent("authenticated");
  });

  test("logout clears token and user", async () => {
    function LogoutButton() {
      const { setToken, logout } = useAuth();
      return (
        <>
          <button onClick={() => setToken("test-jwt")}>login</button>
          <button onClick={() => logout()}>logout</button>
        </>
      );
    }

    render(
      <Wrapper>
        <LogoutButton />
        <AuthStatus />
      </Wrapper>
    );

    await act(async () => screen.getByRole("button", { name: "login" }).click());
    expect(screen.getByTestId("auth")).toHaveTextContent("authenticated");
    await act(async () => screen.getByRole("button", { name: "logout" }).click());
    expect(screen.getByTestId("auth")).toHaveTextContent("unauthenticated");
    expect(screen.getByTestId("token")).toHaveTextContent("null");
  });

  test("ProtectedRoute redirects when no token", async () => {
    render(
      <QueryClientProvider client={makeQueryClient()}>
        <MemoryRouter initialEntries={["/protected"]}>
          <AuthProvider>
            <Routes>
              <Route
                path="/protected"
                element={
                  <ProtectedRoute>
                    <div>Secret</div>
                  </ProtectedRoute>
                }
              />
              <Route path="/login" element={<div>Login page</div>} />
            </Routes>
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.queryByText("Secret")).not.toBeInTheDocument();
    });
  });

  test("AuthCallback exchanges code for token via POST /token", async () => {
    server.use(
      http.post("http://localhost:8081/token", () => {
        return HttpResponse.json({
          access_token: "callback-token",
          token_type: "Bearer",
          expires_in: 3600,
        });
      })
    );

    sessionStorage.setItem("pkce_verifier", "test-verifier");
    sessionStorage.setItem("pkce_state", "test-state");

    const { AuthCallback } = await import("../AuthCallback");

    render(
      <QueryClientProvider client={makeQueryClient()}>
        <MemoryRouter initialEntries={["/auth/callback?code=test-code&state=test-state"]}>
          <AuthProvider>
            <Routes>
              <Route path="/auth/callback" element={<AuthCallback />} />
              <Route path="/graphs" element={<div>Graphs page</div>} />
            </Routes>
          </AuthProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText("Graphs page")).toBeInTheDocument();
    }, { timeout: 3000 });
  });
});
