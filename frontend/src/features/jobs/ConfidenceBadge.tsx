interface ConfidenceBadgeProps {
  /** Human-readable field label shown alongside the score. */
  label: string;
  /** Extraction confidence score 0–100. */
  score: number;
}

/** Confidence tier thresholds — shared with the badge colour logic. */
function confidenceTier(score: number): 'high' | 'medium' | 'low' {
  if (score >= 70) return 'high';
  if (score >= 40) return 'medium';
  return 'low';
}

const TIER_LABELS: Record<ReturnType<typeof confidenceTier>, string> = {
  high: 'élevée',
  medium: 'moyenne',
  low: 'faible',
};

/**
 * Inline badge showing the extraction LLM's confidence for a single field.
 * Tier thresholds: high ≥ 70, medium 40–69, low < 40.
 * Rendered as a `<span>` with `data-tier` so tests and CSS can target the tier
 * without relying on colour values.
 */
function ConfidenceBadge({ label, score }: ConfidenceBadgeProps) {
  const tier = confidenceTier(score);
  const tierLabel = TIER_LABELS[tier];

  return (
    <span data-tier={tier} aria-label={`${label} — confiance ${tierLabel} (${score}%)`}>
      {label} — {score}%
    </span>
  );
}

export { ConfidenceBadge, confidenceTier };
