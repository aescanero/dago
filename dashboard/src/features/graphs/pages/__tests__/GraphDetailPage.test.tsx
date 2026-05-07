import { describe, test, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { AuthContext, type AuthContextValue } from "@/auth/AuthContext";
import { createApiClient } from "@/api/client";
import { GraphDetailPage } from "../GraphDetailPage";
import type { GraphResponse } from "@/api/types.gen";

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

const draftGraph: GraphResponse = {
  id: "graph-123",
  name: "Test Graph",
  version: "1.0.0",
  description: "A test graph",
  entry_node: "llm",
  definition: {
    nodes: { llm: { pattern: "llm_call", config: { model: "claude-sonnet-4-6" } } },
    edges: [],
  },
  memory_config: { semantic_search: false, episode_context: 0 },
  status: "draft",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const archivedGraph: GraphResponse = { ...draftGraph, id: "graph-archived", status: "archived" };

function createWrapper(initialPath = "/graphs/graph-123") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialPath]}>
          <AuthContext.Provider value={mockAuthContext}>
            <Routes>
              <Route path="/graphs/:id" element={children} />
              <Route path="/graphs" element={<div>Graphs List</div>} />
              <Route path="/graphs/:id/edit" element={<div>Edit Page</div>} />
            </Routes>
          </AuthContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>
    );
  };
}

describe("GraphDetailPage", () => {
  test("renders graph metadata", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(draftGraph)
      )
    );

    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Test Graph")).toBeInTheDocument();
    });
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("draft")).toBeInTheDocument();
  });

  test("shows Metadatos tab by default", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(draftGraph)
      )
    );

    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Test Graph")).toBeInTheDocument();
    });

    expect(screen.getByRole("tab", { name: /metadata/i })).toHaveAttribute(
      "data-state",
      "active"
    );
  });

  test("shows definition viewer on Definición tab", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(draftGraph)
      )
    );

    const user = userEvent.setup();
    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Test Graph")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("tab", { name: /definition/i }));

    await waitFor(() => {
      expect(screen.getByText(/llm_call/i)).toBeInTheDocument();
    });
  });

  test("shows Edit button for draft graph", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(draftGraph)
      )
    );

    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByRole("link", { name: /edit/i })).toBeInTheDocument();
    });
  });

  test("hides Edit button for archived graph", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(archivedGraph)
      )
    );

    render(<GraphDetailPage />, {
      wrapper: createWrapper("/graphs/graph-archived"),
    });

    await waitFor(() => {
      expect(screen.getByText("Test Graph")).toBeInTheDocument();
    });

    expect(screen.queryByRole("link", { name: /edit/i })).not.toBeInTheDocument();
  });

  test("shows archive confirmation dialog", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(draftGraph)
      )
    );

    const user = userEvent.setup();
    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /archive/i })).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /archive/i }));

    await waitFor(() => {
      expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    });
  });

  test("shows skeleton while loading", () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", async () => {
        await new Promise(() => {});
      })
    );

    render(<GraphDetailPage />, { wrapper: createWrapper() });

    const skeletons = document.querySelectorAll(".animate-pulse");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  test("shows not found state on 404", async () => {
    server.use(
      http.get("http://localhost:8080/api/v1/graphs/:id", () =>
        HttpResponse.json(
          { code: "GRAPH_NOT_FOUND", message: "Graph not found" },
          { status: 404 }
        )
      )
    );

    render(<GraphDetailPage />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText(/not found/i)).toBeInTheDocument();
    });
  });
});
