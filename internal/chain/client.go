package chain

import (
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	Eth *ethclient.Client
}

func NewClient(rpc string) *Client {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		log.Fatal("failed to connect rpc:", err)
	}

	return &Client{
		Eth: c,
	}
}
