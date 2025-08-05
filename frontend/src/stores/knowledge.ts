import { ref, computed, reactive } from "vue";

import { defineStore } from "pinia";

export const knowledgeStore = defineStore("knowledge", {
  state: () => ({
    cardList: ref([]),
    total: ref(0),
  }),
  actions: {},
});
