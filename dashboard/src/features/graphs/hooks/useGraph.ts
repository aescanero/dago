import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphResponse } from "@/api/types.gen";

export function useGraph(id: string) {
  const { apiClient } = useAuth();

  return useQuery<GraphResponse, Error>({
    queryKey: ["graphs", id],
    queryFn: async () => {
      const res = await apiClient.GET("/api/v1/graphs/{id}" as never, {
        params: { path: { id } },
      } as never);
      const { data, error } = res as {
        data: GraphResponse | undefined;
        error: Error | undefined;
      };
      if (error) throw error;
      return data!;
    },
    enabled: !!id,
  });
}
