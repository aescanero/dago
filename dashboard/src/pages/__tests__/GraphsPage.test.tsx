import { describe, test, expect, vi } from "vitest";
import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, type AuthContextValue } from "../../auth/AuthContext";
import { createApiClient } from "../../api/client";
import { GraphsPage } from "../GraphsPage";

// Mock at the module boundary — component tests verify rendering, not HTTP integration
vi.mock("../../hooks/useGraphs");

import { useGraphs } from "../../hooks/useGraphs";
const mockUseGraphs = vi.mocked(useGraphs);

type UseQueryResult = ReturnType<typeof useGraphs>;

const mockApiClient = createApiClient(() => "mock-token");
const mockAuthContext: AuthContextValue = {
  token: "mock-token",
  user: { sub: "1", email: "test@test.com", tags: [] },
  isAuthenticated: true,
  isLoading: false,
  setToken: vi.fn(),
  logout: vi.fn(),
  startPKCE: vi.fn(),
  handleCallback: vi.fn(),
  apiClient: mockApiClient,
};

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <AuthContext.Provider value={mockAuthContext}>
          {children}
        </AuthContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const sampleGraphs = [
  { id: "1", name: "My Graph", version: "1.0.0", status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
  { id: "2", name: "Another Graph", version: "2.0.0", status: "inactive", created_at: "2026-01-02T00:00:00Z", updated_at: "2026-01-02T00:00:00Z" },
];
const samplePagination = { page: 1, per_page: 20, total: 2, total_pages: 1 };

describe("GraphsPage", () => {
  test("shows skeletons while loading", () => {
    mockUseGraphs.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as unknown as UseQueryResult);

    render(<GraphsPage />, { wrapper: Wrapper });
    const skeletons = document.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("renders graph list from API", () => {
    mockUseGraphs.mockReturnValue({
      data: { items: sampleGraphs, pagination: samplePagination },
      isLoading: false,
      error: null,
    } as unknown as UseQueryResult);

    render(<GraphsPage />, { wrapper: Wrapper });
    expect(screen.getByText("My Graph")).toBeInTheDocument();
    expect(screen.getByText("Another Graph")).toBeInTheDocument();
  });

  test("shows empty state when no graphs", () => {
    mockUseGraphs.mockReturnValue({
      data: { items: [], pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 } },
      isLoading: false,
      error: null,
    } as unknown as UseQueryResult);

    render(<GraphsPage />, { wrapper: Wrapper });
    expect(screen.getByText(/no graphs/i)).toBeInTheDocument();
  });

  test("shows error state on API failure", () => {
    mockUseGraphs.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("server error"),
    } as unknown as UseQueryResult);

    render(<GraphsPage />, { wrapper: Wrapper });
    expect(screen.getByText(/error/i)).toBeInTheDocument();
  });

  test("pagination changes current page", async () => {
    const page1Data = {
      items: [{ id: "1", name: "Graph Page 1", version: "1.0.0", status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }],
      pagination: { page: 1, per_page: 20, total: 2, total_pages: 2 },
    };
    const page2Data = {
      items: [{ id: "3", name: "Graph Page 2", version: "1.0.0", status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }],
      pagination: { page: 2, per_page: 20, total: 2, total_pages: 2 },
    };

    mockUseGraphs.mockReturnValueOnce({
      data: page1Data, isLoading: false, error: null,
    } as unknown as UseQueryResult).mockReturnValue({
      data: page2Data, isLoading: false, error: null,
    } as unknown as UseQueryResult);

    render(<GraphsPage />, { wrapper: Wrapper });
    expect(screen.getByText("Graph Page 1")).toBeInTheDocument();

    await act(async () => {
      await userEvent.click(screen.getByRole("button", { name: /next/i }));
    });

    await waitFor(() => {
      expect(screen.getByText("Graph Page 2")).toBeInTheDocument();
    });
  });
});
