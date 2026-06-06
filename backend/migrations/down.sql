-- ============================================================
-- Learning Platform — Drop Full Schema
-- Order matters: children before parents
-- ============================================================

DROP TABLE IF EXISTS group_tasks;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS class_groups;
DROP TABLE IF EXISTS tutoring_bookings;
DROP TABLE IF EXISTS calendar_events;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS ai_feedback_logs;
DROP TABLE IF EXISTS note_shares;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS teacher_subjects;
DROP TABLE IF EXISTS enrollments;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;

DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS share_role;
DROP TYPE IF EXISTS note_status;
