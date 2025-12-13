-- UUID генерация (gen_random_uuid())
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Создание таблицы отчетов
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_id UUID NOT NULL,
    file_id VARCHAR(255) NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,

    -- Статус анализа
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),

    -- Результаты проверки на плагиат
    plagiarism_flag BOOLEAN NOT NULL DEFAULT FALSE,
    original_work_id UUID,
    match_percentage INTEGER NOT NULL DEFAULT 0 CHECK (match_percentage >= 0 AND match_percentage <= 100),

    -- Хэши файлов
    file_hash VARCHAR(64),
    compared_hashes TEXT[], -- Массив хэшей с которыми сравнивали

    -- Детали анализа
    details JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Метрики производительности
    processing_time_ms INTEGER,
    compared_files_count INTEGER NOT NULL DEFAULT 0,

    -- Временные метки
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(work_id)
);

-- Индексы (PostgreSQL не поддерживает INDEX внутри CREATE TABLE)
CREATE INDEX IF NOT EXISTS idx_reports_work_id ON reports(work_id);
CREATE INDEX IF NOT EXISTS idx_reports_assignment_id ON reports(assignment_id);
CREATE INDEX IF NOT EXISTS idx_reports_student_id ON reports(student_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status);
CREATE INDEX IF NOT EXISTS idx_reports_plagiarism_flag ON reports(plagiarism_flag);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at);
CREATE INDEX IF NOT EXISTS idx_reports_file_hash ON reports(file_hash);

-- Таблица для статистики по заданиям
CREATE TABLE IF NOT EXISTS assignment_stats (
                                                assignment_id UUID PRIMARY KEY,
                                                total_works INTEGER DEFAULT 0,
                                                analyzed_works INTEGER DEFAULT 0,
                                                plagiarized_works INTEGER DEFAULT 0,
                                                avg_match_percentage DECIMAL(5,2) DEFAULT 0,
    last_analyzed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- Таблица для статистики по студентам
CREATE TABLE IF NOT EXISTS student_stats (
                                             student_id UUID PRIMARY KEY,
                                             total_works INTEGER DEFAULT 0,
                                             analyzed_works INTEGER DEFAULT 0,
                                             plagiarized_works INTEGER DEFAULT 0,
                                             avg_match_percentage DECIMAL(5,2) DEFAULT 0,
    last_analyzed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- Таблица для очереди анализа (если нужно сохранять задачи)
CREATE TABLE IF NOT EXISTS analysis_queue (
                                              id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_id UUID NOT NULL,
    file_id VARCHAR(255) NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
    );

CREATE INDEX IF NOT EXISTS idx_analysis_queue_status ON analysis_queue(status);
CREATE INDEX IF NOT EXISTS idx_analysis_queue_priority ON analysis_queue(priority);
CREATE INDEX IF NOT EXISTS idx_analysis_queue_scheduled_at ON analysis_queue(scheduled_at);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_assignment_stats_updated_at
    BEFORE UPDATE ON assignment_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_student_stats_updated_at
    BEFORE UPDATE ON student_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Функция для обновления статистики заданий
CREATE OR REPLACE FUNCTION update_assignment_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Обновляем статистику задания
INSERT INTO assignment_stats (
    assignment_id,
    total_works,
    analyzed_works,
    plagiarized_works,
    avg_match_percentage,
    last_analyzed_at,
    updated_at
)
SELECT
    NEW.assignment_id,
    COUNT(*) as total_works,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as analyzed_works,
    COUNT(CASE WHEN plagiarism_flag = TRUE THEN 1 END) as plagiarized_works,
    COALESCE(AVG(CASE WHEN status = 'completed' THEN match_percentage END), 0) as avg_match_percentage,
    MAX(completed_at) as last_analyzed_at,
    CURRENT_TIMESTAMP
FROM reports
WHERE assignment_id = NEW.assignment_id
    ON CONFLICT (assignment_id) DO UPDATE SET
    total_works = EXCLUDED.total_works,
                                       analyzed_works = EXCLUDED.analyzed_works,
                                       plagiarized_works = EXCLUDED.plagiarized_works,
                                       avg_match_percentage = EXCLUDED.avg_match_percentage,
                                       last_analyzed_at = EXCLUDED.last_analyzed_at,
                                       updated_at = EXCLUDED.updated_at;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_assignment_stats_trigger
    AFTER INSERT OR UPDATE ON reports
                        FOR EACH ROW EXECUTE FUNCTION update_assignment_stats();

-- Функция для обновления статистики студентов
CREATE OR REPLACE FUNCTION update_student_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Обновляем статистику студента
INSERT INTO student_stats (
    student_id,
    total_works,
    analyzed_works,
    plagiarized_works,
    avg_match_percentage,
    last_analyzed_at,
    updated_at
)
SELECT
    NEW.student_id,
    COUNT(*) as total_works,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as analyzed_works,
    COUNT(CASE WHEN plagiarism_flag = TRUE THEN 1 END) as plagiarized_works,
    COALESCE(AVG(CASE WHEN status = 'completed' THEN match_percentage END), 0) as avg_match_percentage,
    MAX(completed_at) as last_analyzed_at,
    CURRENT_TIMESTAMP
FROM reports
WHERE student_id = NEW.student_id
    ON CONFLICT (student_id) DO UPDATE SET
    total_works = EXCLUDED.total_works,
                                    analyzed_works = EXCLUDED.analyzed_works,
                                    plagiarized_works = EXCLUDED.plagiarized_works,
                                    avg_match_percentage = EXCLUDED.avg_match_percentage,
                                    last_analyzed_at = EXCLUDED.last_analyzed_at,
                                    updated_at = EXCLUDED.updated_at;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_student_stats_trigger
    AFTER INSERT OR UPDATE ON reports
                        FOR EACH ROW EXECUTE FUNCTION update_student_stats();