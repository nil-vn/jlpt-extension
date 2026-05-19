# Audit migration Chrome Extension -> Desktop (Wails 3 + Svelte 5)

Ngày audit: 2026-05-19 (UTC)

## Kết luận nhanh

Migration **chưa hoàn tất 100%**. Nền tảng desktop core đã chạy được (Go service, SQLite schema, import JSON, study state, notes/bookmarks, scheduler notification), nhưng vẫn còn một số hạng mục trong kế hoạch migration chưa implement hoặc mới implement ở mức MVP.

## Những phần đã hoàn thành

- Desktop app khởi tạo bằng Wails 3, bind `AppService`, start scheduler, emit event notification vào frontend.
- SQLite schema MVP có `flashcards`, `import_batches`, `study_state`, `card_notes`, `card_bookmarks`, `app_settings`.
- Import JSON có validate server-side, tính SHA256, dry-run preview, upsert, transaction.
- Study flow cơ bản có next/previous, filter theo level/category (qua settings), random/sequential, reveal answer.
- Note và bookmark đã lưu/persist qua SQLite.
- Notification scheduler chạy trong Go, có trigger thủ công `ShowStudyNotificationNow`.
- Frontend desktop có các khu vực Import + Study + Library + Settings (single-page).

## Khoảng trống / tồn đọng so với kế hoạch migration

## 1) Replace library chưa được implement thực tế

- `ImportOptions` có `ReplaceLibrary`, frontend vẫn truyền `replaceLibrary: false` cố định.
- Backend import hiện không có nhánh xử lý xóa/thay toàn bộ thư viện khi `ReplaceLibrary = true`.

**Task đề xuất**
- [ ] Implement `ReplaceLibrary` transactional mode: xóa cards không còn trong dataset mới hoặc truncate + reinsert tùy policy.
- [ ] Thêm UI toggle `Replace library` trong Import section.
- [ ] Bổ sung test đảm bảo notes/bookmarks giữ hoặc reset đúng theo policy đã chọn.

## 2) Chưa có import trực tiếp từ đường dẫn file native (Wails file dialog flow)

- Kế hoạch có API `ImportFlashcardsFromFile(path, options)` ở AppService cho luồng native desktop.
- Hiện UI đang đọc file qua `<input type="file">` / drag-drop và gửi content string sang Go (`ImportFlashcardsFromJSON`).
- `ImportFlashcardsFromFile` mới nằm ở internal service, chưa expose thành desktop UX native end-to-end.

**Task đề xuất**
- [ ] Expose API AppService: `PickAndImportFlashcardsFile()` hoặc `ImportFlashcardsFromFile(path, options)`.
- [ ] Tích hợp Wails file dialog native + validate dung lượng trước khi đọc.
- [ ] Giữ fallback drag-drop hiện tại cho UX nhanh.

## 3) Chưa tách route/view rõ ràng theo kiến trúc mục tiêu

- Kế hoạch đề xuất desktop shell nhiều route (`Study`, `Import`, `Settings`, `Library`).
- Hiện đang là single `App.svelte` lớn, các section chung một màn.

**Task đề xuất**
- [ ] Refactor sang router-based views (hoặc module sections tách file).
- [ ] Tách state management khỏi `App.svelte` sang store/service layer để dễ test.

## 4) Notification native click/deep-link chưa thấy implement

- Kế hoạch yêu cầu notification click mở đúng card trong app.
- Hiện notifier đang emit event `flashcard-notification`; chưa thấy flow bắt click từ OS notification để focus window và open card cụ thể.

**Task đề xuất**
- [ ] Bổ sung notification action/click handler cross-platform.
- [ ] Map payload `card.id` -> điều hướng UI đến card tương ứng.
- [ ] Thêm test integration/manual checklist theo OS.

## 5) Import duplicate by `source_sha256` chưa có UX xử lý

- Kế hoạch có cảnh báo dataset đã import khi hash trùng và cho phép skip/reimport.
- Hiện import có lưu hash nhưng chưa có check/decision flow theo hash.

**Task đề xuất**
- [ ] Thêm query kiểm tra hash đã tồn tại trong `import_batches`.
- [ ] Trả về trạng thái `duplicate_source` để UI hiển thị confirm trước khi import tiếp.

## 6) Migration dữ liệu từ extension cũ sang desktop chưa có tool

- Tài liệu release nêu rõ notes/bookmarks/settings của extension chưa auto migrate.
- Hiện chưa có command/importer chuyển state từ `chrome.storage.local` dump -> SQLite.

**Task đề xuất**
- [ ] Viết tool import state extension (JSON export) vào SQLite (`app_settings`, `card_notes`, `card_bookmarks`, study prefs).
- [ ] Mapping card ID phải dùng cùng stable hash để giữ liên kết chính xác.
- [ ] Thêm docs migration cho user hiện hữu.

## 7) Test coverage chưa bao phủ một số luồng migration quan trọng

- Đã có unit tests cho import/repository/settings/notifications, nhưng chưa thấy test cho:
  - replace library semantics,
  - duplicate hash behavior,
  - notification click action,
  - E2E import->study->restart flow ở desktop runtime.

**Task đề xuất**
- [ ] Bổ sung integration tests cho import policies.
- [ ] Bổ sung smoke test desktop workflow (CI headless + assertions DB).

## Backlog ưu tiên đề xuất

### P0 (chặn “migration done”)
1. Implement `ReplaceLibrary` + tests.
2. Native file-dialog import flow + AppService API path-based.
3. Duplicate hash detection + UI confirm flow.

### P1
4. Refactor route/module hóa frontend desktop.
5. Notification click -> open card/deep-link.

### P2
6. Extension state migration tool + user docs.
7. Mở rộng automated E2E/smoke coverage.

## Định nghĩa “Done migration” đề xuất

Có thể coi migration hoàn thành khi đạt đủ:
- Không phụ thuộc runtime Chrome APIs cho desktop core.
- Import JSON hỗ trợ cả bytes và native file path, có `ReplaceLibrary`, duplicate-hash policy rõ ràng.
- Study + note + bookmark + settings + notification hoạt động ổn định sau restart.
- Có đường migration dữ liệu cho user extension hiện hữu (ít nhất semi-automatic).
- Có checklist test/release đủ để ship desktop bản stable.
