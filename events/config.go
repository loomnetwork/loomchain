package events

const (
	DispatcherDBIndexer = "db_indexer"
	DispatcherRedis     = "redis"
	DispatcherLog       = "log"
)

type EventStoreConfig struct {
	// DBName defines database file name
	DBName string
	// DBBackend defines backend event store type
	// available backend types are 'goleveldb', or 'cleveldb'
	DBBackend string
}

func DefaultEventStoreConfig() *EventStoreConfig {
	return &EventStoreConfig{
		DBName:    "events",
		DBBackend: "goleveldb",
	}
}

// Clone returns a deep clone of the config.
func (c *EventStoreConfig) Clone() *EventStoreConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

type EventDispatcherConfig struct {
	Dispatcher string
	Redis      *RedisEventDispatcherConfig
}

func DefaultEventDispatcherConfig() *EventDispatcherConfig {
	return &EventDispatcherConfig{
		Dispatcher: DispatcherLog,
		Redis: &RedisEventDispatcherConfig{
			URI: "127.0.0.1",
		},
	}
}

// Clone returns a deep clone of the config.
func (c *EventDispatcherConfig) Clone() *EventDispatcherConfig {
	if c == nil {
		return nil
	}
	clone := *c
	*clone.Redis = *c.Redis
	return &clone
}
