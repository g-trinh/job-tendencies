# F3: LLM Extraction Pipeline

**Purpose:** Turn raw job listing into structured data.

**Input:** raw job posting (title + description + board metadata).

## Output fields (structured JSON)
- Skills[]
- Remote policy + office days/week
- Contract type (freelance, CDI, CDD, interim)
- Working days (full-time, part-time, 4-day week)
- Salary range
- Seniority level
- Recruiter: name, email, LinkedIn URL, phone (if present)
- **Per-field confidence score (0–100%)**
- **Per-listing understanding score (0–100%)** — overall parse quality

## Language
- Raw data stored as-is, original language, **never translated**.
- Extracted structured fields **displayed in French**.

## Storage
Keep raw + parsed. Raw retained so listings can be re-processed when extraction improves.

## Edge cases
- French or English listings both handled.
- Salary absent (common) → field null, confidence 0.
- Recruiter behind "Easy Apply" → extract what's visible, flag incomplete.
