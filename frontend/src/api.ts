export interface GameInfo {
  id: string;
  players: number;
  max_players: number;
  has_session: boolean;
}

export const getSessionToken = (): string => {
  let token = localStorage.getItem("paperio_token");
  if (!token) {
    token = "t_" + Math.random().toString(36).substring(2, 9);
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