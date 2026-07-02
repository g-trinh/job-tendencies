import { StatsCards } from './StatsCards';
import { SkillsFrequencyChart } from './SkillsFrequencyChart';
import { SkillsTrendChart } from './SkillsTrendChart';
import { MatchAlerts } from './MatchAlerts';

/** Dashboard page at `/dashboard`, scoped to the active profile. */
function DashboardPage() {
  return (
    <main>
      <h1>Tableau de bord</h1>
      <StatsCards />
      <SkillsFrequencyChart />
      <SkillsTrendChart />
      <MatchAlerts />
    </main>
  );
}

export { DashboardPage };
