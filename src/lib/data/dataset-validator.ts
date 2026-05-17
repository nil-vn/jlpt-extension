import type { Flashcard, FlashcardCategory, JlptLevel } from '../types/flashcard';

export type DatasetValidationError = {
  index?: number;
  field?: keyof Flashcard;
  message: string;
};

export type DatasetValidationResult = {
  validCards: Flashcard[];
  errors: DatasetValidationError[];
};

const requiredFields = ['level', 'category', 'name', 'mean', 'hiragana', 'image', 'audio', 'example'] as const;
const nullableStringFields = ['image', 'audio', 'example'] as const;
const stringFields = ['name', 'mean', 'hiragana'] as const;
const validLevels: JlptLevel[] = ['n5', 'n4', 'n3', 'n2', 'n1'];
const validCategories: FlashcardCategory[] = ['gramma', 'locabulary', 'kanji', 'reading', 'listening'];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function hasOwnField(value: Record<string, unknown>, field: (typeof requiredFields)[number]) {
  return Object.prototype.hasOwnProperty.call(value, field);
}

function isJlptLevel(value: unknown): value is JlptLevel {
  return typeof value === 'string' && validLevels.includes(value as JlptLevel);
}

function isFlashcardCategory(value: unknown): value is FlashcardCategory {
  return typeof value === 'string' && validCategories.includes(value as FlashcardCategory);
}

function isNullableString(value: unknown): value is string | null {
  return value === null || typeof value === 'string';
}

function formatItem(index: number) {
  return `Card #${index + 1}`;
}

export function validateDataset(input: unknown): DatasetValidationResult {
  if (!Array.isArray(input)) {
    return {
      validCards: [],
      errors: [{ message: 'JSON phải là một array các flashcard.' }]
    };
  }

  const validCards: Flashcard[] = [];
  const errors: DatasetValidationError[] = [];

  input.forEach((item, index) => {
    if (!isRecord(item)) {
      errors.push({ index, message: `${formatItem(index)} phải là object.` });
      return;
    }

    const itemErrors: DatasetValidationError[] = [];

    requiredFields.forEach((field) => {
      if (!hasOwnField(item, field)) {
        itemErrors.push({ index, field, message: `${formatItem(index)} thiếu field "${field}".` });
      }
    });

    if (hasOwnField(item, 'level') && !isJlptLevel(item.level)) {
      itemErrors.push({
        index,
        field: 'level',
        message: `${formatItem(index)} field "level" phải là một trong: ${validLevels.join(', ')}.`
      });
    }

    if (hasOwnField(item, 'category') && !isFlashcardCategory(item.category)) {
      itemErrors.push({
        index,
        field: 'category',
        message: `${formatItem(index)} field "category" phải là một trong: ${validCategories.join(', ')}.`
      });
    }

    stringFields.forEach((field) => {
      if (hasOwnField(item, field) && typeof item[field] !== 'string') {
        itemErrors.push({ index, field, message: `${formatItem(index)} field "${field}" phải là string.` });
      }
    });

    nullableStringFields.forEach((field) => {
      if (hasOwnField(item, field) && !isNullableString(item[field])) {
        itemErrors.push({ index, field, message: `${formatItem(index)} field "${field}" phải là string hoặc null.` });
      }
    });

    if (itemErrors.length > 0) {
      errors.push(...itemErrors);
      return;
    }

    validCards.push({
      level: item.level as JlptLevel,
      category: item.category as FlashcardCategory,
      name: item.name as string,
      mean: item.mean as string,
      hiragana: item.hiragana as string,
      image: item.image as string | null,
      audio: item.audio as string | null,
      example: item.example as string | null
    });
  });

  return { validCards, errors };
}
