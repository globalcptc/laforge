package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gen0cide/laforge"
	"github.com/gen0cide/laforge/ent"
	"github.com/gen0cide/laforge/ent/authuser"
	"github.com/gen0cide/laforge/ent/buildcommit"
	"github.com/gen0cide/laforge/ent/ginfilemiddleware"
	"github.com/gen0cide/laforge/ent/migrate"
	"github.com/gen0cide/laforge/ent/servertask"
	"github.com/gen0cide/laforge/ent/status"
	"github.com/gen0cide/laforge/graphql/auth"
	"github.com/gen0cide/laforge/graphql/graph"
	pb "github.com/gen0cide/laforge/grpc/proto"
	"github.com/gen0cide/laforge/grpc/server"
	"github.com/gen0cide/laforge/grpc/server/static"
	"github.com/gen0cide/laforge/logging"
	"github.com/gen0cide/laforge/scheduler"
	"github.com/gen0cide/laforge/server/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultPort = ":8080"

// tempURLHandler Checks ENT to verify that the url results in a file
func tempURLHandler(client *ent.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		urlID := ctx.Param("url_id")
		fileInfo, err := client.GinFileMiddleware.Query().Where(
			// ginfilemiddleware.And(
			ginfilemiddleware.URLIDEQ(urlID),
			// 	ginfilemiddleware.AccessedEQ(false),
			// ),
		).
			Only(ctx)
		if err != nil {
			ctx.AbortWithStatus(404)
			return
		}
		ctx.File(fileInfo.FilePath)
		_, err = fileInfo.Update().SetAccessed(true).Save(ctx)
		if err != nil {
			ctx.AbortWithStatus(404)
			return
		}
		ctx.Next()
	}
}

// Defining the Graphql handler
func graphqlHandler(client *ent.Client, rdb *redis.Client) gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.New(graph.NewSchema(client, rdb))

	h.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			HandshakeTimeout: 30 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			WriteBufferPool:  nil,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			EnableCompression: false,
		},
		KeepAlivePingInterval: 1 * time.Second,
	})
	h.AddTransport(transport.GET{})
	h.AddTransport(transport.POST{})
	h.AddTransport(transport.MultipartForm{})

	h.SetQueryCache(lru.New(1000))

	h.Use(extension.Introspection{})
	h.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/api/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func createDefaultAdminUser(client *ent.Client, ctx context.Context, laforgeConfig *utils.ServerConfig) error {
	entAuthUserExsist, _ := client.AuthUser.Query().Where(
		authuser.And(
			authuser.UsernameEQ(laforgeConfig.Database.AdminUser),
			authuser.ProviderEQ(authuser.ProviderLOCAL),
		)).Exist(ctx)
	if !entAuthUserExsist {
		sshFolderPath := fmt.Sprintf(utils.UserKeyPath, strings.ToLower(authuser.ProviderLOCAL.String()), laforgeConfig.Database.AdminUser)
		err := os.MkdirAll(sshFolderPath, os.ModeAppend|os.ModePerm)
		if err != nil {
			return err
		}
		sshPrivateFile := fmt.Sprintf("%s/id_ed25519", sshFolderPath)
		err = utils.MakeED25519KeyPair(sshPrivateFile)
		if err != nil {
			return err
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(laforgeConfig.Database.AdminPass), 8)
		if err != nil {
			return err
		}
		password := string(hashedPassword[:])
		client.AuthUser.Create().
			SetUsername(laforgeConfig.Database.AdminUser).
			SetPassword(password).
			SetRole(authuser.RoleADMIN).
			SetProvider(authuser.ProviderLOCAL).
			SetPrivateKeyPath(sshPrivateFile).
			Save(ctx)
	}
	return nil
}

// tempServerTaskHandler Ahh
func tempServerTaskHandler(client *ent.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		server_task_id := ctx.Param("server_task_id")
		serverTaskUUID, err := uuid.Parse(server_task_id)

		if err != nil {
			ctx.AbortWithStatus(404)
			return
		}

		entServerTask, err := client.ServerTask.Get(ctx, serverTaskUUID)

		if err != nil {
			ctx.AbortWithStatus(404)
			return
		}
		ctx.File(entServerTask.LogFilePath)
		if err != nil {
			ctx.AbortWithStatus(404)
			return
		}
		ctx.Next()
	}
}

func main() {
	laforge.PrintLogo()

	// Load main server configuration
	laforgeConfig, err := utils.LoadServerConfig()
	if err != nil {
		logrus.Errorf("failed to load LaForge config: %v", err)
		return
	}
	logrus.Infof("\033[1;32mConfig File: \033[0m%s", laforgeConfig.ConfigFile)

	// Start logging all Logrus output to files
	if laforgeConfig.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
		_, err := os.Stat("logs")
		if err != nil {
			if os.IsNotExist(err) {
				mkdirErr := os.Mkdir("logs", os.ModeAppend|os.ModePerm)
				if mkdirErr != nil {
					logrus.Errorf("error while creating logs directory")
				}
			} else {
				logrus.Errorf("error while checking if logs dir exists: %v", err)
			}
		}
		logFile, err := os.OpenFile(fmt.Sprintf("logs/%s.log", time.Now().Format("20060102-15-04-05")), os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModeAppend)
		if err != nil {
			logrus.Errorf("couldn't create log file: %s", err)
		} else {
			logrus.SetOutput(logFile)
		}
	}
	if laforgeConfig.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if laforgeConfig.Database.PostgresUri == "" {
		logrus.Errorf("Database.PostgresUri not set in LaForge config")
		os.Exit(1)
	}
	client := ent.PGOpen(laforgeConfig.Database.PostgresUri)

	ctx := context.Background()
	defer ctx.Done()
	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		logrus.Fatalf("failed creating schema resources: %v", err)
	}

	if err := createDefaultAdminUser(client, ctx, laforgeConfig); err != nil {
		logrus.Fatal(err)
	}

	go func(client *ent.Client, ctx context.Context) {
		ticker := time.NewTicker(time.Minute)
		for {
			<-ticker.C
			go auth.ClearTokens(client, ctx)

		}
	}(client, ctx)

	// Fail all Server Tasks that got interrupted
	go func(client *ent.Client, ctx context.Context) {
		interruptedServerTasks, err := client.ServerTask.Query().Where(servertask.HasStatusWith(status.StateEQ(status.StateINPROGRESS))).All(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				logrus.Info("no interrupted server tasks found.")
			} else {
				logrus.Errorf("error while querying interrupted server tasks: %v", err)
			}
			return
		}
		for _, task := range interruptedServerTasks {
			entStatus, err := task.QueryStatus().Only(ctx)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"taskId": task.ID,
				}).Errorf("error while querying status from server task: %v", err)
				continue
			}
			err = task.Update().SetEndTime(time.Now()).Exec(ctx)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"taskId": task.ID,
				}).Errorf("error while setting end time on server task: %v", err)
				continue
			}
			err = entStatus.Update().SetState(status.StateFAILED).Exec(ctx)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"taskId": task.ID,
				}).Errorf("error while setting FAILED status on server task: %v", err)
				continue
			}
			entBuildCommit, _ := task.QueryBuildCommit().Only(ctx)
			if entBuildCommit != nil && entBuildCommit.State == buildcommit.StateINPROGRESS {
				err := entBuildCommit.Update().SetState(buildcommit.StateCANCELLED).Exec(ctx)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"buildCommitId": entBuildCommit.ID,
					}).Errorf("error while setting CANCELLED state on build commit: %v", err)
					continue
				}
			}
		}
		if len(interruptedServerTasks) == 0 {
			logrus.Info("No interrupted server tasks found")
		} else {
			logrus.Warnf("Failed %d interrupted server tasks", len(interruptedServerTasks))
		}
	}(client, ctx)

	lis, err := net.Listen("tcp", server.Port)

	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	rdb := &redis.Client{}
	if laforgeConfig.Graphql.RedisServerUri != "" && laforgeConfig.Graphql.RedisPassword != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     laforgeConfig.Graphql.RedisServerUri,
			Password: laforgeConfig.Graphql.RedisPassword,
			DB:       0, // use default DB
		})
	} else if laforgeConfig.Graphql.RedisServerUri != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     laforgeConfig.Graphql.RedisServerUri,
			Password: "",
			DB:       0, // use default DB
		})
	}

	go func() {
		logFolder := laforgeConfig.LogFolder
		if logFolder == "" {
			// Default log location
			logFolder = "/var/log/laforge"
		}
		absPath, err := filepath.Abs(logFolder)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"logFolder": logFolder,
			}).Errorf("error getting absolute path from log folder: %v", err)
			return
		}
		err = os.MkdirAll(absPath, 0755)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"logFolder": logFolder,
			}).Errorf("error creating log folder: %v", err)
			return
		}
		filename := fmt.Sprintf("%s_%s.lfglog", time.Now().Format("20060102-15-04-05"), "InternalPubSub")
		logPath := path.Join(absPath, filename)
		logrus.Info(logPath)
		subLog := logging.CreateNewLogger(logPath)
		sub := rdb.Subscribe(ctx, "newAgentStatus", "updatedStatus", "updatedServerTask", "updatedBuild", "updatedBuildCommit", "updatedAgentTask")
		_, err = sub.Receive(ctx)
		if err != nil {
			subLog.Log.Errorf("error reciving from subscription: %v", err)
			return
		}
		ch := sub.Channel()
		for {
			select {
			case message := <-ch:
				subLog.Log.Infof("Message %v received from %v", message.Payload, message.Channel)
			// close when context done
			case <-ctx.Done():
				subLog.Log.Infof("Main Channel CTX Closing, Closing Sub Channel")
				sub.Close()
				return
			}
		}
	}()

	go scheduler.SchedulerWatchdog(ctx, client, rdb, laforgeConfig)

	auth.InitGoth(laforgeConfig)

	router := gin.Default()

	// Add CORS middleware around every request
	// See https://github.com/rs/cors for full option listing
	router.Use(cors.New(cors.Config{
		AllowOrigins:     laforgeConfig.UI.AllowedOrigins,
		AllowMethods:     []string{"GET", "PUT", "PATCH"},
		AllowHeaders:     []string{"Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
	}))

	port, ok := os.LookupEnv("PORT")

	if !ok {
		port = defaultPort
	}

	gqlHandler := graphqlHandler(client, rdb)

	authGroup := router.Group("/auth")
	authGroup.GET("/login", func(c *gin.Context) {
		c.Redirect(301, "/ui/")
	})
	authGroup.POST("/local/login", auth.LocalLogin(client, laforgeConfig))
	authGroup.GET("/:provider/login", auth.GothicBeginAuth())
	authGroup.GET("/:provider/callback", auth.GothicCallbackHandler(client, laforgeConfig))
	authGroup.GET("/logout", auth.Logout(client, laforgeConfig))

	api := router.Group("/api")
	api.Use(auth.Middleware(client, laforgeConfig))

	api.POST("/query", gqlHandler)
	api.GET("/query", gqlHandler)
	api.GET("/download/:url_id", tempURLHandler(client))
	api.GET("/view_server_logs/:server_task_id", tempServerTaskHandler(client))
	api.GET("/playground", playgroundHandler())
	go router.Run(port)

	// secure server
	certPem, certerr := static.ReadFile(server.CertFile)
	if certerr != nil {
		fmt.Println("File reading error", certerr)
		return
	}
	keyPem, keyerr := static.ReadFile(server.KeyFile)
	if keyerr != nil {
		fmt.Println("File reading error", keyerr)
		return
	}

	cert, tlserr := tls.X509KeyPair(certPem, keyPem)
	if tlserr != nil {
		fmt.Println("File reading error", tlserr)
		return
	}

	creds := credentials.NewServerTLSFromCert(&cert)
	s := grpc.NewServer(grpc.Creds(creds))

	logrus.Infof("Starting Laforge Server on port " + server.Port)

	pb.RegisterLaforgeServer(s, &server.Server{
		Client:                     client,
		UnimplementedLaforgeServer: pb.UnimplementedLaforgeServer{},
		RDB:                        rdb,
	})
	if err := s.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}
