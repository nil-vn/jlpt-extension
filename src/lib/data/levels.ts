import type { JlptLevel } from '../types/flashcard';

export const jlptLevels: JlptLevel[] = ['N5', 'N4', 'N3', 'N2', 'N1'];

export const levelDescriptions: Record<JlptLevel, string> = {
  N5: 'Beginner words, simple kanji, and everyday grammar.',
  N4: 'Foundational grammar and vocabulary for basic conversations.',
  N3: 'Intermediate expressions for everyday reading and listening.',
  N2: 'Advanced vocabulary and grammar for newspapers and formal contexts.',
  N1: 'Nuanced language used in complex reading and professional settings.'
};
