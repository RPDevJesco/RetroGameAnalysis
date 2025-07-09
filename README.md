# RetroGameAnalysis

🎮 **Modern retro game memory manipulation with advanced property management**

RetroGameAnalysis is a powerful, next-generation tool for interacting with retro game memory in real-time. Built in Go, it provides a sophisticated web-based interface for monitoring, modifying, and analyzing game state with unprecedented detail and control.

## ✨ What Makes RetroGameAnalysis Special

Unlike traditional memory editors that work with raw bytes and addresses, RetroGameAnalysis operates at a **property level**, treating game data as structured, typed information with rich metadata and validation rules.

### 🚀 Key Features

- **🔄 Real-time Monitoring** - 60fps property change detection and streaming
- **🧊 Property Freezing** - Lock values to prevent changes from the game
- **📊 Advanced Property Types** - Enums, flags, coordinates, colors, percentages, and more
- **⚡ Batch Operations** - Update multiple properties atomically
- **✅ Property Validation** - Enforce constraints and data integrity
- **📈 State Tracking** - Monitor read/write counts, history, and statistics
- **🎨 Rich UI Hints** - Enhanced metadata for beautiful interfaces
- **🔗 Reference Types** - Structured data definitions and lookups
- **📡 WebSocket API** - Real-time bidirectional communication
- **🎯 Event System** - Trigger-based automation and alerts

## 🎯 How It Works

### Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web Browser   │    │RetroGameAnalysis Server│    │   RetroArch     │
│                 │    │                  │    │                 │
│ ┌─────────────┐ │    │ ┌──────────────┐ │    │ ┌─────────────┐ │
│ │ Enhanced UI │ │◄──►│ │ REST API     │ │    │ │ Game Core   │ │
│ └─────────────┘ │    │ └──────────────┘ │    │ └─────────────┘ │
│                 │    │ ┌──────────────┐ │    │       ▲         │
│ ┌─────────────┐ │    │ │ WebSocket    │ │    │       │         │
│ │ Real-time   │ │◄──►│ │ Streaming    │ │    │ ┌─────▼─────┐   │
│ │ Updates     │ │    │ └──────────────┘ │    │ │ Memory    │   │
│ └─────────────┘ │    │ ┌──────────────┐ │◄──►│ │ Interface │   │
└─────────────────┘    │ │ Adaptive     │ │    │ └───────────┘   │
                       │ │ RetroArch    │ │    └─────────────────┘
                       │ │ Driver       │ │
                       │ └──────────────┘ │
                       │ ┌──────────────┐ │
                       │ │ Enhanced     │ │
                       │ │ Memory       │ │
                       │ │ Manager      │ │
                       │ └──────────────┘ │
                       └──────────────────┘
```

### Core Components

1. **Adaptive RetroArch Driver** - Optimized UDP communication with automatic chunking
2. **Enhanced Memory Manager** - Advanced caching, state tracking, and property management
3. **CUE-based Mappers** - Declarative game memory definitions with rich typing
4. **Property Engine** - Real-time monitoring, validation, and transformation
5. **WebSocket Streaming** - 60fps real-time updates to connected clients
6. **REST API** - Comprehensive HTTP API for all operations

## 🔧 Setup & Installation

### Prerequisites

- **RetroArch** with network commands enabled
- **Go 1.19+** for building from source

### RetroArch Configuration

1. Enable network commands in RetroArch:
   ```
   Settings → Network → Network Commands: ON
   Settings → Network → Network Command Port: 55355
   ```

2. Load a compatible game and core (Game Boy games work best)

### Building & Running

```bash
# Clone the repository
git clone <repository-url>
cd gamehook-enhanced

# Build the application
go build -o gamehook-enhanced ./cmd/gamehook

# Run with default settings
./gamehook-enhanced

# Or specify custom configuration
./gamehook-enhanced --port 8080 --retroarch-host 127.0.0.1
```

### Configuration Options

```bash
# Server configuration
--port 8080                    # Web server port
--host 0.0.0.0                # Server host

# RetroArch connection
--retroarch-host 127.0.0.1    # RetroArch host
--retroarch-port 55355        # RetroArch UDP port

# Performance tuning
--update-interval 16ms        # Property monitoring rate (60fps)
--request-timeout 64ms        # RetroArch request timeout

# Directories
--mappers-dir ./mappers       # Mapper definitions directory
--uis-dir ./uis               # Web UI directory
```

## 📝 Mapper System

RetroGameAnalysis uses **CUE** (Configure, Unify, Execute) for defining game memory layouts. This provides type safety, validation, and powerful expressions.

### Simple Property Example

```cue
properties: {
    playerName: {
        type: "string"
        address: "0xD158"
        length: 11
        description: "Player character name"
        charMap: characterMaps.pokemon
        validation: {
            pattern: "^[A-Za-z0-9 ]*$"
        }
        uiHints: {
            icon: "👤"
            editable: true
        }
    }
}
```

### Advanced Property with Freezing

```cue
properties: {
    playerMoney: {
        type: "bcd"
        address: "0xD347"
        length: 3
        description: "Player's money in BCD format"
        freezable: true
        transform: {
            expression: "bcdToDecimal(value)"
        }
        validation: {
            minValue: 0
            maxValue: 999999
        }
        uiHints: {
            displayFormat: "currency"
            unit: "₽"
            icon: "💰"
        }
    }
}
```

### Computed Properties

```cue
computed: {
    teamTotalLevel: {
        expression: """
            properties.pokemon1Level + 
            properties.pokemon2Level + 
            properties.pokemon3Level
        """
        dependencies: ["pokemon1Level", "pokemon2Level", "pokemon3Level"]
        type: "uint16"
    }
}
```

## 🌐 API Reference

### Enhanced REST Endpoints

#### Property Management
```http
GET    /api/properties                    # List all properties
GET    /api/properties/{name}             # Get specific property
PUT    /api/properties/{name}/value       # Set property value
PUT    /api/properties/{name}/bytes       # Set raw bytes
POST   /api/properties/{name}/freeze      # Freeze/unfreeze property
PUT    /api/properties/batch              # Batch property updates
```

#### Enhanced Features
```http
GET    /api/properties/states             # Get all property states
GET    /api/properties/{name}/metadata    # Get property metadata
GET    /api/properties/{name}/ui-hints    # Get UI presentation hints
GET    /api/properties/by-group/{group}   # Get properties by group
```

#### Reference System
```http
GET    /api/references                    # Get reference types
GET    /api/references/{type}             # Get specific reference
```

#### Event System
```http
GET    /api/events                        # Get events
POST   /api/events/{name}/trigger         # Trigger event
```

#### Validation & UI
```http
GET    /api/validation/rules              # Get validation rules
GET    /api/validation/errors             # Get validation errors
GET    /api/ui/themes                     # Get UI themes
GET    /api/ui/layout                     # Get UI layout
```

### WebSocket Streaming

Connect to `/api/stream` for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/stream');

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    switch(data.type) {
        case 'property_changed':
            console.log(`${data.property} = ${data.value}`);
            break;
        case 'property_freeze_changed':
            console.log(`${data.property} freeze: ${data.frozen}`);
            break;
        case 'event_triggered':
            console.log(`Event ${data.event_name} triggered`);
            break;
    }
};
```

## 🎮 Use Cases

### 🕹️ Game Development & Testing
- **Save State Analysis** - Examine save data structure and validation
- **Balancing & Tuning** - Real-time parameter adjustment during gameplay
- **Bug Investigation** - Monitor memory corruption and unexpected changes
- **Feature Testing** - Verify game logic responds correctly to state changes

### 📚 Game Research & Reverse Engineering
- **Memory Layout Discovery** - Map unknown game structures
- **Data Format Analysis** - Understand encoding and compression
- **Behavior Study** - Observe how games respond to different inputs
- **Documentation** - Create comprehensive memory maps

### 🎯 Speedrunning & Competition
- **Route Optimization** - Analyze RNG and optimal strategies
- **Practice Tools** - Set up specific game states for practice
- **Record Analysis** - Verify runs and analyze techniques
- **Training Aids** - Practice difficult sequences repeatedly

### 🔬 Educational & Academic
- **Computer Science Education** - Demonstrate memory management concepts
- **Game Studies** - Research game design and player behavior
- **Preservation** - Document game internals for future preservation

## 🚀 What Makes It Different

### vs. Traditional Memory Editors (Cheat Engine, etc.)

| Feature | Traditional | RetroGameAnalysis |
|---------|-------------|-------------------|
| **Approach** | Raw memory addresses | Structured properties |
| **Type Safety** | Manual casting | Rich type system |
| **Real-time** | Polling-based | 60fps streaming |
| **Validation** | None | Built-in constraints |
| **UI** | Basic tables | Rich metadata-driven |
| **API** | None/Limited | Full REST + WebSocket |
| **Automation** | Scripts | Event system |
| **Collaboration** | File sharing | Web-based, multi-user |

### vs. Save Editors

| Feature | Save Editors | RetroGameAnalysis |
|---------|--------------|-------------------|
| **Timing** | Save file only | Real-time during gameplay |
| **Scope** | Save data only | All game memory |
| **Interaction** | Static | Dynamic with game running |
| **Development** | Game-specific tools | Universal framework |

### vs. Basic RAM Watchers

| Feature | RAM Watchers | RetroGameAnalysis |
|---------|--------------|-------------------|
| **Property Types** | Numbers only | Rich types (enums, colors, etc.) |
| **Validation** | None | Comprehensive |
| **Freezing** | Basic | Advanced with conditions |
| **API** | None | Full REST + WebSocket |
| **UI** | Simple lists | Rich, customizable interface |

## 🏗️ Advanced Features

### Property Freezing
Lock values to prevent the game from changing them:

```bash
# Freeze player health at current value
curl -X POST http://localhost:8080/api/properties/playerHP/freeze \
  -H "Content-Type: application/json" \
  -d '{"freeze": true}'
```

### Batch Operations
Update multiple properties atomically:

```bash
curl -X PUT http://localhost:8080/api/properties/batch \
  -H "Content-Type: application/json" \
  -d '{
    "atomic": true,
    "properties": [
      {"name": "playerHP", "value": 999},
      {"name": "playerMP", "value": 999},
      {"name": "playerLevel", "value": 50}
    ]
  }'
```

### Event Triggers
Automate responses to game state changes:

```cue
events: {
    custom: {
        lowHealth: {
            trigger: "properties.playerHP < 20"
            action: "log('Warning: Low health!')"
            dependencies: ["playerHP"]
        }
    }
}
```

## 🤝 Contributing

RetroGameAnalysis is designed to be extensible and community-driven:

1. **Mapper Development** - Create mappers for new games
2. **Feature Enhancement** - Add new property types and transformations
3. **UI Improvements** - Build better interfaces and visualizations
4. **Driver Support** - Add support for other emulators
5. **Documentation** - Improve guides and examples

## 🙏 Acknowledgments

- **RetroArch Team** - For the excellent emulation platform
- **CUE Language** - For the powerful configuration system
- **Go Community** - For the robust ecosystem

---

**Ready to enhance your retro gaming experience?** 🎮✨

Visit the web interface at `http://localhost:8080` after starting the server to explore your game's memory in real-time!

## Known Bugs

- Not all values are correct in Pokemon Red and Blue Version (WIP)
- Pokemon Stadium is a proof of concept that this works with Mugen-Plus Core, all data is incorrect.
