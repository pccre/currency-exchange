package main

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/earlydata"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/pccre/utils/Mut"
	"github.com/pccre/utils/c"
)

// FEEL FREE TO CONFIGURE
var currencies = map[string]Currency{
	"BTC": {
		Min: 27000,
		Max: 28000,
	},
	"ETH": {
		Min: 1700,
		Max: 2000,
	},
	"DOGE": {
		Min: 580,
		Max: 625,
	},
}

const absInterval = 6
const timelineLimit int = 10

const interval time.Duration = absInterval * time.Second

var json = c.JSON

// CODE STARTS HERE
var timeline = []map[string]int{}
var pool = Mut.Array[*Mut.WS]{Mut: &sync.RWMutex{}}

func randint(min, max int) int {
	return rand.Intn(max-min) + min
}

// keep order
func removeO[T any](s []T, i int) []T {
	return append(s[:i], s[i+1:]...)
}

func BroadcastJSON(content interface{}) {
	pool.Mut.RLock()
	for _, c := range pool.Array {
		c.WriteJSON(content)
	}
	pool.Mut.RUnlock()
}

func makeNewValues() {
	new := make(map[string]int, len(currencies))
	for name, v := range currencies {
		new[name] = randint(v.Min, v.Max)
	}
	if len(timeline) == timelineLimit {
		timeline = removeO(timeline, 0)
	}
	timeline = append(timeline, new)
}

func startPool() {
	for i := 0; i < timelineLimit; i++ {
		makeNewValues()
	}
	for {
		time.Sleep(interval)
		makeNewValues()
		if len(pool.Array) > 0 { // waste less computing power when nobody online
			data, err := json.Marshal(Update{Method: "update", Response: [1]map[string]int{timeline[len(timeline)-1]}})
			if err != nil {
				log.Println("err in startPool: " + err.Error())
				continue
			}
			pool.Mut.RLock()
			for _, c := range pool.Array {
				c.WriteRaw(data)
			}
			pool.Mut.RUnlock()
		}
	}
}

func main() {
	http := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
		GETOnly:     true,
	})

	http.Use(recover.New())
	http.Use(earlydata.New())

	http.Use("/CurrencyRate", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	http.Get("/CurrencyRate", websocket.New(func(cn *websocket.Conn) {
		c := &Mut.WS{WS: cn, Mut: &sync.Mutex{}}
		pool.Append(c)
		var err error

		c.WriteJSON(InitializeBase{Method: "update", Response: Initialize{Interval: absInterval, Timeline: timeline}})
		for {
			if _, _, err = c.WS.ReadMessage(); err != nil {
				log.Println("read err:", err)
				for i, conn := range pool.Array {
					if conn == c {
						pool.Remove(i)
						return
					}
				}
			}
		}
	}, c.WSConfig))
	go startPool()
	log.Fatal(http.Listen(":8082"))
}
