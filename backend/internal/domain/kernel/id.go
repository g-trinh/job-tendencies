// Package kernel contains the shared kernel types used across all bounded contexts.
// These include typed IDs, value objects, enums, domain errors, and pagination DTOs.
// No package in kernel imports anything from app, infra, or handler.
package kernel

// ProfileID uniquely identifies a search profile aggregate.
type ProfileID string

// JobID uniquely identifies a deduplicated job listing aggregate.
type JobID string

// BoardID uniquely identifies a job board source.
type BoardID string

// AdapterID uniquely identifies a board scraping adapter spec.
type AdapterID string

// RawListingID uniquely identifies a captured raw listing.
type RawListingID string

// ContactID uniquely identifies a recruiter contact aggregate.
type ContactID string

// ScrapeRunID uniquely identifies a scrape pipeline execution.
type ScrapeRunID string
