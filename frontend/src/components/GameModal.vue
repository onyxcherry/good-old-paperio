<script setup lang="ts">
defineProps<{
  show: boolean;
  title: string;
  message: string;
  buttonText?: string;
  isError?: boolean;
}>();

const emit = defineEmits<{
  (e: 'action'): void;
}>();
</script>

<template>
  <Transition name="pop">
    <div v-if="show" class="modal-overlay">
      <div class="modal">
        <h2 :class="{ 'text-danger': isError }">{{ title }}</h2>
        <p>{{ message }}</p>
        <button @click="emit('action')" class="btn" :class="isError ? 'btn-danger' : 'btn-primary'">{{ buttonText || 'Back to Lobby' }}</button>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.modal-overlay {
  position: absolute;
  inset: 0;
  background: rgba(15, 23, 42, 0.7);
  backdrop-filter: blur(8px);
  display: flex;
  justify-content: center;
  align-items: center;
}
.modal {
  background: var(--surface);
  padding: 3rem;
  border-radius: 16px;
  text-align: center;
  border: 1px solid #334155;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}
.pop-enter-active, .pop-leave-active { transition: all 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275); }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: scale(0.8); }

.text-danger {
  color: var(--danger);
}
</style>