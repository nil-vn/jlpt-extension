# Kế hoạch phát triển Chrome Extension học JLPT bằng Flashcard

## 1. Mục tiêu sản phẩm

Extension giúp người học JLPT ôn tập bằng flashcard ngay trong Chrome, kết hợp hai luồng sử dụng chính:

1. **GUI chính khi bấm icon extension**: xem, điều hướng, nghe audio, ghi chú, đánh dấu xem lại từng card.
2. **Notification định kỳ trên Windows/Chrome**: hiển thị nội dung flashcard theo khoảng thời gian người dùng cấu hình.

Sản phẩm ưu tiên hiệu năng, trải nghiệm gọn nhẹ, chạy offline sau khi đã có dataset, và cho phép người dùng tự nạp bộ dữ liệu JLPT dạng JSON.

## 2. Tech stack đề xuất

| Hạng mục | Công nghệ | Vai trò |
| --- | --- | --- |
| UI framework | Svelte 5 | Xây dựng popup/options UI dạng static assets để nhúng vào extension. |
| Component system | shadcn-svelte | Tạo UI nhất quán cho card, button, dialog, tabs, switch, select, textarea. |
| Icon | lucide-svelte | Icon cho audio, next/previous, bookmark, settings, pause/resume notification. |
| Build tool | Vite | Build Svelte thành file tĩnh cho Chrome Extension. |
| Extension platform | Chrome Extension Manifest V3 | Popup, options page, service worker, notifications API, storage API. |
| Validation | Zod hoặc validator tự viết | Kiểm tra dataset JSON do người dùng upload. |
| State/storage | chrome.storage.local | Lưu settings, dataset, current index, bookmark, notes, trạng thái pause notification. |

## 3. Phạm vi tính năng MVP

### 3.1. Dataset

Dataset là một file JSON gồm danh sách flashcard. Mỗi item cần đủ field:

```ts
type JlptLevel = "n5" | "n4" | "n3" | "n2" | "n1";
type JlptCategory = "gramma" | "vocabulary" | "kanji" | "reading" | "listening";

type Flashcard = {
  level: JlptLevel;
  category: JlptCategory;
  name: string;
  mean: string;
  hiragana: string;
  image: string | null;
  audio: string | null;
  example: string | null;
};
```

Ghi chú: enum `gramma` và `vocabulary` được giữ đúng theo yêu cầu dataset đầu vào để tương thích. Trong UI nên map nhãn hiển thị thành `Ngữ pháp` và `Từ vựng`.

### 3.2. Màn hình setting/options

Người dùng có thể cấu hình:

- Upload dataset JSON từ máy.
- Validate dataset trước khi lưu.
- Xem số lượng card hợp lệ sau khi load.
- Chọn theme sáng/tối/system.
- Chọn thời gian giữa các notification.
- Chọn thời lượng hiển thị theo khả năng của Chrome Notifications API; notification bắt đầu theo interval khi user enable hoặc extension active.
- Chọn chế độ học:
  - Random.
  - Tuần tự theo thứ tự JSON.
- Reset dataset/settings nếu cần.

### 3.3. GUI chính trong popup

Popup hiển thị flashcard hiện tại:

- Level và category.
- Name, hiragana, meaning.
- Image nếu có.
- Example nếu có.
- Button nghe audio nếu có.
- Button previous/next.
- Button bookmark/đánh dấu xem lại.
- Textarea ghi chú cho card hiện tại.
- Button pause/resume notification.
- Link hoặc button mở settings.

### 3.4. Notification định kỳ

Sau khi user đã cấu hình và load data:

- Service worker dùng `chrome.alarms` để chạy định kỳ.
- Khi alarm kích hoạt, extension chọn card theo mode random/tuần tự.
- Dùng `chrome.notifications.create` để show notification.
- Nội dung notification nên gồm:
  - Line 1/title: `[N5 - Từ vựng]  食べる (たべる)`.
  - Line 2/message: `[THỰC] Ăn`.
  - Line 3 nếu còn đủ chỗ: example nếu có.
- Khi user pause notification, service worker không tạo notification mới.

## 4. Kiến trúc Chrome Extension

```text
jlpt-extension/
├── public/
│   ├── manifest.json
│   └── icons/
├── src/
│   ├── app.css
│   ├── lib/
│   │   ├── components/
│   │   ├── data/
│   │   │   ├── dataset-validator.ts
│   │   │   └── category-labels.ts
│   │   ├── extension/
│   │   │   ├── storage.ts
│   │   │   ├── alarms.ts
│   │   │   └── notifications.ts
│   │   └── types/
│   │       └── flashcard.ts
│   ├── popup/
│   │   ├── Popup.svelte
│   │   └── main.ts
│   ├── options/
│   │   ├── Options.svelte
│   │   └── main.ts
│   └── background/
│       └── service-worker.ts
├── vite.config.ts
├── svelte.config.js
└── package.json
```

### 4.1. Popup

Popup là entry Svelte riêng, build thành `popup.html` và bundle JS/CSS tĩnh. Popup chỉ đọc/ghi state qua `chrome.storage.local`, không tự chạy interval dài hạn vì popup sẽ bị unload khi đóng.

### 4.2. Options page

Options page là entry Svelte riêng, build thành `options.html`. Đây là nơi upload dataset, validate dữ liệu và lưu cấu hình.

### 4.3. Background service worker

Service worker chịu trách nhiệm:

- Lắng nghe `chrome.runtime.onInstalled` để khởi tạo default settings.
- Lắng nghe thay đổi settings/dataset.
- Tạo/cập nhật `chrome.alarms` khi interval thay đổi.
- Lắng nghe `chrome.alarms.onAlarm` để tạo notification.
- Lưu current index tiếp theo nếu học tuần tự.

## 5. Manifest V3 đề xuất

```json
{
  "manifest_version": 3,
  "name": "JLPT Flashcard",
  "version": "0.1.0",
  "description": "Learn JLPT with flashcards and periodic notifications.",
  "action": {
    "default_popup": "popup.html",
    "default_title": "JLPT Flashcard"
  },
  "options_page": "options.html",
  "background": {
    "service_worker": "service-worker.js",
    "type": "module"
  },
  "permissions": ["storage", "alarms", "notifications"],
  "host_permissions": []
}
```

## 6. Thiết kế storage

Không nên ghi note vào file `.txt` trực tiếp vì Chrome Extension không có quyền ghi file tùy ý vào filesystem của người dùng nếu không thông qua File System Access API hoặc download flow. Best practice cho MVP là dùng `chrome.storage.local`.

```ts
type UserSettings = {
  theme: "light" | "dark" | "system";
  notificationIntervalMinutes: number;
  notificationEnabled: boolean;
  orderMode: "random" | "sequential";
};

type ExtensionState = {
  dataset: Flashcard[];
  currentIndex: number;
  bookmarkedCardIds: string[];
  notesByCardId: Record<string, string>;
  settings: UserSettings;
};
```

Card ID nên được sinh ổn định từ nội dung card, ví dụ hash của `level|category|name|hiragana|mean`, vì dataset hiện chưa có field `id`.

## 7. Luồng xử lý chính

### 7.1. Load dataset

1. User mở options page.
2. User chọn file JSON.
3. App parse JSON.
4. App kiểm tra dữ liệu là array.
5. App validate từng item có đủ field, đúng enum và đúng nullable string.
6. Nếu hợp lệ, lưu dataset vào `chrome.storage.local`.
7. Reset `currentIndex` về `0` nếu user chọn thay dataset.
8. Service worker nhận thay đổi và setup alarm nếu notification đang bật.

### 7.2. Điều hướng card trong popup

- Next:
  - Sequential: tăng index, quay vòng về đầu nếu hết dataset.
  - Random: chọn index ngẫu nhiên khác index hiện tại nếu có nhiều hơn một card.
- Previous:
  - Với sequential: giảm index, quay vòng về cuối nếu đang ở đầu.
  - Với random: MVP có thể lưu `historyStack` để quay lại card trước đó; nếu chưa có history thì random lại hoặc disable nút previous.

### 7.3. Notification

1. Alarm được kích hoạt theo interval.
2. Service worker đọc settings, dataset, currentIndex.
3. Nếu notification bị pause hoặc dataset rỗng thì dừng.
4. Chọn card kế tiếp theo mode.
5. Tạo notification.
6. Cập nhật currentIndex/history nếu cần.

## 8. UI/UX đề xuất

### 8.1. Popup layout

- Header:
  - App name.
  - Badge theme/status notification.
  - Settings icon.
- Card area:
  - Badge level.
  - Badge category bằng tiếng Việt.
  - Tên card lớn.
  - Hiragana nhỏ hơn.
  - Meaning nổi bật.
  - Image nếu có.
  - Example trong blockquote/card phụ.
- Actions:
  - Previous.
  - Play audio.
  - Next.
  - Bookmark.
  - Pause/resume notifications.
- Notes:
  - Textarea autosave debounce 300-500ms.

### 8.2. Options layout

- Section Dataset:
  - File input.
  - Validation result.
  - Dataset count.
- Section Study mode:
  - Radio random/sequential.
- Section Notifications:
  - Switch enable/disable.
  - Input interval minutes.
- Section Appearance:
  - Theme selector.

## 9. Rủi ro kỹ thuật và hướng xử lý

| Rủi ro | Hướng xử lý |
| --- | --- |
| Chrome service worker có thể bị unload | Dùng `chrome.alarms` thay vì `setInterval`. |
| Không ghi được file txt trực tiếp | Lưu notes vào `chrome.storage.local`; thêm export notes ở phase sau nếu cần. |
| Dataset có ảnh/audio local path | MVP khuyến nghị dùng URL hoặc data URL; phase sau hỗ trợ import asset dạng packaged/exported bundle. |
| Notification không đảm bảo hiển thị đúng số giây trên mọi OS | Cấu hình interval tạo notification; thời lượng hiển thị thực tế phụ thuộc Chrome/OS. |
| Storage quota | Nếu dataset lớn hoặc có base64 image/audio, cần chuyển sang IndexedDB. MVP dùng `chrome.storage.local`, phase sau đánh giá IndexedDB. |

## 10. Roadmap triển khai

### Phase 1: Scaffold project

- Tạo Svelte 5 + Vite project.
- Cài shadcn-svelte, lucide-svelte.
- Thiết lập multi-entry build cho popup/options/service worker.
- Tạo manifest MV3.

### Phase 2: Data model và storage

- Định nghĩa type `Flashcard`, `UserSettings`, `ExtensionState`.
- Viết validator dataset.
- Viết wrapper cho `chrome.storage.local`.
- Viết helper sinh card ID.

### Phase 3: Options page

- Upload và validate JSON.
- Lưu dataset/settings.
- Hiển thị trạng thái dataset.
- Cho phép đổi theme, interval, notification enabled, order mode.

### Phase 4: Popup UI

- Hiển thị card hiện tại.
- Điều hướng previous/next.
- Play audio nếu có.
- Bookmark card.
- Ghi chú autosave.
- Pause/resume notification.

### Phase 5: Background notifications

- Tạo service worker.
- Setup alarm theo settings.
- Chọn card theo mode.
- Tạo notification.
- Sync current index.

### Phase 6: Polish và kiểm thử

- Empty state khi chưa có dataset.
- Error state khi dataset invalid.
- Dark/light mode.
- Responsive popup size.
- Manual test trên Chrome bằng `chrome://extensions`.
- Build production và kiểm tra static assets.

## 11. Tiêu chí hoàn thành MVP

- Extension build được thành thư mục static để load unpacked trên Chrome.
- User upload được dataset JSON hợp lệ.
- Popup hiển thị đúng flashcard và điều hướng được.
- User ghi chú và bookmark được từng card.
- User bật/tắt notification được.
- Notification định kỳ hiển thị card theo random/sequential mode.
- Settings vẫn tồn tại sau khi đóng/mở Chrome.

## 12. Câu hỏi cần chốt trước khi code sâu

1. Dataset ban đầu có cần bundle sẵn trong extension không, hay chỉ user tự upload?
2. Image/audio sẽ là URL online, base64, hay file local đi kèm dataset?
3. Có cần filter theo level/category trong MVP không?
4. Có cần import/export notes và bookmarks không?
5. Có muốn thêm thống kê học tập như số lần xem, số lần bookmark, hoặc trạng thái đã thuộc/chưa thuộc không?
