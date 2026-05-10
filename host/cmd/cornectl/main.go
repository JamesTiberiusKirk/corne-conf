package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/darthvader/corne-conf/host/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: cornectl <command> [args]\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "get":
		getCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func getCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: cornectl get <resource>\n")
		os.Exit(1)
	}
	switch args[0] {
	case "layer":
		getLayer()
	case "config":
		getConfig()
	default:
		fmt.Fprintf(os.Stderr, "Unknown resource: %s\n", args[0])
		os.Exit(1)
	}
}

func getLayer() {
	data, err := os.ReadFile(config.CachePath())
	if err != nil {
		return
	}
	fmt.Print(string(data))
}

func getConfig() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(out))
}
