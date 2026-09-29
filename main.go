package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

const usage = `Nādabrahma — a Carnatic gem finder.

Finds the specific renditions (kriti × artist × recording) that are hard to skip,
using crowd signal + AI reading of real comments. yt-dlp + Groq. No credit card.

Usage:
  nadabrahma discover [deity] [deep]   deity-shuffle gem feed (the main experience)
  nadabrahma check                     verify yt-dlp, Groq key/model, and cache
  nadabrahma help                      this message

  [deity]  pin the feed to one of: Ganesha Shiva Devi Rama Krishna Vishnu Subrahmanya
  deep     "deep cuts" — only surface lesser-known renditions (view ceiling)

Examples:
  nadabrahma discover Shiva
  nadabrahma discover Rama deep
`

func main() {
	rand.Seed(time.Now().UnixNano())
	cfg := loadConfig()

	cmd := "discover"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "discover", "d":
		deity, deep := "", false
		for _, a := range os.Args[2:] {
			switch strings.ToLower(a) {
			case "deep", "--deep":
				deep = true
			default:
				if deity == "" {
					deity = a
				}
			}
		}
		err = runDiscover(cfg, deity, deep)
	case "check":
		err = runCheck(cfg)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Printf("unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%serror:%s %v\n", "\033[31m", cReset, err)
		os.Exit(1)
	}
}
