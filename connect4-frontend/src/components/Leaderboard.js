import React, { useEffect, useState } from "react";
import "../index.css";

export default function Leaderboard({ bindReload }) {
  const [data, setData] = useState([]);

  // Function to fetch leaderboard data
  const fetchLeaderboard = () => {
    fetch("http://localhost:8080/leaderboard")
      .then((res) => res.json())
      .then((d) => {
        setData(Array.isArray(d) ? d : []);
      })
      .catch(() => setData([]));
  };

  useEffect(() => {
    fetchLeaderboard();

    // Bind the reload function for parent to call later
    if (bindReload) {
      bindReload(fetchLeaderboard);
    }
  }, [bindReload]);

  return (
    <div className="leaderboard-container">
      <h2 className="leaderboard-title">🏆 Leaderboard</h2>

      {data.length === 0 ? (
        <p className="no-games">No games yet — be the first to win!</p>
      ) : (
        <table className="leaderboard-table">
          <thead>
            <tr>
              <th>#</th>
              <th>Player</th>
              <th>Total Games</th>
              <th>Wins</th>
              <th>Win %</th>
              <th>Last Win</th>
            </tr>
          </thead>
          <tbody>
            {data
              .sort((a, b) => b.Wins - a.Wins)
              .map((p, i) => (
                <tr key={i}>
                  <td>{i + 1}</td>
                  <td>{p.Player}</td>
                  <td>{p.Total}</td>
                  <td>{p.Wins}</td>
                  <td>
                    {p.Total > 0
                      ? ((p.Wins / p.Total) * 100).toFixed(1) + "%"
                      : "0%"}
                  </td>
                  <td>
                    {p.LastWin && new Date(p.LastWin).getFullYear() > 1970
                      ? new Date(p.LastWin).toLocaleString([], {
                          hour12: false,
                          hour: "2-digit",
                          minute: "2-digit",
                          day: "2-digit",
                          month: "short",
                        })
                      : new Date().toLocaleString([], {
                          hour12: false,
                          hour: "2-digit",
                          minute: "2-digit",
                          day: "2-digit",
                          month: "short",
                        })}
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
