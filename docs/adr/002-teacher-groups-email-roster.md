# ADR-002: Teacher groups with an email-based student roster

## Status

Proposed

## Date

2026-06-06

## Context

Teachers need a place to organise their classes and post tasks. The driving
requirements are:

- A teacher owns one or more **groups**. A group holds **tasks** (assignments,
  announcements, deadlines).
- A group may optionally include **other teachers** (co-teachers).
- The members of a group are defined by a **list of student emails**. The
  teacher already has this list (the official class roster) and wants to paste
  it in up front.
- That list will contain **emails that have not signed up yet**, mixed with
  emails of students who already have an account.
- A registered student who is on a group's roster can see that group's tasks.
- A student who is **not in any group** sees a "waiting to be added to a group"
  message instead of an empty screen.

The hard part is the roster containing emails of people who are **not yet users**
of the platform. A plain `student ↔ group` join table cannot express that,
because the student row does not exist in `users` yet.

Two modelling strategies were considered:

- **A — Claim on registration.** The roster row stores `email` plus a nullable
  `user_id` and a `pending | active` status. When a person registers, a hook
  links every pending roster row that matches their email. Robust, but adds a
  nullable FK, an enum, and a cross-module hook in the registration flow.
- **B — Match by email.** The roster stores only the email. Access is decided by
  comparing `group_members.email` against the authenticated user's email. No
  linking step, no status column, no registration hook.

## Decision

Adopt **strategy B (match by email)**.

A student is a member of a group when a roster row exists whose email equals the
authenticated user's email. An unregistered email simply sits in the roster;
the moment that person registers with the same email, the comparison starts to
match and they gain access — with no extra code path.

### Data model

Tables are named to avoid the SQL keyword `GROUP` and follow the existing
conventions (UUID PKs, `snake_case`, automatic migrations).

```sql
-- A class group owned by a teacher.
CREATE TABLE IF NOT EXISTS class_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Student roster, keyed by email so people who have not signed up yet can be
-- listed. No user_id: membership is resolved by matching the email.
CREATE TABLE IF NOT EXISTS group_members (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,           -- ALWAYS stored as lower(trim(email))
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, email)
);

-- Tasks posted to a group (view-only for students in v1).
CREATE TABLE IF NOT EXISTS group_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT,
    due_at      TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Phase 2 only (co-teachers). Designed now, exposed in the UI later.
-- CREATE TABLE IF NOT EXISTS group_teachers (
--     group_id   UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
--     teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
--     PRIMARY KEY (group_id, teacher_id)
-- );
```

### Email normalisation (the one non-negotiable rule)

The whole scheme rests on emails matching. Both `users.email` and
`group_members.email` MUST be stored normalised as `lower(trim(email))`, and all
lookups must normalise the input the same way. `users.email` is already unique;
this ADR adds the requirement that it is also normalised on write (verify the
current register handler does this; migrate existing rows if needed).

The `UNIQUE (group_id, email)` constraint assumes normalised input, so duplicate
invites for the same person are rejected.

### Access rules

- **Create group / add members / create tasks:** caller must have role
  `teacher` and be the group's `owner_id`. (Co-teachers in phase 2.)
- **View a group's tasks:** caller is the owner, or a student whose email
  matches a `group_members` row for that group.
- Identity (`user_id`, `role`, `email`) comes from the JWT via the existing
  `RequireAuth` middleware — the same pattern used to protect other routes.

### HTTP endpoints

Teacher:

- `POST   /api/groups` — create `{ name }`.
- `GET    /api/groups` — groups I own.
- `GET    /api/groups/{id}` — detail: roster (each email tagged registered or
  pending via a `LEFT JOIN users`) + tasks.
- `POST   /api/groups/{id}/members` — bulk add `{ emails: ["a@x", "b@y"] }`
  (normalised; duplicates ignored).
- `DELETE /api/groups/{id}/members/{memberId}` — remove a roster entry.
- `POST   /api/groups/{id}/tasks` — `{ title, description?, due_at? }`.

Student / shared:

- `GET /api/me/groups` — groups I belong to. **Empty array drives the
  "waiting to be added" state.**
- `GET /api/groups/{id}/tasks` — tasks, if the caller is a member or the owner.

The "registered vs pending" badge is computed, not stored:

```sql
SELECT m.email, (u.id IS NOT NULL) AS registered
FROM group_members m
LEFT JOIN users u ON u.email = m.email
WHERE m.group_id = $1
ORDER BY m.email;
```

### Frontend states

- **Student, no groups:** `GET /api/me/groups` returns `[]` →
  "Esperando a que te añadan a un grupo." (UI copy in Spanish, per the team's
  decision; identifiers and code stay in English.)
- **Student, in groups:** list groups and their tasks.
- **Teacher:** list/create groups; group detail with a textarea to paste emails
  (one per line or comma-separated), each shown as **Registrado** or
  **Pendiente**, plus a section to post tasks.

### Backend module

A new `backend/internal/groups` package following the existing hexagonal layers
(`domain`, `service`, `repository`, `handler`), wired in
`backend/cmd/server/main.go` like `users` and `calendar`.

## Consequences

**Easier:**

- Minimal schema: three tables, no nullable FKs, no enum, no registration hook.
- A roster of not-yet-registered students is trivial — they are just emails.
- Auto-join on registration is implicit: a newly created account immediately
  matches any roster row with its email, with zero extra code.
- The "waiting" state is a single empty-list check.

**More difficult / trade-offs:**

- Membership is keyed on a mutable string (email) rather than a stable
  `user_id`. The app currently has no "change email" feature, so this is
  acceptable; introducing one later would require updating roster rows or
  migrating to strategy A.
- Per-member metadata (e.g. join date, per-student status) has no natural home.
  If that becomes a requirement, upgrade to strategy A (add `user_id` + status).
- Correctness depends entirely on consistent email normalisation. This must be
  enforced in the users module as well, not just here.

## Open questions (to decide with the team)

1. **Co-teachers:** invite existing teacher accounts only, or also by email like
   students? (Proposal: existing accounts only, phase 2.)
2. **Co-teacher permissions:** same as owner except deleting the group?
3. **Tasks:** stay view-only, or let students submit deliverables later? (The
   `notes` pipeline already handles submissions — keep tasks view-only in v1.)
4. **Relation to `subjects` / `enrollments` / `teacher_subjects`:** groups are
   proposed as independent and lighter (email roster). Decide whether to link a
   group to a `subject` (optional `subject_id`) or leave them separate.
5. **Invitations:** send an email when a student is added? Out of scope for v1;
   the roster acts purely as an allowlist.
```
