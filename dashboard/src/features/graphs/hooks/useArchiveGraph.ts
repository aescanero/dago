import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/auth/useAuth";
import { toast } from "sonner";

const API_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export function useArchiveGraph() {
  const { token } = useAuth();
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: async (id: string) => {
      const response = await fetch(`${API_URL}/api/v1/graphs/${id}`, {
        method: "DELETE",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });

      if (response.status === 409) {
        const body = await response
          .json()
          .catch(() => ({ message: "The graph has active executions" })) as { message?: string };
        throw new Error(body.message ?? "The graph has active executions");
      }

      if (!response.ok && response.status !== 204) {
        throw new Error("Error archiving graph");
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["graphs"] });
      toast.success("Graph archived");
    },
    onError: (error) => {
      toast.error(error.message ?? "Error archiving graph");
    },
  });
}
