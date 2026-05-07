import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { components } from "@/api/types.gen";

type GraphResponse = components["schemas"]["GraphResponse"];
type Pagination = components["schemas"]["Pagination"];

interface UseGraphsOpts {
  page: number;
  perPage: number;
  status?: string;
}

interface GraphListResult {
  items: GraphResponse[];
  pagination: Pagination;
}

export function useGraphs(opts: UseGraphsOpts) {
  const { apiClient } = useAuth();
  return useQuery<GraphListResult, Error>({
    queryKey: ["graphs", opts],
    queryFn: async () => {
      const res = await apiClient.GET("/api/v1/graphs" as never, {
        params: {
          query: {
            page: opts.page,
            per_page: opts.perPage,
            status: opts.status,
          },
        },
      } as never);
      const { data, error } = res as { data: GraphListResult | undefined; error: Error | undefined };
      if (error) throw error;
      return data!;
    },
  });
}
