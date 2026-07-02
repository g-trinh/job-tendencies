import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import { ActiveProfileProvider } from './context/ActiveProfileContext';
import { AuthProvider } from './context/AuthContext';
import { RequireAuth } from './features/auth';
import { JobsPage, JobDetailPage, KanbanPage } from './features/jobs';
import { ProfilesPage } from './features/profiles';
import { BoardsPage } from './features/boards';
import { DashboardPage } from './features/dashboard';
import { ContactsPage } from './features/contacts';
import { PipelinePage } from './features/pipeline';

const queryClient = new QueryClient();

/** Top-level navigation across every Phase 6 feature area. */
function AppNav() {
  return (
    <nav aria-label="Navigation principale">
      <Link to="/">Offres</Link>
      <Link to="/kanban">Candidatures</Link>
      <Link to="/dashboard">Tableau de bord</Link>
      <Link to="/profiles">Profils</Link>
      <Link to="/boards">Boards</Link>
      <Link to="/contacts">Contacts</Link>
      <Link to="/pipeline">Pipeline</Link>
    </nav>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <RequireAuth>
            <ActiveProfileProvider>
              <AppNav />
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
            </ActiveProfileProvider>
          </RequireAuth>
        </BrowserRouter>
      </AuthProvider>
    </QueryClientProvider>
  );
}

export { App };
