-- Создание таблицы метаданных файлов
CREATE TABLE IF NOT EXISTS file_metadata (
                                             id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_name VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_extension VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'minio',
    storage_bucket VARCHAR(255) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    storage_url VARCHAR(500),
    upload_status VARCHAR(50) NOT NULL DEFAULT 'uploaded'
    CHECK (upload_status IN ('uploaded', 'processing', 'failed', 'deleted')),

    -- Информация о загрузке
    uploaded_by VARCHAR(255),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Информация о доступе
    access_count INTEGER DEFAULT 0,
    last_accessed_at TIMESTAMP WITH TIME ZONE,

                                   -- Метаданные
                                   metadata JSONB DEFAULT '{}',

                                   -- Индексы
                                   UNIQUE(hash, file_size),
    UNIQUE(storage_path)
    );

-- Индексы для оптимизации запросов
CREATE INDEX idx_file_metadata_hash ON file_metadata(hash);
CREATE INDEX idx_file_metadata_uploaded_at ON file_metadata(uploaded_at);
CREATE INDEX idx_file_metadata_status ON file_metadata(upload_status);
CREATE INDEX idx_file_metadata_original_name ON file_metadata(original_name);
CREATE INDEX idx_file_metadata_metadata ON file_metadata USING GIN(metadata);

-- Триггер для обновления времени последнего доступа
CREATE OR REPLACE FUNCTION update_last_accessed_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_accessed_at = CURRENT_TIMESTAMP;
    NEW.access_count = COALESCE(OLD.access_count, 0) + 1;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_file_metadata_access
    BEFORE UPDATE OF access_count ON file_metadata
    FOR EACH ROW EXECUTE FUNCTION update_last_accessed_at();

-- Таблица для связей файлов (можно использовать для связывания файлов с работами)
CREATE TABLE IF NOT EXISTS file_associations (
                                                 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES file_metadata(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- 'work', 'assignment', 'student', etc.
    entity_id VARCHAR(255) NOT NULL,
    association_type VARCHAR(50) NOT NULL DEFAULT 'primary', -- 'primary', 'attachment', 'thumbnail', etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                                                           UNIQUE(file_id, entity_type, entity_id, association_type)
    );

-- Индексы для таблицы связей
CREATE INDEX idx_file_associations_file_id ON file_associations(file_id);
CREATE INDEX idx_file_associations_entity ON file_associations(entity_type, entity_id);
CREATE INDEX idx_file_associations_created_at ON file_associations(created_at);