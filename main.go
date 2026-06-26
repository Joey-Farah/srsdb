package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	// "github.com/Joey-Farah/rslp/storage"
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
var slpPath = ""

// /Users/joeyfarah/Documents/slp replays/Game_20260409T184304.slp

var directoryPath = "/Users/joeyfarah/Documents/slp replays"
var fileWriteLines []string

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

	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {

		file := entry.Name()
		fullFilePath := filepath.Join(directoryPath, file)

		out, err := exec.Command("slp", "-s", fullFilePath).Output()
		if err != nil {
			log.Fatal(err)
		}

		var replay rawReplay
		replayPointer := &replay

		err = json.Unmarshal(out, replayPointer)
		if err != nil {
			log.Fatal(err)
		}

		game := (toGame(replay))
		data, err := json.Marshal(game)
		if err != nil {
			log.Fatal(err)
		}

		fileWriteLines = append(fileWriteLines, (string(data)))
	}

	linesList := strings.Join(fileWriteLines, "\n")
	// WriteFile(path, bytes, perm): writes data to disk, creating/overwriting the file.
	// 0644 = owner read+write, others read-only. Returns only an error (data goes IN).
	err = os.WriteFile("data/games.jsonl", []byte(linesList), 0644)
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile("data/games.jsonl")
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")

	var games []Game
	for _, line := range lines {
		var game Game
		err = json.Unmarshal([]byte(line), &game)
		if err != nil {
			log.Fatal(err)
		}
		games = append(games, game)
	}
	fmt.Println(len(games))
	falcoCount := 0
	for _, game := range games {
		for _, char := range game.Players {
			if char.Character == "Falco" {
				falcoCount++
				break
			}
		}
	}
	fmt.Println(falcoCount, "games")
}

// parse replays
func toGame(replay rawReplay) Game {
	var players []Player
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
