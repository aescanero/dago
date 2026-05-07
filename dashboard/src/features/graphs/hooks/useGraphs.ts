import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphResponse, Pagination } from "@/api/types.gen";

interface UseGraphsOpts {
  page?: number;
  perPage?: number;
  status?: string;
}

interface GraphListResult {
  items: GraphResponse[];
  pagination: Pagination;
}

export function useGraphs(opts: UseGraphsOpts = {}) {
  const { apiClient } = useAuth();
  const { page = 1, perPage = 20, status } = opts;

  return useQuery<GraphListResult, Error>({
    queryKey: ["graphs", { page, perPage, status }],
    queryFn: async () => {
      const res = await apiClient.GET("/api/v1/graphs" as never, {
        params: {
          query: { page, per_page: perPage, status },
        },
      } as never);
      const { data, error } = res as {
        data: GraphListResult | undefined;
        error: Error | undefined;
      };
      if (error) throw error;
      return data!;
    },
  });
}
