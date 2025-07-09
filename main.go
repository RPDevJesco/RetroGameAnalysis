// pokemon_web_server.go
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"RetroGameAnalysis/connection"
	"RetroGameAnalysis/server"
	"github.com/gorilla/mux"
)

type GameData struct {
	PlayerName    string    `json:"player_name"`
	PlayerID      uint16    `json:"player_id"`
	Money         uint32    `json:"money"`
	TeamCount     uint8     `json:"team_count"`
	CurrentMap    uint8     `json:"current_map"`
	LocationName  string    `json:"location_name"`
	PlayerX       uint8     `json:"player_x"`
	PlayerY       uint8     `json:"player_y"`
	Badges        []Badge   `json:"badges"`
	PokedexSeen   int       `json:"pokedex_seen"`
	PokedexCaught int       `json:"pokedex_caught"`
	Hours         uint16    `json:"hours"`
	Minutes       uint16    `json:"minutes"`
	Seconds       uint8     `json:"seconds"`
	BagItemCount  uint8     `json:"bag_item_count"`
	BagItems      []Item    `json:"bag_items"`
	Pokemon       []Pokemon `json:"pokemon"`
	BattleMode    string    `json:"battle_mode"`
	BattleType    string    `json:"battle_type"`
	LastUpdated   time.Time `json:"last_updated"`
}

type Pokemon struct {
	Species       uint8   `json:"species"`
	Name          string  `json:"name"`
	PokedexNumber uint8   `json:"pokedex_number"`
	Level         uint8   `json:"level"`
	CurrentHP     uint16  `json:"current_hp"`
	MaxHP         uint16  `json:"max_hp"`
	Attack        uint16  `json:"attack"`
	Defense       uint16  `json:"defense"`
	Speed         uint16  `json:"speed"`
	Special       uint16  `json:"special"`
	Status        uint8   `json:"status"`
	StatusName    string  `json:"status_name"`
	Type1         uint8   `json:"type1"`
	Type1Name     string  `json:"type1_name"`
	Type2         uint8   `json:"type2"`
	Type2Name     string  `json:"type2_name"`
	Moves         []Move  `json:"moves"`
	ExpPoints     uint32  `json:"exp_points"`
	HPPercent     float64 `json:"hp_percent"`
}

type Move struct {
	ID   uint8  `json:"id"`
	Name string `json:"name"`
}

type Item struct {
	ID       uint8  `json:"id"`
	Name     string `json:"name"`
	Quantity uint8  `json:"quantity"`
}

type Badge struct {
	Name     string `json:"name"`
	Obtained bool   `json:"obtained"`
}

// PokemonWebServer handles the web server and Pokemon data
type PokemonWebServer struct {
	wsManager *server.WebSocketManager
	driver    *connection.AdaptiveRetroArchDriver
	gameData  *GameData
	router    *mux.Router
}

func NewPokemonWebServer() *PokemonWebServer {
	wsManager := server.NewWebSocketManager()

	driver := connection.NewAdaptiveRetroArchDriver("localhost", 55355, 5*time.Second)
	driver.SetPlatform("GB")

	return &PokemonWebServer{
		wsManager: wsManager,
		driver:    driver,
		gameData:  &GameData{},
		router:    mux.NewRouter(),
	}
}

func (s *PokemonWebServer) Start(port string) {
	// Start WebSocket manager
	s.wsManager.Start()

	// Connect to RetroArch
	if err := s.driver.Connect(); err != nil {
		log.Fatalf("Failed to connect to RetroArch: %v", err)
	}

	log.Println("✅ Connected to RetroArch")

	// Setup routes
	s.setupRoutes()

	// Start Pokemon data monitoring
	go s.monitorPokemonData()

	// Start server
	log.Printf("🌐 Pokemon Web Server starting on port %s", port)
	log.Printf("📱 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("🌍 Web interface: http://localhost:%s", port)
	log.Printf("📡 REST API: http://localhost:%s/api/", port)

	log.Fatal(http.ListenAndServe(":"+port, s.router))
}

// setupRoutes configures all HTTP routes
func (s *PokemonWebServer) setupRoutes() {
	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.wsManager.HandleWebSocket)

	// REST API endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/gamedata", s.handleGetGameData).Methods("GET")
	api.HandleFunc("/pokemon", s.handleGetPokemon).Methods("GET")
	api.HandleFunc("/pokemon/{id:[0-9]+}", s.handleGetPokemonByID).Methods("GET")
	api.HandleFunc("/player", s.handleGetPlayer).Methods("GET")
	api.HandleFunc("/items", s.handleGetItems).Methods("GET")
	api.HandleFunc("/badges", s.handleGetBadges).Methods("GET")
	api.HandleFunc("/status", s.handleGetStatus).Methods("GET")

	// Static files and web interface
	s.router.HandleFunc("/", s.handleHomePage).Methods("GET")
	s.router.HandleFunc("/live", s.handleLivePage).Methods("GET")
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Enable CORS for API
	s.router.Use(corsMiddleware)
}

// REST API Handlers
func (s *PokemonWebServer) handleGetGameData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.gameData); err != nil {
		http.Error(w, "Failed to encode game data", http.StatusInternalServerError)
		return
	}
}

func (s *PokemonWebServer) handleGetPokemon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.gameData.Pokemon); err != nil {
		http.Error(w, "Failed to encode Pokemon data", http.StatusInternalServerError)
		return
	}
}

func (s *PokemonWebServer) handleGetPokemonByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid Pokemon ID", http.StatusBadRequest)
		return
	}

	if id < 1 || id > len(s.gameData.Pokemon) {
		http.Error(w, "Pokemon not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.gameData.Pokemon[id-1]); err != nil {
		http.Error(w, "Failed to encode Pokemon data", http.StatusInternalServerError)
		return
	}
}

func (s *PokemonWebServer) handleGetPlayer(w http.ResponseWriter, r *http.Request) {
	playerData := map[string]interface{}{
		"name":           s.gameData.PlayerName,
		"id":             s.gameData.PlayerID,
		"money":          s.gameData.Money,
		"location":       s.gameData.LocationName,
		"x":              s.gameData.PlayerX,
		"y":              s.gameData.PlayerY,
		"hours":          s.gameData.Hours,
		"minutes":        s.gameData.Minutes,
		"pokedex_seen":   s.gameData.PokedexSeen,
		"pokedex_caught": s.gameData.PokedexCaught,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playerData)
}

func (s *PokemonWebServer) handleGetItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.gameData.BagItems)
}

func (s *PokemonWebServer) handleGetBadges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.gameData.Badges)
}

func (s *PokemonWebServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"connected":         true,
		"last_updated":      s.gameData.LastUpdated,
		"websocket_clients": s.wsManager.GetClientCount(),
		"game_loaded":       s.gameData.PlayerName != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Web Interface Handlers
func (s *PokemonWebServer) handleHomePage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pokemon Red/Blue Analyzer</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #fff;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            text-align: center;
            margin-bottom: 40px;
        }
        
        .header h1 {
            font-size: 3rem;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        
        .status-card {
            background: rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 30px;
            margin-bottom: 30px;
            border: 1px solid rgba(255,255,255,0.2);
        }
        
        .player-info {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .info-card {
            background: rgba(255,255,255,0.1);
            padding: 20px;
            border-radius: 10px;
            border: 1px solid rgba(255,255,255,0.2);
        }
        
        .info-card h3 {
            margin-bottom: 15px;
            color: #FFD700;
        }
        
        .pokemon-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .pokemon-card {
            background: rgba(255,255,255,0.15);
            border-radius: 15px;
            padding: 20px;
            border: 1px solid rgba(255,255,255,0.3);
            transition: transform 0.3s ease;
        }
        
        .pokemon-card:hover {
            transform: translateY(-5px);
            background: rgba(255,255,255,0.2);
        }
        
        .pokemon-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }
        
        .pokemon-name {
            font-size: 1.4rem;
            font-weight: bold;
            color: #FFD700;
        }
        
        .pokemon-level {
            background: #4CAF50;
            color: white;
            padding: 5px 10px;
            border-radius: 20px;
            font-size: 0.9rem;
        }
        
        .hp-bar {
            background: #333;
            border-radius: 10px;
            height: 20px;
            margin: 10px 0;
            overflow: hidden;
        }
        
        .hp-fill {
            height: 100%;
            border-radius: 10px;
            transition: width 0.3s ease;
        }
        
        .hp-excellent { background: #4CAF50; }
        .hp-good { background: #FFC107; }
        .hp-okay { background: #FF9800; }
        .hp-critical { background: #F44336; }
        .hp-fainted { background: #9E9E9E; }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 10px;
            margin: 15px 0;
        }
        
        .stat {
            background: rgba(0,0,0,0.2);
            padding: 8px;
            border-radius: 5px;
        }
        
        .moves {
            margin-top: 15px;
        }
        
        .move-tag {
            display: inline-block;
            background: #6C63FF;
            color: white;
            padding: 4px 8px;
            border-radius: 15px;
            font-size: 0.8rem;
            margin: 2px;
        }
        
        .badges {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 10px;
            margin: 20px 0;
        }
        
        .badge {
            text-align: center;
            padding: 10px;
            border-radius: 10px;
            border: 2px solid;
        }
        
        .badge.obtained {
            background: rgba(76, 175, 80, 0.3);
            border-color: #4CAF50;
            color: #4CAF50;
        }
        
        .badge.not-obtained {
            background: rgba(158, 158, 158, 0.3);
            border-color: #9E9E9E;
            color: #9E9E9E;
        }
        
        .nav-buttons {
            display: flex;
            gap: 15px;
            justify-content: center;
            margin: 30px 0;
        }
        
        .btn {
            background: #6C63FF;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 25px;
            cursor: pointer;
            text-decoration: none;
            font-size: 1rem;
            transition: all 0.3s ease;
        }
        
        .btn:hover {
            background: #5A52E5;
            transform: translateY(-2px);
        }
        
        .loading {
            text-align: center;
            padding: 50px;
            font-size: 1.2rem;
        }
        
        .error {
            background: rgba(244, 67, 54, 0.2);
            border: 1px solid #F44336;
            color: #F44336;
            padding: 20px;
            border-radius: 10px;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎮 Pokemon Red/Blue Analyzer</h1>
            <p>Real-time Pokemon game data analysis</p>
        </div>
        
        <div class="nav-buttons">
            <a href="/" class="btn">📊 Overview</a>
            <a href="/live" class="btn">📡 Live Data</a>
            <a href="/api/gamedata" class="btn" target="_blank">🔗 JSON API</a>
        </div>
        
        <div id="content" class="loading">
            Loading Pokemon data...
        </div>
    </div>

    <script>
        async function loadGameData() {
            try {
                const response = await fetch('/api/gamedata');
                const data = await response.json();
                displayGameData(data);
            } catch (error) {
                displayError('Failed to load game data: ' + error.message);
            }
        }
        
        function displayGameData(data) {
            const content = document.getElementById('content');
            
            if (!data.player_name) {
                content.innerHTML = '<div class="error">No Pokemon game detected. Make sure RetroArch is running with Pokemon Red/Blue.</div>';
                return;
            }
            
            const hpColor = (percent) => {
                if (percent > 75) return 'hp-excellent';
                if (percent > 50) return 'hp-good';
                if (percent > 25) return 'hp-okay';
                if (percent > 0) return 'hp-critical';
                return 'hp-fainted';
            };
            
            let html = ` + "`" + `
                <div class="status-card">
                    <h2>🎮 Game Status</h2>
                    <div class="player-info">
                        <div class="info-card">
                            <h3>👤 Trainer</h3>
                            <p><strong>Name:</strong> ${data.player_name}</p>
                            <p><strong>ID:</strong> ${data.player_id}</p>
                            <p><strong>Money:</strong> $${data.money.toLocaleString()}</p>
                        </div>
                        <div class="info-card">
                            <h3>📍 Location</h3>
                            <p><strong>Area:</strong> ${data.location_name}</p>
                            <p><strong>Position:</strong> (${data.player_x}, ${data.player_y})</p>
                        </div>
                        <div class="info-card">
                            <h3>⏰ Playtime</h3>
                            <p><strong>Time:</strong> ${String(data.hours).padStart(2, '0')}:${String(data.minutes).padStart(2, '0')}</p>
                        </div>
                        <div class="info-card">
                            <h3>📚 Pokedex</h3>
                            <p><strong>Seen:</strong> ${data.pokedex_seen}</p>
                            <p><strong>Caught:</strong> ${data.pokedex_caught}</p>
                        </div>
                    </div>
                </div>
            ` + "`" + `;
            
            if (data.badges && data.badges.length > 0) {
                html += ` + "`" + `
                    <div class="status-card">
                        <h2>🏆 Gym Badges</h2>
                        <div class="badges">
                            ${data.badges.map(badge => ` + "`" + `
                                <div class="badge ${badge.obtained ? 'obtained' : 'not-obtained'}">
                                    ${badge.obtained ? '✅' : '❌'}<br>
                                    ${badge.name.replace(' Badge', '')}
                                </div>
                            ` + "`" + `).join('')}
                        </div>
                    </div>
                ` + "`" + `;
            }
            
            if (data.pokemon && data.pokemon.length > 0) {
                html += ` + "`" + `
                    <div class="status-card">
                        <h2>🐾 Pokemon Team</h2>
                        <div class="pokemon-grid">
                            ${data.pokemon.map(pokemon => ` + "`" + `
                                <div class="pokemon-card">
                                    <div class="pokemon-header">
                                        <div class="pokemon-name">${pokemon.name}</div>
                                        <div class="pokemon-level">Lv. ${pokemon.level}</div>
                                    </div>
                                    <p><strong>Type:</strong> ${pokemon.type1_name}${pokemon.type1_name !== pokemon.type2_name ? '/' + pokemon.type2_name : ''}</p>
                                    <p><strong>Status:</strong> ${pokemon.status_name}</p>
                                    
                                    <div class="hp-bar">
                                        <div class="hp-fill ${hpColor(pokemon.hp_percent)}" style="width: ${pokemon.hp_percent}%"></div>
                                    </div>
                                    <p><strong>HP:</strong> ${pokemon.current_hp}/${pokemon.max_hp} (${pokemon.hp_percent.toFixed(1)}%)</p>
                                    
                                    <div class="stats-grid">
                                        <div class="stat"><strong>ATK:</strong> ${pokemon.attack}</div>
                                        <div class="stat"><strong>DEF:</strong> ${pokemon.defense}</div>
                                        <div class="stat"><strong>SPD:</strong> ${pokemon.speed}</div>
                                        <div class="stat"><strong>SPC:</strong> ${pokemon.special}</div>
                                    </div>
                                    
                                    <p><strong>EXP:</strong> ${pokemon.exp_points.toLocaleString()}</p>
                                    
                                    ${pokemon.moves && pokemon.moves.length > 0 ? ` + "`" + `
                                        <div class="moves">
                                            <strong>Moves:</strong><br>
                                            ${pokemon.moves.map(move => ` + "`" + `<span class="move-tag">${move.name}</span>` + "`" + `).join('')}
                                        </div>
                                    ` + "`" + ` : ''}
                                </div>
                            ` + "`" + `).join('')}
                        </div>
                    </div>
                ` + "`" + `;
            }
            
            if (data.bag_items && data.bag_items.length > 0) {
                html += ` + "`" + `
                    <div class="status-card">
                        <h2>🎒 Bag Items</h2>
                        <div class="pokemon-grid">
                            ${data.bag_items.map(item => ` + "`" + `
                                <div class="info-card">
                                    <strong>${item.name}</strong><br>
                                    Quantity: ${item.quantity}
                                </div>
                            ` + "`" + `).join('')}
                        </div>
                    </div>
                ` + "`" + `;
            }
            
            content.innerHTML = html;
        }
        
        function displayError(message) {
            document.getElementById('content').innerHTML = ` + "`" + `<div class="error">${message}</div>` + "`" + `;
        }
        
        // Load data on page load
        loadGameData();
        
        // Refresh every 5 seconds
        setInterval(loadGameData, 5000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, tmpl)
}

func (s *PokemonWebServer) handleLivePage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pokemon Live Data</title>
    <style>
        /* Same styles as home page */
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #fff;
        }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 40px; }
        .header h1 { font-size: 3rem; margin-bottom: 10px; text-shadow: 2px 2px 4px rgba(0,0,0,0.3); }
        .status-indicator { 
            display: inline-block; 
            width: 10px; 
            height: 10px; 
            border-radius: 50%; 
            margin-right: 10px;
        }
        .connected { background: #4CAF50; }
        .disconnected { background: #F44336; }
        .live-data { 
            background: rgba(255,255,255,0.1); 
            backdrop-filter: blur(10px); 
            border-radius: 15px; 
            padding: 30px; 
            margin-bottom: 30px; 
            border: 1px solid rgba(255,255,255,0.2); 
        }
        .btn {
            background: #6C63FF;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 25px;
            cursor: pointer;
            text-decoration: none;
            font-size: 1rem;
            transition: all 0.3s ease;
            display: inline-block;
            margin: 5px;
        }
        .btn:hover { background: #5A52E5; transform: translateY(-2px); }
        pre { background: rgba(0,0,0,0.3); padding: 20px; border-radius: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📡 Pokemon Live Data</h1>
            <p>Real-time WebSocket connection</p>
        </div>
        
        <div style="text-align: center; margin-bottom: 30px;">
            <a href="/" class="btn">📊 Back to Overview</a>
            <button onclick="toggleConnection()" class="btn" id="connectBtn">🔌 Connect</button>
            <button onclick="clearLog()" class="btn">🗑️ Clear Log</button>
        </div>
        
        <div class="live-data">
            <h2>
                <span id="status-indicator" class="status-indicator disconnected"></span>
                WebSocket Status: <span id="connection-status">Disconnected</span>
            </h2>
            <p>Messages: <span id="message-count">0</span></p>
            <p>Last update: <span id="last-update">Never</span></p>
        </div>
        
        <div class="live-data">
            <h3>📊 Live Pokemon Data</h3>
            <pre id="pokemon-data">Connecting...</pre>
        </div>
        
        <div class="live-data">
            <h3>📝 WebSocket Log</h3>
            <pre id="websocket-log">WebSocket log will appear here...</pre>
        </div>
    </div>

    <script>
        let ws = null;
        let messageCount = 0;
        let isConnected = false;
        
        function updateStatus(connected) {
            isConnected = connected;
            const indicator = document.getElementById('status-indicator');
            const status = document.getElementById('connection-status');
            const connectBtn = document.getElementById('connectBtn');
            
            if (connected) {
                indicator.className = 'status-indicator connected';
                status.textContent = 'Connected';
                connectBtn.textContent = '🔌 Disconnect';
            } else {
                indicator.className = 'status-indicator disconnected';
                status.textContent = 'Disconnected';
                connectBtn.textContent = '🔌 Connect';
            }
        }
        
        function addToLog(message) {
            const log = document.getElementById('websocket-log');
            const timestamp = new Date().toLocaleTimeString();
            log.textContent += ` + "`" + `[${timestamp}] ${message}\n` + "`" + `;
            log.scrollTop = log.scrollHeight;
        }
        
        function connectWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = ` + "`" + `${protocol}//${window.location.host}/ws` + "`" + `;
            
            addToLog('Connecting to ' + wsUrl);
            
            ws = new WebSocket(wsUrl);
            
            ws.onopen = function() {
                updateStatus(true);
                addToLog('WebSocket connected successfully');
                
                // Request status
                ws.send(JSON.stringify({
                    type: 'get_status'
                }));
            };
            
            ws.onmessage = function(event) {
                messageCount++;
                document.getElementById('message-count').textContent = messageCount;
                document.getElementById('last-update').textContent = new Date().toLocaleString();
                
                try {
                    const message = JSON.parse(event.data);
                    addToLog(` + "`" + `Received: ${message.type}` + "`" + `);
                    
                    if (message.type === 'pokemon_update') {
                        document.getElementById('pokemon-data').textContent = JSON.stringify(message.data, null, 2);
                    }
                } catch (e) {
                    addToLog('Failed to parse message: ' + e.message);
                }
            };
            
            ws.onclose = function() {
                updateStatus(false);
                addToLog('WebSocket connection closed');
            };
            
            ws.onerror = function(error) {
                addToLog('WebSocket error: ' + error.message);
            };
        }
        
        function disconnectWebSocket() {
            if (ws) {
                ws.close();
                ws = null;
            }
        }
        
        function toggleConnection() {
            if (isConnected) {
                disconnectWebSocket();
            } else {
                connectWebSocket();
            }
        }
        
        function clearLog() {
            document.getElementById('websocket-log').textContent = '';
            messageCount = 0;
            document.getElementById('message-count').textContent = '0';
        }
        
        // Auto-connect on page load
        connectWebSocket();
        
        // Fetch and display Pokemon data via REST API
        async function fetchPokemonData() {
            try {
                const response = await fetch('/api/gamedata');
                const data = await response.json();
                document.getElementById('pokemon-data').textContent = JSON.stringify(data, null, 2);
            } catch (error) {
                document.getElementById('pokemon-data').textContent = 'Error fetching data: ' + error.message;
            }
        }
        
        // Refresh Pokemon data every 2 seconds
        setInterval(fetchPokemonData, 2000);
        fetchPokemonData(); // Initial load
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, tmpl)
}

// monitorPokemonData continuously reads Pokemon data and broadcasts changes
func (s *PokemonWebServer) monitorPokemonData() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("🔄 Starting Pokemon data monitoring...")

	for {
		select {
		case <-ticker.C:
			newData := s.readCompleteGameData()
			if newData != nil {
				// Check for changes and broadcast via WebSocket
				if s.hasDataChanged(newData) {
					s.gameData = newData
					s.gameData.LastUpdated = time.Now()

					// Broadcast update to WebSocket clients
					s.wsManager.BroadcastMessage(server.Message{
						Type:      "pokemon_update",
						Data:      s.gameData,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}
}

// hasDataChanged compares new data with existing data
func (s *PokemonWebServer) hasDataChanged(newData *GameData) bool {
	if s.gameData == nil {
		return true
	}

	// Simple comparison - might want a more sophisticated change detection
	return s.gameData.PlayerName != newData.PlayerName ||
		s.gameData.Money != newData.Money ||
		s.gameData.LocationName != newData.LocationName ||
		len(s.gameData.Pokemon) != len(newData.Pokemon) ||
		(len(newData.Pokemon) > 0 && len(s.gameData.Pokemon) > 0 &&
			s.gameData.Pokemon[0].CurrentHP != newData.Pokemon[0].CurrentHP)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	server := NewPokemonWebServer()
	server.Start("8080")
}

// Pokemon Red/Blue Memory Layout
const (
	// Player data
	PLAYER_NAME_ADDR = 0xD158 // Player name (11 bytes)
	PLAYER_ID_ADDR   = 0xD359 // Player ID (2 bytes)
	MONEY_ADDR       = 0xD347 // Money (3 bytes, BCD)
	TEAM_COUNT_ADDR  = 0xD163 // Number of Pokemon in party

	// Overworld
	CURRENT_MAP_ADDR = 0xD35E // Current map ID
	PLAYER_X_ADDR    = 0xD362 // Player X position
	PLAYER_Y_ADDR    = 0xD361 // Player Y position

	// Badges (bitfield at 0xD356)
	BADGES_ADDR = 0xD356

	// Pokedex
	POKEDEX_SEEN_ADDR   = 0xD30A // Seen Pokemon (19 bytes bitarray)
	POKEDEX_CAUGHT_ADDR = 0xD2F7 // Caught Pokemon (19 bytes bitarray)

	// Game time addresses
	GAME_HOURS_ADDR   = 0xDA40 // Hours (2 bytes, big endian)
	GAME_MINUTES_ADDR = 0xDA45 // Minutes (1 byte) - CORRECTED from XML
	GAME_SECONDS_ADDR = 0xDA44 // Seconds (1 byte)
	GAME_FRAMES_ADDR  = 0xDA45 // Frames (1 byte)

	// Bag
	BAG_ITEM_COUNT_ADDR = 0xD31D // Number of items in bag
	BAG_ITEMS_ADDR      = 0xD31E // Start of bag items (2 bytes per item)

	// Pokemon party base addresses
	POKEMON_1_ADDR = 0xD16B // Pokemon #1 base address
	POKEMON_2_ADDR = 0xD197 // Pokemon #2 base address
	POKEMON_3_ADDR = 0xD1C3 // Pokemon #3 base address
	POKEMON_4_ADDR = 0xD1EF // Pokemon #4 base address
	POKEMON_5_ADDR = 0xD21B // Pokemon #5 base address
	POKEMON_6_ADDR = 0xD247 // Pokemon #6 base address

	// Pokemon structure offsets
	OFFSET_SPECIES    = 0  // +0: Species ID
	OFFSET_CURRENT_HP = 2  // +2: Current HP (2 bytes)
	OFFSET_LEVEL      = 33 // +33: Level
	OFFSET_MAX_HP     = 35 // +35: Max HP (2 bytes)
	OFFSET_ATTACK     = 37 // +37: Attack stat (2 bytes)
	OFFSET_DEFENSE    = 39 // +39: Defense stat (2 bytes)
	OFFSET_SPEED      = 41 // +41: Speed stat (2 bytes)
	OFFSET_SPECIAL    = 43 // +43: Special stat (2 bytes)
	OFFSET_STATUS     = 4  // +4: Status condition
	OFFSET_TYPE1      = 5  // +5: Type 1
	OFFSET_TYPE2      = 6  // +6: Type 2
	OFFSET_MOVES      = 8  // +8: Moves (4 bytes)
	OFFSET_EXP_POINTS = 14 // +14: Experience points (3 bytes)

	// Battle data (when in battle)
	BATTLE_MODE_ADDR = 0xD057 // Battle mode
	BATTLE_TYPE_ADDR = 0xD05A // Battle type
)

func (s *PokemonWebServer) readCompleteGameData() *GameData {
	data := &GameData{}

	// Read player data
	if nameBytes, err := s.driver.ReadMemory(PLAYER_NAME_ADDR, 11); err == nil {
		data.PlayerName = convertPokemonText(nameBytes)
	}

	if idBytes, err := s.driver.ReadMemory(PLAYER_ID_ADDR, 2); err == nil {
		data.PlayerID = binary.LittleEndian.Uint16(idBytes)
	}

	if moneyBytes, err := s.driver.ReadMemory(MONEY_ADDR, 3); err == nil {
		data.Money = decodeBCD(moneyBytes)
	}

	if teamBytes, err := s.driver.ReadMemory(TEAM_COUNT_ADDR, 1); err == nil {
		data.TeamCount = teamBytes[0]
	}

	if mapBytes, err := s.driver.ReadMemory(CURRENT_MAP_ADDR, 1); err == nil {
		data.CurrentMap = mapBytes[0]
		data.LocationName = getLocationName(data.CurrentMap)
	}

	if xBytes, err := s.driver.ReadMemory(PLAYER_X_ADDR, 1); err == nil {
		data.PlayerX = xBytes[0]
	}

	if yBytes, err := s.driver.ReadMemory(PLAYER_Y_ADDR, 1); err == nil {
		data.PlayerY = yBytes[0]
	}

	data.Badges = s.readBadges()

	data.PokedexSeen, data.PokedexCaught = s.readPokedexCounts()

	// Read game time (corrected format)
	if hoursBytes, err := s.driver.ReadMemory(GAME_HOURS_ADDR, 2); err == nil {
		// Hours stored as 2 bytes, big endian works better
		data.Hours = binary.BigEndian.Uint16(hoursBytes)
	}

	if minutesBytes, err := s.driver.ReadMemory(GAME_MINUTES_ADDR, 1); err == nil {
		// Minutes stored as single byte
		data.Minutes = uint16(minutesBytes[0])
	}

	if secondsBytes, err := s.driver.ReadMemory(GAME_SECONDS_ADDR, 1); err == nil {
		data.Seconds = secondsBytes[0]
	}

	data.BagItems = s.readBagItems()
	data.BagItemCount = uint8(len(data.BagItems))

	pokemonAddresses := []uint32{
		POKEMON_1_ADDR, POKEMON_2_ADDR, POKEMON_3_ADDR,
		POKEMON_4_ADDR, POKEMON_5_ADDR, POKEMON_6_ADDR,
	}

	data.Pokemon = make([]Pokemon, 0, data.TeamCount)

	for i := 0; i < int(data.TeamCount) && i < len(pokemonAddresses); i++ {
		pokemon := s.readPokemon(pokemonAddresses[i])
		if pokemon != nil {
			data.Pokemon = append(data.Pokemon, *pokemon)
		}
	}

	if battleModeBytes, err := s.driver.ReadMemory(BATTLE_MODE_ADDR, 1); err == nil {
		data.BattleMode = getBattleMode(battleModeBytes[0])
	}

	if battleTypeBytes, err := s.driver.ReadMemory(BATTLE_TYPE_ADDR, 1); err == nil {
		data.BattleType = getBattleType(battleTypeBytes[0])
	}

	return data
}

func (s *PokemonWebServer) readPokemon(baseAddr uint32) *Pokemon {
	pokemon := &Pokemon{}

	// Read species
	if speciesBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_SPECIES, 1); err == nil {
		pokemon.Species = speciesBytes[0]
		pokemon.Name = getPokemonName(pokemon.Species)
		pokemon.PokedexNumber = getPokedexNumber(pokemon.Species)
	} else {
		return nil
	}

	// Read current HP (2 bytes, little endian)
	if hpBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_CURRENT_HP, 2); err == nil {
		pokemon.CurrentHP = binary.LittleEndian.Uint16(hpBytes)
	}

	// Read max HP (2 bytes, little endian)
	if maxHPBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_MAX_HP, 2); err == nil {
		pokemon.MaxHP = binary.LittleEndian.Uint16(maxHPBytes)
	}

	// Read level
	if levelBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_LEVEL, 1); err == nil {
		pokemon.Level = levelBytes[0]
	}

	// Read stats (all 2 bytes, little endian)
	if attackBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_ATTACK, 2); err == nil {
		pokemon.Attack = binary.LittleEndian.Uint16(attackBytes)
	}

	if defenseBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_DEFENSE, 2); err == nil {
		pokemon.Defense = binary.LittleEndian.Uint16(defenseBytes)
	}

	if speedBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_SPEED, 2); err == nil {
		pokemon.Speed = binary.LittleEndian.Uint16(speedBytes)
	}

	if specialBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_SPECIAL, 2); err == nil {
		pokemon.Special = binary.LittleEndian.Uint16(specialBytes)
	}

	// Read status and types
	if statusBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_STATUS, 1); err == nil {
		pokemon.Status = statusBytes[0]
		pokemon.StatusName = getStatusCondition(pokemon.Status)
	}

	if type1Bytes, err := s.driver.ReadMemory(baseAddr+OFFSET_TYPE1, 1); err == nil {
		pokemon.Type1 = type1Bytes[0]
		pokemon.Type1Name = getTypeName(pokemon.Type1)
	}

	if type2Bytes, err := s.driver.ReadMemory(baseAddr+OFFSET_TYPE2, 1); err == nil {
		pokemon.Type2 = type2Bytes[0]
		pokemon.Type2Name = getTypeName(pokemon.Type2)
	}

	// Read moves (4 bytes)
	if moveBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_MOVES, 4); err == nil {
		pokemon.Moves = make([]Move, 0, 4)
		for i := 0; i < 4; i++ {
			if moveBytes[i] != 0 {
				pokemon.Moves = append(pokemon.Moves, Move{
					ID:   moveBytes[i],
					Name: getMoveName(moveBytes[i]),
				})
			}
		}
	}

	// Read experience points (3 bytes, big endian format)
	if expBytes, err := s.driver.ReadMemory(baseAddr+OFFSET_EXP_POINTS, 3); err == nil {
		// Experience is stored as big endian binary (not BCD) in Pokemon Red/Blue
		pokemon.ExpPoints = uint32(expBytes[2]) | uint32(expBytes[1])<<8 | uint32(expBytes[0])<<16
	}

	// Calculate HP percentage
	if pokemon.MaxHP > 0 {
		pokemon.HPPercent = (float64(pokemon.CurrentHP) / float64(pokemon.MaxHP)) * 100
	}

	return pokemon
}

func (s *PokemonWebServer) readBadges() []Badge {
	badges := []Badge{
		{"Boulder Badge", false},
		{"Cascade Badge", false},
		{"Thunder Badge", false},
		{"Rainbow Badge", false},
		{"Soul Badge", false},
		{"Marsh Badge", false},
		{"Volcano Badge", false},
		{"Earth Badge", false},
	}

	if badgeBytes, err := s.driver.ReadMemory(BADGES_ADDR, 1); err == nil {
		badgeByte := badgeBytes[0]
		for i := 0; i < 8; i++ {
			badges[i].Obtained = (badgeByte & (1 << i)) != 0
		}
	}

	return badges
}

func (s *PokemonWebServer) readPokedexCounts() (int, int) {
	seen := 0
	caught := 0

	// Count seen Pokemon (19 bytes bitarray)
	if seenBytes, err := s.driver.ReadMemory(POKEDEX_SEEN_ADDR, 19); err == nil {
		for _, b := range seenBytes {
			for i := 0; i < 8; i++ {
				if (b & (1 << i)) != 0 {
					seen++
				}
			}
		}
	}

	// Count caught Pokemon (19 bytes bitarray)
	if caughtBytes, err := s.driver.ReadMemory(POKEDEX_CAUGHT_ADDR, 19); err == nil {
		for _, b := range caughtBytes {
			for i := 0; i < 8; i++ {
				if (b & (1 << i)) != 0 {
					caught++
				}
			}
		}
	}

	return seen, caught
}

func (s *PokemonWebServer) readBagItems() []Item {
	items := []Item{}

	if countBytes, err := s.driver.ReadMemory(BAG_ITEM_COUNT_ADDR, 1); err == nil {
		itemCount := countBytes[0]
		if itemCount > 20 {
			itemCount = 20
		}

		for i := 0; i < int(itemCount); i++ {
			itemAddr := BAG_ITEMS_ADDR + uint32(i*2)
			if itemBytes, err := s.driver.ReadMemory(itemAddr, 2); err == nil {
				if itemBytes[0] != 0 {
					items = append(items, Item{
						ID:       itemBytes[0],
						Name:     getItemName(itemBytes[0]),
						Quantity: itemBytes[1],
					})
				}
			}
		}
	}

	return items
}

// Helper functions
func convertPokemonText(data []byte) string {
	pokemonChars := map[byte]string{
		0x80: "A", 0x81: "B", 0x82: "C", 0x83: "D", 0x84: "E", 0x85: "F", 0x86: "G",
		0x87: "H", 0x88: "I", 0x89: "J", 0x8A: "K", 0x8B: "L", 0x8C: "M", 0x8D: "N",
		0x8E: "O", 0x8F: "P", 0x90: "Q", 0x91: "R", 0x92: "S", 0x93: "T", 0x94: "U",
		0x95: "V", 0x96: "W", 0x97: "X", 0x98: "Y", 0x99: "Z", 0x9A: "(", 0x9B: ")",
		0x9C: ":", 0x9D: ";", 0x9E: "[", 0x9F: "]", 0xA0: "a", 0xA1: "b", 0xA2: "c",
		0xA3: "d", 0xA4: "e", 0xA5: "f", 0xA6: "g", 0xA7: "h", 0xA8: "i", 0xA9: "j",
		0xAA: "k", 0xAB: "l", 0xAC: "m", 0xAD: "n", 0xAE: "o", 0xAF: "p", 0xB0: "q",
		0xB1: "r", 0xB2: "s", 0xB3: "t", 0xB4: "u", 0xB5: "v", 0xB6: "w", 0xB7: "x",
		0xB8: "y", 0xB9: "z", 0xF6: "0", 0xF7: "1", 0xF8: "2", 0xF9: "3", 0xFA: "4",
		0xFB: "5", 0xFC: "6", 0xFD: "7", 0xFE: "8", 0xFF: "9", 0x50: " ", 0x00: "",
		0xEF: "♂", 0xF5: "♀",
	}

	result := ""
	for _, b := range data {
		if b == 0x50 || b == 0x00 {
			break
		}
		if char, exists := pokemonChars[b]; exists {
			result += char
		} else {
			result += "?"
		}
	}
	return result
}

// Complete Pokemon species mapping
func getPokemonName(species uint8) string {
	names := map[uint8]string{
		0x99: "Bulbasaur", 0x09: "Ivysaur", 0x9A: "Venusaur",
		0xB0: "Charmander", 0xB2: "Charmeleon", 0xB4: "Charizard",
		0xB1: "Squirtle", 0xB3: "Wartortle", 0x1C: "Blastoise",
		0x7B: "Caterpie", 0x7C: "Metapod", 0x7D: "Butterfree",
		0x70: "Weedle", 0x71: "Kakuna", 0x72: "Beedrill",
		0x24: "Pidgey", 0x96: "Pidgeotto", 0x97: "Pidgeot",
		0xA5: "Rattata", 0xA6: "Raticate", 0x05: "Spearow", 0x23: "Fearow",
		0x6C: "Ekans", 0x2D: "Arbok", 0x54: "Pikachu", 0x55: "Raichu",
		0x60: "Sandshrew", 0x61: "Sandslash", 0x0F: "Nidoran♀", 0xA8: "Nidorina",
		0x10: "Nidoqueen", 0x03: "Nidoran♂", 0xA7: "Nidorino", 0x07: "Nidoking",
		0x04: "Clefairy", 0x8E: "Clefable", 0x52: "Vulpix", 0x53: "Ninetales",
		0x64: "Jigglypuff", 0x65: "Wigglytuff", 0x6B: "Zubat", 0x82: "Golbat",
		0xB9: "Oddish", 0xBA: "Gloom", 0xBB: "Vileplume", 0x6D: "Paras", 0x2E: "Parasect",
		0x41: "Venonat", 0x77: "Venomoth", 0x3B: "Diglett", 0x76: "Dugtrio",
		0x4D: "Meowth", 0x90: "Persian", 0x2F: "Psyduck", 0x80: "Golduck",
		0x39: "Mankey", 0x75: "Primeape", 0x21: "Growlithe", 0x14: "Arcanine",
		0x47: "Poliwag", 0x6E: "Poliwhirl", 0x6F: "Poliwrath", 0x94: "Abra",
		0x26: "Kadabra", 0x95: "Alakazam", 0x6A: "Machop", 0x29: "Machoke",
		0x7E: "Machamp", 0xBC: "Bellsprout", 0xBD: "Weepinbell", 0xBE: "Victreebel",
		0x18: "Tentacool", 0x9B: "Tentacruel", 0xA9: "Geodude", 0x27: "Graveler",
		0x31: "Golem", 0xA3: "Ponyta", 0xA4: "Rapidash", 0x25: "Slowpoke",
		0x08: "Slowbro", 0xAD: "Magnemite", 0x36: "Magneton", 0x40: "Farfetch'd",
		0x46: "Doduo", 0x74: "Dodrio", 0x3A: "Seel", 0x78: "Dewgong",
		0x0D: "Grimer", 0x88: "Muk", 0x17: "Shellder", 0x8B: "Cloyster",
		0x19: "Gastly", 0x93: "Haunter", 0x0E: "Gengar", 0x22: "Onix",
		0x30: "Drowzee", 0x81: "Hypno", 0x4E: "Krabby", 0x8A: "Kingler",
		0x06: "Voltorb", 0x8D: "Electrode", 0x0C: "Exeggcute", 0x0A: "Exeggutor",
		0x11: "Cubone", 0x91: "Marowak", 0x2B: "Hitmonlee", 0x2C: "Hitmonchan",
		0x0B: "Lickitung", 0x37: "Koffing", 0x8F: "Weezing", 0x01: "Rhydon",
		0x12: "Rhyhorn", 0x28: "Chansey", 0x1E: "Tangela", 0x02: "Kangaskhan",
		0x5C: "Horsea", 0x5D: "Seadra", 0x9D: "Goldeen", 0x9E: "Seaking",
		0x98: "Starmie", 0x1B: "Staryu", 0x2A: "Mr. Mime", 0x1A: "Scyther",
		0x48: "Jynx", 0x35: "Electabuzz", 0x33: "Magmar", 0x1D: "Pinsir",
		0x3C: "Tauros", 0x85: "Magikarp", 0x16: "Gyarados", 0x13: "Lapras",
		0x4C: "Ditto", 0x66: "Eevee", 0x69: "Vaporeon", 0x68: "Jolteon",
		0x67: "Flareon", 0xAA: "Porygon", 0x62: "Omanyte", 0x63: "Omastar",
		0x5A: "Kabuto", 0x5B: "Kabutops", 0xAB: "Aerodactyl", 0x84: "Snorlax",
		0x4A: "Articuno", 0x4B: "Zapdos", 0x49: "Moltres", 0x58: "Dratini",
		0x59: "Dragonair", 0x42: "Dragonite", 0x83: "Mewtwo", 0x15: "Mew",
	}

	if name, exists := names[species]; exists {
		return name
	}
	return fmt.Sprintf("Pokemon #%d", species)
}

// Pokedex number mapping
func getPokedexNumber(species uint8) uint8 {
	numbers := map[uint8]uint8{
		0x99: 1, 0x09: 2, 0x9A: 3, 0xB0: 4, 0xB2: 5, 0xB4: 6,
		0xB1: 7, 0xB3: 8, 0x1C: 9, 0x7B: 10, 0x7C: 11, 0x7D: 12,
		0x70: 13, 0x71: 14, 0x72: 15, 0x24: 16, 0x96: 17, 0x97: 18,
		0xA5: 19, 0xA6: 20, 0x05: 21, 0x23: 22, 0x6C: 23, 0x2D: 24,
		0x54: 25, 0x55: 26, 0x60: 27, 0x61: 28, 0x0F: 29, 0xA8: 30,
		0x10: 31, 0x03: 32, 0xA7: 33, 0x07: 34, 0x04: 35, 0x8E: 36,
		0x52: 37, 0x53: 38, 0x64: 39, 0x65: 40, 0x6B: 41, 0x82: 42,
		0xB9: 43, 0xBA: 44, 0xBB: 45, 0x6D: 46, 0x2E: 47, 0x41: 48,
		0x77: 49, 0x3B: 50, 0x76: 51, 0x4D: 52, 0x90: 53, 0x2F: 54,
		0x80: 55, 0x39: 56, 0x75: 57, 0x21: 58, 0x14: 59, 0x47: 60,
		0x6E: 61, 0x6F: 62, 0x94: 63, 0x26: 64, 0x95: 65, 0x6A: 66,
		0x29: 67, 0x7E: 68, 0xBC: 69, 0xBD: 70, 0xBE: 71, 0x18: 72,
		0x9B: 73, 0xA9: 74, 0x27: 75, 0x31: 76, 0xA3: 77, 0xA4: 78,
		0x25: 79, 0x08: 80, 0xAD: 81, 0x36: 82, 0x40: 83, 0x46: 84,
		0x74: 85, 0x3A: 86, 0x78: 87, 0x0D: 88, 0x88: 89, 0x17: 90,
		0x8B: 91, 0x19: 92, 0x93: 93, 0x0E: 94, 0x22: 95, 0x30: 96,
		0x81: 97, 0x4E: 98, 0x8A: 99, 0x06: 100, 0x8D: 101, 0x0C: 102,
		0x0A: 103, 0x11: 104, 0x91: 105, 0x2B: 106, 0x2C: 107, 0x0B: 108,
		0x37: 109, 0x8F: 110, 0x01: 111, 0x12: 112, 0x28: 113, 0x1E: 114,
		0x02: 115, 0x5C: 116, 0x5D: 117, 0x9D: 118, 0x9E: 119, 0x1B: 120,
		0x98: 121, 0x2A: 122, 0x1A: 123, 0x48: 124, 0x35: 125, 0x33: 126,
		0x1D: 127, 0x3C: 128, 0x85: 129, 0x16: 130, 0x13: 131, 0x4C: 132,
		0x66: 133, 0x68: 134, 0x67: 135, 0x69: 136, 0xAA: 137, 0x62: 138,
		0x63: 139, 0x5A: 140, 0x5B: 141, 0xAB: 142, 0x84: 143, 0x4A: 144,
		0x4B: 145, 0x49: 146, 0x58: 147, 0x59: 148, 0x42: 149, 0x83: 150,
		0x15: 151,
	}

	if number, exists := numbers[species]; exists {
		return number
	}
	return 0
}

// Type mapping
func getTypeName(typeID uint8) string {
	types := map[uint8]string{
		0x00: "Normal", 0x01: "Fighting", 0x02: "Flying", 0x03: "Poison",
		0x04: "Ground", 0x05: "Rock", 0x07: "Bug", 0x08: "Ghost",
		0x14: "Fire", 0x15: "Water", 0x16: "Grass", 0x17: "Electric",
		0x18: "Psychic", 0x19: "Ice", 0x1A: "Dragon",
	}
	if name, exists := types[typeID]; exists {
		return name
	}
	return "Unknown"
}

// Status conditions
func getStatusCondition(status uint8) string {
	switch status {
	case 0x00:
		return "Normal"
	case 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07:
		return "Asleep"
	case 0x08:
		return "Poisoned"
	case 0x10:
		return "Burned"
	case 0x20:
		return "Frozen"
	case 0x40:
		return "Paralyzed"
	default:
		return "Unknown"
	}
}

// Battle modes
func getBattleMode(mode uint8) string {
	switch mode {
	case 0x00:
		return "None"
	case 0x01:
		return "Wild"
	case 0x02:
		return "Trainer"
	case 0xFF:
		return "Lost Battle"
	default:
		return "Unknown"
	}
}

// Battle types
func getBattleType(battleType uint8) string {
	switch battleType {
	case 0x00:
		return "Normal"
	case 0x01:
		return "Old Man Battle"
	case 0x02:
		return "Safari Zone"
	case 0x04:
		return "Oak Catching Starter"
	default:
		return "Unknown"
	}
}

// Complete move names
func getMoveName(moveID uint8) string {
	moves := map[uint8]string{
		0x00: "", 0x01: "Pound", 0x02: "Karate Chop", 0x03: "DoubleSlap",
		0x04: "Comet Punch", 0x05: "Mega Punch", 0x06: "Pay Day", 0x07: "Fire Punch",
		0x08: "Ice Punch", 0x09: "ThunderPunch", 0x0A: "Scratch", 0x0B: "ViceGrip",
		0x0C: "Guillotine", 0x0D: "Razor Wind", 0x0E: "Swords Dance", 0x0F: "Cut",
		0x10: "Gust", 0x11: "Wing Attack", 0x12: "Whirlwind", 0x13: "Fly",
		0x14: "Bind", 0x15: "Slam", 0x16: "Vine Whip", 0x17: "Stomp",
		0x18: "Double Kick", 0x19: "Mega Kick", 0x1A: "Jump Kick", 0x1B: "Rolling Kick",
		0x1C: "Sand-Attack", 0x1D: "Headbutt", 0x1E: "Horn Attack", 0x1F: "Fury Attack",
		0x20: "Horn Drill", 0x21: "Tackle", 0x22: "Body Slam", 0x23: "Wrap",
		0x24: "Take Down", 0x25: "Thrash", 0x26: "Double-Edge", 0x27: "Tail Whip",
		0x28: "Poison Sting", 0x29: "Twineedle", 0x2A: "Pin Missile", 0x2B: "Leer",
		0x2C: "Bite", 0x2D: "Growl", 0x2E: "Roar", 0x2F: "Sing",
		0x30: "Supersonic", 0x31: "SonicBoom", 0x32: "Disable", 0x33: "Acid",
		0x34: "Ember", 0x35: "Flamethrower", 0x36: "Mist", 0x37: "Water Gun",
		0x38: "Hydro Pump", 0x39: "Surf", 0x3A: "Ice Beam", 0x3B: "Blizzard",
		0x3C: "Psybeam", 0x3D: "BubbleBeam", 0x3E: "Aurora Beam", 0x3F: "Hyper Beam",
		0x40: "Peck", 0x41: "Drill Peck", 0x42: "Submission", 0x43: "Low Kick",
		0x44: "Counter", 0x45: "Seismic Toss", 0x46: "Strength", 0x47: "Absorb",
		0x48: "Mega Drain", 0x49: "Leech Seed", 0x4A: "Growth", 0x4B: "Razor Leaf",
		0x4C: "SolarBeam", 0x4D: "PoisonPowder", 0x4E: "Stun Spore", 0x4F: "Sleep Powder",
		0x50: "Petal Dance", 0x51: "String Shot", 0x52: "Dragon Rage", 0x53: "Fire Spin",
		0x54: "Thundershock", 0x55: "Thunderbolt", 0x56: "Thunder Wave", 0x57: "Thunder",
		0x58: "Rock Throw", 0x59: "Earthquake", 0x5A: "Fissure", 0x5B: "Dig",
		0x5C: "Toxic", 0x5D: "Confusion", 0x5E: "Psychic", 0x5F: "Hypnosis",
		0x60: "Meditate", 0x61: "Agility", 0x62: "Quick Attack", 0x63: "Rage",
		0x64: "Teleport", 0x65: "Night Shade", 0x66: "Mimic", 0x67: "Screech",
		0x68: "Double Team", 0x69: "Recover", 0x6A: "Harden", 0x6B: "Minimize",
		0x6C: "Smokescreen", 0x6D: "Confuse Ray", 0x6E: "Withdraw", 0x6F: "Defense Curl",
		0x70: "Barrier", 0x71: "Light Screen", 0x72: "Haze", 0x73: "Reflect",
		0x74: "Focus Energy", 0x75: "Bide", 0x76: "Metronome", 0x77: "Mirror Move",
		0x78: "Selfdestruct", 0x79: "Egg Bomb", 0x7A: "Lick", 0x7B: "Smog",
		0x7C: "Sludge", 0x7D: "Bone Club", 0x7E: "Fire Blast", 0x7F: "Waterfall",
		0x80: "Clamp", 0x81: "Swift", 0x82: "Skull Bash", 0x83: "Spike Cannon",
		0x84: "Constrict", 0x85: "Amnesia", 0x86: "Kinesis", 0x87: "Softboiled",
		0x88: "Hi Jump Kick", 0x89: "Glare", 0x8A: "Dream Eater", 0x8B: "Poison Gas",
		0x8C: "Barrage", 0x8D: "Leech Life", 0x8E: "Lovely Kiss", 0x8F: "Sky Attack",
		0x90: "Transform", 0x91: "Bubble", 0x92: "Dizzy Punch", 0x93: "Spore",
		0x94: "Flash", 0x95: "Psywave", 0x96: "Splash", 0x97: "Acid Armor",
		0x98: "Crabhammer", 0x99: "Explosion", 0x9A: "Fury Swipes", 0x9B: "Bonemerang",
		0x9C: "Rest", 0x9D: "Rock Slide", 0x9E: "Hyper Fang", 0x9F: "Sharpen",
		0xA0: "Conversion", 0xA1: "Tri Attack", 0xA2: "Super Fang", 0xA3: "Slash",
		0xA4: "Substitute", 0xA5: "Struggle",
	}

	if name, exists := moves[moveID]; exists {
		return name
	}
	return fmt.Sprintf("Move #%d", moveID)
}

// Complete item names
func getItemName(itemID uint8) string {
	items := map[uint8]string{
		0x00: "", 0x01: "MASTER BALL", 0x02: "ULTRA BALL", 0x03: "GREAT BALL",
		0x04: "POKé BALL", 0x05: "TOWN MAP", 0x06: "BICYCLE", 0x07: "?????",
		0x08: "SAFARI BALL", 0x09: "POKéDEX", 0x0A: "MOON STONE", 0x0B: "ANTIDOTE",
		0x0C: "BURN HEAL", 0x0D: "ICE HEAL", 0x0E: "AWAKENING", 0x0F: "PARLYZ HEAL",
		0x10: "FULL RESTORE", 0x11: "MAX POTION", 0x12: "HYPER POTION", 0x13: "SUPER POTION",
		0x14: "POTION", 0x15: "BOULDERBADGE", 0x16: "CASCADEBADGE", 0x17: "THUNDERBADGE",
		0x18: "RAINBOWBADGE", 0x19: "SOULBADGE", 0x1A: "MARSHBADGE", 0x1B: "VOLCANOBADGE",
		0x1C: "EARTHBADGE", 0x1D: "ESCAPE ROPE", 0x1E: "REPEL", 0x1F: "OLD AMBER",
		0x20: "FIRE STONE", 0x21: "THUNDERSTONE", 0x22: "WATER STONE", 0x23: "HP UP",
		0x24: "PROTEIN", 0x25: "IRON", 0x26: "CARBOS", 0x27: "CALCIUM",
		0x28: "RARE CANDY", 0x29: "DOME FOSSIL", 0x2A: "HELIX FOSSIL", 0x2B: "SECRET KEY",
		0x2C: "?????", 0x2D: "BIKE VOUCHER", 0x2E: "X ACCURACY", 0x2F: "LEAF STONE",
		0x30: "CARD KEY", 0x31: "NUGGET", 0x32: "PP UP", 0x33: "POKé DOLL",
		0x34: "FULL HEAL", 0x35: "REVIVE", 0x36: "MAX REVIVE", 0x37: "GUARD SPEC.",
		0x38: "SUPER REPEL", 0x39: "MAX REPEL", 0x3A: "DIRE HIT", 0x3B: "COIN",
		0x3C: "FRESH WATER", 0x3D: "SODA POP", 0x3E: "LEMONADE", 0x3F: "S.S.TICKET",
		0x40: "GOLD TEETH", 0x41: "X ATTACK", 0x42: "X DEFEND", 0x43: "X SPEED",
		0x44: "X SPECIAL", 0x45: "COIN CASE", 0x46: "OAK's PARCEL", 0x47: "ITEMFINDER",
		0x48: "SILPH SCOPE", 0x49: "POKé FLUTE", 0x4A: "LIFT KEY", 0x4B: "EXP.ALL",
		0x4C: "OLD ROD", 0x4D: "GOOD ROD", 0x4E: "SUPER ROD", 0x4F: "PP UP",
		0x50: "ETHER", 0x51: "MAX ETHER", 0x52: "ELIXER", 0x53: "MAX ELIXER",
		0xC4: "HM01: Cut", 0xC5: "HM02: Fly", 0xC6: "HM03: Surf", 0xC7: "HM04: Strength",
		0xC8: "HM05: Flash", 0xC9: "TM01: Mega Punch", 0xCA: "TM02: Razor Wind", 0xCB: "TM03: Swords Dance",
		0xCC: "TM04: Whirlwind", 0xCD: "TM05: Mega Kick", 0xCE: "TM06: Toxic", 0xCF: "TM07: Horn Drill",
		0xD0: "TM08: Body Slam", 0xD1: "TM09: Take Down", 0xD2: "TM10: Double-Edge", 0xD3: "TM11: BubbleBeam",
		0xD4: "TM12: Water Gun", 0xD5: "TM13: Ice Beam", 0xD6: "TM14: Blizzard", 0xD7: "TM15: Hyper Beam",
		0xD8: "TM16: Pay Day", 0xD9: "TM17: Submission", 0xDA: "TM18: Counter", 0xDB: "TM19: Seismic Toss",
		0xDC: "TM20: Rage", 0xDD: "TM21: Mega Drain", 0xDE: "TM22: SolarBeam", 0xDF: "TM23: Dragon Rage",
		0xE0: "TM24: Thunderbolt", 0xE1: "TM25: Thunder", 0xE2: "TM26: Earthquake", 0xE3: "TM27: Fissure",
		0xE4: "TM28: Dig", 0xE5: "TM29: Psychic", 0xE6: "TM30: Teleport", 0xE7: "TM31: Mimic",
		0xE8: "TM32: Double Team", 0xE9: "TM33: Reflect", 0xEA: "TM34: Bide", 0xEB: "TM35: Metronome",
		0xEC: "TM36: Selfdestruct", 0xED: "TM37: Egg Bomb", 0xEE: "TM38: Fire Blast", 0xEF: "TM39: Swift",
		0xF0: "TM40: Skull Bash", 0xF1: "TM41: Softboiled", 0xF2: "TM42: Dream Eater", 0xF3: "TM43: Sky Attack",
		0xF4: "TM44: Rest", 0xF5: "TM45: Thunder Wave", 0xF6: "TM46: Psywave", 0xF7: "TM47: Explosion",
		0xF8: "TM48: Rock Slide", 0xF9: "TM49: Tri Attack", 0xFA: "TM50: Substitute", 0xFF: "",
	}

	if name, exists := items[itemID]; exists {
		return name
	}
	return fmt.Sprintf("Item #%d", itemID)
}

// Map names
func getLocationName(mapID uint8) string {
	locations := map[uint8]string{
		0x00: "Pallet Town", 0x01: "Viridian City", 0x02: "Pewter City",
		0x03: "Cerulean City", 0x04: "Lavender Town", 0x05: "Vermilion City",
		0x06: "Celadon City", 0x07: "Fuchsia City", 0x08: "Cinnabar Island",
		0x09: "Indigo Plateau", 0x0A: "Saffron City", 0x0C: "Route 1",
		0x0D: "Route 2", 0x0E: "Route 3", 0x0F: "Route 4", 0x10: "Route 5",
		0x11: "Route 6", 0x12: "Route 7", 0x13: "Route 8", 0x14: "Route 9",
		0x15: "Route 10", 0x16: "Route 11", 0x17: "Route 12", 0x18: "Route 13",
		0x19: "Route 14", 0x1A: "Route 15", 0x1B: "Route 16", 0x1C: "Route 17",
		0x1D: "Route 18", 0x1E: "Route 19", 0x1F: "Route 20", 0x20: "Route 21",
		0x21: "Route 22", 0x22: "Route 23", 0x23: "Route 24", 0x24: "Route 25",
		0x25: "Red's House 1F", 0x26: "Red's House 2F", 0x27: "Blue's House",
		0x28: "Oak's Lab", 0x29: "Viridian Pokecenter", 0x2A: "Viridian Mart",
		0x2B: "Viridian School", 0x2C: "Viridian House", 0x2D: "Viridian Gym",
		0x33: "Viridian Forest", 0x34: "Pewter Museum 1F", 0x35: "Pewter Museum 2F",
		0x36: "Pewter Gym", 0x37: "Pewter House 1", 0x38: "Pewter Mart",
		0x39: "Pewter House 2", 0x3A: "Pewter Pokecenter", 0x3B: "Mt Moon 1",
		0x3C: "Mt Moon 2", 0x3D: "Mt Moon 3", 0x3E: "Cerulean Trashed House",
		0x3F: "Cerulean House", 0x40: "Cerulean Pokecenter", 0x41: "Cerulean Gym",
		0x42: "Cerulean Bike Shop", 0x43: "Cerulean Mart", 0x5C: "Vermilion Gym",
		0x85: "Celadon Gym", 0x9D: "Fuchsia Gym", 0xA6: "Cinnabar Gym",
		0xAE: "Indigo Plateau Lobby", 0xB2: "Saffron Gym",
	}
	if location, exists := locations[mapID]; exists {
		return location
	}
	return fmt.Sprintf("Map %d", mapID)
}

func decodeBCD(data []byte) uint32 {
	result := uint32(0)
	multiplier := uint32(1)

	for i := len(data) - 1; i >= 0; i-- {
		byte := data[i]

		digit := byte & 0x0F
		if digit <= 9 {
			result += uint32(digit) * multiplier
			multiplier *= 10
		}

		digit = (byte & 0xF0) >> 4
		if digit <= 9 {
			result += uint32(digit) * multiplier
			multiplier *= 10
		}
	}
	return result
}
