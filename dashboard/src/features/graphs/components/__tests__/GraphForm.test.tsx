import { describe, test, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, type AuthContextValue } from "@/auth/AuthContext";
import { createApiClient } from "@/api/client";
import { GraphForm } from "../GraphForm";
import type { GraphFormValues } from "@/features/graphs/schemas/graph-form.schema";

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
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <MemoryRouter>
        <AuthContext.Provider value={mockAuthContext}>
          {children}
        </AuthContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const validDefinition = JSON.stringify({ nodes: {}, edges: [] });

describe("GraphForm", () => {
  test("renders all required fields", () => {
    render(
      <GraphForm
        onSubmit={vi.fn()}
        isPending={false}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/version/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/entry node/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create/i })
    ).toBeInTheDocument();
  });

  test("shows version format error for non-semver", async () => {
    const user = userEvent.setup();
    render(
      <GraphForm
        onSubmit={vi.fn()}
        isPending={false}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    await user.type(screen.getByLabelText(/version/i), "abc");
    await user.click(screen.getByRole("button", { name: /create/i }));

    await waitFor(() => {
      expect(screen.getByText("Formato: 1.0.0")).toBeInTheDocument();
    });
  });

  test("shows JSON error for invalid definition", async () => {
    const user = userEvent.setup();
    render(
      <GraphForm
        onSubmit={vi.fn()}
        isPending={false}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    const definitionTextarea = screen.getByRole("textbox", {
      name: /definition/i,
    });
    await user.clear(definitionTextarea);
    await user.type(definitionTextarea, "not valid json");
    await user.tab();

    await waitFor(() => {
      expect(screen.getByText("JSON inválido")).toBeInTheDocument();
    });
  });

  test("fills definition textarea on template select", async () => {
    const user = userEvent.setup();
    render(
      <GraphForm
        onSubmit={vi.fn()}
        isPending={false}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    await user.click(screen.getByRole("combobox", { name: /template/i }));
    await user.click(screen.getByText(/LLM Call simple/i));

    await waitFor(() => {
      const textarea = screen.getByRole("textbox", { name: /definition/i }) as HTMLTextAreaElement;
      expect(textarea.value).toContain("llm_call");
    });
  });

  test("calls onSubmit with transformed values on valid submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(
      <GraphForm
        onSubmit={onSubmit}
        isPending={false}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    await user.type(screen.getByLabelText(/^name/i), "mi-grafo");
    await user.type(screen.getByLabelText(/version/i), "1.0.0");
    await user.type(screen.getByLabelText(/entry node/i), "llm");

    const definitionTextarea = screen.getByRole("textbox", {
      name: /definition/i,
    });
    await user.clear(definitionTextarea);
    await user.type(definitionTextarea, validDefinition);

    await user.click(screen.getByRole("button", { name: /create/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "mi-grafo",
          version: "1.0.0",
          entry_node: "llm",
        }) as GraphFormValues
      );
    });
  });

  test("disables submit while pending", () => {
    render(
      <GraphForm
        onSubmit={vi.fn()}
        isPending={true}
        submitLabel="Create"
      />,
      { wrapper: Wrapper }
    );

    expect(screen.getByRole("button", { name: /create/i })).toBeDisabled();
  });
});
