import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import type { GraphInput, GraphResponse } from "@/api/types.gen";

export function useCreateGraph() {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<GraphResponse, Error, GraphInput>({
    mutationFn: async (input: GraphInput) => {
      const res = await apiClient.POST("/api/v1/graphs" as never, {
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
    },
  });
}
