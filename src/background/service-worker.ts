chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.sync.set({
    jlptSettings: {
      dailyGoal: 20,
      selectedLevels: ['N5', 'N4'],
      enabledCategories: ['vocabulary', 'kanji', 'grammar']
    }
  });
});
