export type JlptLevel = 'N5' | 'N4' | 'N3' | 'N2' | 'N1';

export type FlashcardCategory = 'vocabulary' | 'kanji' | 'grammar';

export interface Flashcard {
  id: string;
  level: JlptLevel;
  category: FlashcardCategory;
  prompt: string;
  answer: string;
  reading?: string;
  example?: string;
  notes?: string;
}

export interface StudySettings {
  dailyGoal: number;
  selectedLevels: JlptLevel[];
  enabledCategories: FlashcardCategory[];
}
