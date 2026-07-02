import { Link } from 'react-router-dom';
import { useDashboardMatches } from './useDashboard';

/** Top job matches ranked by weighted fit score; dealbreaker-failed jobs are excluded server-side. */
function MatchAlerts() {
  const { data: matches, isPending, isError } = useDashboardMatches();

  return (
    <section aria-label="Meilleures correspondances">
      <h2>Meilleures correspondances</h2>
      {isPending && <p>Chargement…</p>}
      {isError && <p role="alert">Impossible de charger les correspondances.</p>}
      {matches !== undefined && matches.length === 0 && (
        <p>Aucune correspondance pour le moment.</p>
      )}
      {matches !== undefined && matches.length > 0 && (
        <ul aria-label="Offres correspondantes">
          {matches.map((m) => (
            <li key={m.id}>
              <Link to={`/jobs/${m.id}`}>{m.title || "Voir l'offre"}</Link>
              {(m.company || m.location) && (
                <p>{[m.company, m.location].filter(Boolean).join(' — ')}</p>
              )}
              {m.weighted_score != null && (
                <p>Score pondéré : {Math.round(m.weighted_score)}/100</p>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export { MatchAlerts };
