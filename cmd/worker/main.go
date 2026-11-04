package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"earthmanlabs/resumaker/internal/app"
	"earthmanlabs/resumaker/internal/config"
	"earthmanlabs/resumaker/internal/envelope"
	"earthmanlabs/resumaker/internal/store"
	"earthmanlabs/resumaker/internal/worker"
	"earthmanlabs/resumaker/render"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	keys, err := envelope.NewKeyring(cfg.MasterKey)
	if err != nil {
		return err
	}

	renderer, err := render.NewRenderer(render.RendererOptions{
		Timeout:  cfg.RenderTimeout,
		ExecPath: cfg.ChromePath,
	})
	if err != nil {
		return err
	}
	defer renderer.Close()

	return worker.Run(ctx, app.New(st, keys), renderer, worker.Options{
		Concurrency: cfg.RenderConcurrency,
	})
}
