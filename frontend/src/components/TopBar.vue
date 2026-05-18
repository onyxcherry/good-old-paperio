<script setup lang="ts">
defineProps<{
  inLobby: boolean;
}>();

const emit = defineEmits<{
  (e: 'exit'): void;
}>();
</script>

<template>
  <header class="topbar">
    <div class="brand">Good old paper.io</div>
    <button v-if="!inLobby" @click="emit('exit')" class="btn btn-danger">Exit Game</button>
  </header>
</template>

<style scoped>
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background: rgba(30, 41, 59, 0.8);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid #334155;
  z-index: 10;
}
.brand {
  font-size: 1.5rem;
  font-weight: 800;
  letter-spacing: -0.5px;
}
.brand span {
  color: var(--primary);
}
</style>