package main

import (
	"context"
	"fmt"
	"os"

	"github.com/flexer2006/pes-api/internal/service/app"
)

const envs = "./deploy/.env"

func main() {
	if err := app.Run(context.Background(), envs); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "service startup failed:", err)
		os.Exit(1)
	}
}
