package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// type struct declarations

type rawStart struct {
	Stage   int         `json:"stage"`
	Players []rawPlayer `json:"players"`
}

type rawPlayer struct {
	Character int    `json:"character"`
	Port      string `json:"port"`
}

type rawMetadata struct {
	LastFrame int `json:"lastFrame"`
}

type rawReplay struct {
	Start    rawStart    `json:"start"`
	Metadata rawMetadata `json:"metadata"`
}

var slpPath = "/Users/joeyfarah/Documents/slp replays/Game_20260409T184304.slp"

var replay rawReplay

var replayPointer = &replay

// main
func main() {
	out, err := exec.Command("slp", "-s", slpPath).Output()
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(out, replayPointer)
	if err != nil {
		log.Fatal(err)
	}
	// outString := string(out)
	fmt.Println(replay)
}
