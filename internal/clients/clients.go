// Package clients holds the configured SharePoint admin API client shared by all
// resources.
package clients

import (
	"github.com/terraprovider/go-spo/spo"
	"github.com/terraprovider/go-spo/spoapi"
)

// Client is passed to every resource/data source via Configure.
type Client struct {
	API      *spoapi.Client
	SPO      *spo.Service
	AdminURL string
}
