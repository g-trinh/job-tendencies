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
    <form aria-label="Ajouter un board" onSubmit={handleSubmit}>
      <h2>Ajouter un board</h2>
      <div>
        <label htmlFor="board-name">Nom</label>
        <input
          id="board-name"
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="board-base-url">URL de base</label>
        <input
          id="board-base-url"
          type="url"
          required
          value={baseUrl}
          onChange={(e) => setBaseUrl(e.target.value)}
        />
      </div>
      <button type="submit" disabled={isPending}>
        Ajouter
      </button>
    </form>
  );
}

/** One board row: enabled toggle, delete, and the adapter review flow. */
function BoardRow({ board }: { board: BoardDto }) {
  const { mutate: update } = useUpdateBoardMutation();
  const { mutate: remove, isPending: isDeleting } = useDeleteBoardMutation();

  return (
    <li>
      <article aria-label={board.name}>
        <h3>{board.name}</h3>
        <p>{board.base_url}</p>
        <label htmlFor={`board-enabled-${board.id}`}>
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
          Activé
        </label>
        <button
          type="button"
          disabled={isDeleting}
          onClick={() => remove(board.id)}
        >
          Supprimer
        </button>
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

  return (
    <main>
      <h1>Boards</h1>

      {isPending && <p>Chargement des boards…</p>}
      {isError && <p role="alert">Impossible de charger les boards.</p>}

      {allDisabled && (
        <p role="alert">
          Tous les boards sont désactivés : aucune offre ne sera récupérée.
        </p>
      )}

      {boards !== undefined && boards.length === 0 && (
        <p>Aucun board pour l'instant. Ajoutez-en un pour commencer.</p>
      )}

      {boards !== undefined && boards.length > 0 && (
        <ul aria-label="Boards">
          {boards.map((board) => (
            <BoardRow key={board.id} board={board} />
          ))}
        </ul>
      )}

      <CreateBoardForm />
      <ScheduleEditor />
    </main>
  );
}

export { BoardsPage };
