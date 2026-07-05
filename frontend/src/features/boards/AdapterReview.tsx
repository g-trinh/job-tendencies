import { useState } from 'react';
import {
  useApproveAdapterMutation,
  useGenerateAdapterMutation,
} from './useBoardMutations';
import type { AdapterDto } from './types';

interface AdapterReviewProps {
  boardId: string;
  adapter: AdapterDto | null;
}

/**
 * Adapter generate → review draft → approve flow for a single board.
 * `adapter` is the board's most-recently-approved adapter (null if none
 * approved yet); a freshly generated draft replaces the reviewed spec until
 * approved, at which point `useBoards` re-fetches and shows the approved one.
 */
function AdapterReview({ boardId, adapter }: AdapterReviewProps) {
  const [exampleResponse, setExampleResponse] = useState('');
  const {
    mutate: generate,
    data: draft,
    isPending: isGenerating,
    isError: generateFailed,
  } = useGenerateAdapterMutation(boardId);
  const {
    mutate: approve,
    isPending: isApproving,
    isSuccess: approved,
    isError: approveFailed,
  } = useApproveAdapterMutation(boardId);

  const reviewed = draft ?? adapter;

  return (
    <section aria-label="Adaptateur de scraping">
      <div className="card__head">
        <h4 className="card__title">Adaptateur de scraping</h4>
        {adapter ? (
          <span
            className={`badge ${adapter.status === 'approved' ? 'badge--success' : 'badge--warning'}`}
          >
            {adapter.status === 'approved' ? 'Prêt' : 'Brouillon à valider'} — v
            {adapter.version}
          </span>
        ) : (
          <span className="badge badge--neutral">Aucun</span>
        )}
      </div>

      <div className="field mbe-4">
        <label className="field__label" htmlFor={`adapter-example-${boardId}`}>
          Page d'exemple (HTML ou JSON de la page de recherche)
        </label>
        <textarea
          className="textarea mono"
          id={`adapter-example-${boardId}`}
          value={exampleResponse}
          onChange={(e) => setExampleResponse(e.target.value)}
        />
      </div>
      <button
        className="btn btn--secondary btn--sm"
        type="button"
        disabled={isGenerating || !exampleResponse}
        onClick={() => generate(exampleResponse)}
      >
        Générer un brouillon
      </button>
      {generateFailed && (
        <div className="banner banner--danger" role="alert">
          Échec de la génération de l'adaptateur.
        </div>
      )}

      {reviewed && (
        <div className="card card--pad-sm mbs-4">
          <div className="field">
            <span className="field__label">
              Aperçu du brouillon (version {reviewed.version})
            </span>
            <pre className="code-block">
              {JSON.stringify(reviewed.spec, null, 2)}
            </pre>
          </div>
          <div className="row justify-end mbs-4">
            {approved && (
              <span className="badge badge--success" role="status">
                Adaptateur approuvé.
              </span>
            )}
            <button
              className="btn btn--primary btn--sm"
              type="button"
              disabled={isApproving}
              onClick={() => approve()}
            >
              Approuver l'adaptateur
            </button>
          </div>
          {approveFailed && (
            <div className="banner banner--danger" role="alert">
              Échec de l'approbation. Vérifiez le brouillon et réessayez.
            </div>
          )}
        </div>
      )}
    </section>
  );
}

export { AdapterReview };
