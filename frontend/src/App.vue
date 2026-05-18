<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { GameClient } from './game';
import { fetchAvailableGames, getSessionToken, type GameInfo } from './api';
import TopBar from './components/TopBar.vue';
import GameLobby from './components/GameLobby.vue';
import GameModal from './components/GameModal.vue';

const canvasRef = ref<HTMLCanvasElement | null>(null);
const isDead = ref(false);
const isWinner = ref(false);
const isExited = ref(false);
const errorMessage = ref("");
const inLobby = ref(true);
const games = ref<GameInfo[]>([]);
const sessionToken = ref("");
let game: GameClient | null = null;
let fetchInterval: number | null = null;

const loadGames = async () => {
  if (!inLobby.value) return;
  try {
    games.value = await fetchAvailableGames(sessionToken.value);
  } catch (error: any) {
    errorMessage.value = "Failed to fetch games: " + (error.message || String(error));
  }
};

onMounted(() => {
  sessionToken.value = getSessionToken();
  loadGames();
  fetchInterval = window.setInterval(loadGames, 2000);
});

onUnmounted(() => {
  if (game) {
    game.destroy();
  }
  if (fetchInterval !== null) {
    clearInterval(fetchInterval);
  }
});

const ADJS = ["swift", "brave", "mighty", "clever", "silent", "happy", "lucky", "fierce"];
const NOUNS = ["tiger", "eagle", "dragon", "panther", "wolf", "bear", "fox", "shark"];

const generateGameName = () => {
  const adj = ADJS[Math.floor(Math.random() * ADJS.length)];
  const noun = NOUNS[Math.floor(Math.random() * NOUNS.length)];
  return `${adj}-${noun}-${Math.floor(Math.random() * 1000)}`;
};

const joinGame = (gameId: string) => {
  inLobby.value = false;
  isDead.value = false;
  isWinner.value = false;
  isExited.value = false;
  errorMessage.value = "";
  setTimeout(() => {
    if (canvasRef.value) {
      game = new GameClient(
        canvasRef.value,
        sessionToken.value,
        gameId,
        () => { isDead.value = true; },
        () => { isWinner.value = true; },
        (err) => { errorMessage.value = err; }
      );
    }
  }, 0);
};

const createGame = () => {
  joinGame(generateGameName());
};

const handleManualExit = () => {
  if (game) {
    game.destroy(true);
    game = null;
  }
  isExited.value = true;
};

const respawn = () => {
  if (game) {
    game.destroy(false);
    game = null;
  }
  inLobby.value = true;
  isDead.value = false;
  isWinner.value = false;
  isExited.value = false;
  errorMessage.value = "";
  loadGames();
};
</script>

<template>
  <div class="app-layout">
    <TopBar :in-lobby="inLobby" @exit="handleManualExit" />

    <main class="game-container">
      <GameLobby v-if="inLobby" :games="games" @create="createGame" @join="joinGame" />
      <canvas v-else ref="canvasRef"></canvas>

      <GameModal :show="isWinner" title="Victory!" message="You captured at least 99% of the map!" @action="respawn" />

      <GameModal :show="isExited" title="Session Ended"
        message="You have left the game and your session was invalidated." @action="respawn" />

      <GameModal :show="isDead" title="Game Over" message="Your territory was overrun or you crashed."
        @action="respawn" />

      <GameModal :show="!!errorMessage" title="Error" :message="errorMessage" buttonText="Dismiss"
        @action="errorMessage = ''" />
    </main>
  </div>
</template>

<style>
:root {
  --bg-color: #0f172a;
  --surface: #1e293b;
  --text: #f8fafc;
  --primary: #3b82f6;
  --danger: #ef4444;
}

body,
html {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  background-color: var(--bg-color);
  color: var(--text);
  font-family: 'Inter', sans-serif;
  overflow: hidden;
}

.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.game-container {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  background: radial-gradient(circle at center, #1e293b 0%, #0f172a 100%);
}

canvas {
  width: 100%;
  height: 100%;
  display: block;
  background: #09090b;
}

.btn {
  padding: 0.6rem 1.2rem;
  border: none;
  border-radius: 6px;
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.1s, opacity 0.2s;
}

.btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-primary {
  background: var(--primary);
  color: white;
}

.btn-danger {
  background: rgba(239, 68, 68, 0.1);
  color: var(--danger);
  border: 1px solid rgba(239, 68, 68, 0.3);
}
</style>