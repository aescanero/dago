import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import { toast } from "sonner";

export function useArchiveGraph() {
  const { apiClient } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: async (id: string) => {
      const res = await apiClient.DELETE("/api/v1/graphs/{id}" as never, {
        params: { path: { id } },
      } as never);
      const { error } = res as { error: Error | undefined };
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["graphs"] });
      toast.success("Graph archived");
    },
    onError: (error) => {
      const msg =
        (error as unknown as { code?: string; message?: string }).message ??
        "Error archiving graph";
      toast.error(msg);
    },
  });
}
