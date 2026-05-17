import type { JlptLevel } from '../types/flashcard';

export const jlptLevels: JlptLevel[] = ['n5', 'n4', 'n3', 'n2', 'n1'];

export const levelDescriptions: Record<JlptLevel, string> = {
  n5: 'Beginner words, simple kanji, and everyday grammar.',
  n4: 'Foundational grammar and vocabulary for basic conversations.',
  n3: 'Intermediate expressions for everyday reading and listening.',
  n2: 'Advanced vocabulary and grammar for newspapers and formal contexts.',
  n1: 'Nuanced language used in complex reading and professional settings.'
};
