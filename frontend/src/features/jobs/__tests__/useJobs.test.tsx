import { type ReactNode } from 'react';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { useJobs } from '../useJobs';
import { jobsFixture, toPagedJobsFixture } from '../fixtures';

const ACTIVE_PROFILE_ID = 'profile-123';

function renderUseJobs(
  ...args: Parameters<typeof useJobs>
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ActiveProfileProvider>{children}</ActiveProfileProvider>
      </QueryClientProvider>
    );
  }

  return renderHook(() => useJobs(...args), { wrapper: Wrapper });
}

describe('useJobs', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, { id: ACTIVE_PROFILE_ID });
  });

  afterEach(() => {
    mock.restore();
  });

  // ADR-007: the hook maps the paginated envelope to camelCase PagedJobs
  it('maps the paginated envelope to camelCase fields', async () => {
    mock
      .onGet('/jobs')
      .reply(
        200,
        toPagedJobsFixture(jobsFixture, { page: 2, page_size: 25, total: 132 }),
      );

    const { result } = renderUseJobs();

    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(result.current.data?.page).toBe(2);
    expect(result.current.data?.pageSize).toBe(25);
    expect(result.current.data?.total).toBe(132);
    expect(result.current.data?.totalPages).toBe(6);
    expect(result.current.data?.items).toHaveLength(jobsFixture.length);
    expect(result.current.data?.items[0].id).toBe(jobsFixture[0].id);
  });

  // ADR-007: page/pageSize are sent as query params, defaulting to 1/25
  it('sends page and page_size as query params, defaulting to page 1 and size 25', async () => {
    let sentParams: Record<string, unknown> = {};
    mock.onGet('/jobs').reply((config) => {
      sentParams = config.params as Record<string, unknown>;
      return [200, toPagedJobsFixture(jobsFixture)];
    });

    const { result } = renderUseJobs();

    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(sentParams['page']).toBe(1);
    expect(sentParams['page_size']).toBe(25);
  });

  // ADR-007: an explicit page/pageSize is forwarded and included in the cache key
  it('forwards an explicit page and pageSize as query params', async () => {
    let sentParams: Record<string, unknown> = {};
    mock.onGet('/jobs').reply((config) => {
      sentParams = config.params as Record<string, unknown>;
      return [200, toPagedJobsFixture(jobsFixture, { page: 3, page_size: 50 })];
    });

    const { result } = renderUseJobs(undefined, { page: 3, pageSize: 50 });

    await waitFor(() => expect(result.current.data).toBeDefined());

    expect(sentParams['page']).toBe(3);
    expect(sentParams['page_size']).toBe(50);
    expect(result.current.data?.page).toBe(3);
    expect(result.current.data?.pageSize).toBe(50);
  });
});
