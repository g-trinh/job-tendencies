import { useDashboardStats } from './useDashboard';
import { t } from '../../i18n/fr';

/** Four stat cards: total, new today, new this week, remote %, avg salary, top contract. */
function StatsCards() {
  const { data: stats, isPending, isError } = useDashboardStats();

  if (isPending) return <p>Chargement des statistiques…</p>;
  if (isError) return <p role="alert">Impossible de charger les statistiques.</p>;
  if (!stats) return null;

  return (
    <section aria-label="Statistiques">
      <h2>Statistiques</h2>
      <dl>
        <div>
          <dt>Total des offres</dt>
          <dd>{stats.total}</dd>
        </div>
        <div>
          <dt>Nouvelles aujourd'hui</dt>
          <dd>{stats.new_today}</dd>
        </div>
        <div>
          <dt>Nouvelles cette semaine</dt>
          <dd>{stats.new_this_week}</dd>
        </div>
        <div>
          <dt>% en télétravail</dt>
          <dd>{Math.round(stats.pct_remote)}%</dd>
        </div>
        <div>
          <dt>Salaire moyen</dt>
          <dd>
            {stats.avg_salary != null
              ? `${Math.round(stats.avg_salary).toLocaleString('fr-FR')} €`
              : 'Non communiqué'}
          </dd>
        </div>
        <div>
          <dt>Contrat le plus fréquent</dt>
          <dd>
            {stats.top_contract_type
              ? t(`job.contract.${stats.top_contract_type}`)
              : 'Indéterminé'}
          </dd>
        </div>
      </dl>
    </section>
  );
}

export { StatsCards };
