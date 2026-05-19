export interface GameInfo {
  id: string;
  players: number;
  max_players: number;
  has_session: boolean;
}

export const getSessionToken = async (): Promise<string> => {
  let token = localStorage.getItem("paperio_token");
  if (!token) {
    const res = await fetch(`/api/session`);
    if (!res.ok) {
      throw new Error(`HTTP error! status: ${res.status}`);
    }
    const data = await res.json();
    token = data.token;
    localStorage.setItem("paperio_token", token);
  }
  return token;
};

export const fetchAvailableGames = async (token: string): Promise<GameInfo[]> => {
  const res = await fetch(`/api/games?token=${encodeURIComponent(token)}`);
  if (!res.ok) {
    throw new Error(`HTTP error! status: ${res.status}`);
  }
  return await res.json();
};

export const createNewGame = async (): Promise<string> => {
  const res = await fetch(`/api/games/create`, { method: "POST" });
  if (!res.ok) {
    throw new Error(`HTTP error! status: ${res.status}`);
  }
  const data = await res.json();
  return data.id;
};