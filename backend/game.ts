const CELL_SIZE = 8;

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
    private particles: any[] = [];
    
    constructor(canvas: HTMLCanvasElement, token: string, private onDeathCallback: () => void) {
        this.canvas = canvas;
        this.ctx = canvas.getContext("2d")!;
        
        // Detect if we're on Vite dev server or production Go server
        const isDev = window.location.port === "5173";
        const wsHost = isDev ? "localhost:8080" : window.location.host;
        
        this.ws = new WebSocket(`ws://${wsHost}/ws?game=lobby1&token=${token}`);
        this.ws.binaryType = "arraybuffer";
        
        this.ws.onmessage = this.handleMessage.bind(this);
        this.ws.onclose = this.handleClose.bind(this);
        
        window.addEventListener("keydown", this.handleInput);
        requestAnimationFrame(this.gameLoop);
    }

    private handleMessage(event: MessageEvent) {
        const view = new DataView(event.data);
        const msgType = view.getUint8(0);

        if (msgType === 0) { // Init
            this.myId = view.getUint32(1);
            this.gridWidth = view.getUint16(5);
            this.gridHeight = view.getUint16(7);
            this.canvas.width = this.gridWidth * CELL_SIZE;
            this.canvas.height = this.gridHeight * CELL_SIZE;
        } 
        else if (msgType === 1) { // Sync State
            let offset = 1;
            const numPlayers = view.getUint16(offset);
            offset += 2;

            const players: Player[] = [];
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
            }

            const numGrid = view.getUint16(offset);
            offset += 2;
            const grid = new Uint32Array(this.gridWidth * this.gridHeight);
            for(let i = 0; i < numGrid; i++) {
                grid[i] = view.getUint32(offset);
                offset += 4;
            }

            this.gameState = { players, grid };
        }
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
        
        const px = me.x * CELL_SIZE + CELL_SIZE / 2;
        const py = me.y * CELL_SIZE + CELL_SIZE / 2;
        
        for (let i = 0; i < 60; i++) {
            this.particles.push({
                x: px,
                y: py,
                vx: (Math.random() - 0.5) * 12,
                vy: (Math.random() - 0.5) * 12,
                life: 0,
                maxLife: 20 + Math.random() * 40,
                color: this.getColor(this.myId)
            });
        }
    }

    private gameLoop = () => {
        if (!this.isRunning) return;
        
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

        if (this.gameState) {
            for (let x = 0; x < this.gridWidth; x++) {
                for (let y = 0; y < this.gridHeight; y++) {
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
                    this.ctx.globalAlpha = 0.8;
                    this.ctx.fillRect(tp.x * CELL_SIZE, tp.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
                });
                this.ctx.globalAlpha = 1.0;

                this.ctx.fillStyle = p.id === this.myId ? "#fff" : playerColor;
                this.ctx.fillRect(p.x * CELL_SIZE, p.y * CELL_SIZE, CELL_SIZE, CELL_SIZE);
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
            this.ctx.arc(p.x, p.y, 4, 0, Math.PI * 2);
            this.ctx.fill();
        }
        this.ctx.globalAlpha = 1.0;

        requestAnimationFrame(this.gameLoop);
    }

    public destroy() {
        this.isRunning = false;
        window.removeEventListener("keydown", this.handleInput);
        if (this.ws.readyState === WebSocket.OPEN) {
            this.ws.close();
        }
    }
}