import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphResponse } from "@/api/types.gen";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export function useGraph(id: string) {
  const { token } = useAuth();

  return useQuery<GraphResponse, Error>({
    queryKey: ["graphs", id],
    queryFn: async () => {
      const response = await fetch(`${API_URL}/api/v1/graphs/${id}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });

      if (!response.ok) {
        const body = await response
          .json()
          .catch(() => ({ message: "Graph not found" })) as { message?: string };
        throw new Error(body.message ?? "Graph not found");
      }

      return response.json() as Promise<GraphResponse>;
    },
    enabled: !!id,
    retry: false,
  });
}
