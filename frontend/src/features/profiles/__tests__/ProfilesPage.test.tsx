import { type ReactNode } from 'react';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { ProfilesPage } from '../ProfilesPage';
import type { ProfileDto } from '../types';

const ACTIVE_PROFILE_ID = 'profile-123';

const profileFixture: ProfileDto = {
  id: ACTIVE_PROFILE_ID,
  name: 'Backend Go',
  search_keywords: ['golang', 'backend'],
  location: 'Paris',
  is_active: true,
  skills: ['Go', 'PostgreSQL'],
  seniority: 'senior',
  raw_experience: '',
  conditions: {
    dealbreaker_contract_type: null,
    dealbreaker_remote_policy: null,
    dealbreaker_salary_min: null,
    dealbreaker_required_skills: [],
    preferred_skills: [],
    preferred_max_office_days: null,
    preferred_location: '',
    preferred_working_days: '',
  },
  weights: {
    preferred_skills: 40,
    salary: 20,
    location: 20,
    office_days: 10,
    working_days: 10,
  },
};

function renderProfilesPage() {
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

  return render(<ProfilesPage />, { wrapper: Wrapper });
}

describe('ProfilesPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, profileFixture);
  });

  afterEach(() => {
    mock.restore();
  });

  it('lists profiles and shows editors for the active profile', async () => {
    mock.onGet('/profiles').reply(200, [profileFixture]);

    renderProfilesPage();

    expect(await screen.findByText('Backend Go')).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Identité et compétences' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Conditions de recherche' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', {
        name: 'Pondération du score de pertinence',
      }),
    ).toBeInTheDocument();
  });

  it('warns when the fit-score weights do not sum to 100', async () => {
    mock.onGet('/profiles').reply(200, [profileFixture]);

    renderProfilesPage();

    const weightsSection = await screen.findByRole('region', {
      name: 'Pondération du score de pertinence',
    });
    const salarySlider = within(weightsSection).getByLabelText(/Salaire/);

    fireEvent.change(salarySlider, { target: { value: '50' } });

    expect(
      within(weightsSection).getByRole('alert'),
    ).toHaveTextContent("n'est pas égale à 100");
  });

  it('shows an empty-state message when there are no profiles', async () => {
    mock.onGet('/profiles').reply(200, []);

    renderProfilesPage();

    expect(
      await screen.findByText(
        "Aucun profil pour l'instant. Créez-en un pour commencer.",
      ),
    ).toBeInTheDocument();
  });
});
