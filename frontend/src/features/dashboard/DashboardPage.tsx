import { StatsCards } from './StatsCards';
import { SkillsFrequencyChart } from './SkillsFrequencyChart';
import { SkillsTrendChart } from './SkillsTrendChart';
import { MatchAlerts } from './MatchAlerts';

/** Dashboard page at `/dashboard`, scoped to the active profile. */
function DashboardPage() {
  return (
    <main>
      <header className="page__head">
        <h1 className="page__title">Tableau de bord</h1>
        <p className="page__sub">Vue d'ensemble pour le profil actif.</p>
      </header>
      <div className="stack stack-5">
        <StatsCards />
        <div className="grid-2">
          <SkillsFrequencyChart />
          <SkillsTrendChart />
        </div>
        <MatchAlerts />
      </div>
    </main>
  );
}

export { DashboardPage };
