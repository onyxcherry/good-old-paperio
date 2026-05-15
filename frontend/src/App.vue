<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { GameClient } from '../../backend/game';

const canvasRef = ref<HTMLCanvasElement | null>(null);
const isDead = ref(false);
let game: GameClient | null = null;

onMounted(() => {
  let sessionToken = localStorage.getItem("paperio_token");
  if (!sessionToken) {
    sessionToken = "t_" + Math.random().toString(36).substring(2, 9);
    localStorage.setItem("paperio_token", sessionToken);
  }

  if (canvasRef.value) {
    game = new GameClient(canvasRef.value, sessionToken, () => {
      isDead.value = true;
    });
  }
});

onUnmounted(() => {
  if (game) {
    game.destroy();
  }
});

const exitSession = () => {
  localStorage.removeItem("paperio_token");
  window.location.reload();
};

const respawn = () => {
  window.location.reload();
};
</script>

<template>
  <div class="app-layout">
    <header class="topbar">
      <div class="brand">Paper.io <span>Enterprise</span></div>
      <button @click="exitSession" class="btn btn-danger">Exit Session</button>
    </header>

    <main class="game-container">
      <canvas ref="canvasRef"></canvas>
      
      <Transition name="pop">
        <div v-if="isDead" class="modal-overlay">
          <div class="modal">
            <h2>Game Over</h2>
            <p>Your territory was overrun.</p>
            <button @click="respawn" class="btn btn-primary">Play Again</button>
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
  background: #1e1e2e;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  border-radius: 8px;
  border: 1px solid #334155;
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
</style>