package main

import (
	"context"
	"os"

	"github.com/productive-k3s/productive-k3s-cli/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], app.DefaultDependencies()))
}
