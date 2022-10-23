package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"jaipur/pkg/read"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "remove", Usage: "A word you want to remove. ex '--remove a,b,c' "},
			&cli.UintFlag{Name: "quality", Aliases: []string{"q"}, Value: 70, Usage: "Set quality for mozjpeg"},
		},
		Action: func(cCtx *cli.Context) error {
			fmt.Printf("Hello %q\n", cCtx.Args().Get(0))
			remove := cCtx.String("remove")
			fmt.Println("Remove word: ", remove)

			dir := cCtx.Args().Get(0)
			read.Read(dir)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
