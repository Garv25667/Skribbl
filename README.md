# Skribbl Clone

A high-performance, real-time multiplayer drawing and guessing game (similar to Skribbl.io) built with Go, WebSockets, PostgreSQL, and HTML5 Canvas.

This project was built from scratch to demonstrate production-grade Go backend engineering, concurrent programming using WebSockets, database design with PostgreSQL & sqlc, and scalability testing under high concurrency.

## 🚀 Key Features

- **Real-Time Multiplayer drawing sync**: Fully synchronized HTML5 Canvas using binary WebSockets for ultra-low latency drawing replication.
- **Turn-Based Game Loop**: Automated game lobby state machine handling player turns, drawing states, secret word selection, hints, and countdown timers.
- **Robust Room Management**: Dynamic lobby creation and joining via room codes, with host-only start controls.
- **JWT-Based Authentication**: Secure registration and token-based authentication with custom JWT payload validation integrated into WebSocket upgrades.
- **Interactive Guessing System**: In-game chat which automatically intercepts, scores, and updates leaderboards for correct guesses.
- **Polished Frontend**: Responsive, modern dark-themed user interface featuring tooltips, sound effects, custom color palettes, brush size adjusters, and mobile touch support.
- **Scalability & Load Tested**: Proven stability and high throughput simulating **8,000 concurrent connections** (1,000 rooms × 8 players).

---

## 🛠️ Tech Stack

- **Backend**: Go (Golang)
- **WebSockets**: `gorilla/websocket`
- **Database**: PostgreSQL
- **SQL Compiler**: `sqlc` (generates type-safe Go code from SQL schema)
- **Migrations**: `goose`
- **Frontend**: Vanilla HTML5, CSS3 (Glassmorphism design), and modern JavaScript (no bloated frameworks)

---

## ⚡ Performance & Load Testing

To verify the scalability and performance of the Go backend under heavy network loads, the repository includes a custom simulation/load test runner ([loadtest.go](file:///home/garv_sharma/Skribbl/loadtest/loadtest.go)):

- **Test Volume**: Spawns **1,000 distinct game rooms** concurrently.
- **Connections**: Drives **8,000 concurrent WebSockets** (8 players per room).
- **Simulation**: Orchestrates full game loops including room registration, WebSocket handshakes, simulated host start game commands, real-time drawing sync, and automated word guessing.
- **Result**: Successfully handles the load on a local server, showcasing the efficiency of Go's goroutines and scheduler for I/O bound WebSocket connections.

Run the load test using:
```bash
cd loadtest && go run loadtest.go
```

---

## 📂 Project Structure

```
├── auth.go             # JWT generation, validation, and registration endpoints
├── client.go           # WebSocket client wrapper (read/write pumps)
├── events.go           # WebSocket event payload schemas
├── server.go           # Game state manager, room manager, and broadcast hub
├── main.go             # Entry point (initializes DB, router, and ws handler)
├── room.go             # Game room logic, state machine, and player list
├── message.go          # Message parsing and JSON processing
├── sqlc.yaml           # SQL compiler configuration
├── sql/                # SQL Schema migrations (goose) and query definition (sqlc)
│   ├── schema/
│   └── queries/
├── internal/
│   └── database/       # Compiled, type-safe Go database queries
├── frontend/           # Web client UI (Glassmorphic canvas and chat)
└── loadtest/           # High-concurrency simulation runner
```

---

## ⚙️ Getting Started

### Prerequisites
- Go 1.21+
- PostgreSQL
- `goose` (optional, for manual database migrations)

### 1. Database Setup
Create a PostgreSQL database and configure the `.env` file in the root directory:
```env
DB_URL=postgres://<user>:<password>@localhost:5432/skribbl?sslmode=disable
JWT_SECRET=your_super_secret_jwt_key
```

Run the schema migrations using `goose` (or copy the schema from [001_players.sql](file:///home/garv_sharma/Skribbl/sql/schema/001_players.sql)):
```bash
cd sql/schema
goose postgres "postgres://<user>:<password>@localhost:5432/skribbl?sslmode=disable" up
```

### 2. Start the Backend Server
Run the Go application:
```bash
go run .
```
The server will start listening for HTTP and WebSocket connections on port `:3223`.

### 3. Run the Frontend Client
You can serve the frontend files using any simple web server (e.g., Python's built-in server):
```bash
cd frontend
python3 -m http.server 8080
```
Then navigate to `http://localhost:8080` in your web browser. Enjoy drawing!
