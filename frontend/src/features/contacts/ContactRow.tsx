import { useState } from 'react';
import {
  useDeleteContactMutation,
  useUpdateContactMutation,
} from './useContactMutations';
import type { ContactDto } from './types';

interface ContactRowProps {
  contact: ContactDto;
}

/** One editable contact row: inline tags + notes editing, delete. */
function ContactRow({ contact }: ContactRowProps) {
  const [tags, setTags] = useState(contact.tags.join(', '));
  const [notes, setNotes] = useState(contact.notes);
  const { mutate: update, isPending: isSaving } = useUpdateContactMutation();
  const { mutate: remove, isPending: isDeleting } = useDeleteContactMutation();

  function handleSave() {
    update({
      id: contact.id,
      name: contact.name,
      company: contact.company,
      email: contact.email,
      linkedin_url: contact.linkedin_url,
      phone: contact.phone,
      notes,
      tags: tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
    });
  }

  return (
    <tr>
      <td>{contact.name}</td>
      <td>{contact.company}</td>
      <td>{contact.email}</td>
      <td className="text-xs">
        {contact.linkedin_url && (
          <a href={contact.linkedin_url} target="_blank" rel="noreferrer">
            Profil
          </a>
        )}
      </td>
      <td>
        <label className="sr-only" htmlFor={`tags-${contact.id}`}>
          Tags
        </label>
        <input
          className="input"
          id={`tags-${contact.id}`}
          type="text"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
        />
      </td>
      <td>
        <label className="sr-only" htmlFor={`notes-${contact.id}`}>
          Notes
        </label>
        <textarea
          className="textarea"
          id={`notes-${contact.id}`}
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </td>
      <td>
        <div className="row">
          <button
            className="btn btn--secondary btn--sm"
            type="button"
            disabled={isSaving}
            onClick={handleSave}
          >
            Enregistrer
          </button>
          <button
            className="btn btn--ghost btn--sm"
            type="button"
            disabled={isDeleting}
            onClick={() => remove(contact.id)}
          >
            Supprimer
          </button>
        </div>
      </td>
    </tr>
  );
}

export { ContactRow };
