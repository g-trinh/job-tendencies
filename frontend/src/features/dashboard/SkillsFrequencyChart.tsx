import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { useSkillFrequency } from './useDashboard';

/** Horizontal bar chart of the most frequent skills across visible jobs. */
function SkillsFrequencyChart() {
  const { data, isPending, isError } = useSkillFrequency();

  return (
    <section className="card" aria-label="Fréquence des compétences">
      <div className="card__head">
        <h2 className="card__title">Fréquence des compétences</h2>
      </div>
      {isPending && <p className="muted">Chargement…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger les compétences.
        </div>
      )}
      {data !== undefined && data.length === 0 && (
        <p className="muted">Aucune donnée de compétence disponible.</p>
      )}
      {data !== undefined && data.length > 0 && (
        <ResponsiveContainer width="100%" height={Math.max(200, data.length * 32)}>
          <BarChart data={data} layout="vertical">
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis type="number" allowDecimals={false} />
            <YAxis type="category" dataKey="skill" width={120} />
            <Tooltip />
            <Bar dataKey="count" fill="#2563eb" name="Occurrences" />
          </BarChart>
        </ResponsiveContainer>
      )}
    </section>
  );
}

export { SkillsFrequencyChart };
