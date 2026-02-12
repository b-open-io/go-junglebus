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
		log.Fatalln("no address given")
	}
	address := argsWithoutProg[0]

	var addr []*models.AddressTx
	if addr, err = junglebusClient.GetAddressTransactions(context.Background(), address, 0); err != nil {
		log.Printf("ERROR: failed getting address transactions %s", err.Error())
	} else {
		j, _ := json.Marshal(addr)
		log.Printf("Got address %s", string(j))
	}
	os.Exit(0)
}
