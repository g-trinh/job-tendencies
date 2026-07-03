import { useDashboardStats } from './useDashboard';
import { t } from '../../i18n/fr';

/** Four stat cards: total, new today, new this week, remote %, avg salary, top contract. */
function StatsCards() {
  const { data: stats, isPending, isError } = useDashboardStats();

  if (isPending) return <p className="muted">Chargement des statistiques…</p>;
  if (isError)
    return (
      <div className="banner banner--danger" role="alert">
        Impossible de charger les statistiques.
      </div>
    );
  if (!stats) return null;

  return (
    <section className="grid-stats" aria-label="Statistiques">
      <div className="stat">
        <div className="stat__label">Total des offres</div>
        <div className="stat__value">{stats.total}</div>
      </div>
      <div className="stat">
        <div className="stat__label">Nouvelles aujourd'hui</div>
        <div className="stat__value">{stats.new_today}</div>
      </div>
      <div className="stat">
        <div className="stat__label">Nouvelles cette semaine</div>
        <div className="stat__value">{stats.new_this_week}</div>
      </div>
      <div className="stat">
        <div className="stat__label">% en télétravail</div>
        <div className="stat__value">{Math.round(stats.pct_remote)}%</div>
      </div>
      <div className="stat">
        <div className="stat__label">Salaire moyen</div>
        <div className="stat__value">
          {stats.avg_salary != null
            ? `${Math.round(stats.avg_salary).toLocaleString('fr-FR')} €`
            : 'Non communiqué'}
        </div>
      </div>
      <div className="stat">
        <div className="stat__label">Contrat le plus fréquent</div>
        <div className="stat__value" style={{ fontSize: 'var(--font-size-lg)' }}>
          {stats.top_contract_type
            ? t(`job.contract.${stats.top_contract_type}`)
            : 'Indéterminé'}
        </div>
      </div>
    </section>
  );
}

export { StatsCards };
