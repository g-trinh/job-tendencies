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
    <section aria-label="Fréquence des compétences">
      <h2>Fréquence des compétences</h2>
      {isPending && <p>Chargement…</p>}
      {isError && <p role="alert">Impossible de charger les compétences.</p>}
      {data !== undefined && data.length === 0 && (
        <p>Aucune donnée de compétence disponible.</p>
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
