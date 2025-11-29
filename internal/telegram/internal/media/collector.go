package media

import (
	"print3d-order-bot/internal/pkg/model"
	"sync"
	"time"

	"github.com/go-telegram/bot/models"
)

type Collector struct {
	windows map[int64]*Window
	mu      sync.Mutex
}

type Window struct {
	Media    []model.TGOrderFile
	Contacts []string
	Links    []string
	Timer    *time.Timer
	mu       sync.Mutex
}

func NewCollector() *Collector {
	return &Collector{
		windows: make(map[int64]*Window),
		mu:      sync.Mutex{},
	}
}

func (c *Collector) GetOrCreateWindow(id int64) *Window {
	c.mu.Lock()
	defer c.mu.Unlock()

	if window, ok := c.windows[id]; ok {
		return window
	}

	window := &Window{
		Media:    make([]model.TGOrderFile, 0),
		Contacts: make([]string, 0),
		Links:    make([]string, 0),
		mu:       sync.Mutex{},
	}

	c.windows[id] = window

	return window
}

func (c *Collector) SetWindow(id int64, window *Window) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.windows[id] = window
}

func (c *Collector) DeleteWindow(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.windows, id)
}

func (c *Collector) ProcessMessage(message *models.Message, onSuccess func(userID int64, window *Window)) {
	if !HasMediaOrResources(message) {
		return
	}

	window := c.GetOrCreateWindow(message.From.ID)

	media := ExtractMedia(message)
	contacts, links := ExtractResources(message)

	window.mu.Lock()
	defer window.mu.Unlock()

	window.Media = append(window.Media, media...)
	window.Contacts = append(window.Contacts, contacts...)
	window.Links = append(window.Links, links...)

	if window.Timer != nil {
		window.Timer.Stop()
	}

	window.Timer = time.AfterFunc(time.Second*2, func() {
		onSuccess(message.From.ID, window)
		c.DeleteWindow(message.From.ID)
	})

	c.SetWindow(message.From.ID, window)
}
