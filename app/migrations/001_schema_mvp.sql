CREATE TABLE import_batches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_filename TEXT NOT NULL,
  source_sha256 TEXT NOT NULL,
  total_rows INTEGER NOT NULL,
  valid_rows INTEGER NOT NULL,
  invalid_rows INTEGER NOT NULL,
  imported_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE flashcards (
  id TEXT PRIMARY KEY,
  level TEXT NOT NULL CHECK (level IN ('n5', 'n4', 'n3', 'n2', 'n1')),
  category TEXT NOT NULL CHECK (category IN ('gramma', 'vocabulary', 'kanji', 'reading', 'listening')),
  name TEXT NOT NULL,
  mean TEXT NOT NULL,
  hiragana TEXT NOT NULL,
  image TEXT NULL,
  audio TEXT NULL,
  example TEXT NULL,
  source_batch_id INTEGER NULL REFERENCES import_batches(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_flashcards_level_category ON flashcards(level, category);
CREATE INDEX idx_flashcards_name ON flashcards(name);

CREATE TABLE study_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  current_card_id TEXT NULL REFERENCES flashcards(id) ON DELETE SET NULL,
  order_mode TEXT NOT NULL CHECK (order_mode IN ('random', 'sequential')) DEFAULT 'sequential',
  reveal_answers INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE card_notes (
  card_id TEXT PRIMARY KEY REFERENCES flashcards(id) ON DELETE CASCADE,
  note TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE card_bookmarks (
  card_id TEXT PRIMARY KEY REFERENCES flashcards(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE app_settings (
  key TEXT PRIMARY KEY,
  value_json TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
