package pkg

import (
	"net/http"
	"time"
)

// TODO: implement client with rate limiting

var HTTPClient = &http.Client{
	Timeout: time.Second * 10,
}
