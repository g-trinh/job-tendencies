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
      <h3>Adaptateur de scraping</h3>

      {adapter && (
        <p>
          Statut actuel : {adapter.status === 'approved' ? 'Approuvé' : 'Brouillon'}{' '}
          (version {adapter.version})
        </p>
      )}
      {!adapter && <p>Aucun adaptateur approuvé pour ce board.</p>}

      <div>
        <label htmlFor={`adapter-example-${boardId}`}>
          Page d'exemple (HTML ou JSON de la page de recherche)
        </label>
        <textarea
          id={`adapter-example-${boardId}`}
          value={exampleResponse}
          onChange={(e) => setExampleResponse(e.target.value)}
        />
      </div>
      <button
        type="button"
        disabled={isGenerating || !exampleResponse}
        onClick={() => generate(exampleResponse)}
      >
        Générer un brouillon
      </button>
      {generateFailed && (
        <p role="alert">Échec de la génération de l'adaptateur.</p>
      )}

      {reviewed && (
        <div>
          <h4>Aperçu du brouillon (version {reviewed.version})</h4>
          <pre>{JSON.stringify(reviewed.spec, null, 2)}</pre>
          <button
            type="button"
            disabled={isApproving}
            onClick={() => approve()}
          >
            Approuver l'adaptateur
          </button>
          {approved && <p role="status">Adaptateur approuvé.</p>}
          {approveFailed && (
            <p role="alert">
              Échec de l'approbation. Vérifiez le brouillon et réessayez.
            </p>
          )}
        </div>
      )}
    </section>
  );
}

export { AdapterReview };
