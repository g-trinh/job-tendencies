import { useState } from 'react';
import { useBoards } from './useBoards';
import {
  useCreateBoardMutation,
  useDeleteBoardMutation,
  useUpdateBoardMutation,
} from './useBoardMutations';
import { AdapterReview } from './AdapterReview';
import { ScheduleEditor } from './ScheduleEditor';
import type { BoardDto } from './types';

/** Inline form to create a new board. */
function CreateBoardForm() {
  const [name, setName] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const { mutate, isPending } = useCreateBoardMutation();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    mutate(
      { name, base_url: baseUrl },
      { onSuccess: () => { setName(''); setBaseUrl(''); } },
    );
  }

  return (
    <form className="card" aria-label="Ajouter un board" onSubmit={handleSubmit}>
      <div className="card__head">
        <h2 className="card__title">Ajouter un board</h2>
      </div>
      <div className="stack stack-4">
        <div className="field">
          <label className="field__label" htmlFor="board-name">
            Nom
          </label>
          <input
            className="input"
            id="board-name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="field__label" htmlFor="board-base-url">
            URL de base
          </label>
          <input
            className="input"
            id="board-base-url"
            type="url"
            required
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
          />
        </div>
        <button className="btn btn--primary" type="submit" disabled={isPending}>
          Ajouter
        </button>
      </div>
    </form>
  );
}

/** One board row: enabled toggle, delete, and the adapter review flow. */
function BoardRow({ board }: { board: BoardDto }) {
  const { mutate: update } = useUpdateBoardMutation();
  const { mutate: remove, isPending: isDeleting } = useDeleteBoardMutation();

  return (
    <li>
      <article className="card" aria-label={board.name}>
        <div className="card__head">
          <h3 className="card__title">{board.name}</h3>
          <label className="toggle" htmlFor={`board-enabled-${board.id}`}>
            <input
              id={`board-enabled-${board.id}`}
              type="checkbox"
              checked={board.enabled}
              onChange={(e) =>
                update({
                  id: board.id,
                  name: board.name,
                  base_url: board.base_url,
                  enabled: e.target.checked,
                })
              }
            />
            <span className="toggle__track" />
            <span className="text-sm">Activé</span>
          </label>
        </div>
        <div className="row-between mbe-4">
          <p className="mono text-xs muted">{board.base_url}</p>
          <button
            className="btn btn--ghost btn--sm"
            type="button"
            disabled={isDeleting}
            onClick={() => remove(board.id)}
          >
            Supprimer
          </button>
        </div>
        <AdapterReview boardId={board.id} adapter={board.adapter} />
      </article>
    </li>
  );
}

/**
 * Boards page at `/boards`: board CRUD with enabled toggles (warns when every
 * board is disabled, since the pipeline would then have nothing to scrape),
 * the global schedule editor, and per-board adapter generate/review/approve.
 */
function BoardsPage() {
  const { data: boards, isPending, isError } = useBoards();
  const allDisabled =
    boards !== undefined &&
    boards.length > 0 &&
    boards.every((b) => !b.enabled);

  const enabledCount = boards?.filter((b) => b.enabled).length ?? 0;

  return (
    <>
      <header className="page__head">
        <div>
          <h1 className="page__title">Boards</h1>
          <p className="page__sub">
            Gérez les sites scrapés, la planification globale et les
            adaptateurs générés par IA.
          </p>
        </div>
        {boards !== undefined && (
          <span className="badge badge--neutral num">
            {boards.length} source{boards.length > 1 ? 's' : ''} ·{' '}
            {enabledCount} activée{enabledCount > 1 ? 's' : ''}
          </span>
        )}
      </header>

      <div className="stack stack-5">
        {isPending && <p className="muted">Chargement des boards…</p>}
        {isError && (
          <div className="banner banner--danger" role="alert">
            Impossible de charger les boards.
          </div>
        )}

        {allDisabled && (
          <div className="banner banner--warning" role="alert">
            <span aria-hidden="true">⚠️</span>
            <span>
              Tous les boards sont désactivés : aucune offre ne sera récupérée.
            </span>
          </div>
        )}

        <ScheduleEditor />

        {boards !== undefined && boards.length === 0 && (
          <div className="card">
            <div className="state">
              <span className="state__icon" aria-hidden="true">
                🗂️
              </span>
              <span className="state__title">
                Aucun board pour l'instant. Ajoutez-en un pour commencer.
              </span>
            </div>
          </div>
        )}

        {boards !== undefined && boards.length > 0 && (
          <ul className="stack stack-4" aria-label="Boards">
            {boards.map((board) => (
              <BoardRow key={board.id} board={board} />
            ))}
          </ul>
        )}

        <CreateBoardForm />
      </div>
    </>
  );
}

export { BoardsPage };
