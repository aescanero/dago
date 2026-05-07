import { Routes, Route, Navigate } from "react-router-dom";
import { AuthCallback } from "@/auth/AuthCallback";
import { ProtectedRoute } from "@/auth/ProtectedRoute";
import { AppLayout } from "@/layouts/AppLayout";
import {
  GraphsPage,
  GraphDetailPage,
  GraphCreatePage,
  GraphEditPage,
} from "@/features/graphs";
import { NotFoundPage } from "@/pages/NotFoundPage";

export function App() {
  return (
    <Routes>
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Navigate to="/graphs" replace />} />
        <Route path="/graphs" element={<GraphsPage />} />
        <Route path="/graphs/new" element={<GraphCreatePage />} />
        <Route path="/graphs/:id" element={<GraphDetailPage />} />
        <Route path="/graphs/:id/edit" element={<GraphEditPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
