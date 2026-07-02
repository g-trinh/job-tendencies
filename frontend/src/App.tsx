import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ActiveProfileProvider } from './context/ActiveProfileContext';
import { AuthProvider } from './context/AuthContext';
import { RequireAuth } from './features/auth';
import { JobsPage, JobDetailPage, KanbanPage } from './features/jobs';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <RequireAuth>
            <ActiveProfileProvider>
              <Routes>
                <Route path="/" element={<JobsPage />} />
                <Route path="/jobs/:id" element={<JobDetailPage />} />
                <Route path="/kanban" element={<KanbanPage />} />
              </Routes>
            </ActiveProfileProvider>
          </RequireAuth>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export { App };
