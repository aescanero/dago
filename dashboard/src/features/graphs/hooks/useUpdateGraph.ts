import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphInput, GraphResponse } from "@/api/types.gen";

export function useUpdateGraph(id: string) {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<GraphResponse, Error, GraphInput>({
    mutationFn: async (input: GraphInput) => {
      const res = await apiClient.PUT("/api/v1/graphs/{id}" as never, {
        params: { path: { id } },
        body: input,
      } as never);
      const { data, error } = res as {
        data: GraphResponse | undefined;
        error: Error | undefined;
      };
      if (error) throw error;
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["graphs"] });
      queryClient.invalidateQueries({ queryKey: ["graphs", id] });
    },
  });
}
