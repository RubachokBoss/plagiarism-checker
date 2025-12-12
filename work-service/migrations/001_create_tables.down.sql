-- Удаление триггеров
DROP TRIGGER IF EXISTS update_works_updated_at ON works;
DROP TRIGGER IF EXISTS update_students_updated_at ON students;
DROP TRIGGER IF EXISTS update_assignments_updated_at ON assignments;

-- Удаление функции
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаление таблиц в правильном порядке (из-за foreign keys)
DROP TABLE IF EXISTS works;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS assignments;