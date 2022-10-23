package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "remove", Usage: "A word you want to remove"},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Printf("Hello %q\n", cCtx.Args().Get(0))
			remove := cCtx.String("remove")
			fmt.Println("Remove word: ", remove)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
