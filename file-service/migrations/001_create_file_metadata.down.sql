-- Удаление триггера
DROP TRIGGER IF EXISTS update_file_metadata_access ON file_metadata;

-- Удаление функции
DROP FUNCTION IF EXISTS update_last_accessed_at();

-- Удаление таблиц в правильном порядке (из-за foreign keys)
DROP TABLE IF EXISTS file_associations;
DROP TABLE IF EXISTS file_metadata;