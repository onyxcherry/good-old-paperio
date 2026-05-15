<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { GameClient } from './game';

const canvasRef = ref<HTMLCanvasElement | null>(null);
const isDead = ref(false);
const isWinner = ref(false);
const isExited = ref(false);
const inLobby = ref(true);
const games = ref<any[]>([]);
const sessionToken = ref("");
let game: GameClient | null = null;

const fetchGames = async () => {
  if (!inLobby.value) return;
  const res = await fetch(`/api/games?token=${sessionToken.value}`);
  if (res.ok) {
    games.value = await res.json();
  }
};

onMounted(() => {
  let token = localStorage.getItem("paperio_token");
  if (!token) {
    token = "t_" + Math.random().toString(36).substring(2, 9);
    localStorage.setItem("paperio_token", token);
  }

  sessionToken.value = token;
  fetchGames();
  setInterval(fetchGames, 2000);
});

onUnmounted(() => {
  if (game) {
    game.destroy();
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
  setTimeout(() => {
    if (canvasRef.value) {
      game = new GameClient(
        canvasRef.value, 
        sessionToken.value, 
        gameId, 
        () => { isDead.value = true; },
        () => { isWinner.value = true; }
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
  fetchGames();
};
</script>

<template>
  <div class="app-layout">
    <header class="topbar">
      <div class="brand">Paper.io <span>Enterprise</span></div>
      <button v-if="!inLobby" @click="handleManualExit" class="btn btn-danger">Exit Game</button>
    </header>

    <main class="game-container">
      <div v-if="inLobby" class="lobby">
        <h2>Available Games</h2>
        <button @click="createGame" class="btn btn-primary create-btn">Create New Game</button>
        <div class="games-list">
          <div v-for="g in games" :key="g.id" class="game-card">
            <div class="game-info">
              <h3>{{ g.id }}</h3>
              <p>{{ g.players }} / {{ g.max_players }} Players</p>
            </div>
            <div class="game-actions">
              <button v-if="g.has_session" @click="joinGame(g.id)" class="btn btn-primary">Rejoin</button>
              <button v-else :disabled="g.players >= g.max_players" @click="joinGame(g.id)" class="btn btn-primary">
                {{ g.players >= g.max_players ? 'Full' : 'Join' }}
              </button>
            </div>
          </div>
          <div v-if="games.length === 0" class="no-games">
            No games found. Create one!
          </div>
        </div>
      </div>
      <canvas v-else ref="canvasRef"></canvas>
      
      <Transition name="pop">
        <div v-if="isWinner" class="modal-overlay">
          <div class="modal">
            <h2>Victory!</h2>
            <p>You captured 99% of the map!</p>
            <button @click="respawn" class="btn btn-primary">Back to Lobby</button>
          </div>
        </div>
      </Transition>
      <Transition name="pop">
        <div v-if="isExited" class="modal-overlay">
          <div class="modal">
            <h2>Session Ended</h2>
            <p>You have left the game and your session was invalidated.</p>
            <button @click="respawn" class="btn btn-primary">Back to Lobby</button>
          </div>
        </div>
      </Transition>
      <Transition name="pop">
        <div v-if="isDead" class="modal-overlay">
          <div class="modal">
            <h2>Game Over</h2>
            <p>Your territory was overrun or you crashed.</p>
            <button @click="respawn" class="btn btn-primary">Back to Lobby</button>
          </div>
        </div>
      </Transition>
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

body, html {
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

/* Modal Transition */
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

/* Lobby Styles */
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
.create-btn {
  align-self: flex-start;
}
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
.no-games {
  text-align: center;
  color: #94a3b8;
  padding: 2rem;
}
</style>