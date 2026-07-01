export type View = 'card' | 'table';

interface ViewToggleProps {
  view: View;
  onChange: (view: View) => void;
}

/**
 * Button group that switches the jobs list between card and table views.
 * Uses `aria-pressed` on each button so screen readers announce the active
 * selection without requiring a visually-hidden label.
 */
function ViewToggle({ view, onChange }: ViewToggleProps) {
  return (
    <div role="group" aria-label="Mode d'affichage">
      <button
        type="button"
        aria-pressed={view === 'card'}
        onClick={() => onChange('card')}
      >
        Cartes
      </button>
      <button
        type="button"
        aria-pressed={view === 'table'}
        onClick={() => onChange('table')}
      >
        Tableau
      </button>
    </div>
  );
}

export { ViewToggle };
