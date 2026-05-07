import { describe, test, expect } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { AuthContext, type AuthContextValue } from "@/auth/AuthContext";
import { createApiClient } from "@/api/client";
import { useArchiveGraph } from "../useArchiveGraph";
import React from "react";

const mockApiClient = createApiClient(() => "mock-token");
const mockAuthContext: AuthContextValue = {
  token: "mock-token",
  user: { sub: "1", email: "test@test.com", tags: [] },
  isAuthenticated: true,
  isLoading: false,
  setToken: () => {},
  logout: () => {},
  startPKCE: async () => {},
  handleCallback: async () => {},
  apiClient: mockApiClient,
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(
          AuthContext.Provider,
          { value: mockAuthContext },
          children
        )
      )
    );
  };
}

describe("useArchiveGraph", () => {
  test("invalidates graphs cache on success (204)", async () => {
    server.use(
      http.delete("http://localhost:8080/api/v1/graphs/:id", () => {
        return new HttpResponse(null, { status: 204 });
      })
    );

    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(
            AuthContext.Provider,
            { value: mockAuthContext },
            children
          )
        )
      );

    const { result } = renderHook(() => useArchiveGraph(), { wrapper });

    result.current.mutate("test-uuid");

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(invalidateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ["graphs"] })
    );
  });

  test("throws on 409 conflict", async () => {
    server.use(
      http.delete("http://localhost:8080/api/v1/graphs/:id", () => {
        return HttpResponse.json(
          { code: "GRAPH_HAS_ACTIVE_EXECUTIONS", message: "The graph has active executions" },
          { status: 409 }
        );
      })
    );

    const { result } = renderHook(() => useArchiveGraph(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("conflict-uuid");

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
  });
});
