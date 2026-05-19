1. Install dependences: `npm install`
2. Build: `npm run build`
3. Load unpacked from `dist`
4. Open config page
5. Load data from `/data`
6. Customize: make json data, for example:
```json
[{
      "level": "n2",
      "category": "vocabulary",
      "name": "一家",
      "mean": "Một nhà, cả nhà, cả gia đình",
      "hiragana": "いっか",
      "image": null,
      "audio": null,
      "example": "兄（あに）が私（わたし）達（たち）一家（いっか）を支（ささ）えてくれている。"
  }, ...]
  ```

## Desktop migration

Migration planning for the Golang + Wails v3 + Svelte 5 desktop app is documented in [`docs/desktop-migration-plan.md`](docs/desktop-migration-plan.md).

Phase 1 Wails v3 + Svelte 5 spike notes are documented in [`docs/desktop-phase-1.md`](docs/desktop-phase-1.md).

Desktop release and extension migration instructions are documented in [`docs/desktop-release.md`](docs/desktop-release.md).
