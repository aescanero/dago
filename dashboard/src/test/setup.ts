import "@testing-library/jest-dom";
import { server } from "./server";

beforeAll(() => server.listen({ onUnhandledRequest: "warn" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// Polyfill crypto.subtle for jsdom if not available
if (!globalThis.crypto.subtle) {
  const { webcrypto } = await import("node:crypto");
  Object.defineProperty(globalThis, "crypto", {
    value: webcrypto,
    writable: false,
  });
}
