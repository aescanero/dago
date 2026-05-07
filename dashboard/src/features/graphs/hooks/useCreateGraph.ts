import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphInput, GraphResponse } from "@/api/types.gen";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export function useCreateGraph() {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<GraphResponse, Error, GraphInput>({
    mutationFn: async (input: GraphInput) => {
      const response = await fetch(`${API_URL}/api/v1/graphs`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(input),
      });

      const body = await response.json() as GraphResponse & { message?: string };

      if (!response.ok) {
        throw Object.assign(new Error(body.message ?? "Error creating graph"), {
          status: response.status,
          body,
        });
      }

      return body;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["graphs"] });
    },
  });
}
