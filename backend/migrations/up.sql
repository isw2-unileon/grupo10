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
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id        UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id       UUID         REFERENCES subjects(id), -- Le quitamos el NOT NULL
    title            VARCHAR(300) NOT NULL,
    content          TEXT         NOT NULL,
    status           note_status  NOT NULL DEFAULT 'draft',
    ai_feedback      TEXT,
    teacher_feedback TEXT,        -- Añadimos la columna para el profesor
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
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

DROP TABLE IF EXISTS note_shares CASCADE;

CREATE TABLE IF NOT EXISTS note_shares (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id           UUID        NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    shared_with_email VARCHAR(255),
    shared_with_group UUID        REFERENCES class_groups(id) ON DELETE CASCADE,
    role              share_role  NOT NULL DEFAULT 'viewer',
    shared_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (shared_with_email IS NOT NULL OR shared_with_group IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_note_shares_note  ON note_shares(note_id);
CREATE INDEX IF NOT EXISTS idx_note_shares_email ON note_shares(shared_with_email);

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

-- ============================================================
-- Teacher groups (ADR-002): a class group with an email-based
-- student roster. Membership is resolved by matching the email,
-- so students who have not signed up yet can already be listed.
-- ============================================================

-- A class group owned by a teacher.
CREATE TABLE IF NOT EXISTS class_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,
    owner_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_class_groups_owner ON class_groups(owner_id);

-- Student roster, keyed by email (always stored as lower(trim(email))).
CREATE TABLE IF NOT EXISTS group_members (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  UUID         NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    email     VARCHAR(255) NOT NULL,
    added_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, email)
);

CREATE INDEX IF NOT EXISTS idx_group_members_email ON group_members(email);

-- Eliminar tablas previas para recrearlas con soporte de archivos y cuestionarios
DROP TABLE IF EXISTS student_submissions CASCADE;
DROP TABLE IF EXISTS quiz_answers CASCADE;
DROP TABLE IF EXISTS quiz_options CASCADE;
DROP TABLE IF EXISTS quiz_questions CASCADE;
DROP TABLE IF EXISTS group_resources CASCADE;
DROP TABLE IF EXISTS group_sections CASCADE;

-- 1. Secciones de la asignatura
CREATE TABLE IF NOT EXISTS group_sections (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id  UUID NOT NULL REFERENCES class_groups(id) ON DELETE CASCADE,
    title     VARCHAR(200) NOT NULL,
    position  INT NOT NULL DEFAULT 0
);
-- 1.5 Crear el tipo ENUM para los recursos (AÑADIR ESTE BLOQUE)
DO $$ BEGIN
    CREATE TYPE resource_type AS ENUM ('file', 'assignment', 'quiz');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 2. Recursos, Tareas y Cuestionarios
CREATE TABLE IF NOT EXISTS group_resources (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id  UUID NOT NULL REFERENCES group_sections(id) ON DELETE CASCADE,
    type        resource_type NOT NULL, -- 'file', 'assignment', 'quiz'
    title       VARCHAR(300) NOT NULL,
    content     TEXT, -- Descripción de la tarea o instrucciones
    file_path   VARCHAR(500), -- Ruta del archivo subido por el profesor (.docx, .pptx, etc)
    due_at      TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Cuestionarios: Preguntas
CREATE TABLE IF NOT EXISTS quiz_questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES group_resources(id) ON DELETE CASCADE,
    question_text TEXT NOT NULL,
    position    INT NOT NULL DEFAULT 0
);

-- 4. Cuestionarios: Opciones de respuesta
CREATE TABLE IF NOT EXISTS quiz_options (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    is_correct  BOOLEAN NOT NULL DEFAULT FALSE
);

-- 5. Entregas de tareas de los alumnos (Soporta archivos físicos)
CREATE TABLE IF NOT EXISTS student_submissions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id  UUID NOT NULL REFERENCES group_resources(id) ON DELETE CASCADE,
    student_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text_content TEXT, -- Texto opcional que deje el alumno
    file_path    VARCHAR(500), -- Ruta del archivo entregado por el alumno (.pdf, .zip)
    grade        NUMERIC(4,2), -- Nota asignada por el profesor (Ej: 8.50)
    feedback     TEXT, -- Comentarios del profesor
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource_id, student_id)
);

-- 6. Respuestas de los alumnos a los cuestionarios
CREATE TABLE IF NOT EXISTS quiz_submissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES group_resources(id) ON DELETE CASCADE,
    student_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score       NUMERIC(4,2) NOT NULL, -- Nota calculada automáticamente
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource_id, student_id)
);

-- 7. Respuestas individuales de los alumnos a cada pregunta del cuestionario
CREATE TABLE IF NOT EXISTS student_quiz_answers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id   UUID NOT NULL REFERENCES group_resources(id) ON DELETE CASCADE,
    student_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id   UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    option_id     UUID NOT NULL REFERENCES quiz_options(id) ON DELETE CASCADE,
    UNIQUE(student_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_student_quiz_answers_lookup ON student_quiz_answers(resource_id, student_id);