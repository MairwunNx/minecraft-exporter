package worldapi

import "encoding/json"

// Server is GET /server, cut to what the exporter uses.
type Server struct {
	Name             string  `json:"name"`
	MinecraftVersion string  `json:"minecraftVersion"`
	Build            *int    `json:"build"`
	APIVersion       string  `json:"apiVersion"`
	WorldAPIVersion  string  `json:"worldApiVersion"`
	MaxPlayers       int     `json:"maxPlayers"`
	OnlinePlayers    int     `json:"onlinePlayers"`
	StartedAt        string  `json:"startedAt"`
	TickRate         float64 `json:"tickRate"`
	Frozen           bool    `json:"frozen"`
	Whitelist        bool    `json:"whitelist"`
	Datapacks        []struct {
		Enabled bool `json:"enabled"`
	} `json:"datapacks"`
}

// Performance is GET /server/performance.
type Performance struct {
	TPS struct {
		OneMinute      float64 `json:"oneMinute"`
		FiveMinutes    float64 `json:"fiveMinutes"`
		FifteenMinutes float64 `json:"fifteenMinutes"`
	} `json:"tps"`
	MSPT struct {
		Average float64 `json:"average"`
		P50     float64 `json:"p50"`
		P95     float64 `json:"p95"`
		P99     float64 `json:"p99"`
		Max     float64 `json:"max"`
	} `json:"mspt"`
	CPU struct {
		Process *float64 `json:"process"`
		System  *float64 `json:"system"`
		Cores   int      `json:"cores"`
	} `json:"cpu"`
	Worlds []struct {
		World        string         `json:"world"`
		LoadedChunks int            `json:"loadedChunks"`
		Entities     int            `json:"entities"`
		TileEntities int            `json:"tileEntities"`
		Players      int            `json:"players"`
		EntityTypes  map[string]int `json:"entityTypes"`
	} `json:"worlds"`
	JVM *struct {
		Heap    Memory `json:"heap"`
		NonHeap Memory `json:"nonHeap"`
		Pools   []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Memory
		} `json:"pools"`
		GC []struct {
			Name       string `json:"name"`
			Count      int64  `json:"count"`
			TimeMillis int64  `json:"timeMillis"`
		} `json:"gc"`
		Threads struct {
			Live   int `json:"live"`
			Daemon int `json:"daemon"`
			Peak   int `json:"peak"`
		} `json:"threads"`
		ClassesLoaded int64 `json:"classesLoaded"`
		Buffers       []struct {
			Name          string `json:"name"`
			Count         int64  `json:"count"`
			UsedBytes     int64  `json:"usedBytes"`
			CapacityBytes int64  `json:"capacityBytes"`
		} `json:"buffers"`
	} `json:"jvm"`
}

// Memory is a used, committed, and maximum size; the maximum is missing when it's unbounded.
type Memory struct {
	UsedBytes      int64  `json:"usedBytes"`
	CommittedBytes int64  `json:"committedBytes"`
	MaxBytes       *int64 `json:"maxBytes"`
}

// World is GET /worlds/{world}.
type World struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Time        struct {
		DayTime  int64 `json:"dayTime"`
		FullTime int64 `json:"fullTime"`
		Day      int64 `json:"day"`
	} `json:"time"`
	Weather struct {
		Raining    bool `json:"raining"`
		Thundering bool `json:"thundering"`
	} `json:"weather"`
	Border struct {
		Size float64 `json:"size"`
	} `json:"border"`
}

// OnlinePlayers is GET /players/online.
type OnlinePlayers struct {
	Count   int `json:"count"`
	Max     int `json:"max"`
	Players []struct {
		UUID        string  `json:"uuid"`
		Name        string  `json:"name"`
		World       string  `json:"world"`
		GameMode    string  `json:"gameMode"`
		Health      float64 `json:"health"`
		Food        int     `json:"food"`
		Level       int     `json:"level"`
		Ping        int     `json:"ping"`
		IdleSeconds int64   `json:"idleSeconds"`
		// Location comes only to clients that may see coordinates.
		Location *struct {
			World string  `json:"world"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
			Z     float64 `json:"z"`
		} `json:"location"`
	} `json:"players"`
}

// PlayerSummary is one item of GET /players.
type PlayerSummary struct {
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	FirstPlayed *string `json:"firstPlayed"`
	LastSeen    *string `json:"lastSeen"`
}

// Leaderboard is GET /leaderboards/{statistic}.
type Leaderboard struct {
	Total   int `json:"total"`
	Entries []struct {
		Player struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		} `json:"player"`
		Value float64 `json:"value"`
	} `json:"entries"`
}

// Extension is one plugin's namespace in GET /extensions.
type Extension struct {
	Namespace string `json:"namespace"`
	Topics    []struct {
		Topic    string `json:"topic"`
		Recorded bool   `json:"recorded"`
	} `json:"topics"`
}

// EventPage is GET /events.
type EventPage struct {
	Items      []Event `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// Event is one recorded event; Data depends on the type.
type Event struct {
	ID       int64           `json:"id"`
	Type     string          `json:"type"`
	AtMillis int64           `json:"atMillis"`
	Player   *PlayerRef      `json:"player"`
	Data     json.RawMessage `json:"data"`
}

// PlayerRef names a player.
type PlayerRef struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// System is GET /server/system; Linux fills in more than other systems, so most fields may be missing.
type System struct {
	CPUs        int       `json:"cpus"`
	LoadAverage []float64 `json:"loadAverage"`
	Memory      struct {
		TotalBytes     *int64 `json:"totalBytes"`
		AvailableBytes *int64 `json:"availableBytes"`
		SwapTotalBytes *int64 `json:"swapTotalBytes"`
		SwapFreeBytes  *int64 `json:"swapFreeBytes"`
	} `json:"memory"`
	Disk struct {
		Path        string `json:"path"`
		TotalBytes  int64  `json:"totalBytes"`
		UsableBytes int64  `json:"usableBytes"`
	} `json:"disk"`
	Container *struct {
		MemoryLimitBytes *int64   `json:"memoryLimitBytes"`
		MemoryUsedBytes  *int64   `json:"memoryUsedBytes"`
		CPULimit         *float64 `json:"cpuLimit"`
	} `json:"container"`
	OpenFiles            *int64 `json:"openFiles"`
	MaxOpenFiles         *int64 `json:"maxOpenFiles"`
	ProcessResidentBytes *int64 `json:"processResidentBytes"`
	Java                 struct {
		Name    string `json:"name"`
		Vendor  string `json:"vendor"`
		Version string `json:"version"`
	} `json:"java"`
}

// LogLine is one console line.
type LogLine struct {
	At      string  `json:"at"`
	Level   string  `json:"level"`
	Logger  *string `json:"logger"`
	Message string  `json:"message"`
}

// Plugin is one item of GET /server/plugins.
type Plugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
}

// Moderation counts bans, operators, and whitelisted players.
type Moderation struct {
	BannedPlayers, BannedAddresses, Operators, Whitelisted int
}

// EntitySummary is GET /worlds/{world}/entities/summary.
type EntitySummary struct {
	DroppedItems     int `json:"droppedItems"`
	DroppedItemCount int `json:"droppedItemCount"`
	Villagers        int `json:"villagers"`
	Named            int `json:"named"`
	Tamed            int `json:"tamed"`
	TopChunks        []struct {
		X            int `json:"x"`
		Z            int `json:"z"`
		Entities     int `json:"entities"`
		TileEntities int `json:"tileEntities"`
		Hoppers      int `json:"hoppers"`
	} `json:"topChunks"`
}
