-- Удаление триггеров
DROP TRIGGER IF EXISTS update_reports_updated_at ON reports;
DROP TRIGGER IF EXISTS update_assignment_stats_updated_at ON assignment_stats;
DROP TRIGGER IF EXISTS update_student_stats_updated_at ON student_stats;
DROP TRIGGER IF EXISTS update_assignment_stats_trigger ON reports;
DROP TRIGGER IF EXISTS update_student_stats_trigger ON reports;

-- Удаление функций
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS update_assignment_stats();
DROP FUNCTION IF EXISTS update_student_stats();

-- Удаление таблиц в правильном порядке
DROP TABLE IF EXISTS analysis_queue;
DROP TABLE IF EXISTS student_stats;
DROP TABLE IF EXISTS assignment_stats;
DROP TABLE IF EXISTS reports;