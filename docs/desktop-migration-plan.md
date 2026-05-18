# Kế hoạch migrate JLPT Flashcard Extension sang Desktop App

## 1. Bối cảnh và quyết định kỹ thuật

Tài liệu này lập kế hoạch migrate từ Chrome Extension hiện tại sang desktop app native-like nhưng vẫn tái sử dụng tối đa UI Svelte. Toàn bộ code migration, PR và bước merge phải xuất phát từ nhánh feature hiện tại, không tạo nhánh tách rời ngoài luồng này.

### 1.1. Mục tiêu migration

- Chuyển runtime từ Chrome Extension Manifest V3 sang desktop app dùng **Golang + Wails v3 + Svelte 5**.
- Cho phép người dùng upload file JSON từ máy, validate theo schema flashcard hiện có, rồi insert dữ liệu vào **SQLite3**.
- Duy trì trải nghiệm học hiện tại: xem flashcard, điều hướng, bookmark, note, filter theo level/category, cài đặt notification/review.
- Loại bỏ phụ thuộc `chrome.storage.local`, `chrome.alarms`, `chrome.notifications` ở core business logic; thay bằng Go services, SQLite và API desktop.
- Thiết kế để có thể giữ extension cũ trong một thời gian chuyển tiếp nếu cần rollback.

### 1.2. Ràng buộc theo nhánh feature

- Mọi thay đổi migration phải commit trên nhánh feature hiện tại.
- PR migration phải target nhánh tích hợp chính của dự án, nhưng source branch luôn là nhánh feature này.
- Không merge trực tiếp từ branch khác vào main cho các thay đổi desktop app; nếu cần lấy code từ nhánh khác thì merge/cherry-pick vào nhánh feature trước, resolve conflict tại đây, rồi mở PR từ nhánh feature.
- Mỗi milestone nên có PR nhỏ, có trạng thái build/check rõ ràng, thay vì một PR migration quá lớn.

## 2. Hiện trạng extension cần migrate

### 2.1. Data model hiện tại

Schema flashcard hiện tại gồm:

```ts
type JlptLevel = 'n5' | 'n4' | 'n3' | 'n2' | 'n1';
type FlashcardCategory = 'gramma' | 'vocabulary' | 'kanji' | 'reading' | 'listening';

interface Flashcard {
  level: JlptLevel;
  category: FlashcardCategory;
  name: string;
  mean: string;
  hiragana: string;
  image: string | null;
  audio: string | null;
  example: string | null;
}
```

Lưu ý quan trọng: category `gramma` đang là giá trị tương thích dữ liệu đầu vào và cần giữ backward-compatible trong migration. Nếu muốn sửa chính tả thành `grammar`, nên làm bằng migration/alias riêng, không đổi breaking trong lần import đầu tiên.

### 2.2. Các phần có thể tái sử dụng

- UI Svelte: các component flashcard, selector level/category, button/card/badge.
- Validator TypeScript hiện có: có thể dùng tạm ở frontend để preview lỗi trước khi gọi Go import service.
- Dataset mẫu: dùng làm seed/demo data cho database.
- Kiểu dữ liệu `Flashcard`, `StudySettings`: chuyển thành contract chung giữa frontend và Go bindings.

### 2.3. Các phần cần thay thế

| Extension hiện tại | Desktop app thay thế |
| --- | --- |
| `chrome.storage.local` | SQLite3 qua Go repository/service |
| `chrome.alarms` | Go scheduler/ticker hoặc OS notification schedule trong app process |
| `chrome.notifications` | Wails notifications service hoặc abstraction notification riêng |
| Options page/popup tách biệt | Một desktop shell nhiều route/view |
| File upload qua browser input | Wails file dialog + drag/drop + fallback input file |
| Build output extension `dist` | Binary desktop + bundled web assets |

## 3. Stack mục tiêu

| Layer | Công nghệ | Vai trò |
| --- | --- | --- |
| Desktop shell | Wails v3 | Đóng gói Go backend + Svelte frontend, native WebView, bindings Go-to-JS |
| Backend | Go | Import JSON, validate server-side, SQLite repository, scheduler, settings |
| Database | SQLite3 | Lưu cards, imports, notes, bookmarks, study progress, settings |
| Frontend | Svelte 5 + TypeScript | UI desktop, upload/import flow, study dashboard |
| Build frontend | Vite | Build static assets cho Wails |
| Tests backend | Go test | Unit test validator/importer/repository |
| Tests frontend | svelte-check + TypeScript | Kiểm tra type/component |

Ghi chú cập nhật ngày 2026-05-18: Wails v3 vẫn được tài liệu chính thức gắn nhãn **ALPHA**, nhưng API được mô tả là tương đối ổn định và đã có app chạy production. Vì vậy kế hoạch nên pin version Wails v3 cụ thể trong `go.mod`/tooling, tránh dùng floating `latest` trong CI release.

## 4. Kiến trúc thư mục đề xuất

```text
jlpt-extension/
├── app/                         # Go desktop application
│   ├── cmd/jlpt-desktop/         # main package
│   ├── internal/
│   │   ├── app/                  # Wails bootstrap, lifecycle, DI
│   │   ├── database/             # sqlite open, migrations, transaction helpers
│   │   ├── flashcards/           # import, validation, query, study logic
│   │   ├── settings/             # app settings repository/service
│   │   ├── notifications/        # scheduler + notification adapter
│   │   └── testdata/             # json fixtures
│   ├── migrations/               # SQL migrations embedded by Go
│   ├── go.mod
│   └── wails.json hoặc tương đương cấu hình Wails v3
├── desktop/                      # Svelte 5 frontend cho desktop
│   ├── src/
│   │   ├── lib/api/              # wrapper quanh Wails generated bindings
│   │   ├── lib/components/       # tái sử dụng/chuyển component hiện có
│   │   ├── lib/types/            # TS contract mirror từ Go bindings
│   │   ├── routes/               # Study, Import, Settings, Library
│   │   └── main.ts
│   ├── package.json
│   └── vite.config.ts
├── src/                          # Extension legacy giữ tạm trong giai đoạn chuyển tiếp
├── data/                         # Dataset mẫu hiện có
└── docs/
```

Có thể chọn cách đặt frontend trong `frontend/` theo template Wails, nhưng `desktop/` giúp tránh nhầm với extension `src/` hiện tại.

## 5. Database design SQLite3

### 5.1. Nguyên tắc

- SQLite là source of truth cho desktop app.
- JSON upload chỉ là nguồn import; sau khi import thành công, app đọc từ DB.
- Import phải chạy trong transaction để tránh trạng thái nửa vời.
- Sinh `card_id` ổn định từ nội dung cũ để bảo toàn bookmark/note khi import lại cùng dataset.
- Dùng `import_batches` để audit mỗi lần user nạp file.

### 5.2. Schema SQL MVP

```sql
CREATE TABLE schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
```

### 5.3. Upsert policy khi import

- `source_sha256` giống import trước: cảnh báo user dataset có thể đã import; cho phép skip hoặc import lại.
- `flashcards.id` trùng: `UPSERT` các field content, giữ note/bookmark/study history.
- Card không còn xuất hiện trong file mới: MVP không xóa tự động; thêm tùy chọn `Replace library` ở giai đoạn sau nếu user muốn đồng bộ tuyệt đối.
- Import lỗi: rollback toàn batch, trả danh sách lỗi kèm index dòng JSON.

## 6. JSON upload và import flow

### 6.1. Flow UI

1. User mở màn hình **Import JSON**.
2. User chọn file bằng native file dialog hoặc kéo thả file JSON.
3. Frontend hiển thị preview: filename, size, số item đọc được, lỗi parse cơ bản nếu có.
4. Frontend gọi Go service `ImportFlashcardsFromFile(path, options)` hoặc `ImportFlashcardsFromBytes(name, content, options)`.
5. Go service parse JSON, validate toàn bộ item, tính hash file, insert trong transaction.
6. Frontend hiển thị kết quả: số card inserted/updated/skipped, danh sách lỗi nếu import fail.
7. Nếu thành công, chuyển sang Library/Study view và refresh query từ SQLite.

### 6.2. Validation server-side bắt buộc

Go backend phải validate lại dù frontend đã validate để đảm bảo DB không nhận dữ liệu bẩn:

- Root JSON phải là array.
- Mỗi item phải là object.
- Bắt buộc đủ field: `level`, `category`, `name`, `mean`, `hiragana`, `image`, `audio`, `example`.
- `level` thuộc enum `n5/n4/n3/n2/n1`.
- `category` thuộc enum hiện tại, bao gồm `gramma`.
- `name`, `mean`, `hiragana` là string không rỗng sau trim.
- `image`, `audio`, `example` là string hoặc null.
- Giới hạn file: MVP đề xuất 25 MB; nếu cần lớn hơn thì parse streaming ở milestone sau.

### 6.3. API bindings đề xuất

```go
type ImportOptions struct {
    ReplaceLibrary bool `json:"replaceLibrary"`
    DryRun         bool `json:"dryRun"`
}

type ImportResult struct {
    BatchID     int64             `json:"batchId"`
    TotalRows   int               `json:"totalRows"`
    ValidRows   int               `json:"validRows"`
    InvalidRows int               `json:"invalidRows"`
    Inserted    int               `json:"inserted"`
    Updated     int               `json:"updated"`
    Errors      []ValidationError `json:"errors"`
}

type FlashcardService struct { /* repositories */ }

func (s *FlashcardService) ImportFlashcardsFromFile(path string, options ImportOptions) (ImportResult, error)
func (s *FlashcardService) ListFlashcards(filter FlashcardFilter) ([]FlashcardDTO, error)
func (s *FlashcardService) GetStudyState() (StudyStateDTO, error)
func (s *FlashcardService) MoveNext() (FlashcardDTO, error)
func (s *FlashcardService) MovePrevious() (FlashcardDTO, error)
func (s *FlashcardService) SaveNote(cardID string, note string) error
func (s *FlashcardService) ToggleBookmark(cardID string) (bool, error)
```

## 7. Frontend Svelte 5 plan

### 7.1. Route/view MVP

- **Study**: flashcard chính, next/previous, reveal answer, audio/image/example, note, bookmark.
- **Import**: upload JSON, dry-run validation, import result.
- **Library**: list/search/filter cards từ SQLite.
- **Settings**: theme, language, daily goal, enabled levels/categories, notification interval, order mode.

### 7.2. Migration từ extension UI

- Tách component thuần UI khỏi API chrome-specific.
- Thay import `src/lib/extension/storage.ts` bằng `desktop/src/lib/api/*.ts` gọi Wails bindings.
- Giữ type TS tương thích với Go JSON tags.
- Svelte 5 dùng runes cho state local của view mới; các component legacy có thể migrate dần nếu đang dùng cú pháp Svelte cũ.

### 7.3. State management

- Không lưu dataset lớn trong localStorage.
- Frontend chỉ cache query result ngắn hạn.
- Settings đọc/ghi qua Go service để đồng bộ SQLite.
- Với màn hình Study, state hiện tại lấy từ `study_state` để app resume được sau khi đóng/mở.

## 8. Backend Go plan

### 8.1. Packages chính

- `internal/database`: mở DB trong app data dir, chạy migrations, helper transaction.
- `internal/flashcards`: domain model, validator JSON, repository, import service, study service.
- `internal/settings`: key-value JSON settings repository.
- `internal/notifications`: scheduler và adapter notification.
- `internal/app`: Wails bootstrap, bind services, lifecycle startup/shutdown.

### 8.2. SQLite driver

Có hai hướng:

1. Dùng Wails v3 built-in SQLite service nếu đáp ứng đủ nhu cầu migration/query.
2. Dùng `database/sql` với driver SQLite phổ biến để kiểm soát migration, transaction, test dễ hơn.

Khuyến nghị MVP: bắt đầu bằng repository qua `database/sql` để domain không phụ thuộc chặt vào Wails. Nếu sau spike Wails SQLite service ổn định và tiện hơn, bọc nó sau interface `DBTX` tương tự để không ảnh hưởng service layer.

### 8.3. App data path

DB nên đặt trong app data dir theo OS, không nằm cạnh binary:

- Windows: `%AppData%/JLPT Flashcard/jlpt.db`
- macOS: `~/Library/Application Support/JLPT Flashcard/jlpt.db`
- Linux: `${XDG_DATA_HOME:-~/.local/share}/jlpt-flashcard/jlpt.db`

## 9. Notification/scheduler desktop

### 9.1. MVP

- Scheduler chạy trong process app bằng Go ticker.
- Khi app đang mở và notification enabled, chọn card kế tiếp từ SQLite rồi gọi notification adapter.
- Settings interval và pause/resume lưu trong SQLite.

### 9.2. Sau MVP

- System tray để app chạy nền.
- Autostart tuỳ chọn theo OS.
- Native notification click action mở đúng card trong app.
- Nếu Wails v3 notifications API thay đổi, giữ interface nội bộ:

```go
type Notifier interface {
    ShowFlashcard(ctx context.Context, card FlashcardDTO) error
}
```

## 10. Milestones đề xuất

### Milestone 0 — Chốt nền tảng và spike Wails v3

- Pin Wails v3 version và Go version.
- Tạo app skeleton Wails v3 + Svelte 5 trong repo.
- Chứng minh frontend gọi được Go service đơn giản.
- Chạy build dev trên ít nhất một OS chính của team.

**Exit criteria:** `wails3 dev` chạy được, bindings sinh ra ổn định, CI cài được toolchain.

### Milestone 1 — SQLite foundation

- Thêm Go module, DB open, migrations, schema MVP.
- Thêm repository flashcards/settings/study_state.
- Unit test migrations và repository cơ bản.

**Exit criteria:** `go test ./...` pass, DB tạo được ở temp dir, migration idempotent.

### Milestone 2 — JSON import service

- Port validator sang Go.
- Implement hash file, dry-run, import transaction, upsert.
- Test fixtures hợp lệ/lỗi/trùng card/file lớn gần limit.

**Exit criteria:** Import được `data/n2.json` vào SQLite, lỗi validation trả đúng index/field.

### Milestone 3 — Desktop import UI

- Màn hình chọn/kéo thả JSON.
- Hiển thị validation/import result.
- Refresh Library sau khi import.

**Exit criteria:** User import file JSON thật từ máy và thấy card trong app.

### Milestone 4 — Study UI parity

- Port StudyCard, LevelSelector, settings cơ bản.
- Implement next/previous/random/sequential qua Go service.
- Notes/bookmarks lưu SQLite.

**Exit criteria:** Feature chính của popup extension có mặt trong desktop app.

### Milestone 5 — Notifications desktop

- Implement scheduler trong Go.
- Notification settings/pause/resume.
- Notification content format tương đương extension.

**Exit criteria:** App đang mở có thể phát notification định kỳ dựa trên data SQLite.

### Milestone 6 — Packaging và release

- Build binary cho OS mục tiêu.
- Icon/app metadata.
- Smoke test install/run/import.
- Viết hướng dẫn user migration từ extension: export JSON hoặc import lại dataset gốc.

**Exit criteria:** Có artifact release desktop app và hướng dẫn cài đặt.

## 11. Chiến lược PR từ nhánh feature

Để giảm rủi ro, chia PR theo milestone nhưng đều từ nhánh feature hiện tại hoặc các commit tuần tự trên nhánh này:

1. **PR 1: Desktop migration plan** — tài liệu này, không đổi runtime.
2. **PR 2: Wails/Svelte skeleton** — thêm app chạy hello service.
3. **PR 3: SQLite schema + repositories** — backend DB foundation.
4. **PR 4: JSON import backend** — import service + tests.
5. **PR 5: Import UI** — Svelte desktop import/library.
6. **PR 6: Study UI parity** — học flashcard, settings, note/bookmark.
7. **PR 7: Notifications + packaging** — scheduler, native packaging, docs.

Quy tắc merge:

- Không squash mất lịch sử nếu team cần audit từng milestone; nếu squash thì body PR phải liệt kê đầy đủ migration step.
- Không merge PR sau nếu PR trước chưa pass build/check tối thiểu.
- Nếu cần sửa hotfix extension legacy trong lúc migration, commit hotfix trên nhánh feature rồi mở PR riêng từ chính nhánh này hoặc rebase nhánh feature sau khi hotfix vào main.

## 12. Testing và CI checklist

### 12.1. Backend

- `go test ./...`
- Test validator với missing field, enum sai, nullable field sai, JSON root không phải array.
- Test import transaction rollback khi có lỗi.
- Test idempotent import cùng file.
- Test upsert giữ `card_notes` và `card_bookmarks`.

### 12.2. Frontend

- `npm run check` cho desktop frontend.
- Component smoke test nếu thêm test runner sau.
- Manual test import file thật, file lỗi, file rỗng.

### 12.3. End-to-end manual smoke

- Launch app.
- Import `data/n2.json`.
- Search/filter card.
- Next/previous/random/sequential.
- Add note, bookmark, restart app, kiểm tra dữ liệu vẫn còn.
- Bật notification interval ngắn, xác nhận notification hiển thị.

## 13. Rủi ro và phương án giảm thiểu

| Rủi ro | Tác động | Giảm thiểu |
| --- | --- | --- |
| Wails v3 còn alpha | API/tooling có thể đổi | Pin version, bọc Wails API sau service nội bộ, spike sớm |
| SQLite driver khác nhau giữa OS | Build/package lỗi | Chọn driver sớm, setup CI matrix, tránh logic phụ thuộc OS |
| JSON user upload lớn | UI đơ hoặc memory cao | Giới hạn MVP 25 MB, milestone sau parse streaming |
| Category `gramma` sai chính tả | Dữ liệu legacy break nếu đổi | Giữ enum cũ, thêm display label `Ngữ pháp` |
| Notification background khi app đóng | User kỳ vọng như extension | MVP chỉ khi app chạy; sau đó thêm tray/autostart |
| Nhánh feature kéo dài | Conflict với main | Rebase/merge main định kỳ vào feature, PR nhỏ theo milestone |

## 14. Definition of Done cho migration MVP

- Desktop app build được bằng Wails v3 và frontend Svelte 5.
- User import được JSON theo schema hiện tại vào SQLite3.
- Card hiển thị từ SQLite, không còn đọc dataset chính từ `chrome.storage.local` trong desktop path.
- Notes/bookmarks/settings/study progress persist sau restart.
- Có test Go cho validator/import/repository.
- Có hướng dẫn chạy dev/build desktop trong README hoặc docs.
- PR cuối cùng từ nhánh feature được merge sau khi các check pass.
