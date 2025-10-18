import React, { useEffect, useRef, useState, useCallback } from "react";
import GameBoard from "./components/GameBoard";
import Leaderboard from "./components/Leaderboard";
import "./index.css";

export default function App() {
  const [username, setUsername] = useState("");
  const [start, setStart] = useState(false);
  const [ws, setWs] = useState(null);
  const [status, setStatus] = useState("");
  const loadLeaderboardRef = useRef(null);

  const bindReload = useCallback((fn) => {
    loadLeaderboardRef.current = fn;
  }, []);

  const handleSearchOpponent = () => {
    if (!username.trim()) return;
    setStatus("Searching for opponent...");
    setStart(true);
  };

  useEffect(() => {
    if (!start) return;

    const socket = new WebSocket("ws://localhost:8080/ws");
    setWs(socket);

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "join", username }));
      setStatus("Connected! Waiting for game to start...");
    };

    socket.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);

        // Example of receiving status messages from server
        if (msg.type === "status" && msg.message) {
          setStatus(msg.message);
        }

        // If server sends reload_leaderboard message, refresh leaderboard
        if (msg.type === "reload_leaderboard" && loadLeaderboardRef.current) {
          loadLeaderboardRef.current();
        }
      } catch {
        // Ignore JSON parse errors
      }
    };

    socket.onclose = () => {
      setWs(null);
      setStatus("Connection closed.");
    };

    socket.onerror = () => {
      setStatus("Connection error.");
    };

    return () => {
      socket.close();
    };
  }, [start, username]);

  return (
    <div className="page">
      <header className="hero">
        <div className="title">
          <span className="emoji">🎯</span> Connect4 Arena
        </div>

        {!start && (
          <div className="join">
            <input
              placeholder="Enter username..."
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={start}
            />
            <button onClick={handleSearchOpponent} className="primary">
              Search for Opponent
            </button>
            {status && (
              <p style={{ marginTop: "10px", color: "#888" }}>{status}</p>
            )}
          </div>
        )}
      </header>

      {start && ws && (
        <GameBoard
          ws={ws}
          username={username}
          onReloadLeaderboard={() =>
            loadLeaderboardRef.current && loadLeaderboardRef.current()
          }
          status={status}
        />
      )}

      <Leaderboard bindReload={bindReload} />

      <footer className="footer">
        © 2025 Connect4 Arena · Thank you Emitrr 💜 from Prasanna
      </footer>
    </div>
  );
}
