package flashcards_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nil-vn/jlpt-extension/app/internal/database"
	"github.com/nil-vn/jlpt-extension/app/internal/flashcards"
)

func TestImportServiceImportsValidFixtureAndDryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	service := flashcards.NewImportService(db)
	content := readFixture(t, "valid.json")
	dryRun, err := service.ImportFlashcardsFromBytes(ctx, "valid.json", content, flashcards.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run import error = %v", err)
	}
	if dryRun.TotalRows != 2 || dryRun.ValidRows != 2 || dryRun.InvalidRows != 0 || dryRun.Inserted != 2 || dryRun.Updated != 0 || dryRun.BatchID != 0 {
		t.Fatalf("dry-run result = %+v, want 2 valid rows, 2 inserts preview, no batch", dryRun)
	}
	assertCount(t, db, "flashcards", 0)
	assertCount(t, db, "import_batches", 0)

	result, err := service.ImportFlashcardsFromBytes(ctx, "valid.json", content, flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("import error = %v", err)
	}
	wantHash := sha256.Sum256(content)
	if result.SourceSHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("SourceSHA256 = %q, want content hash", result.SourceSHA256)
	}
	if result.BatchID == 0 || result.Inserted != 2 || result.Updated != 0 || len(result.Errors) != 0 {
		t.Fatalf("import result = %+v, want 2 inserted and no errors", result)
	}
	assertCount(t, db, "flashcards", 2)
	assertCount(t, db, "import_batches", 1)
}

func TestImportServiceValidationErrorsIncludeIndexAndFieldAndRollback(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	service := flashcards.NewImportService(db)
	result, err := service.ImportFlashcardsFromBytes(ctx, "invalid.json", readFixture(t, "invalid.json"), flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("import returned unexpected Go error = %v", err)
	}
	if result.TotalRows != 2 || result.ValidRows != 0 || result.InvalidRows != 2 {
		t.Fatalf("result counts = %+v, want all rows invalid", result)
	}
	assertValidationError(t, result.Errors, 0, "level")
	assertValidationError(t, result.Errors, 1, "mean")
	assertValidationError(t, result.Errors, 1, "image")
	assertCount(t, db, "flashcards", 0)
	assertCount(t, db, "import_batches", 0)
}

func TestImportServiceRejectsDuplicateCardsByStableID(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	service := flashcards.NewImportService(db)
	result, err := service.ImportFlashcardsFromBytes(ctx, "duplicate.json", readFixture(t, "duplicate.json"), flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("import returned unexpected Go error = %v", err)
	}
	if result.ValidRows != 1 || result.InvalidRows != 1 {
		t.Fatalf("result = %+v, want one valid row and one duplicate error", result)
	}
	assertValidationError(t, result.Errors, 1, "name")
	assertCount(t, db, "flashcards", 0)
	assertCount(t, db, "import_batches", 0)
}

func TestImportServiceUpsertsExistingCards(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()
	service := flashcards.NewImportService(db)

	content := readFixture(t, "valid.json")
	first, err := service.ImportFlashcardsFromBytes(ctx, "valid.json", content, flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("first import error = %v", err)
	}
	secondContent := bytes.Replace(content, []byte("猫がいます。"), []byte("猫が寝ています。"), 1)
	second, err := service.ImportFlashcardsFromBytes(ctx, "valid-v2.json", secondContent, flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("second import error = %v", err)
	}
	if first.Inserted != 2 || second.Inserted != 0 || second.Updated != 2 {
		t.Fatalf("first = %+v second = %+v, want second import to update existing stable IDs", first, second)
	}
	assertCount(t, db, "flashcards", 2)
	assertCount(t, db, "import_batches", 2)
}

func TestImportServiceImportsRepositoryN2Dataset(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()

	path := filepath.Join("..", "..", "..", "data", "n2.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	result, err := flashcards.NewImportService(db).ImportFlashcardsFromFile(ctx, path, flashcards.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportFlashcardsFromFile(data/n2.json) error = %v", err)
	}
	if result.TotalRows != 1546 || result.ValidRows != 1546 || result.Inserted != 1546 || len(result.Errors) != 0 {
		t.Fatalf("n2 import result = %+v, want all rows imported", result)
	}
	wantHash := sha256.Sum256(content)
	if result.SourceSHA256 != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("n2 SourceSHA256 = %q, want file hash", result.SourceSHA256)
	}
	assertCount(t, db, "flashcards", 1546)
}

func TestImportServiceAcceptsLargeFileNearLimitAndRejectsOverLimit(t *testing.T) {
	ctx := context.Background()
	db, _, err := database.OpenInDir(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("OpenInDir() error = %v", err)
	}
	defer db.Close()
	service := flashcards.NewImportService(db)

	nearLimit := buildLargeImportJSON(t, flashcards.MaxImportFileSizeBytes-1024)
	result, err := service.ImportFlashcardsFromBytes(ctx, "near-limit.json", nearLimit, flashcards.ImportOptions{DryRun: true})
	if err != nil {
		t.Fatalf("near-limit dry-run import error = %v", err)
	}
	if result.TotalRows != 1 || result.ValidRows != 1 || result.Inserted != 1 {
		t.Fatalf("near-limit result = %+v, want one valid insert preview", result)
	}

	overLimit := append(nearLimit, bytes.Repeat([]byte(" "), 2048)...)
	if _, err := service.ImportFlashcardsFromBytes(ctx, "over-limit.json", overLimit, flashcards.ImportOptions{DryRun: true}); err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("over-limit error = %v, want size limit error", err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return content
}

func assertValidationError(t *testing.T, errors []flashcards.ValidationError, index int, field string) {
	t.Helper()
	for _, validationError := range errors {
		if validationError.Index != nil && *validationError.Index == index && validationError.Field == field {
			return
		}
	}
	t.Fatalf("errors = %+v, want index %d field %q", errors, index, field)
}

func assertCount(t *testing.T, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, table string, want int) {
	t.Helper()
	var got int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := db.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}

func buildLargeImportJSON(t *testing.T, targetSize int64) []byte {
	t.Helper()
	base := `[{"level":"n5","category":"vocabulary","name":"large","mean":"large","hiragana":"large","image":null,"audio":null,"example":"%s"}]`
	overhead := len(fmt.Sprintf(base, ""))
	payloadSize := int(targetSize) - overhead
	if payloadSize <= 0 {
		t.Fatalf("target size %d too small for fixture overhead %d", targetSize, overhead)
	}
	return []byte(fmt.Sprintf(base, strings.Repeat("あ", payloadSize/3)))
}
