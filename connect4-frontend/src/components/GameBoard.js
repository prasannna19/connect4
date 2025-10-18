import React, { useEffect, useState } from "react";
import "../index.css";

export default function GameBoard({ ws, username, onReloadLeaderboard }) {
  const [grid, setGrid] = useState(Array.from({ length: 6 }, () => Array(7).fill(0)));
  const [status, setStatus] = useState("Your turn!");
  const [over, setOver] = useState(false);

  useEffect(() => {
    if (!ws) return;

    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === "grid") setGrid(msg.grid);
      if (msg.type === "status") setStatus(msg.message);
      if (msg.type === "result") {
        setGrid(msg.grid);
        setStatus(msg.result);
        setOver(true);
      }
    };

    ws.onclose = () => setStatus("Connection closed — refresh to play again.");
  }, [ws]);

  const dropDisc = (col) => {
    if (over) return;
    ws.send(JSON.stringify({ type: "drop", column: col }));
  };

  const getClass = (v) => (v === 1 ? "red" : v === 2 ? "yel" : "");

  return (
    <div>
      <h2 style={{ textAlign: "center", color: "#5e35b1" }}>
        🎯 Playing as {username} (Red)
      </h2>
      <p style={{ textAlign: "center", marginTop: "-8px" }}>{status}</p>

      <div className="board">
        {grid.map((r, i) => (
          <div key={i} style={{ display: "flex" }}>
            {r.map((c, j) => (
              <div
                key={j}
                className={`cell ${getClass(c)}`}
                onClick={() => dropDisc(j)}
              />
            ))}
          </div>
        ))}
      </div>

      {over && (
        <div style={{ textAlign: "center", marginTop: "12px" }}>
          <button
            className="primary"
            onClick={() => window.location.reload()}
            style={{
              backgroundColor: "#5e35b1",
              color: "white",
              padding: "10px 20px",
              borderRadius: "8px",
              marginRight: "10px",
            }}
          >
            🔁 Play Again
          </button>
          <button
            className="secondary"
            onClick={() => {
              if (onReloadLeaderboard) onReloadLeaderboard();
            }}
            style={{
              backgroundColor: "#ffb300",
              color: "#fff",
              padding: "10px 20px",
              borderRadius: "8px",
            }}
          >
            📊 Refresh Leaderboard
          </button>
        </div>
      )}
    </div>
  );
}
