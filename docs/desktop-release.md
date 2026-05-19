# JLPT Flashcard Desktop — release, cài đặt và migration từ extension

Tài liệu này là hướng dẫn Milestone 6 cho desktop app Golang + Wails v3 + Svelte 5. Mục tiêu release hiện tại là Linux `amd64` theo môi trường build chính; Windows/macOS dùng cùng metadata/icon và cần build trên runner tương ứng hoặc Docker cross image của Wails.

## 1. Artifact release

Artifact chuẩn của Milestone 6 là tarball:

```text
app/bin/release/jlpt-desktop_<version>_linux_amd64.tar.gz
app/bin/release/jlpt-desktop_<version>_linux_amd64.tar.gz.sha256
```

Nội dung tarball gồm:

- `jlpt-desktop`: binary desktop đã bundle frontend Svelte.
- `jlpt-desktop.desktop`: Linux launcher metadata.
- `jlpt-desktop.png`: app icon.
- `install-linux-user.sh`: script cài vào `$HOME/.local` hoặc `PREFIX` tùy chọn.
- `INSTALL_AND_MIGRATION.md`: bản copy của tài liệu này để user đọc offline.

## 2. Build release artifact

Yêu cầu build Linux:

- Go đúng version trong `app/go.mod`.
- Node/npm theo `app/frontend/package-lock.json`.
- Wails CLI pinned: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.93`.
- GTK/WebKitGTK dev packages cho Linux, ví dụ Ubuntu 24.04: `libgtk-4-dev libwebkitgtk-6.0-dev pkg-config`.

Chạy từ repo root:

```bash
cd app
./scripts/package-release.sh
```

Hoặc dùng Wails task alias:

```bash
cd app
wails3 task release:linux:tar
```

Script sẽ build binary production, chạy smoke test import JSON cấp service, sau đó đóng gói tarball và checksum SHA-256.

## 3. Cài đặt trên Linux

Tải hoặc copy tarball release, sau đó:

```bash
tar -xzf jlpt-desktop_0.1.0_linux_amd64.tar.gz
cd jlpt-desktop_0.1.0_linux_amd64
./install-linux-user.sh
```

Mặc định script cài vào:

- Binary: `$HOME/.local/bin/jlpt-desktop`
- Launcher: `$HOME/.local/share/applications/jlpt-desktop.desktop`
- Icon: `$HOME/.local/share/icons/hicolor/1024x1024/apps/jlpt-desktop.png`

Nếu muốn cài vào prefix khác:

```bash
PREFIX=/opt/jlpt-desktop ./install-linux-user.sh
```

Sau khi cài, mở app bằng menu application của desktop environment hoặc chạy:

```bash
$HOME/.local/bin/jlpt-desktop
```

Database SQLite của app được tạo trong app data directory theo OS, không nằm cạnh binary.

## 4. Smoke test release

Chạy smoke test tarball từ repo root:

```bash
cd app
./scripts/smoke-release.sh bin/release/jlpt-desktop_0.1.0_linux_amd64.tar.gz
```

Smoke test thực hiện các bước:

1. Giải nén artifact vào thư mục tạm.
2. Cài user-local vào prefix tạm bằng `install-linux-user.sh`.
3. Kiểm tra binary, `.desktop`, icon và dynamic links bằng `ldd`.
4. Chạy `TestReleaseSmokeImportRunAndStudy` để preview/import JSON mẫu vào SQLite, list library, và tạo study state.

Vì CI/container thường không có graphical session, bước launch GUI thủ công nên được chạy trên máy Linux có desktop session:

```bash
$HOME/.local/bin/jlpt-desktop
```

Sau khi app mở, vào màn hình **Import JSON**, import một file JSON thật, rồi xác nhận card xuất hiện ở Library/Study.

## 5. Migration từ Chrome Extension sang Desktop

Desktop app không đọc trực tiếp `chrome.storage.local`. User cần đưa dữ liệu sang desktop theo một trong hai cách dưới đây.

### Cách A — Export dataset JSON từ extension hiện tại

Dùng cách này nếu bạn đã import/custom dataset trong extension và muốn giữ chính dataset đó.

1. Mở trang extension/options của JLPT Flashcard extension trong Chrome.
2. Mở DevTools cho trang extension/options.
3. Chạy snippet sau trong Console để tải dataset đang lưu trong `chrome.storage.local`:

```js
chrome.storage.local.get(['jlptExtensionState', 'jlptDataset', 'dataset'], (stored) => {
  const dataset = stored.jlptExtensionState?.dataset ?? stored.jlptDataset ?? stored.dataset ?? [];
  const blob = new Blob([JSON.stringify(dataset, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = 'jlpt-extension-dataset-export.json';
  link.click();
  URL.revokeObjectURL(url);
});
```

4. Mở desktop app, vào **Import JSON**, chọn `jlpt-extension-dataset-export.json`, preview validation, rồi bấm import.

Lưu ý: Milestone 6 import dataset/card content. Notes/bookmarks/settings trong extension chưa có migration tự động sang SQLite; chúng sẽ được tạo lại trong desktop app sau khi import.

### Cách B — Import lại dataset gốc

Dùng cách này nếu bạn vẫn còn file JSON gốc hoặc chỉ dùng dataset mẫu của extension.

1. Tìm file JSON gốc mà bạn đã từng import vào extension, hoặc lấy dataset mẫu từ repo/nguồn release.
2. Mở desktop app → **Import JSON**.
3. Chọn file JSON, đọc preview lỗi nếu có, rồi import.

Schema JSON cần là array các object có đủ field:

```json
[
  {
    "level": "n2",
    "category": "vocabulary",
    "name": "一家",
    "mean": "Một nhà, cả nhà, cả gia đình",
    "hiragana": "いっか",
    "image": null,
    "audio": null,
    "example": "兄が私達一家を支えてくれている。"
  }
]
```

Category `gramma` vẫn được giữ để tương thích dữ liệu extension cũ.

## 6. Checklist release thủ công

- Build artifact bằng `./scripts/package-release.sh`.
- Verify checksum: `sha256sum -c app/bin/release/*.sha256`.
- Smoke test artifact bằng `./scripts/smoke-release.sh <artifact>`.
- Cài trên máy Linux thật và launch GUI.
- Import file export từ extension hoặc dataset gốc.
- Upload tarball + `.sha256` vào GitHub Release/kanban release storage.
