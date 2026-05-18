<script setup lang="ts">
import type { GameInfo } from '../api';

defineProps<{
  games: GameInfo[];
}>();

const emit = defineEmits<{
  (e: 'create'): void;
  (e: 'join', gameId: string): void;
}>();
</script>

<template>
  <div class="lobby">
    <h2>Available Games</h2>
    <button @click="emit('create')" class="btn btn-primary create-btn">Create New Game</button>
    <div class="games-list">
      <div v-for="g in games" :key="g.id" class="game-card">
        <div class="game-info">
          <h3>{{ g.id }}</h3>
          <p>{{ g.players }} / {{ g.max_players }} Players</p>
        </div>
        <div class="game-actions">
          <button v-if="g.has_session" @click="emit('join', g.id)" class="btn btn-primary">Rejoin</button>
          <button v-else :disabled="g.players >= g.max_players" @click="emit('join', g.id)" class="btn btn-primary">
            {{ g.players >= g.max_players ? 'Full' : 'Join' }}
          </button>
        </div>
      </div>
      <div v-if="games.length === 0" class="no-games">
        No games found. Create one!
      </div>
    </div>
  </div>
</template>

<style scoped>
.lobby {
  background: var(--surface);
  padding: 2rem;
  border-radius: 12px;
  width: 100%;
  max-width: 600px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  border: 1px solid #334155;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.create-btn { align-self: flex-start; }
.games-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-height: 400px;
  overflow-y: auto;
}
.game-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #0f172a;
  padding: 1rem;
  border-radius: 8px;
  border: 1px solid #334155;
}
.game-card h3 { margin: 0 0 0.5rem 0; }
.game-card p { margin: 0; color: #94a3b8; }
.no-games { text-align: center; color: #94a3b8; padding: 2rem; }
</style>