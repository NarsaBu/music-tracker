package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	httpadapter "music-tracker/backend/internal/adapter/http"
	"music-tracker/backend/internal/adapter/musicbrainz"
	"music-tracker/backend/internal/adapter/postgres"
	"music-tracker/backend/internal/infrastructure/config"
	"music-tracker/backend/internal/infrastructure/observability"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/artist"
)

func serve(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := observability.NewLogger(os.Stdout, cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return err
	}
	slog.SetDefault(log)

	if !cfg.AuthEnabled() {
		log.Warn("APP_PASSWORD_HASH is not set: authentication is disabled, expose the app only on localhost")
	}

	if cfg.MigrateOnStart {
		if err := migrateUp(cfg.DatabaseURL); err != nil {
			return err
		}
		log.Info("migrations applied")
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	catalog := musicbrainz.NewClient(musicbrainz.Config{
		BaseURL:   cfg.MusicBrainz.BaseURL,
		UserAgent: cfg.MusicBrainz.UserAgent,
		RateLimit: cfg.MusicBrainz.RateLimit,
	})
	tx := postgres.NewTxManager(pool)
	artists := postgres.NewArtistRepository(pool)
	subs := postgres.NewSubscriptionRepository(pool)
	settings := postgres.NewSettingsRepository(pool)
	clock := usecase.SystemClock{}

	router := httpadapter.NewRouter(httpadapter.Deps{
		Logger:    log,
		Readiness: pool,
		Artists: httpadapter.ArtistUseCases{
			Search:      artist.NewSearchArtists(catalog, subs),
			Subscribe:   artist.NewSubscribe(catalog, artists, subs, settings, tx, clock),
			Unsubscribe: artist.NewUnsubscribe(subs),
			List:        artist.NewListSubscriptions(subs, settings),
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", slog.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
