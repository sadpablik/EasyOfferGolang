CREATE TABLE IF NOT EXISTS questions (
    id uuid PRIMARY KEY,
    title varchar(255),
    content text NOT NULL,
    category varchar(255),
    answer_format varchar(255),
    language varchar(255),
    starter_code text,
    author_id varchar(255) NOT NULL,
    created_at timestamp
);

CREATE INDEX IF NOT EXISTS idx_questions_category ON questions (category);
CREATE INDEX IF NOT EXISTS idx_questions_answer_format ON questions (answer_format);
CREATE INDEX IF NOT EXISTS idx_questions_language ON questions (language);
CREATE INDEX IF NOT EXISTS idx_questions_created_at ON questions (created_at);
CREATE INDEX IF NOT EXISTS idx_questions_author_id ON questions (author_id);

CREATE TABLE IF NOT EXISTS question_reviews (
    id uuid PRIMARY KEY,
    user_id varchar(255) NOT NULL,
    question_id uuid NOT NULL,
    status varchar(255),
    user_answer text,
    note text,
    reviewed_at timestamp
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_question ON question_reviews (user_id, question_id);
CREATE INDEX IF NOT EXISTS idx_user_status_question ON question_reviews (user_id, status, question_id);
