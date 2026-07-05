import type { CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { useDashboardMatches } from './useDashboard';

/** Top job matches ranked by weighted fit score; dealbreaker-failed jobs are excluded server-side. */
function MatchAlerts() {
  const { data: matches, isPending, isError } = useDashboardMatches();

  return (
    <section className="card" aria-label="Meilleures correspondances">
      <div className="card__head">
        <h2 className="card__title">Meilleures correspondances</h2>
        <Link className="text-sm" to="/">
          Voir toutes
        </Link>
      </div>
      {isPending && <p className="muted">Chargement…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger les correspondances.
        </div>
      )}
      {matches !== undefined && matches.length === 0 && (
        <p className="muted">Aucune correspondance pour le moment.</p>
      )}
      {matches !== undefined && matches.length > 0 && (
        <ul className="stack stack-3" aria-label="Offres correspondantes">
          {matches.map((m) => (
            <li key={m.id} className="list-row">
              <div className="row">
                {m.weighted_score != null && (
                  <span
                    className="fit-score"
                    style={{ '--v': Math.round(m.weighted_score) } as CSSProperties}
                    aria-hidden="true"
                  >
                    {Math.round(m.weighted_score)}
                  </span>
                )}
                <div className="stack">
                  <Link className="text-sm" to={`/jobs/${m.id}`}>
                    {m.title || "Voir l'offre"}
                  </Link>
                  {(m.company || m.location) && (
                    <span className="muted text-xs">
                      {[m.company, m.location].filter(Boolean).join(' — ')}
                    </span>
                  )}
                </div>
              </div>
              {m.weighted_score != null && (
                <span className="muted text-xs">
                  Score pondéré :{' '}
                  <span className="num">{Math.round(m.weighted_score)}/100</span>
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export { MatchAlerts };
