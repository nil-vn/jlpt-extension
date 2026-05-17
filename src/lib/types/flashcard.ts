export type JlptLevel = 'n5' | 'n4' | 'n3' | 'n2' | 'n1';

export type FlashcardCategory = 'gramma' | 'locabulary' | 'kanji' | 'reading' | 'listening';

export interface Flashcard {
  level: JlptLevel;
  category: FlashcardCategory;
  name: string;
  mean: string;
  hiragana: string;
  image: string | null;
  audio: string | null;
  example: string | null;
}

export interface StudySettings {
  dailyGoal: number;
  selectedLevels: JlptLevel[];
  enabledCategories: FlashcardCategory[];
}

export function createFlashcardId(card: Pick<Flashcard, 'level' | 'category' | 'name' | 'hiragana' | 'mean'>) {
  return [card.level, card.category, card.name, card.hiragana, card.mean]
    .map((part) => String(part ?? '').trim())
    .join('|');
}
