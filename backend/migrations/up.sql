-- ============================================================
-- Learning Platform — Full Schema
-- ============================================================

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO roles (name) VALUES ('student'), ('teacher')
    ON CONFLICT DO NOTHING;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id       UUID         NOT NULL REFERENCES roles(id),
    name          VARCHAR(150) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email   ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);

-- Subjects
CREATE TABLE IF NOT EXISTS subjects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,
    code       VARCHAR(20)  NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Enrollments (student <-> subject)
CREATE TABLE IF NOT EXISTS enrollments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id  UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(student_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_enrollments_student ON enrollments(student_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_subject ON enrollments(subject_id);

-- Teacher assignments (teacher <-> subject)
CREATE TABLE IF NOT EXISTS teacher_subjects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id  UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(teacher_id, subject_id)
);

CREATE INDEX IF NOT EXISTS idx_teacher_subjects_teacher ON teacher_subjects(teacher_id);

-- Notes (3-layer review pipeline)
-- draft → ai_reviewed → pending → approved → shared
DO $$ BEGIN
    CREATE TYPE note_status AS ENUM (
        'draft',
        'ai_reviewed',
        'pending',
        'approved',
        'shared'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS notes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id  UUID        NOT NULL REFERENCES subjects(id),
    title       VARCHAR(300) NOT NULL,
    content     TEXT         NOT NULL,
    status      note_status  NOT NULL DEFAULT 'draft',
    ai_feedback TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_author  ON notes(author_id);
CREATE INDEX IF NOT EXISTS idx_notes_subject ON notes(subject_id);
CREATE INDEX IF NOT EXISTS idx_notes_status  ON notes(status);

-- Note shares (viewer | editor)
DO $$ BEGIN
    CREATE TYPE share_role AS ENUM ('viewer', 'editor');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS note_shares (
    id        UUID       PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id   UUID       NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id   UUID       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      share_role NOT NULL DEFAULT 'viewer',
    shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(note_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_note_shares_note ON note_shares(note_id);

-- AI feedback logs
CREATE TABLE IF NOT EXISTS ai_feedback_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id     UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    prompt_used TEXT NOT NULL,
    response    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_logs_note ON ai_feedback_logs(note_id);

-- Messages (async student <-> teacher)
CREATE TABLE IF NOT EXISTS messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT        NOT NULL,
    read        BOOLEAN     NOT NULL DEFAULT FALSE,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (sender_id <> receiver_id)
);

CREATE INDEX IF NOT EXISTS idx_messages_sender   ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver ON messages(receiver_id, read);

-- Calendar events
DO $$ BEGIN
    CREATE TYPE event_type AS ENUM ('tutoring', 'deadline', 'exam', 'other');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS calendar_events (
    id         UUID       PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id UUID       REFERENCES subjects(id),
    title      VARCHAR(300) NOT NULL,
    type       event_type   NOT NULL DEFAULT 'other',
    starts_at  TIMESTAMPTZ  NOT NULL,
    ends_at    TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_calendar_owner     ON calendar_events(owner_id);
CREATE INDEX IF NOT EXISTS idx_calendar_starts_at ON calendar_events(starts_at);


-- Tutoring Bookings (student reserves a tutoring slot)
CREATE TABLE IF NOT EXISTS tutoring_bookings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'confirmed', 'cancelled'
    booked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_tutoring_event ON tutoring_bookings(event_id);