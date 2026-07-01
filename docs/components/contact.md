# Component: Contact

A recruiter record. Auto-populated from the extraction pipeline; also manually
editable. Deduplicated by email or LinkedIn URL via a computed `dedup_key`.

## Properties

| Property    | Type      | Description                                              |
|-------------|-----------|----------------------------------------------------------|
| ID          | ContactID | Stable identifier.                                       |
| Name        | string    | Recruiter's full name.                                   |
| Company     | string    | Recruiter's employer.                                    |
| Email       | string    | Recruiter's email. Primary dedup key when non-empty.     |
| LinkedInURL | string    | LinkedIn profile URL. Fallback dedup key.                |
| Phone       | string    | Phone number (optional).                                 |
| Notes       | string    | Free-text notes.                                         |
| Tags        | []string  | Labels (e.g. "in-house", "responsive").                  |
| DedupKey    | string    | `"email:<email>"` or `"linkedin:<url>"`. Unique.         |

## Dedup key computation

`dedup_key = "email:<lower(email)>"` when email is non-empty, otherwise
`"linkedin:<linkedin_url>"`. Email takes priority. The DB unique index on
`dedup_key` enforces deduplication at the storage level.

## Upsert behaviour

`POST /api/contacts` upserts: when a contact with the same `dedup_key` already
exists, name, company, and phone are updated; tags are merged (union); notes are
updated only when the new value is non-empty. Returns **201** when created, **200**
when merged.

## API

| Method | Path                 | Description                       |
|--------|----------------------|-----------------------------------|
| GET    | /api/contacts        | List all contacts.                |
| POST   | /api/contacts        | Upsert by email or linkedin_url.  |
| GET    | /api/contacts/{id}   | Get one contact.                  |
| PUT    | /api/contacts/{id}   | Update all editable fields.       |
| DELETE | /api/contacts/{id}   | Delete contact.                   |

## Notes

- At least one of `email` or `linkedin_url` must be provided.
- Tags and notes (P3-CO-2) and CSV export (P3-CO-3) are later tasks.
- `job.contact_id` FK is added in migration 00010 alongside this table.
