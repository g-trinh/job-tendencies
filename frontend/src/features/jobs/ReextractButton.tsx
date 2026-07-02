import { useReextractMutation } from './useReextractMutation';

interface ReextractButtonProps {
  jobId: string;
}

/**
 * Triggers `POST /api/jobs/{id}/reextract` (P5-4) to re-queue extraction from
 * the job's retained raw listing — e.g. after an extractor improvement.
 * Re-extraction is asynchronous on the backend (202 Accepted, processed later
 * by extract-worker), so this only confirms the request was queued; it never
 * polls or blocks for the result. Disabled while a request is in flight to
 * prevent duplicate re-extraction requests.
 */
function ReextractButton({ jobId }: ReextractButtonProps) {
  const { mutate, isPending, isSuccess, isError } =
    useReextractMutation(jobId);

  return (
    <div>
      <button type="button" disabled={isPending} onClick={() => mutate()}>
        Relancer l&apos;extraction
      </button>
      {isPending && <span aria-live="polite">Ré-extraction en cours…</span>}
      {isSuccess && (
        <span aria-live="polite">Ré-extraction demandée avec succès.</span>
      )}
      {isError && (
        <span role="alert">Impossible de relancer l&apos;extraction.</span>
      )}
    </div>
  );
}

export { ReextractButton };
