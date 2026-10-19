PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS applications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_name TEXT NOT NULL,
    role_title TEXT NOT NULL,
    location TEXT DEFAULT '',
    job_url TEXT DEFAULT '',
    salary_range TEXT DEFAULT '',
    notes TEXT DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS application_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    application_id INTEGER NOT NULL,
    status TEXT NOT NULL CHECK(
        status IN (
            'SAVED', 'APPLIED', 'RESPONSE_RECEIVED', 'INTERVIEWING',
            'OFFER', 'ACCEPTED', 'REJECTED', 'GHOSTED', 'WITHDRAWN'
        )
    ),
    notes TEXT DEFAULT '',
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_status_history_app_id_date
ON application_status_history(application_id, changed_at DESC, id DESC);
