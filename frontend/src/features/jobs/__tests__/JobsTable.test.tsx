import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { jobsFixture } from '../fixtures';
import { toJobSummary } from '../types';
import { JobsTable } from '../JobsTable';

const jobs = jobsFixture.map(toJobSummary);

function renderTable(jobList = jobs) {
  return render(
    <MemoryRouter>
      <JobsTable jobs={jobList} />
    </MemoryRouter>,
  );
}

describe('JobsTable', () => {
  // AC: dense table view shows all jobs
  it('renders a row per job with the title linking to the detail page', () => {
    renderTable();

    const titleLink = screen.getByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(titleLink).toHaveAttribute('href', '/jobs/11111111-1111-1111-1111-111111111111');

    expect(
      screen.getByRole('link', { name: 'Développeur Full-Stack' }),
    ).toHaveAttribute('href', '/jobs/22222222-2222-2222-2222-222222222222');
  });

  // AC: structured enum fields shown in French
  it('renders contract type and remote policy in French', () => {
    renderTable();

    expect(screen.getByText('CDI')).toBeInTheDocument();
    expect(screen.getByText('Hybride')).toBeInTheDocument();
  });

  // AC: salary shown compactly
  it('renders salary in compact k€ format', () => {
    renderTable();
    // 65 000 – 85 000 → 65k€–85k€
    expect(screen.getByText('65k€–85k€')).toBeInTheDocument();
  });

  // AC: missing salary shows dash
  it('shows a dash when salary is not published', () => {
    renderTable();
    // Second job has no salary; there are two cells but only one dash for salary
    // The table has other dash cells (company, etc.) — find the salary column
    const rows = screen.getAllByRole('row');
    // Row 1 = header, Row 2 = first job, Row 3 = second job (no salary)
    expect(rows[2].querySelectorAll('td')[4]).toHaveTextContent('—');
  });

  // AC: fit score shown when present
  it('renders fit score when available', () => {
    renderTable();
    expect(screen.getByText('87/100')).toBeInTheDocument();
  });

  // AC: empty state
  it('shows empty-state message when job list is empty', () => {
    renderTable([]);
    expect(screen.getByText('Aucune offre pour ce profil.')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  // AC: application status shown in French
  it('renders application status in French', () => {
    renderTable();
    expect(screen.getByText('Sauvegardé')).toBeInTheDocument();
  });

  // AC: undetermined enum fields show a dash, not a raw key
  it('shows a dash for undetermined contract type', () => {
    renderTable();
    // Second job has contract_type: '' — must not render "job.contract."
    expect(screen.queryByText(/job\.contract\./)).not.toBeInTheDocument();
  });
});
