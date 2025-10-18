# 🎯 4-in-a-Row (Connect Four) — Real-Time Multiplayer Game

## 🚀 Overview
A real-time multiplayer **Connect Four** game built with **GoLang**, **WebSockets**, **Kafka**, and **PostgreSQL**.  
If no opponent joins within 10 seconds, the user plays against a **competitive bot**.
This project is a **real-time, backend-driven version of the classic Connect Four game**, built as part of the Backend Engineering Intern assignment.
Players can:
- Play **1v1 against a competitive bot** (fallback if no opponent found).
- View a **real-time game board**.
- Track performance via a **live leaderboard**.
- Observe **Kafka-based analytics events** (simulating production metrics).


---

## 🧠 Features
- Real-time gameplay with WebSockets  
- 1v1 matchmaking or vs bot  
- Reconnection within 30s  
- Auto-forfeit after 30s inactivity  
- Game analytics via Kafka  
- PostgreSQL leaderboard  
- Simple HTML frontend

---
## 🧩 Tech Stack

**Backend**
- Go (Golang 1.22+)
- Gorilla WebSocket
- PostgreSQL (via `github.com/lib/pq`)
- Kafka (Producer/Consumer)
- CORS middleware (`github.com/rs/cors`)

**Frontend**
- React.js (create-react-app)
- WebSocket client for real-time play
- Responsive 7×6 grid UI
- Live leaderboard refresh

## ⚙️ Setup Instructions


2.  Start ZooKeeper & Kafka

cd C:\kafka\kafka_2.13-3.7.0
bin\windows\zookeeper-server-start.bat config\zookeeper.properties
bin\windows\kafka-server-start.bat config\server.properties
bin\windows\kafka-topics.bat --create --topic connect4-events --bootstrap-server localhost:9092

3. Run Backend
   
cd backend
go mod tidy
go run cmd/api/main.go

4. Run Frontend

Open frontend/index.html in your browser or with VS Code Live Server.


API Endpoints
Endpoint	Method	Description
/ws	WebSocket	Real-time game connection
/leaderboard	GET	Player stats

Author
CHERUKURI PRASANNA LAKSHMI
2200030023cseh@gmail.com
Github: https://github.com/prasannna19/connect4/
