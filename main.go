package main

import "fmt"

// type struct declarations

type rawStart struct {
	Stage int `json:"stage"`
	Players []rawPlayer `json:"players"`
}

type rawPlayer struct {
	Character int `json:"character"`
	Port string `json:"port"`
}

type rawMetadata struct {
	LastFrame int `json:"lastFrame"`
}

type rawReplay struct {
	Start rawStart `json:"start"`
	Metadata rawMetadata `json:"metadata"`
}

//main
func main() {
	fmt.Println()
}


