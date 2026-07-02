/**
 * API contract types for the Contacts feature. Mirror
 * `backend/internal/handler/http/contacts.go` response/request shapes.
 * Contacts are global (not profile-scoped).
 */

export interface ContactDto {
  id: string;
  name: string;
  company: string;
  email: string;
  linkedin_url: string;
  phone: string;
  notes: string;
  tags: string[];
  dedup_key: string;
}

/** Shared request body for POST (upsert by dedup_key) and PUT /api/contacts. */
export interface ContactWriteRequest {
  name: string;
  company: string;
  email: string;
  linkedin_url: string;
  phone: string;
  notes: string;
  tags: string[];
}
