CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    email varchar(255) NOT NULL,
    password_hash text NOT NULL,
    created_at timestamp NOT NULL,
    role varchar(50) NOT NULL DEFAULT 'user'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
