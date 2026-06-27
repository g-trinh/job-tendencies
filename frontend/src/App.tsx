import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ActiveProfileProvider } from './context/ActiveProfileContext';
import { JobsPage, JobDetailPage, KanbanPage } from './features/jobs';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ActiveProfileProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<JobsPage />} />
            <Route path="/jobs/:id" element={<JobDetailPage />} />
            <Route path="/kanban" element={<KanbanPage />} />
          </Routes>
        </BrowserRouter>
      </ActiveProfileProvider>
    </QueryClientProvider>
  );
}

export { App };
