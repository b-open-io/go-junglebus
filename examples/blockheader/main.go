package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/b-open-io/go-junglebus"
	"github.com/b-open-io/go-junglebus/models"
)

func main() {
	junglebusClient, err := junglebus.New(
		junglebus.WithHTTP("https://junglebus.gorillapool.io"),
	)
	if err != nil {
		log.Fatalln(err.Error())
	}

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		// If no block is specified, get the chain tip
		var blockHeader *models.BlockHeader
		if blockHeader, err = junglebusClient.GetChainTip(context.Background()); err != nil {
			log.Printf("ERROR: failed getting chain tip %s", err.Error())
		} else {
			j, _ := json.Marshal(blockHeader)
			log.Printf("Got chain tip %s", string(j))
		}
		os.Exit(0)
	}

	block := argsWithoutProg[0]
	var blockHeader *models.BlockHeader
	if blockHeader, err = junglebusClient.GetBlockHeader(context.Background(), block); err != nil {
		log.Printf("ERROR: failed getting block header %s", err.Error())
	} else {
		j, _ := json.Marshal(blockHeader)
		log.Printf("Got block header %s", string(j))
	}
	os.Exit(0)
}
