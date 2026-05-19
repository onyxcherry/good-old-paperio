export interface GameInfo {
  id: string;
  players: number;
  max_players: number;
  has_session: boolean;
}

export const getSessionToken = async (): Promise<string> => {
  let token = localStorage.getItem("paperio_token");
  if (!token) {
    try {
      const res = await fetch(`/api/session`);
      if (!res.ok) throw new Error();
      const data = await res.json();
      token = data.token;
      localStorage.setItem("paperio_token", token as string);
    } catch (err) {
      throw new Error("Unable to obtain a session token. The server might be unavailable.");
    }
  }
  return token as string;
};

export const fetchAvailableGames = async (token: string): Promise<GameInfo[]> => {
  try {
    const res = await fetch(`/api/games?token=${encodeURIComponent(token)}`);
    if (!res.ok) throw new Error();
    return await res.json();
  } catch (err) {
    throw new Error("Unable to fetch available games. Please check your connection.");
  }
};

export const createNewGame = async (): Promise<string> => {
  try {
    const res = await fetch(`/api/games/create`, { method: "POST" });
    if (!res.ok) throw new Error();
    const data = await res.json();
    return data.id;
  } catch (err) {
    throw new Error("Unable to create a new game. Please try again.");
  }
};