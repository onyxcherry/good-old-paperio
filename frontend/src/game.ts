const CELL_SIZE = 24;

export interface Player {
    id: number;
    x: number;
    y: number;
    dir: number;
    tail: { x: number; y: number }[];
}

export interface GameState {
    players: Player[];
    grid: Uint32Array;
}

export class GameClient {
    private canvas: HTMLCanvasElement;
    private ctx: CanvasRenderingContext2D;
    private ws: WebSocket;
    
    private myId: number = 0;
    private gridWidth: number = 0;
    private gridHeight: number = 0;
    private gameState: GameState | null = null;
    
    private isRunning: boolean = true;
    private isDead: boolean = false;
    
    private renderStates = new Map<number, {x: number, y: number, prevX: number, prevY: number}>();
    private lastSyncTime: number = Date.now();
    private particles: any[] = [];
    
    constructor(
        canvas: HTMLCanvasElement, 
        token: string, 
        gameId: string, 
        private onDeathCallback: () => void, 
        private onWinCallback: () => void,
        private onErrorCallback: (error: string) => void
    ) {
        this.canvas = canvas;
        this.ctx = canvas.getContext("2d")!;
        
        // Detect if we're on Vite dev server or production Go server
        const isDev = window.location.port === "5173";
        const wsHost = isDev ? "localhost:8080" : window.location.host;
        const wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        
        this.ws = new WebSocket(`${wsProtocol}//${wsHost}/ws?game=${gameId}&token=${token}`);
        this.ws.binaryType = "arraybuffer";
        
        this.ws.onmessage = this.handleMessage.bind(this);
        this.ws.onclose = this.handleClose.bind(this);
        this.ws.onerror = this.handleError.bind(this);
        
        window.addEventListener("keydown", this.handleInput);
        window.addEventListener("resize", this.resizeCanvas);
        requestAnimationFrame(this.gameLoop);
    }

    private handleMessage(event: MessageEvent) {
        const view = new DataView(event.data);
        const msgType = view.getUint8(0);

        if (msgType === 0) { // Init
            this.myId = view.getUint32(1);
            this.gridWidth = view.getUint16(5);
            this.gridHeight = view.getUint16(7);
            this.resizeCanvas();
        } 
        else if (msgType === 1) { // Sync State
            this.lastSyncTime = Date.now();
            let offset = 1;
            const numPlayers = view.getUint16(offset);
            offset += 2;

            const players: Player[] = [];
            const currentIds = new Set<number>();
            for (let i = 0; i < numPlayers; i++) {
                const id = view.getUint32(offset);
                const x = view.getUint16(offset + 4);
                const y = view.getUint16(offset + 6);
                const dir = view.getUint8(offset + 8);
                const tailLen = view.getUint16(offset + 9);
                offset += 11;

                const tail = [];
                for (let t = 0; t < tailLen; t++) {
                    tail.push({ x: view.getUint16(offset), y: view.getUint16(offset + 2) });
                    offset += 4;
                }
                players.push({ id, x, y, dir, tail });
                currentIds.add(id);

                const rs = this.renderStates.get(id);
                if (rs) {
                    rs.prevX = rs.x;
                    rs.prevY = rs.y;
                    rs.x = x;
                    rs.y = y;
                } else {
                    this.renderStates.set(id, {x, y, prevX: x, prevY: y});
                }
            }

            // Cleanup disconnected players from render states
            for (const key of this.renderStates.keys()) {
                if (!currentIds.has(key)) {
                    this.renderStates.delete(key);
                }
            }

            const numGrid = view.getUint32(offset);
            offset += 4;
            const grid = new Uint32Array(this.gridWidth * this.gridHeight);
            for(let i = 0; i < numGrid; i++) {
                grid[i] = view.getUint32(offset);
                offset += 4;
            }

            this.gameState = { players, grid };
        }
        else if (msgType === 4) { // Win
            const winnerId = view.getUint32(1);
            if (winnerId === this.myId) {
                this.onWinCallback();
            } else {
                this.onDeathCallback();
            }
        }
    }

    private resizeCanvas = () => {
        if (this.canvas.parentElement) {
            this.canvas.width = this.canvas.parentElement.clientWidth;
            this.canvas.height = this.canvas.parentElement.clientHeight;
        } else {
            this.canvas.width = window.innerWidth;
            this.canvas.height = window.innerHeight;
        }
    }

    private handleError(event: Event) {
        let errorMsg = "WebSocket connection error occurred.";
        if (event instanceof ErrorEvent && event.message) {
            errorMsg = `WebSocket error: ${event.message}`;
        }
        this.onErrorCallback(errorMsg);
    }

    private handleClose() {
        this.isDead = true;
        this.createExplosion();
        this.onDeathCallback();
    }

    private handleInput = (e: KeyboardEvent) => {
        if (this.ws.readyState === WebSocket.OPEN && !this.isDead) {
            let dirByte = -1;
            switch(e.key) {
                case "ArrowUp":    dirByte = 0; break;
                case "ArrowRight": dirByte = 1; break;
                case "ArrowDown":  dirByte = 2; break;
                case "ArrowLeft":  dirByte = 3; break;
            }
            
            if (dirByte !== -1) {
                const buf = new ArrayBuffer(2);
                const view = new DataView(buf);
                view.setUint8(0, 2); 
                view.setUint8(1, dirByte);
                this.ws.send(buf);
            }
        }
    }

    private getColor(id: number): string {
        if (id === 0) return "transparent";
        const hue = (id * 137) % 360;
        return `hsl(${hue}, 70%, 60%)`;
    }

    private createExplosion() {
        const me = this.gameState?.players.find(p => p.id === this.myId);
        if (!me) return;
        
        const rs = this.renderStates.get(this.myId);
        const rx = rs ? rs.x : me.x;
        const ry = rs ? rs.y : me.y;
        
        const px = rx * CELL_SIZE + CELL_SIZE / 2;
        const py = ry * CELL_SIZE + CELL_SIZE / 2;
        
        for (let i = 0; i < 60; i++) {
            this.particles.push({
                x: px,
                y: py,
                    vx: (Math.random() - 0.5) * 36,
                    vy: (Math.random() - 0.5) * 36,
                life: 0,
                maxLife: 20 + Math.random() * 40,
                color: this.getColor(this.myId)
            });
        }
    }

    private gameLoop = () => {
        if (!this.isRunning) return;
        
        const now = Date.now();
        // Tick rate is 100ms, cap progress at 1.0 to prevent overshooting if server lags
        const progress = Math.min((now - this.lastSyncTime) / 100, 1.0);
        
        this.ctx.fillStyle = "#09090b"; // Dark out-of-bounds area
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);

        let camX = 0;
        let camY = 0;

        if (this.gameState) {
            const me = this.gameState.players.find(p => p.id === this.myId);
            if (me) {
                const rs = this.renderStates.get(this.myId);
                const rx = rs ? rs.prevX + (rs.x - rs.prevX) * progress : me.x;
                const ry = rs ? rs.prevY + (rs.y - rs.prevY) * progress : me.y;
                camX = rx * CELL_SIZE - this.canvas.width / 2 + CELL_SIZE / 2;
                camY = ry * CELL_SIZE - this.canvas.height / 2 + CELL_SIZE / 2;
            }
        }

        this.ctx.save();
        this.ctx.translate(-camX, -camY);

        if (this.gameState) {
            // Inner map background
            this.ctx.fillStyle = "#1e1e2e";
            this.ctx.fillRect(0, 0, this.gridWidth * CELL_SIZE, this.gridHeight * CELL_SIZE);

            // Map bounds border
            this.ctx.strokeStyle = "#ef4444"; // Bold red border
            this.ctx.lineWidth = 8;
            this.ctx.strokeRect(0, 0, this.gridWidth * CELL_SIZE, this.gridHeight * CELL_SIZE);

            // Optimize: calculate visible view bounds to avoid iterating non-visible grid nodes
            const startX = Math.max(0, Math.floor(camX / CELL_SIZE));
            const endX = Math.min(this.gridWidth, Math.ceil((camX + this.canvas.width) / CELL_SIZE));
            const startY = Math.max(0, Math.floor(camY / CELL_SIZE));
            const endY = Math.min(this.gridHeight, Math.ceil((camY + this.canvas.height) / CELL_SIZE));

            for (let x = startX; x < endX; x++) {
                for (let y = startY; y < endY; y++) {
                    const ownerId = this.gameState.grid[x * this.gridHeight + y];
                    if (ownerId !== 0) {
                        this.ctx.fillStyle = this.getColor(ownerId);
                        this.ctx.globalAlpha = 0.25; 
                        this.ctx.fillRect(x * CELL_SIZE, y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
                        this.ctx.globalAlpha = 1.0;
                    }
                }
            }

            this.gameState.players.forEach(p => {
                const playerColor = this.getColor(p.id);
                
                this.ctx.fillStyle = playerColor;
                p.tail.forEach(tp => {
                    // Prevent drawing the tail segment at the current destination 
                    // to avoid seeing a square jutting out in front of the interpolating head
                    if (tp.x === p.x && tp.y === p.y) return;

                    this.ctx.globalAlpha = 0.8;
                    this.ctx.fillRect(tp.x * CELL_SIZE, tp.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
                });
                this.ctx.globalAlpha = 1.0;
                
                const rs = this.renderStates.get(p.id);
                const rx = rs ? rs.prevX + (rs.x - rs.prevX) * progress : p.x;
                const ry = rs ? rs.prevY + (rs.y - rs.prevY) * progress : p.y;
                
                const headX = rx * CELL_SIZE;
                const headY = ry * CELL_SIZE;
                
                const hSize = CELL_SIZE * 1.4; // Make head 40% bigger than body
                const offset = (CELL_SIZE - hSize) / 2;

                this.ctx.fillStyle = p.id === this.myId ? "#fff" : playerColor;
                
                // Draw Shadow
                this.ctx.shadowColor = 'rgba(0,0,0,0.4)';
                this.ctx.shadowBlur = 6;
                this.ctx.shadowOffsetY = 2;
                this.ctx.fillRect(headX + offset, headY + offset, hSize, hSize);
                
                // Reset shadow & draw border
                this.ctx.shadowBlur = 0;
                this.ctx.shadowOffsetY = 0;
                this.ctx.strokeStyle = "rgba(0,0,0,0.8)";
                this.ctx.lineWidth = 1.5;
                this.ctx.strokeRect(headX + offset, headY + offset, hSize, hSize);

                // Draw Eyes based on direction
                this.ctx.fillStyle = "#111";
                const eyeSize = Math.max(2, CELL_SIZE * 0.25);
                let e1x = 0, e1y = 0, e2x = 0, e2y = 0;

                if (p.dir === 0) { // Up
                    e1x = headX + CELL_SIZE * 0.1; e1y = headY - CELL_SIZE * 0.1;
                    e2x = headX + CELL_SIZE * 0.6; e2y = headY - CELL_SIZE * 0.1;
                } else if (p.dir === 1) { // Right
                    e1x = headX + CELL_SIZE * 0.8; e1y = headY + CELL_SIZE * 0.1;
                    e2x = headX + CELL_SIZE * 0.8; e2y = headY + CELL_SIZE * 0.6;
                } else if (p.dir === 2) { // Down
                    e1x = headX + CELL_SIZE * 0.1; e1y = headY + CELL_SIZE * 0.8;
                    e2x = headX + CELL_SIZE * 0.6; e2y = headY + CELL_SIZE * 0.8;
                } else if (p.dir === 3) { // Left
                    e1x = headX - CELL_SIZE * 0.1; e1y = headY + CELL_SIZE * 0.1;
                    e2x = headX - CELL_SIZE * 0.1; e2y = headY + CELL_SIZE * 0.6;
                }
                
                this.ctx.fillRect(e1x, e1y, eyeSize, eyeSize);
                this.ctx.fillRect(e2x, e2y, eyeSize, eyeSize);
            });
        }

        // Render explosion overlay
        for (let i = this.particles.length - 1; i >= 0; i--) {
            const p = this.particles[i];
            p.x += p.vx; p.y += p.vy; p.life++;
            
            if (p.life >= p.maxLife) {
                this.particles.splice(i, 1);
                continue;
            }
            
            this.ctx.globalAlpha = 1 - (p.life / p.maxLife);
            this.ctx.fillStyle = p.color;
            this.ctx.beginPath();
            this.ctx.arc(p.x, p.y, 12, 0, Math.PI * 2);
            this.ctx.fill();
        }
        this.ctx.globalAlpha = 1.0;

        this.ctx.restore();

        requestAnimationFrame(this.gameLoop);
    }

    public destroy(isManualExit: boolean = false) {
        this.isRunning = false;
        window.removeEventListener("keydown", this.handleInput);
        window.removeEventListener("resize", this.resizeCanvas);
        this.ws.onclose = null; // Prevent handleClose from firing

        if (this.ws.readyState === WebSocket.OPEN) {
            if (isManualExit) {
                const buf = new ArrayBuffer(1);
                new DataView(buf).setUint8(0, 3); // Leave Game
                this.ws.send(buf);
            }
            this.ws.close();
        }
    }
}