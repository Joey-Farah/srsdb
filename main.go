package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// type struct declarations

type rawMetadata struct {
	LastFrame int `json:"lastFrame"`
}

type rawPlayer struct {
	Character int    `json:"character"`
	Port      string `json:"port"`
}

type rawStart struct {
	Stage   int         `json:"stage"`
	Players []rawPlayer `json:"players"`
}

type rawReplay struct {
	Start    rawStart    `json:"start"`
	Metadata rawMetadata `json:"metadata"`
}

type Player struct {
	Character string
	Port      string
}

type Game struct {
	Players  []Player
	Stage    string
	Duration int
}

// replays
var slpPath = "/Users/joeyfarah/Documents/slp replays/Game_20260409T184304.slp"
var replay rawReplay
var replayPointer = &replay

// stage map
var stageNames = map[int]string{
	2:  "Fountain of Dreams",
	3:  "Pokémon Stadium",
	8:  "Yoshi's Story",
	28: "Dream Land",
	31: "Battlefield",
	32: "Final Destination",
}

// character map
var characterNames = map[int]string{
	0:  "Captain Falcon",
	1:  "Donkey Kong",
	2:  "Fox",
	3:  "Mr. Game & Watch",
	4:  "Kirby",
	5:  "Bowser",
	6:  "Link",
	7:  "Luigi",
	8:  "Mario",
	9:  "Marth",
	10: "Mewtwo",
	11: "Ness",
	12: "Peach",
	13: "Pikachu",
	14: "Ice Climbers",
	15: "Jigglypuff",
	16: "Samus",
	17: "Yoshi",
	18: "Zelda",
	19: "Sheik",
	20: "Falco",
	21: "Young Link",
	22: "Dr. Mario",
	23: "Roy",
	24: "Pichu",
	25: "Ganondorf",
}

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
	fmt.Println(toGame(replay))
}

// parse replays
func toGame(replay rawReplay) Game {
	var players []Player
	// go loops only have the for keyword and each slice of a loop produces TWO values, the index AND the element. We're gonna use i for index and rp for the element
	// if you dont have a need for the index, you can supply a _ instead
	for _, rp := range replay.Start.Players {
		p := Player{
			Character: characterNames[rp.Character],
			Port:      rp.Port,
		}
		players = append(players, p)
	}

	return Game{
		Players:  players,
		Stage:    stageNames[replay.Start.Stage],
		Duration: replay.Metadata.LastFrame,
	}

}
