import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ActiveProfileProvider } from './context/ActiveProfileContext';
import { AuthProvider } from './context/AuthContext';
import { RequireAuth } from './features/auth';
import { JobsPage, JobDetailPage, KanbanPage } from './features/jobs';
import { ProfilesPage } from './features/profiles';
import { BoardsPage } from './features/boards';
import { DashboardPage } from './features/dashboard';
import { ContactsPage } from './features/contacts';
import { PipelinePage } from './features/pipeline';
import { AppShell } from './components/AppShell';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <RequireAuth>
            <ActiveProfileProvider>
              <AppShell>
                <Routes>
                  <Route path="/" element={<JobsPage />} />
                  <Route path="/jobs/:id" element={<JobDetailPage />} />
                  <Route path="/kanban" element={<KanbanPage />} />
                  <Route path="/dashboard" element={<DashboardPage />} />
                  <Route path="/profiles" element={<ProfilesPage />} />
                  <Route path="/boards" element={<BoardsPage />} />
                  <Route path="/contacts" element={<ContactsPage />} />
                  <Route path="/pipeline" element={<PipelinePage />} />
                </Routes>
              </AppShell>
            </ActiveProfileProvider>
          </RequireAuth>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export { App };
