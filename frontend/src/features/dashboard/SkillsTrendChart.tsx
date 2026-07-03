import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useSkillTrend } from './useDashboard';

const LINE_COLORS = ['#2563eb', '#16a34a', '#dc2626', '#ca8a04', '#7c3aed'];

/** Pivots flat (period, skill, count) rows into one row per period, one column per skill. */
function pivotBySkill(
  rows: { period: string; skill: string; count: number }[],
): { rows: Record<string, number | string>[]; skills: string[] } {
  const skills = Array.from(new Set(rows.map((r) => r.skill)));
  const periods = Array.from(new Set(rows.map((r) => r.period))).sort();

  const pivoted = periods.map((period) => {
    const row: Record<string, number | string> = { period };
    for (const skill of skills) {
      const match = rows.find((r) => r.period === period && r.skill === skill);
      row[skill] = match?.count ?? 0;
    }
    return row;
  });

  return { rows: pivoted, skills };
}

/** Line chart of skill counts over time (weekly/monthly buckets from the API). */
function SkillsTrendChart() {
  const { data, isPending, isError } = useSkillTrend();
  const pivoted = data ? pivotBySkill(data) : null;

  return (
    <section className="card" aria-label="Évolution des compétences">
      <div className="card__head">
        <h2 className="card__title">Évolution des compétences</h2>
        <span className="badge badge--neutral">Tendance</span>
      </div>
      {isPending && <p className="muted">Chargement…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger la tendance des compétences.
        </div>
      )}
      {pivoted !== null && pivoted.rows.length === 0 && (
        <p className="muted">Aucune donnée de tendance disponible.</p>
      )}
      {pivoted !== null && pivoted.rows.length > 0 && (
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={pivoted.rows}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="period"
              tickFormatter={(v: string) => new Date(v).toLocaleDateString('fr-FR')}
            />
            <YAxis allowDecimals={false} />
            <Tooltip
              labelFormatter={(v: string) => new Date(v).toLocaleDateString('fr-FR')}
            />
            <Legend />
            {pivoted.skills.map((skill, i) => (
              <Line
                key={skill}
                type="monotone"
                dataKey={skill}
                stroke={LINE_COLORS[i % LINE_COLORS.length]}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      )}
    </section>
  );
}

export { SkillsTrendChart };
