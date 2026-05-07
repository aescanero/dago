import { describe, test, expect, vi } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";
import { AuthContext, type AuthContextValue } from "@/auth/AuthContext";
import { createApiClient } from "@/api/client";
import { GraphCreatePage } from "../GraphCreatePage";

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

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/graphs/new"]}>
          <AuthContext.Provider value={mockAuthContext}>
            <Toaster />
            <Routes>
              <Route path="/graphs/new" element={children} />
              <Route
                path="/graphs/:id"
                element={<div data-testid="detail-page">Detail</div>}
              />
              <Route path="/graphs" element={<div>Graphs List</div>} />
            </Routes>
          </AuthContext.Provider>
        </MemoryRouter>
      </QueryClientProvider>
    );
  };
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/^name/i), "mi-grafo");
  await user.type(screen.getByLabelText(/version/i), "1.0.0");
  await user.type(screen.getByLabelText(/entry node/i), "llm");

  const defTextarea = screen.getByRole("textbox", { name: /definition/i });
  // Use fireEvent.change for JSON to avoid userEvent interpreting { as key sequences
  fireEvent.change(defTextarea, { target: { value: '{"nodes":{},"edges":[]}' } });
}

describe("GraphCreatePage", () => {
  test("renders the create form", () => {
    render(<GraphCreatePage />, { wrapper: createWrapper() });

    expect(screen.getByLabelText(/^name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/version/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create/i })).toBeInTheDocument();
  });

  test("shows success toast and navigates on 201", async () => {
    server.use(
      http.post("http://localhost:8080/api/v1/graphs", () =>
        HttpResponse.json(
          {
            id: "new-graph-id",
            name: "mi-grafo",
            version: "1.0.0",
            status: "draft",
            entry_node: "llm",
            definition: { nodes: {}, edges: [] },
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
          },
          { status: 201 }
        )
      )
    );

    const user = userEvent.setup();
    render(<GraphCreatePage />, { wrapper: createWrapper() });

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create/i }));

    await waitFor(() => {
      expect(screen.getByTestId("detail-page")).toBeInTheDocument();
    });
  });

  test("shows conflict toast on 409", async () => {
    server.use(
      http.post("http://localhost:8080/api/v1/graphs", () =>
        HttpResponse.json(
          { code: "GRAPH_DUPLICATE_VERSION", message: "Version 1.0.0 of this graph already exists" },
          { status: 409 }
        )
      )
    );

    const user = userEvent.setup();
    render(<GraphCreatePage />, { wrapper: createWrapper() });

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create/i }));

    await waitFor(() => {
      expect(screen.getByText(/already exists/i)).toBeInTheDocument();
    });
  });

  test("shows inline errors on 422", async () => {
    server.use(
      http.post("http://localhost:8080/api/v1/graphs", () =>
        HttpResponse.json(
          {
            code: "VALIDATION_ERROR",
            message: "Validation failed",
            details: { name: ["Name is invalid"] },
          },
          { status: 422 }
        )
      )
    );

    const user = userEvent.setup();
    render(<GraphCreatePage />, { wrapper: createWrapper() });

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create/i }));

    await waitFor(() => {
      expect(screen.getByText(/name is invalid/i)).toBeInTheDocument();
    });
  });

  test("renders correct breadcrumbs", () => {
    render(<GraphCreatePage />, { wrapper: createWrapper() });

    expect(screen.getByRole("link", { name: /graphs/i })).toBeInTheDocument();
    expect(screen.getByText(/new graph/i)).toBeInTheDocument();
  });
});
