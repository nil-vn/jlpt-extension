import type { Flashcard } from '../types/flashcard';
import n2Vocabulary from './n2-vocabulary.json';

export const defaultN2Vocabulary = n2Vocabulary as Flashcard[];
export const starterFlashcards: Flashcard[] = defaultN2Vocabulary;
