import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphResponse, Pagination } from "@/api/types.gen";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

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
  const { token } = useAuth();
  const { page = 1, perPage = 20, status } = opts;

  return useQuery<GraphListResult, Error>({
    queryKey: ["graphs", { page, perPage, status }],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: String(page),
        per_page: String(perPage),
        ...(status ? { status } : {}),
      });
      const response = await fetch(
        `${API_URL}/api/v1/graphs?${params.toString()}`,
        { headers: token ? { Authorization: `Bearer ${token}` } : {} }
      );
      if (!response.ok) throw new Error("Failed to load graphs");
      return response.json() as Promise<GraphListResult>;
    },
  });
}
