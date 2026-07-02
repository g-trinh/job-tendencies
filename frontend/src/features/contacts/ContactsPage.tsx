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
      <header className="page__head row-between">
        <div className="stack">
          <h1 className="page__title">Contacts</h1>
          <p className="page__sub">
            Recruteurs auto-remplis depuis l'extraction. Dédupliqués par e-mail
            ou LinkedIn.
          </p>
        </div>
        <a className="btn btn--secondary" href="/api/contacts/export.csv">
          Exporter en CSV
        </a>
      </header>

      <div className="stack stack-5">
        <section aria-label="Liste des contacts">
          <div className="card__head">
            <div className="field">
              <label className="field__label" htmlFor="contact-tag-filter">
                Filtrer par tag
              </label>
              <input
                className="input"
                id="contact-tag-filter"
                type="text"
                value={tagFilter}
                onChange={(e) => setTagFilter(e.target.value)}
              />
            </div>
          </div>

          {isPending && <p className="muted">Chargement des contacts…</p>}
          {isError && (
            <div className="banner banner--danger" role="alert">
              Impossible de charger les contacts.
            </div>
          )}

          {contacts !== undefined && contacts.length === 0 && (
            <div className="state">
              <span className="state__title">Aucun contact pour l'instant.</span>
            </div>
          )}

          {contacts !== undefined && contacts.length > 0 && (
            <div className="table-wrap">
              <table className="table">
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
            </div>
          )}
        </section>

        <ContactForm />
      </div>
    </main>
  );
}

export { ContactsPage };
