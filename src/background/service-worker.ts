chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.sync.set({
    jlptSettings: {
      dailyGoal: 20,
      selectedLevels: ['n5', 'n4'],
      enabledCategories: ['locabulary', 'kanji', 'gramma']
    }
  });
});
