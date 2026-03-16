CREATE UNIQUE INDEX IF NOT EXISTS uq_questions_author_title_content_norm
ON questions (
    author_id,
    COALESCE(lower(btrim(title)), ''),
    COALESCE(lower(btrim(content)), '')
);