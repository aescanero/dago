import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphInput, GraphResponse } from "@/api/types.gen";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export function useUpdateGraph(id: string) {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<GraphResponse, Error, GraphInput>({
    mutationFn: async (input: GraphInput) => {
      const response = await fetch(`${API_URL}/api/v1/graphs/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(input),
      });

      const body = await response.json() as GraphResponse & { message?: string };

      if (!response.ok) {
        throw Object.assign(new Error(body.message ?? "Error updating graph"), {
          status: response.status,
          body,
        });
      }

      return body;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["graphs"] });
      queryClient.invalidateQueries({ queryKey: ["graphs", id] });
    },
  });
}
