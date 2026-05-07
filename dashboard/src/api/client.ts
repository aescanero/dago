import createClient from "openapi-fetch";
import type { paths } from "./types.gen";

export type ApiClient = ReturnType<typeof createClient<paths>>;

export function createApiClient(getToken: () => string | null): ApiClient {
  const client = createClient<paths>({ baseUrl: import.meta.env.VITE_API_URL });

  client.use({
    onRequest({ request }) {
      const token = getToken();
      if (token) {
        request.headers.set("Authorization", `Bearer ${token}`);
      }
      return request;
    },
  });

  return client;
}
