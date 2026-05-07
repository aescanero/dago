import { http, HttpResponse } from "msw";

export const handlers = [
  http.get("http://localhost:8080/api/v1/graphs", () => {
    return HttpResponse.json({
      items: [],
      pagination: { page: 1, per_page: 20, total: 0, total_pages: 0 },
    });
  }),

  http.post("http://localhost:8081/token", () => {
    return HttpResponse.json({
      access_token: "mock-jwt-token",
      token_type: "Bearer",
      expires_in: 3600,
    });
  }),
];
