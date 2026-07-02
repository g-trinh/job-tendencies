import { useState } from 'react';
import { useContacts } from './useContacts';
import { ContactRow } from './ContactRow';
import { ContactForm } from './ContactForm';

/**
 * Contacts page at `/contacts`: filterable table with inline tag/notes
 * editing, manual add form, and a CSV export download link. Contacts are
 * global (not scoped to the active profile).
 */
function ContactsPage() {
  const [tagFilter, setTagFilter] = useState('');
  const { data: contacts, isPending, isError } = useContacts(
    tagFilter || undefined,
  );

  return (
    <main>
      <h1>Contacts</h1>

      <div>
        <label htmlFor="contact-tag-filter">Filtrer par tag</label>
        <input
          id="contact-tag-filter"
          type="text"
          value={tagFilter}
          onChange={(e) => setTagFilter(e.target.value)}
        />
      </div>

      <a href="/api/contacts/export.csv">Exporter en CSV</a>

      {isPending && <p>Chargement des contacts…</p>}
      {isError && <p role="alert">Impossible de charger les contacts.</p>}

      {contacts !== undefined && contacts.length === 0 && (
        <p>Aucun contact pour l'instant.</p>
      )}

      {contacts !== undefined && contacts.length > 0 && (
        <table>
          <thead>
            <tr>
              <th>Nom</th>
              <th>Entreprise</th>
              <th>E-mail</th>
              <th>LinkedIn</th>
              <th>Tags</th>
              <th>Notes</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {contacts.map((contact) => (
              <ContactRow key={contact.id} contact={contact} />
            ))}
          </tbody>
        </table>
      )}

      <ContactForm />
    </main>
  );
}

export { ContactsPage };
