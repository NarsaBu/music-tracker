package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"music-tracker/backend/internal/adapter/email"
	httpadapter "music-tracker/backend/internal/adapter/http"
	"music-tracker/backend/internal/adapter/musicbrainz"
	"music-tracker/backend/internal/adapter/postgres"
	"music-tracker/backend/internal/domain"
	"music-tracker/backend/internal/infrastructure/config"
	"music-tracker/backend/internal/infrastructure/jobs"
	"music-tracker/backend/internal/infrastructure/observability"
	"music-tracker/backend/internal/infrastructure/scheduler"
	"music-tracker/backend/internal/usecase"
	"music-tracker/backend/internal/usecase/artist"
	"music-tracker/backend/internal/usecase/notify"
	settingsuc "music-tracker/backend/internal/usecase/settings"
	syncuc "music-tracker/backend/internal/usecase/sync"
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
	releases := postgres.NewReleaseGroupRepository(pool)
	runs := postgres.NewSyncRunRepository(pool)
	queue := postgres.NewJobQueue(pool)
	notifications := postgres.NewNotificationRepository(pool)
	clock := usecase.SystemClock{}

	mailer, err := email.NewMailer(email.Config{
		Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, User: cfg.SMTP.User, Password: cfg.SMTP.Password,
		From: cfg.SMTP.From, AppURL: cfg.PublicBaseURL,
	})
	if err != nil {
		return err
	}
	if cfg.SMTP.Host == "" {
		log.Warn("SMTP_HOST is not set: email digests cannot be sent")
	}

	fanout := notify.NewFanOut(settings, subs, notifications, clock)
	syncArtist := syncuc.NewArtist(catalog, artists, subs, releases, runs, fanout, tx, clock, syncuc.Config{
		StaleThreshold: time.Duration(cfg.Sync.StaleDays) * 24 * time.Hour,
	})
	scheduleDigest := notify.NewScheduleDigest(settings, queue, clock)
	sendDigest := notify.NewSendDigest(settings, notifications, mailer, tx, clock)
	settingsCfg := settingsuc.Config{PublicBaseURL: cfg.PublicBaseURL}
	scheduleRun := syncuc.NewScheduleSyncRun(artists, runs, queue, tx, clock)

	router := httpadapter.NewRouter(httpadapter.Deps{
		Logger:    log,
		Readiness: pool,
		Artists: httpadapter.ArtistUseCases{
			Search:        artist.NewSearchArtists(catalog, subs),
			Subscribe:     artist.NewSubscribe(catalog, artists, subs, settings, queue, tx, clock),
			Unsubscribe:   artist.NewUnsubscribe(subs),
			List:          artist.NewListSubscriptions(subs, settings),
			Get:           artist.NewGetSubscription(subs, settings),
			Sync:          syncuc.NewRequestArtist(subs, queue, clock),
			ReleaseGroups: artist.NewListReleaseGroups(artists, releases),
			Update:        artist.NewUpdateSubscription(subs, settings),
		},
		Sync: httpadapter.SyncUseCases{
			Start:  scheduleRun,
			Latest: syncuc.NewLatestSyncRun(runs),
		},
		Settings: httpadapter.SettingsUseCases{
			Get:       settingsuc.NewGet(settings, settingsCfg),
			Update:    settingsuc.NewUpdate(settings, clock, settingsCfg),
			Rotate:    settingsuc.NewRotateFeedToken(settings, settingsCfg),
			TestEmail: notify.NewSendTestEmail(settings, mailer),
		},
		Releases: notify.NewListReleases(notifications),
		Feeds:    notify.NewBuildFeed(settings, notifications),
		AppURL:   cfg.PublicBaseURL,
	})

	// Background work stops with ctx; serve returns only after it has finished.
	ctx, stopBackground := context.WithCancel(ctx)
	var background sync.WaitGroup
	defer func() {
		stopBackground()
		background.Wait()
	}()

	runner := jobs.NewRunner(queue, jobs.Config{Concurrency: cfg.Worker.Concurrency}, log)
	runner.Register(usecase.JobSyncArtist, syncArtist)
	runner.Register(usecase.JobSendDigest, sendDigest)
	background.Go(func() { runner.Run(ctx) })

	sched := scheduler.New(log)
	err = sched.Add(ctx, "sync", cfg.Sync.Cron, func(ctx context.Context) error {
		run, started, err := scheduleRun.Execute(ctx, domain.TriggerSchedule)
		if err == nil && started {
			log.InfoContext(ctx, "sync run started", slog.String("run_id", run.ID), slog.Int("artists", run.ArtistsTotal))
		}
		return err
	})
	if err != nil {
		return err
	}
	// The digest time is per user setting, so it is checked every minute (docs/08 §2.1).
	if err := sched.Add(ctx, "digest", "* * * * *", scheduleDigest.Execute); err != nil {
		return err
	}
	background.Go(func() { sched.Run(ctx) })

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
