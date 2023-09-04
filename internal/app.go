package internal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	coresdk "github.com/goverland-labs/core-web-sdk"
	"github.com/goverland-labs/inbox-api/protobuf/inboxapi"
	"github.com/goverland-labs/platform-events/pkg/natsclient"
	"github.com/nats-io/nats.go"
	"github.com/s-larionov/process-manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/goverland-labs/inbox-feed/internal/config"
	"github.com/goverland-labs/inbox-feed/internal/feed"
	"github.com/goverland-labs/inbox-feed/pkg/grpcsrv"
	"github.com/goverland-labs/inbox-feed/pkg/health"
	"github.com/goverland-labs/inbox-feed/pkg/prometheus"
)

type Application struct {
	sigChan <-chan os.Signal
	manager *process.Manager
	cfg     config.App

	natsConn      *nats.Conn
	publisher     *natsclient.Publisher
	subscriptions inboxapi.SubscriptionClient
	feedRepo      *feed.Repo
	feedService   *feed.Service
	coreSDK       *coresdk.Client
}

func NewApplication(cfg config.App) (*Application, error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	a := &Application{
		sigChan: sigChan,
		cfg:     cfg,
		manager: process.NewManager(),
	}

	err := a.bootstrap()
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *Application) Run() {
	a.manager.StartAll()
	a.registerShutdown()
}

func (a *Application) bootstrap() error {
	initializers := []func() error{
		// Init Dependencies
		a.initDatabase,
		a.initNats,
		a.initInboxAPI,
		a.initCodeSDK,
		a.initServices,

		// Init Workers: Application
		a.initGRPCServer,
		a.initFeedConsumer,

		// Init Workers: System
		a.initPrometheusWorker,
		a.initHealthWorker,
	}

	for _, initializer := range initializers {
		if err := initializer(); err != nil {
			return err
		}
	}

	return nil
}

func (a *Application) initDatabase() error {
	conn, err := gorm.Open(postgres.Open(a.cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlConnection, err := conn.DB()
	if err != nil {
		return err
	}
	sqlConnection.SetMaxOpenConns(a.cfg.Database.MaxOpenConnections)
	sqlConnection.SetMaxIdleConns(a.cfg.Database.MaxIdleConnections)

	if a.cfg.Database.Debug {
		conn = conn.Debug()
	}

	// TODO: Use real migrations intead of auto migrations from gorm
	if err := conn.AutoMigrate(&feed.Item{}); err != nil {
		return err
	}

	a.feedRepo = feed.NewRepo(conn)

	return err
}

func (a *Application) initNats() error {
	nc, err := nats.Connect(
		a.cfg.Nats.URL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(a.cfg.Nats.MaxReconnects),
		nats.ReconnectWait(a.cfg.Nats.ReconnectTimeout),
	)
	if err != nil {
		return err
	}

	publisher, err := natsclient.NewPublisher(nc)
	if err != nil {
		return err
	}
	a.publisher = publisher

	a.natsConn = nc

	return nil
}

func (a *Application) initInboxAPI() error {
	conn, err := grpc.Dial(a.cfg.Inbox.StorageAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create connection with storage server: %v", err)
	}

	a.subscriptions = inboxapi.NewSubscriptionClient(conn)

	return nil
}

func (a *Application) initCodeSDK() error {
	a.coreSDK = coresdk.NewClient(a.cfg.Core.CoreURL)

	return nil
}

func (a *Application) initServices() error {
	a.feedService = feed.NewService(a.feedRepo, a.subscriptions, a.coreSDK, a.publisher)

	return nil
}

func (a *Application) initFeedConsumer() error {
	consumer := feed.NewConsumer(a.natsConn, a.feedService)
	a.manager.AddWorker(process.NewCallbackWorker("feed consumer", consumer.Start))

	return nil
}

func (a *Application) initPrometheusWorker() error {
	srv := prometheus.NewServer(a.cfg.Prometheus.Listen, "/metrics")
	a.manager.AddWorker(process.NewServerWorker("prometheus", srv))

	return nil
}

func (a *Application) initHealthWorker() error {
	srv := health.NewHealthCheckServer(a.cfg.Health.Listen, "/status", health.DefaultHandler(a.manager))
	a.manager.AddWorker(process.NewServerWorker("health", srv))

	return nil
}

func (a *Application) registerShutdown() {
	go func(manager *process.Manager) {
		<-a.sigChan

		manager.StopAll()
	}(a.manager)

	a.manager.AwaitAll()
}

func (a *Application) initGRPCServer() error {
	srv := grpcsrv.NewGrpcServer()
	inboxapi.RegisterFeedServer(srv, feed.NewServer(a.feedService))

	a.manager.AddWorker(grpcsrv.NewGrpcServerWorker("gRPC server", srv, a.cfg.Inbox.Bind))

	return nil
}
