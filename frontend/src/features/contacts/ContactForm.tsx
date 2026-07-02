import { useState } from 'react';
import { useUpsertContactMutation } from './useContactMutations';

/**
 * Manual add-contact form. POST /api/contacts is an upsert keyed on
 * email/LinkedIn URL, so submitting an existing contact's details merges
 * into that record instead of erroring — the confirmation message reflects
 * which happened.
 */
function ContactForm() {
  const [name, setName] = useState('');
  const [company, setCompany] = useState('');
  const [email, setEmail] = useState('');
  const [linkedinUrl, setLinkedinUrl] = useState('');
  const [phone, setPhone] = useState('');
  const [notes, setNotes] = useState('');
  const [tags, setTags] = useState('');
  const { mutate, isPending, data } = useUpsertContactMutation();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    mutate(
      {
        name,
        company,
        email,
        linkedin_url: linkedinUrl,
        phone,
        notes,
        tags: tags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
      },
      {
        onSuccess: () => {
          setName('');
          setCompany('');
          setEmail('');
          setLinkedinUrl('');
          setPhone('');
          setNotes('');
          setTags('');
        },
      },
    );
  }

  return (
    <form aria-label="Ajouter un contact" onSubmit={handleSubmit}>
      <h2>Ajouter un contact</h2>
      <div>
        <label htmlFor="contact-name">Nom</label>
        <input
          id="contact-name"
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-company">Entreprise</label>
        <input
          id="contact-company"
          type="text"
          value={company}
          onChange={(e) => setCompany(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-email">E-mail</label>
        <input
          id="contact-email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-linkedin">Profil LinkedIn</label>
        <input
          id="contact-linkedin"
          type="url"
          value={linkedinUrl}
          onChange={(e) => setLinkedinUrl(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-phone">Téléphone</label>
        <input
          id="contact-phone"
          type="tel"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-notes">Notes</label>
        <textarea
          id="contact-notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="contact-tags">Tags (séparés par des virgules)</label>
        <input
          id="contact-tags"
          type="text"
          value={tags}
          onChange={(e) => setTags(e.target.value)}
        />
      </div>
      <button type="submit" disabled={isPending}>
        Enregistrer
      </button>
      {data !== undefined && (
        <p role="status">
          {data.created
            ? 'Contact créé.'
            : 'Un contact existant a été mis à jour (même e-mail ou profil LinkedIn).'}
        </p>
      )}
    </form>
  );
}

export { ContactForm };
